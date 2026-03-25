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
	"github.com/onflux-tech/dockeeper/internal/shared/constants"
	agentpb "github.com/onflux-tech/dockeeper/proto/gen/dockeeper/agent/v1"
)

const agentLogsPort = "50052"

// RegisterContainerLogRoutes adds SSE streaming log endpoints.
func RegisterContainerLogRoutes(se *core.ServeEvent, app core.App) {
	se.Router.GET("/api/dk/containers/{id}/logs/stream", func(e *core.RequestEvent) error {
		return handleLogStream(e, app)
	}).BindFunc(middleware.RequireSuperuser)
}

type logEvent struct {
	Timestamp string `json:"timestamp"`
	Stream    string `json:"stream"`
	Message   string `json:"message"`
}

func handleLogStream(e *core.RequestEvent, app core.App) error {
	containerID := e.Request.PathValue("id")
	follow := e.Request.URL.Query().Get("follow") != "false"
	tail := e.Request.URL.Query().Get("tail")
	if tail == "" {
		tail = "100"
	}

	// Resolve container → server.
	container, err := app.FindRecordById(constants.CollContainers, containerID)
	if err != nil {
		return e.JSON(http.StatusNotFound, map[string]string{"error": "container not found"})
	}

	serverID := container.GetString("server")
	server, err := app.FindRecordById(constants.CollServers, serverID)
	if err != nil {
		return e.JSON(http.StatusNotFound, map[string]string{"error": "server not found"})
	}

	if server.GetString("status") != "online" {
		return e.JSON(http.StatusServiceUnavailable, map[string]string{"error": "server is offline"})
	}

	agentAddr := net.JoinHostPort(server.GetString("ip_address"), agentLogsPort)
	dockerContainerID := container.GetString("container_id")

	// Dial agent gRPC.
	conn, err := grpc.NewClient(agentAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		slog.Error("failed to dial agent for logs", "addr", agentAddr, "error", err)
		return e.JSON(http.StatusBadGateway, map[string]string{"error": "cannot reach agent"})
	}
	defer func() {
		if closeErr := conn.Close(); closeErr != nil {
			slog.Warn("failed to close log connection", "error", closeErr)
		}
	}()

	client := agentpb.NewAgentLogsServiceClient(conn)

	var tailLines uint32
	if _, scanErr := fmt.Sscanf(tail, "%d", &tailLines); scanErr != nil {
		tailLines = 100
	}

	stream, err := client.StreamLogs(e.Request.Context(), &agentpb.StreamLogsRequest{
		ContainerId: dockerContainerID,
		Follow:      follow,
		TailLines:   tailLines,
	})
	if err != nil {
		slog.Error("failed to open log stream", "error", err)
		return e.JSON(http.StatusBadGateway, map[string]string{"error": "failed to stream logs"})
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

	// Disable write deadline for this long-lived SSE connection.
	rc := http.NewResponseController(w)
	_ = rc.SetWriteDeadline(time.Time{})

	// Send initial connection confirmation event.
	_, _ = fmt.Fprintf(w, "event: connected\ndata: {}\n\n")
	flusher.Flush()

	// Start keepalive ticker: sends SSE comments to prevent proxy/server timeout.
	keepalive := time.NewTicker(15 * time.Second)
	defer keepalive.Stop()

	// Channel receives log entries (or errors) from the gRPC stream.
	type recvResult struct {
		entry *agentpb.StreamLogsResponse
		err   error
	}
	ch := make(chan recvResult, 1)

	go func() {
		for {
			entry, recvErr := stream.Recv()
			ch <- recvResult{entry: entry, err: recvErr}
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
				slog.Debug("log stream ended", "error", res.err)
				_, _ = fmt.Fprintf(w, "event: error\ndata: {\"message\":\"stream ended\"}\n\n")
				flusher.Flush()
				return nil
			}

			ev := logEvent{
				Stream:  res.entry.GetStream(),
				Message: res.entry.GetMessage(),
			}
			if res.entry.GetTimestamp() != nil {
				ev.Timestamp = res.entry.GetTimestamp().AsTime().Format("2006-01-02T15:04:05.000Z")
			}

			data, _ := json.Marshal(ev)
			_, _ = fmt.Fprintf(w, "event: log\ndata: %s\n\n", data)
			flusher.Flush()
		}
	}
}
