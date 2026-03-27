package routes

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"

	"github.com/coder/websocket"
	"github.com/pocketbase/pocketbase/core"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/onflux-tech/dockeeper/internal/manager/middleware"
	"github.com/onflux-tech/dockeeper/internal/manager/services"
	agentpb "github.com/onflux-tech/dockeeper/proto/gen/dockeeper/agent/v1"
)

// RegisterTerminalRoutes adds the WebSocket terminal proxy endpoint.
func RegisterTerminalRoutes(se *core.ServeEvent, app core.App, termSvc *services.TerminalService, rbac *middleware.RBAC) {
	se.Router.GET("/api/dk/containers/{id}/terminal", func(e *core.RequestEvent) error {
		return handleTerminal(e, app, termSvc, rbac)
	})
}

// wsControl is the JSON format for text-frame control messages from the client.
type wsControl struct {
	Type string `json:"type"`
	Rows uint32 `json:"rows,omitempty"`
	Cols uint32 `json:"cols,omitempty"`
}

func handleTerminal(e *core.RequestEvent, app core.App, termSvc *services.TerminalService, rbac *middleware.RBAC) error {
	containerID := e.Request.PathValue("id")
	if containerID == "" {
		return e.JSON(http.StatusBadRequest, map[string]string{"error": "container ID is required"})
	}

	// Auth: extract token from query param (WS can't set headers).
	token := e.Request.URL.Query().Get("token")
	if token == "" {
		return e.JSON(http.StatusUnauthorized, map[string]string{"error": "token is required"})
	}

	authRecord, err := app.FindAuthRecordByToken(token, core.TokenTypeAuth)
	if err != nil {
		return e.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid token"})
	}

	// RBAC: superuser bypasses, others need containers.manage capability.
	if !authRecord.IsSuperuser() && !rbac.HasCapability(authRecord.Id, "containers.manage") {
		return e.JSON(http.StatusForbidden, map[string]string{"error": "insufficient permissions: containers.manage required"})
	}

	userID := authRecord.Id
	clientIP := e.RealIP()

	// Parse optional query params.
	cmd := e.Request.URL.Query().Get("cmd")
	if cmd == "" {
		cmd = "/bin/sh"
	}
	tty := e.Request.URL.Query().Get("tty") != "false"

	// Open session (resolve container, create record, audit).
	info, err := termSvc.OpenSession(containerID, userID, cmd, clientIP)
	if err != nil {
		return mapServiceError(e, err)
	}

	// Upgrade to WebSocket.
	ws, err := websocket.Accept(e.Response, e.Request, &websocket.AcceptOptions{
		InsecureSkipVerify: true, // allow any origin for dev; tighten in production
	})
	if err != nil {
		slog.Error("websocket accept failed", "error", err)
		termSvc.CloseSession(info.SessionID, userID, containerID, clientIP)
		return nil // error already written to response by Accept
	}
	defer func() {
		_ = ws.Close(websocket.StatusNormalClosure, "session ended")
		termSvc.CloseSession(info.SessionID, userID, containerID, clientIP)
	}()

	ctx := e.Request.Context()

	// Dial agent gRPC.
	conn, err := grpc.NewClient(info.AgentAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		slog.Error("failed to dial agent for exec", "addr", info.AgentAddr, "error", err)
		_ = ws.Close(websocket.StatusInternalError, "cannot reach agent")
		return nil
	}
	defer func() {
		if closeErr := conn.Close(); closeErr != nil {
			slog.Warn("failed to close exec gRPC connection", "error", closeErr)
		}
	}()

	client := agentpb.NewAgentExecServiceClient(conn)
	stream, err := client.Exec(ctx)
	if err != nil {
		slog.Error("failed to open exec stream", "error", err)
		_ = ws.Close(websocket.StatusInternalError, "failed to start exec")
		return nil
	}

	// Send ExecStart as the first gRPC message.
	cmdParts := parseCommand(cmd)
	if err := stream.Send(&agentpb.ExecRequest{
		Input: &agentpb.ExecRequest_Start{
			Start: &agentpb.ExecStart{
				ContainerId: info.DockerContainerID,
				Command:     cmdParts,
				Tty:         tty,
			},
		},
	}); err != nil {
		slog.Error("failed to send exec start", "error", err)
		_ = ws.Close(websocket.StatusInternalError, "exec start failed")
		return nil
	}

	slog.Info("terminal session started",
		"session_id", info.SessionID,
		"container_id", containerID,
	)

	// Bridge: 2 goroutines.
	var wg sync.WaitGroup
	wg.Add(2)

	// gRPC → WS (agent output → browser).
	go func() {
		defer wg.Done()
		bridgeGRPCToWS(ctx, stream, ws)
	}()

	// WS → gRPC (browser input → agent).
	go func() {
		defer wg.Done()
		bridgeWSToGRPC(ctx, ws, stream)
	}()

	wg.Wait()

	slog.Info("terminal session ended",
		"session_id", info.SessionID,
		"container_id", containerID,
	)

	return nil
}

// bridgeGRPCToWS reads ExecResponse from the gRPC stream and writes to WebSocket.
func bridgeGRPCToWS(ctx context.Context, stream agentpb.AgentExecService_ExecClient, ws *websocket.Conn) {
	for {
		resp, err := stream.Recv()
		if err != nil {
			if err != io.EOF {
				slog.Debug("exec stream recv ended", "error", err)
			}
			return
		}

		// Send stdout/stderr as binary frames.
		if data := resp.GetStdout(); len(data) > 0 {
			if writeErr := ws.Write(ctx, websocket.MessageBinary, data); writeErr != nil {
				return
			}
		}
		if data := resp.GetStderr(); len(data) > 0 {
			if writeErr := ws.Write(ctx, websocket.MessageBinary, data); writeErr != nil {
				return
			}
		}

		// If exit_code is present, send as text JSON and close.
		if resp.ExitCode != nil {
			msg, _ := json.Marshal(map[string]any{
				"type":      "exit",
				"exit_code": resp.GetExitCode(),
			})
			_ = ws.Write(ctx, websocket.MessageText, msg)
			return
		}
	}
}

// bridgeWSToGRPC reads WebSocket messages and sends to the gRPC stream.
func bridgeWSToGRPC(ctx context.Context, ws *websocket.Conn, stream agentpb.AgentExecService_ExecClient) {
	for {
		msgType, data, err := ws.Read(ctx)
		if err != nil {
			// Client disconnected or context cancelled.
			_ = stream.CloseSend()
			return
		}

		switch msgType {
		case websocket.MessageBinary:
			// Raw stdin data.
			if sendErr := stream.Send(&agentpb.ExecRequest{
				Input: &agentpb.ExecRequest_StdinData{
					StdinData: data,
				},
			}); sendErr != nil {
				return
			}

		case websocket.MessageText:
			// JSON control message (resize).
			var ctrl wsControl
			if jsonErr := json.Unmarshal(data, &ctrl); jsonErr != nil {
				continue
			}
			if ctrl.Type == "resize" && ctrl.Rows > 0 && ctrl.Cols > 0 {
				if sendErr := stream.Send(&agentpb.ExecRequest{
					Input: &agentpb.ExecRequest_Resize{
						Resize: &agentpb.ExecResize{
							Rows: ctrl.Rows,
							Cols: ctrl.Cols,
						},
					},
				}); sendErr != nil {
					return
				}
			}
		}
	}
}

// parseCommand splits a simple command string into parts.
func parseCommand(cmd string) []string {
	parts := strings.Fields(cmd)
	if len(parts) == 0 {
		return []string{"/bin/sh"}
	}
	return parts
}
