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

	"github.com/onflux-tech/dockeeper/internal/manager/channel"
	"github.com/onflux-tech/dockeeper/internal/manager/middleware"
	"github.com/onflux-tech/dockeeper/internal/manager/services"
	agentpb "github.com/onflux-tech/dockeeper/proto/gen/dockeeper/agent/v1"
)

// RegisterTerminalRoutes adds the WebSocket terminal proxy endpoint.
func RegisterTerminalRoutes(se *core.ServeEvent, app core.App, termSvc *services.TerminalService, registry *channel.Registry, rbac *middleware.RBAC) {
	se.Router.GET("/api/dk/containers/{id}/terminal", func(e *core.RequestEvent) error {
		return handleTerminal(e, app, termSvc, registry, rbac)
	})
}

// wsControl is the JSON format for text-frame control messages from the client.
type wsControl struct {
	Type string `json:"type"`
	Rows uint32 `json:"rows,omitempty"`
	Cols uint32 `json:"cols,omitempty"`
}

func handleTerminal(e *core.RequestEvent, app core.App, termSvc *services.TerminalService, registry *channel.Registry, rbac *middleware.RBAC) error {
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

	// Get the agent channel from the registry.
	ch, ok := registry.Get(info.ServerID)
	if !ok {
		_ = ws.Close(websocket.StatusInternalError, "agent not connected")
		return nil
	}

	sessionID := info.SessionID
	cmdParts := parseCommand(cmd)

	signal := &agentpb.OpenExecStreamSignal{
		SessionId:   sessionID,
		ContainerId: info.DockerContainerID,
		Command:     cmdParts,
		Tty:         tty,
	}

	streamCh, err := ch.SignalOpenExecStream(ctx, signal)
	if err != nil {
		_ = ws.Close(websocket.StatusInternalError, "failed to signal exec stream")
		return nil
	}

	// Wait for the agent to open the ExecStream.
	var stream agentpb.AgentExecStreamService_OpenExecStreamServer
	select {
	case <-ctx.Done():
		_ = ws.Close(websocket.StatusInternalError, "timeout waiting for exec stream")
		return nil
	case s, ok := <-streamCh:
		if !ok {
			_ = ws.Close(websocket.StatusInternalError, "exec stream channel closed")
			return nil
		}
		stream = s
	}

	slog.Info("terminal session started",
		"session_id", info.SessionID,
		"container_id", containerID,
	)

	// Bridge: 2 goroutines.
	var wg sync.WaitGroup
	wg.Add(2)

	// ExecStream → WS (agent output → browser).
	go func() {
		defer wg.Done()
		bridgeExecStreamToWS(ctx, stream, ws)
	}()

	// WS → ExecStream (browser input → agent).
	go func() {
		defer wg.Done()
		bridgeWSToExecStream(ctx, ws, stream, sessionID)
	}()

	wg.Wait()

	slog.Info("terminal session ended",
		"session_id", info.SessionID,
		"container_id", containerID,
	)

	return nil
}

// bridgeExecStreamToWS reads OpenExecStreamRequest from the agent and writes to WebSocket.
func bridgeExecStreamToWS(ctx context.Context, stream agentpb.AgentExecStreamService_OpenExecStreamServer, ws *websocket.Conn) {
	for {
		msg, err := stream.Recv()
		if err != nil {
			if err != io.EOF {
				slog.Debug("exec stream recv ended", "error", err)
			}
			return
		}

		if data := msg.GetStdout(); len(data) > 0 {
			if writeErr := ws.Write(ctx, websocket.MessageBinary, data); writeErr != nil {
				return
			}
		}

		if exit := msg.GetExit(); exit != nil {
			exitMsg, _ := json.Marshal(map[string]any{
				"type":      "exit",
				"exit_code": exit.GetExitCode(),
			})
			_ = ws.Write(ctx, websocket.MessageText, exitMsg)
			return
		}
	}
}

// bridgeWSToExecStream reads WebSocket messages and sends to the ExecStream.
func bridgeWSToExecStream(ctx context.Context, ws *websocket.Conn, stream agentpb.AgentExecStreamService_OpenExecStreamServer, sessionID string) {
	for {
		msgType, data, err := ws.Read(ctx)
		if err != nil {
			return
		}

		switch msgType {
		case websocket.MessageBinary:
			// Send stdin via the stream's Send (manager→agent direction = OpenExecStreamResponse).
			if sendErr := stream.Send(&agentpb.OpenExecStreamResponse{
				SessionId: sessionID,
				Payload:   &agentpb.OpenExecStreamResponse_Stdin{Stdin: data},
			}); sendErr != nil {
				return
			}

		case websocket.MessageText:
			var ctrl wsControl
			if jsonErr := json.Unmarshal(data, &ctrl); jsonErr != nil {
				continue
			}
			if ctrl.Type == "resize" && ctrl.Rows > 0 && ctrl.Cols > 0 {
				if sendErr := stream.Send(&agentpb.OpenExecStreamResponse{
					SessionId: sessionID,
					Payload: &agentpb.OpenExecStreamResponse_Resize{
						Resize: &agentpb.ExecStreamResize{Rows: ctrl.Rows, Cols: ctrl.Cols},
					},
				}); sendErr != nil {
					return
				}
			}
			if ctrl.Type == "close" {
				_ = stream.Send(&agentpb.OpenExecStreamResponse{
					SessionId: sessionID,
					Payload:   &agentpb.OpenExecStreamResponse_Close{Close: &agentpb.ExecStreamClose{}},
				})
				return
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
