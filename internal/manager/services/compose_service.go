package services

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/pocketbase/pocketbase/core"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/jkdesenvolvedor/dockeeper/internal/manager/store"
	"github.com/jkdesenvolvedor/dockeeper/internal/shared/constants"
	agentpb "github.com/jkdesenvolvedor/dockeeper/proto/gen/dockeeper/agent/v1"
)

// ComposeService handles compose stack operations, managing PocketBase records
// and dispatching compose commands to agents via gRPC.
type ComposeService struct {
	app   core.App
	audit *store.AuditStore
}

// NewComposeService creates a ComposeService.
func NewComposeService(app core.App, audit *store.AuditStore) *ComposeService {
	return &ComposeService{app: app, audit: audit}
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
	addr := info.AgentAddr + ":" + agentCommandPort
	conn, err := grpc.NewClient(addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return fmt.Errorf("dial agent: %w", err)
	}
	defer closeConn(conn, addr)

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
