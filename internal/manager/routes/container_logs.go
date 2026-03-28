package routes

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/pocketbase/pocketbase/core"

	"github.com/onflux-tech/dockeeper/internal/manager/channel"
	"github.com/onflux-tech/dockeeper/internal/manager/middleware"
	"github.com/onflux-tech/dockeeper/internal/shared/constants"
	agentpb "github.com/onflux-tech/dockeeper/proto/gen/dockeeper/agent/v1"
)

// RegisterContainerLogRoutes adds SSE streaming log endpoints.
func RegisterContainerLogRoutes(se *core.ServeEvent, app core.App, registry *channel.Registry, rbac *middleware.RBAC) {
	se.Router.GET("/api/dk/containers/{id}/logs/stream", func(e *core.RequestEvent) error {
		return handleLogStream(e, app, registry)
	}).BindFunc(rbac.RequireCapability(constants.CapContainersView))

	se.Router.GET("/api/dk/containers/multi-logs", func(e *core.RequestEvent) error {
		return handleMultiLogStream(e, app, registry)
	}).BindFunc(rbac.RequireCapability(constants.CapContainersView))
}

type logEvent struct {
	Timestamp string `json:"timestamp"`
	Stream    string `json:"stream"`
	Message   string `json:"message"`
}

func handleLogStream(e *core.RequestEvent, app core.App, registry *channel.Registry) error {
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

	dockerContainerID := container.GetString("container_id")

	// Get channel from registry.
	ch, okCh := registry.Get(serverID)
	if !okCh {
		return e.JSON(http.StatusBadGateway, map[string]string{"error": "agent not connected"})
	}

	var tailLines uint32
	if _, scanErr := fmt.Sscanf(tail, "%d", &tailLines); scanErr != nil {
		tailLines = 100
	}

	streamID := channel.NewRequestID()
	signal := &agentpb.OpenDataStreamSignal{
		StreamId:   streamID,
		StreamType: agentpb.DataStreamType_DATA_STREAM_TYPE_LOGS,
		Params: &agentpb.OpenDataStreamSignal_StreamLogs{
			StreamLogs: &agentpb.StreamLogsRequest{
				ContainerId: dockerContainerID,
				Follow:      follow,
				TailLines:   tailLines,
			},
		},
	}

	streamCh, err := ch.SignalOpenDataStream(e.Request.Context(), signal)
	if err != nil {
		return e.JSON(http.StatusBadGateway, map[string]string{"error": "failed to signal log stream"})
	}

	// Wait for the agent to open the DataStream.
	var stream agentpb.AgentDataStreamService_OpenDataStreamServer
	select {
	case <-e.Request.Context().Done():
		return nil
	case s, ok := <-streamCh:
		if !ok {
			return e.JSON(http.StatusBadGateway, map[string]string{"error": "log stream not available"})
		}
		stream = s
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

	// Channel receives DataStream messages (or errors) from the agent.
	type recvResult struct {
		entry *agentpb.OpenDataStreamRequest
		err   error
	}
	recvCh := make(chan recvResult, 1)

	go func() {
		for {
			entry, recvErr := stream.Recv()
			recvCh <- recvResult{entry: entry, err: recvErr}
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

		case res := <-recvCh:
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

			// Handle text messages (log lines).
			if text := res.entry.GetText(); text != nil {
				ev := logEvent{
					Stream:    text.GetStream(),
					Message:   text.GetLine(),
					Timestamp: time.Now().UTC().Format("2006-01-02T15:04:05.000Z"),
				}
				data, _ := json.Marshal(ev)
				_, _ = fmt.Fprintf(w, "event: log\ndata: %s\n\n", data)
				flusher.Flush()
			}

			// Complete or error signals end the stream.
			if res.entry.GetComplete() != nil || res.entry.GetError() != nil {
				_, _ = fmt.Fprintf(w, "event: done\ndata: {}\n\n")
				flusher.Flush()
				return nil
			}
		}
	}
}

// multiLogEvent extends logEvent with a container identifier.
type multiLogEvent struct {
	ContainerID   string `json:"container_id"`
	ContainerName string `json:"container_name"`
	Timestamp     string `json:"timestamp"`
	Stream        string `json:"stream"`
	Message       string `json:"message"`
}

// handleMultiLogStream opens log streams for multiple containers and merges
// them into a single SSE connection. Query: ids=a,b,c&tail=100&follow=true.
func handleMultiLogStream(e *core.RequestEvent, app core.App, registry *channel.Registry) error {
	idsParam := e.Request.URL.Query().Get("ids")
	if idsParam == "" {
		return e.JSON(http.StatusBadRequest, map[string]string{"error": "ids query parameter required"})
	}

	ids := strings.Split(idsParam, ",")
	if len(ids) < 2 || len(ids) > 6 {
		return e.JSON(http.StatusBadRequest, map[string]string{"error": "provide 2-6 container IDs"})
	}

	follow := e.Request.URL.Query().Get("follow") != "false"
	tail := e.Request.URL.Query().Get("tail")
	if tail == "" {
		tail = "100"
	}
	var tailLines uint32
	if _, scanErr := fmt.Sscanf(tail, "%d", &tailLines); scanErr != nil {
		tailLines = 100
	}

	// Resolve all containers and their servers.
	type containerInfo struct {
		recordID          string
		name              string
		dockerContainerID string
		serverID          string
	}
	var containers []containerInfo

	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		container, err := app.FindRecordById(constants.CollContainers, id)
		if err != nil {
			return e.JSON(http.StatusNotFound, map[string]string{"error": "container not found: " + id})
		}
		srvID := container.GetString("server")
		server, err := app.FindRecordById(constants.CollServers, srvID)
		if err != nil {
			return e.JSON(http.StatusNotFound, map[string]string{"error": "server not found for container: " + id})
		}
		if server.GetString("status") != "online" {
			return e.JSON(http.StatusServiceUnavailable, map[string]string{"error": "server offline for container: " + id})
		}
		containers = append(containers, containerInfo{
			recordID:          id,
			name:              container.GetString("name"),
			dockerContainerID: container.GetString("container_id"),
			serverID:          srvID,
		})
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

	// Merged channel for all container log entries.
	type mergedEntry struct {
		containerID   string
		containerName string
		entry         *agentpb.OpenDataStreamRequest
		err           error
	}
	mergedCh := make(chan mergedEntry, 16)

	// Open a DataStream for each container.
	var wg sync.WaitGroup
	for _, c := range containers {
		wg.Add(1)
		go func(ci containerInfo) {
			defer wg.Done()

			ch, okCh := registry.Get(ci.serverID)
			if !okCh {
				slog.Error("multi-log: agent not connected", "container", ci.recordID, "server", ci.serverID)
				return
			}

			streamID := channel.NewRequestID()
			signal := &agentpb.OpenDataStreamSignal{
				StreamId:   streamID,
				StreamType: agentpb.DataStreamType_DATA_STREAM_TYPE_LOGS,
				Params: &agentpb.OpenDataStreamSignal_StreamLogs{
					StreamLogs: &agentpb.StreamLogsRequest{
						ContainerId: ci.dockerContainerID,
						Follow:      follow,
						TailLines:   tailLines,
					},
				},
			}

			streamCh, err := ch.SignalOpenDataStream(e.Request.Context(), signal)
			if err != nil {
				slog.Error("multi-log: signal failed", "container", ci.recordID, "error", err)
				return
			}

			// Wait for agent to open the DataStream.
			var stream agentpb.AgentDataStreamService_OpenDataStreamServer
			select {
			case <-e.Request.Context().Done():
				return
			case s, ok := <-streamCh:
				if !ok {
					slog.Error("multi-log: stream channel closed", "container", ci.recordID)
					return
				}
				stream = s
			}

			for {
				entry, recvErr := stream.Recv()
				mergedCh <- mergedEntry{
					containerID:   ci.recordID,
					containerName: ci.name,
					entry:         entry,
					err:           recvErr,
				}
				if recvErr != nil {
					return
				}
			}
		}(c)
	}

	// Close merged channel when all goroutines finish.
	go func() {
		wg.Wait()
		close(mergedCh)
	}()

	keepalive := time.NewTicker(15 * time.Second)
	defer keepalive.Stop()

	for {
		select {
		case <-e.Request.Context().Done():
			return nil

		case <-keepalive.C:
			_, _ = fmt.Fprintf(w, ": keepalive\n\n")
			flusher.Flush()

		case res, ok := <-mergedCh:
			if !ok {
				// All streams closed.
				_, _ = fmt.Fprintf(w, "event: done\ndata: {}\n\n")
				flusher.Flush()
				return nil
			}
			if res.err != nil {
				if res.err == io.EOF {
					continue // One stream ended, others may still be active.
				}
				slog.Debug("multi-log stream error", "container", res.containerID, "error", res.err)
				continue
			}

			// Handle text messages (log lines).
			if text := res.entry.GetText(); text != nil {
				ev := multiLogEvent{
					ContainerID:   res.containerID,
					ContainerName: res.containerName,
					Stream:        text.GetStream(),
					Message:       text.GetLine(),
					Timestamp:     time.Now().UTC().Format("2006-01-02T15:04:05.000Z"),
				}
				data, _ := json.Marshal(ev)
				_, _ = fmt.Fprintf(w, "event: log\ndata: %s\n\n", data)
				flusher.Flush()
			}

			// Complete or error from one container: just continue, others may still be active.
			if res.entry.GetComplete() != nil || res.entry.GetError() != nil {
				continue
			}
		}
	}
}
