package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/pocketbase/pocketbase/core"

	"github.com/onflux-tech/dockeeper/internal/manager/grpc_pool"
	"github.com/onflux-tech/dockeeper/internal/manager/store"
	"github.com/onflux-tech/dockeeper/internal/shared/constants"
	agentpb "github.com/onflux-tech/dockeeper/proto/gen/dockeeper/agent/v1"
)

// ComposeService handles compose stack operations, managing PocketBase records
// and dispatching compose commands to agents via gRPC.
type ComposeService struct {
	app   core.App
	audit *store.AuditStore
	pool  *grpc_pool.Pool
}

// NewComposeService creates a ComposeService.
func NewComposeService(app core.App, audit *store.AuditStore, pool *grpc_pool.Pool) *ComposeService {
	return &ComposeService{app: app, audit: audit, pool: pool}
}

// StackInfo holds resolved stack/server information for agent operations.
type StackInfo struct {
	AgentAddr  string
	ServerID   string
	StackName  string
	WorkingDir string
}

// CreateStack creates a new compose_stacks record in PocketBase.
func (s *ComposeService) CreateStack(serverID, name, yamlContent string) (*core.Record, error) {
	agentIP, err := s.resolveServer(serverID)
	if err != nil {
		return nil, err
	}
	_ = agentIP // validated server is online

	collection, err := s.app.FindCollectionByNameOrId(constants.CollComposeStacks)
	if err != nil {
		return nil, fmt.Errorf("find compose_stacks collection: %w", err)
	}

	record := core.NewRecord(collection)
	record.Set("server", serverID)
	record.Set("name", name)
	record.Set("yaml_content", yamlContent)
	record.Set("status", "stopped")
	record.Set("services_count", 0)

	if err := s.app.Save(record); err != nil {
		return nil, fmt.Errorf("save compose stack: %w", err)
	}

	return record, nil
}

// UpdateStack updates the YAML content of an existing compose stack.
func (s *ComposeService) UpdateStack(stackID, yamlContent string) (*core.Record, error) {
	record, err := s.app.FindRecordById(constants.CollComposeStacks, stackID)
	if err != nil {
		return nil, fmt.Errorf("stack not found")
	}

	record.Set("yaml_content", yamlContent)
	record.Set("updated", time.Now().UTC())

	if err := s.app.Save(record); err != nil {
		return nil, fmt.Errorf("update compose stack: %w", err)
	}

	return record, nil
}

// ResolveStack finds a compose stack record and its server info.
func (s *ComposeService) ResolveStack(stackID string) (*core.Record, *StackInfo, error) {
	record, err := s.app.FindRecordById(constants.CollComposeStacks, stackID)
	if err != nil {
		return nil, nil, fmt.Errorf("stack not found")
	}

	serverID := record.GetString("server")
	agentIP, err := s.resolveServer(serverID)
	if err != nil {
		return nil, nil, err
	}

	return record, &StackInfo{
		AgentAddr: agentIP,
		ServerID:  serverID,
		StackName: record.GetString("name"),
	}, nil
}

// DeleteStack removes a compose stack from PocketBase after running compose down.
func (s *ComposeService) DeleteStack(ctx context.Context, stackID, userID, ip string) error {
	record, info, err := s.ResolveStack(stackID)
	if err != nil {
		return err
	}

	// Run compose down synchronously on agent.
	if downErr := s.runComposeDown(ctx, info); downErr != nil {
		slog.Warn("compose down failed during delete, proceeding", "error", downErr)
	}

	// Delete PocketBase record.
	if err := s.app.Delete(record); err != nil {
		return fmt.Errorf("delete compose stack: %w", err)
	}

	s.audit.Log("compose.delete", userID, "compose_stack", stackID, ip, map[string]any{
		"stack_name": info.StackName,
		"server_id":  info.ServerID,
	})

	return nil
}

// MarkStackStatus updates the status field of a compose stack.
func (s *ComposeService) MarkStackStatus(stackID, status string, servicesCount int) {
	record, err := s.app.FindRecordById(constants.CollComposeStacks, stackID)
	if err != nil {
		return
	}
	record.Set("status", status)
	record.Set("services_count", servicesCount)
	if status == "running" {
		record.Set("deployed_at", time.Now().UTC())
	}
	if saveErr := s.app.Save(record); saveErr != nil {
		slog.Warn("failed to update compose stack status", "error", saveErr)
	}
}

// ImportedStack holds info about a single imported stack.
type ImportedStack struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	ServicesCount int      `json:"services_count"`
	Services      []string `json:"services"`
}

// ImportStacks scans containers for a server and imports compose stacks
// based on Docker Compose labels.
func (s *ComposeService) ImportStacks(serverID string) ([]ImportedStack, error) {
	// Validate server exists and get agent address.
	agentAddr, err := s.resolveServer(serverID)
	if err != nil {
		return nil, err
	}

	// Fetch all containers for this server.
	containers, err := s.app.FindRecordsByFilter(
		constants.CollContainers,
		"server = {:sid}",
		"", 0, 0,
		map[string]any{"sid": serverID},
	)
	if err != nil {
		return nil, fmt.Errorf("fetch containers: %w", err)
	}

	// Group containers by compose project label, extracting compose metadata from labels.
	type svcInfo struct {
		recordID string
		service  string
	}
	type projectInfo struct {
		svcs       []svcInfo
		workingDir string
		files      []string
	}
	projects := make(map[string]*projectInfo)
	for _, c := range containers {
		project := c.GetString("compose_project")
		service := c.GetString("compose_service")
		if project == "" {
			continue
		}
		pi, ok := projects[project]
		if !ok {
			pi = &projectInfo{}
			// Extract compose working_dir and config_files from first container's labels.
			labelsRaw := c.GetString("labels")
			if labelsRaw != "" {
				var labels map[string]string
				if err := json.Unmarshal([]byte(labelsRaw), &labels); err == nil {
					pi.workingDir = labels["com.docker.compose.project.working_dir"]
					if cf := labels["com.docker.compose.project.config_files"]; cf != "" {
						pi.files = strings.Split(cf, ",")
					}
				}
			}
			projects[project] = pi
		}
		pi.svcs = append(pi.svcs, svcInfo{
			recordID: c.Id,
			service:  service,
		})
	}

	if len(projects) == 0 {
		return []ImportedStack{}, nil
	}

	stackCol, err := s.app.FindCollectionByNameOrId(constants.CollComposeStacks)
	if err != nil {
		return nil, fmt.Errorf("find compose_stacks collection: %w", err)
	}

	// Get gRPC connection to fetch YAML content from agent.
	var composeClient agentpb.AgentComposeServiceClient
	if conn, connErr := s.pool.Get(agentAddr); connErr == nil {
		composeClient = agentpb.NewAgentComposeServiceClient(conn)
	}

	var imported []ImportedStack

	for projectName, pi := range projects {
		// Upsert compose_stacks record.
		existing, _ := s.app.FindRecordsByFilter(
			constants.CollComposeStacks,
			"server = {:sid} && name = {:name}",
			"", 1, 0,
			map[string]any{"sid": serverID, "name": projectName},
		)

		var rec *core.Record
		if len(existing) > 0 {
			rec = existing[0]
		} else {
			rec = core.NewRecord(stackCol)
			rec.Set("server", serverID)
			rec.Set("name", projectName)
			rec.Set("status", "running")
		}

		rec.Set("source", "imported")
		rec.Set("services_count", len(pi.svcs))

		// Fetch compose YAML from agent via `docker compose config`.
		if composeClient != nil {
			ctx, cancel := context.WithTimeout(context.Background(), commandTimeout)
			resp, configErr := composeClient.ComposeConfig(ctx, &agentpb.ComposeConfigRequest{
				ProjectName: projectName,
				WorkingDir:  pi.workingDir,
				Files:       pi.files,
			})
			cancel()
			if configErr == nil && resp.GetYamlContent() != "" {
				rec.Set("yaml_content", resp.GetYamlContent())
			} else if configErr != nil {
				slog.Warn("import stack: compose config failed", "project", projectName, "error", configErr)
			}
		}

		if err := s.app.Save(rec); err != nil {
			slog.Error("import stack: save failed", "project", projectName, "error", err)
			continue
		}

		// Link containers to this stack.
		serviceNames := make([]string, 0, len(pi.svcs))
		for _, si := range pi.svcs {
			serviceNames = append(serviceNames, si.service)
			cRec, findErr := s.app.FindRecordById(constants.CollContainers, si.recordID)
			if findErr != nil {
				continue
			}
			if cRec.GetString("compose_stack") != rec.Id {
				cRec.Set("compose_stack", rec.Id)
				_ = s.app.Save(cRec)
			}
		}

		imported = append(imported, ImportedStack{
			ID:            rec.Id,
			Name:          projectName,
			ServicesCount: len(pi.svcs),
			Services:      serviceNames,
		})
	}

	return imported, nil
}

func (s *ComposeService) resolveServer(serverID string) (string, error) {
	server, err := s.app.FindRecordById(constants.CollServers, serverID)
	if err != nil {
		return "", ErrServerNotFound
	}
	if server.GetString("status") != "online" {
		return "", ErrServerOffline
	}
	return server.GetString("ip_address"), nil
}

// runComposeDown executes compose down on the agent synchronously.
func (s *ComposeService) runComposeDown(ctx context.Context, info *StackInfo) error {
	conn, err := s.pool.Get(info.AgentAddr)
	if err != nil {
		return fmt.Errorf("dial agent: %w", err)
	}

	client := agentpb.NewAgentComposeServiceClient(conn)

	cmdCtx, cancel := context.WithTimeout(ctx, 2*commandTimeout)
	defer cancel()

	stream, err := client.ComposeDown(cmdCtx, &agentpb.ComposeRequest{
		ProjectName: info.StackName,
	})
	if err != nil {
		return fmt.Errorf("compose down: %w", err)
	}

	// Drain the stream.
	for {
		msg, recvErr := stream.Recv()
		if recvErr != nil {
			break
		}
		if msg.GetDone() {
			if msg.GetExitCode() != 0 {
				return fmt.Errorf("compose down exited with code %d", msg.GetExitCode())
			}
			break
		}
	}

	return nil
}
