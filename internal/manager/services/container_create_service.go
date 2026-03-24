package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/pocketbase/pocketbase/core"

	"github.com/jkdesenvolvedor/dockeeper/internal/manager/store"
	"github.com/jkdesenvolvedor/dockeeper/internal/shared/constants"
	"github.com/jkdesenvolvedor/dockeeper/internal/shared/models"
	agentpb "github.com/jkdesenvolvedor/dockeeper/proto/gen/dockeeper/agent/v1"
)

// ContainerCreateService handles container creation requests.
type ContainerCreateService struct {
	app        core.App
	dispatcher CommandDispatcher
	audit      *store.AuditStore
}

// NewContainerCreateService creates a ContainerCreateService.
func NewContainerCreateService(app core.App, dispatcher CommandDispatcher, audit *store.AuditStore) *ContainerCreateService {
	return &ContainerCreateService{
		app:        app,
		dispatcher: dispatcher,
		audit:      audit,
	}
}

// CreateResult is the outcome of a container creation.
type CreateResult struct {
	Success bool   `json:"success"`
	Output  string `json:"output,omitempty"`
	Error   string `json:"error,omitempty"`
}

// CreateContainer validates the spec, dispatches the CREATE command to the
// agent, and logs the action.
func (s *ContainerCreateService) CreateContainer(
	ctx context.Context,
	spec *models.ContainerSpec,
	userID, ip string,
) (*CreateResult, error) {
	// 1. Validate spec.
	if errs := spec.Validate(); len(errs) > 0 {
		return nil, &ValidationError{Errors: errs}
	}

	// 2. Find server.
	server, err := s.app.FindRecordById(constants.CollServers, spec.ServerID)
	if err != nil {
		return nil, ErrServerNotFound
	}

	// 3. Check server is online.
	if server.GetString("status") != "online" {
		return nil, ErrServerOffline
	}

	// 4. Check for duplicate container name on this server.
	nameFilter := fmt.Sprintf("server = %q && name = %q", spec.ServerID, spec.Name)
	existing, _ := s.app.FindFirstRecordByFilter(constants.CollContainers, nameFilter)
	if existing != nil {
		return nil, &ConflictError{Message: fmt.Sprintf("container name %q already exists on this server", spec.Name)}
	}

	// 5. Serialize spec for agent.
	specJSON, err := json.Marshal(spec)
	if err != nil {
		return nil, fmt.Errorf("marshal container spec: %w", err)
	}

	// 6. Build gRPC request.
	agentAddr := server.GetString("ip_address")
	req := &agentpb.ExecuteCommandRequest{
		ServerId: spec.ServerID,
		Type:     agentpb.CommandType_COMMAND_TYPE_CREATE,
		Params: map[string]string{
			"spec": string(specJSON),
		},
	}

	// 7. Dispatch to agent.
	resp, err := s.dispatcher.Dispatch(ctx, agentAddr, req)

	// 8. Audit log.
	details := map[string]any{
		"command":   "CREATE",
		"name":      spec.Name,
		"image":     spec.Image,
		"server_id": spec.ServerID,
	}

	if err != nil {
		details["error"] = err.Error()
		s.audit.Log("container.create.failed", userID, "server", spec.ServerID, ip, details)
		return nil, fmt.Errorf("dispatch create command: %w", err)
	}

	details["success"] = resp.GetSuccess()
	if resp.GetError() != "" {
		details["agent_error"] = resp.GetError()
	}

	s.audit.Log("container.create", userID, "server", spec.ServerID, ip, details)

	slog.Info("container created",
		"name", spec.Name,
		"image", spec.Image,
		"server", spec.ServerID,
		"success", resp.GetSuccess(),
	)

	return &CreateResult{
		Success: resp.GetSuccess(),
		Output:  resp.GetOutput(),
		Error:   resp.GetError(),
	}, nil
}

// ValidationError holds multiple validation errors.
type ValidationError struct {
	Errors []string `json:"errors"`
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation failed: %v", e.Errors)
}

// ConflictError indicates a resource conflict (e.g. duplicate name).
type ConflictError struct {
	Message string `json:"message"`
}

func (e *ConflictError) Error() string {
	return e.Message
}
