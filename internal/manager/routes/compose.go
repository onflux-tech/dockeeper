package routes

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/pocketbase/pocketbase/core"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/onflux-tech/dockeeper/internal/manager/middleware"
	"github.com/onflux-tech/dockeeper/internal/manager/services"
	"github.com/onflux-tech/dockeeper/internal/shared/constants"
	"github.com/onflux-tech/dockeeper/internal/shared/security"
	agentpb "github.com/onflux-tech/dockeeper/proto/gen/dockeeper/agent/v1"
)

const agentComposePort = "50052"

// RegisterComposeRoutes adds compose stack management endpoints.
func RegisterComposeRoutes(se *core.ServeEvent, app core.App, svc *services.ComposeService, rbac *middleware.RBAC) {
	compose := se.Router.Group("/api/dk/compose")

	compose.GET("", func(e *core.RequestEvent) error {
		return handleListStacks(e, app)
	}).BindFunc(rbac.RequireCapability(constants.CapComposeView))

	compose.GET("/{id}", func(e *core.RequestEvent) error {
		return handleGetStack(e, app)
	}).BindFunc(rbac.RequireCapability(constants.CapComposeView))

	compose.POST("", func(e *core.RequestEvent) error {
		return handleCreateStack(e, svc)
	}).BindFunc(rbac.RequireCapability(constants.CapComposeManage))

	compose.PUT("/{id}", func(e *core.RequestEvent) error {
		return handleUpdateStack(e, svc)
	}).BindFunc(rbac.RequireCapability(constants.CapComposeManage))

	compose.DELETE("/{id}", func(e *core.RequestEvent) error {
		return handleDeleteStack(e, svc)
	}).BindFunc(rbac.RequireCapability(constants.CapComposeManage))

	// SSE streaming endpoints for compose operations.
	compose.GET("/{id}/up", func(e *core.RequestEvent) error {
		return handleComposeSSE(e, svc, "up")
	}).BindFunc(rbac.RequireCapability(constants.CapComposeManage))

	compose.GET("/{id}/down", func(e *core.RequestEvent) error {
		return handleComposeSSE(e, svc, "down")
	}).BindFunc(rbac.RequireCapability(constants.CapComposeManage))

	compose.GET("/{id}/pull", func(e *core.RequestEvent) error {
		return handleComposeSSE(e, svc, "pull")
	}).BindFunc(rbac.RequireCapability(constants.CapComposeManage))
}

func handleListStacks(e *core.RequestEvent, app core.App) error {
	serverID := e.Request.URL.Query().Get("server_id")

	filter := ""
	if serverID != "" {
		filter = fmt.Sprintf("server = '%s'", security.SanitizeFilterParam(serverID))
	}

	records, err := app.FindRecordsByFilter(
		constants.CollComposeStacks,
		filter,
		"-created",
		0, 0,
	)
	if err != nil {
		return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to list stacks"})
	}

	type stackEntry struct {
		ID            string `json:"id"`
		Server        string `json:"server"`
		Name          string `json:"name"`
		Status        string `json:"status"`
		ServicesCount int    `json:"services_count"`
		DeployedAt    string `json:"deployed_at"`
		Created       string `json:"created"`
	}

	items := make([]stackEntry, 0, len(records))
	for _, r := range records {
		items = append(items, stackEntry{
			ID:            r.Id,
			Server:        r.GetString("server"),
			Name:          r.GetString("name"),
			Status:        r.GetString("status"),
			ServicesCount: int(r.GetFloat("services_count")),
			DeployedAt:    r.GetString("deployed_at"),
			Created:       r.GetString("created"),
		})
	}

	return e.JSON(http.StatusOK, items)
}

func handleGetStack(e *core.RequestEvent, app core.App) error {
	stackID := e.Request.PathValue("id")

	record, err := app.FindRecordById(constants.CollComposeStacks, stackID)
	if err != nil {
		return e.JSON(http.StatusNotFound, map[string]string{"error": "stack not found"})
	}

	return e.JSON(http.StatusOK, map[string]any{
		"id":             record.Id,
		"server":         record.GetString("server"),
		"name":           record.GetString("name"),
		"yaml_content":   record.GetString("yaml_content"),
		"status":         record.GetString("status"),
		"services_count": record.GetFloat("services_count"),
		"deployed_at":    record.GetString("deployed_at"),
		"created":        record.GetString("created"),
		"updated":        record.GetString("updated"),
	})
}

func handleCreateStack(e *core.RequestEvent, svc *services.ComposeService) error {
	var body struct {
		ServerID    string `json:"server_id"`
		Name        string `json:"name"`
		YAMLContent string `json:"yaml_content"`
	}
	if err := e.BindBody(&body); err != nil {
		return e.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	if body.ServerID == "" || body.Name == "" || body.YAMLContent == "" {
		return e.JSON(http.StatusBadRequest, map[string]string{"error": "server_id, name, and yaml_content are required"})
	}

	if err := security.ValidateContainerName(body.Name); err != nil {
		return e.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	record, err := svc.CreateStack(body.ServerID, body.Name, body.YAMLContent)
	if err != nil {
		return mapComposeServiceError(e, err)
	}

	return e.JSON(http.StatusCreated, map[string]any{
		"id":     record.Id,
		"server": record.GetString("server"),
		"name":   record.GetString("name"),
		"status": record.GetString("status"),
	})
}

func handleUpdateStack(e *core.RequestEvent, svc *services.ComposeService) error {
	stackID := e.Request.PathValue("id")

	var body struct {
		YAMLContent string `json:"yaml_content"`
	}
	if err := e.BindBody(&body); err != nil || body.YAMLContent == "" {
		return e.JSON(http.StatusBadRequest, map[string]string{"error": "yaml_content is required"})
	}

	record, err := svc.UpdateStack(stackID, body.YAMLContent)
	if err != nil {
		return mapComposeServiceError(e, err)
	}

	return e.JSON(http.StatusOK, map[string]any{
		"id":     record.Id,
		"server": record.GetString("server"),
		"name":   record.GetString("name"),
		"status": record.GetString("status"),
	})
}

func handleDeleteStack(e *core.RequestEvent, svc *services.ComposeService) error {
	stackID := e.Request.PathValue("id")

	userID := ""
	if e.Auth != nil {
		userID = e.Auth.Id
	}

	if err := svc.DeleteStack(e.Request.Context(), stackID, userID, e.RealIP()); err != nil {
		return mapComposeServiceError(e, err)
	}

	return e.JSON(http.StatusOK, map[string]string{"message": "stack deleted"})
}

// composeOutputEvent is the SSE data for compose operations.
type composeOutputEvent struct {
	Line     string `json:"line,omitempty"`
	Stream   string `json:"stream,omitempty"`
	Done     bool   `json:"done,omitempty"`
	ExitCode int32  `json:"exit_code,omitempty"`
	Error    string `json:"error,omitempty"`
}

// handleComposeSSE streams compose operation output via SSE.
func handleComposeSSE(e *core.RequestEvent, svc *services.ComposeService, operation string) error {
	stackID := e.Request.PathValue("id")

	record, info, err := svc.ResolveStack(stackID)
	if err != nil {
		return mapComposeServiceError(e, err)
	}

	// Dial agent gRPC.
	addr := net.JoinHostPort(info.AgentAddr, agentComposePort)
	conn, err := grpc.NewClient(addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		slog.Error("failed to dial agent for compose", "addr", addr, "error", err)
		return e.JSON(http.StatusBadGateway, map[string]string{"error": "cannot reach agent"})
	}
	defer func() {
		if closeErr := conn.Close(); closeErr != nil {
			slog.Warn("failed to close agent connection", "addr", addr, "error", closeErr)
		}
	}()

	client := agentpb.NewAgentComposeServiceClient(conn)

	req := &agentpb.ComposeRequest{
		ProjectName: info.StackName,
	}

	var stream agentpb.AgentComposeService_ComposeUpClient
	switch operation {
	case "up":
		stream, err = client.ComposeUp(e.Request.Context(), req)
	case "down":
		stream, err = client.ComposeDown(e.Request.Context(), req)
	case "pull":
		stream, err = client.ComposePull(e.Request.Context(), req)
	default:
		return e.JSON(http.StatusBadRequest, map[string]string{"error": "invalid operation"})
	}
	if err != nil {
		slog.Error("failed to open compose stream", "operation", operation, "error", err)
		return e.JSON(http.StatusBadGateway, map[string]string{"error": "failed to start compose " + operation})
	}

	// Set SSE headers.
	w := e.Response
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)

	flusher, ok := w.(http.Flusher)
	if !ok {
		return fmt.Errorf("response writer does not support flushing")
	}

	rc := http.NewResponseController(w)
	_ = rc.SetWriteDeadline(time.Time{})

	_, _ = fmt.Fprintf(w, "event: connected\ndata: {}\n\n")
	flusher.Flush()

	keepalive := time.NewTicker(15 * time.Second)
	defer keepalive.Stop()

	type recvResult struct {
		msg *agentpb.ComposeOutput
		err error
	}
	ch := make(chan recvResult, 1)

	go func() {
		for {
			msg, recvErr := stream.Recv()
			ch <- recvResult{msg: msg, err: recvErr}
			if recvErr != nil {
				return
			}
		}
	}()

	for {
		select {
		case <-e.Request.Context().Done():
			return nil

		case <-keepalive.C:
			_, _ = fmt.Fprintf(w, ": keepalive\n\n")
			flusher.Flush()

		case res := <-ch:
			if res.err != nil {
				if res.err == io.EOF {
					_, _ = fmt.Fprintf(w, "event: done\ndata: {}\n\n")
					flusher.Flush()
					return nil
				}
				data, _ := json.Marshal(composeOutputEvent{Error: res.err.Error(), Done: true})
				_, _ = fmt.Fprintf(w, "event: error\ndata: %s\n\n", data)
				flusher.Flush()
				return nil
			}

			ev := composeOutputEvent{
				Line:     res.msg.GetLine(),
				Stream:   res.msg.GetStream(),
				Done:     res.msg.GetDone(),
				ExitCode: res.msg.GetExitCode(),
				Error:    res.msg.GetError(),
			}

			data, _ := json.Marshal(ev)
			_, _ = fmt.Fprintf(w, "event: output\ndata: %s\n\n", data)
			flusher.Flush()

			if ev.Done {
				// Update stack status based on operation + exit code.
				if ev.ExitCode == 0 {
					switch operation {
					case "up":
						svc.MarkStackStatus(record.Id, "running", 0)
					case "down":
						svc.MarkStackStatus(record.Id, "stopped", 0)
					}
				}
				return nil
			}
		}
	}
}

func mapComposeServiceError(e *core.RequestEvent, err error) error {
	switch err {
	case services.ErrServerNotFound:
		return e.JSON(http.StatusNotFound, map[string]string{"error": "server not found"})
	case services.ErrServerOffline:
		return e.JSON(http.StatusServiceUnavailable, map[string]string{"error": "server is offline"})
	default:
		if err.Error() == "stack not found" {
			return e.JSON(http.StatusNotFound, map[string]string{"error": "stack not found"})
		}
		return e.JSON(http.StatusBadGateway, map[string]string{"error": err.Error()})
	}
}
