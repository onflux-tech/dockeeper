package routes

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/pocketbase/pocketbase/core"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/jkdesenvolvedor/dockeeper/internal/manager/middleware"
	"github.com/jkdesenvolvedor/dockeeper/internal/manager/services"
	"github.com/jkdesenvolvedor/dockeeper/internal/shared/constants"
	agentpb "github.com/jkdesenvolvedor/dockeeper/proto/gen/dockeeper/agent/v1"
)

const agentImagePort = "50052"

// RegisterImageRoutes adds image management endpoints.
func RegisterImageRoutes(se *core.ServeEvent, app core.App, svc *services.ImageService) {
	images := se.Router.Group("/api/dk/images")

	images.GET("", func(e *core.RequestEvent) error {
		return handleListImages(e, app)
	}).BindFunc(middleware.RequireSuperuser)

	images.GET("/pull", func(e *core.RequestEvent) error {
		return handlePullImageSSE(e, svc)
	}).BindFunc(middleware.RequireSuperuser)

	images.DELETE("/{id}", func(e *core.RequestEvent) error {
		return handleRemoveImage(e, svc)
	}).BindFunc(middleware.RequireSuperuser)

	images.POST("/prune", func(e *core.RequestEvent) error {
		return handlePruneImages(e, svc)
	}).BindFunc(middleware.RequireSuperuser)
}

// handleListImages returns images from PocketBase, optionally filtered by server.
func handleListImages(e *core.RequestEvent, app core.App) error {
	serverID := e.Request.URL.Query().Get("server_id")

	filter := ""
	if serverID != "" {
		filter = fmt.Sprintf("server = '%s'", serverID)
	}

	records, err := app.FindRecordsByFilter(
		constants.CollImages,
		filter,
		"-created",
		0, 0,
	)
	if err != nil {
		return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to list images"})
	}

	type imageEntry struct {
		ID        string `json:"id"`
		Server    string `json:"server"`
		ImageID   string `json:"image_id"`
		Tags      any    `json:"tags"`
		Size      int64  `json:"size"`
		InUse     bool   `json:"in_use"`
		CreatedAt string `json:"created_at"`
	}

	items := make([]imageEntry, 0, len(records))
	for _, r := range records {
		items = append(items, imageEntry{
			ID:        r.Id,
			Server:    r.GetString("server"),
			ImageID:   r.GetString("image_id"),
			Tags:      r.Get("tags"),
			Size:      int64(r.GetFloat("size")),
			InUse:     r.GetBool("in_use"),
			CreatedAt: r.GetString("created"),
		})
	}

	return e.JSON(http.StatusOK, items)
}

// pullProgressEvent is the SSE data for image pull progress.
type pullProgressEvent struct {
	ID      string `json:"id,omitempty"`
	Status  string `json:"status"`
	Current int64  `json:"current,omitempty"`
	Total   int64  `json:"total,omitempty"`
	Done    bool   `json:"done,omitempty"`
	Error   string `json:"error,omitempty"`
}

// handlePullImageSSE streams image pull progress via SSE.
func handlePullImageSSE(e *core.RequestEvent, svc *services.ImageService) error {
	serverID := e.Request.URL.Query().Get("server_id")
	imageRef := e.Request.URL.Query().Get("image")

	if serverID == "" || imageRef == "" {
		return e.JSON(http.StatusBadRequest, map[string]string{"error": "server_id and image query params are required"})
	}

	agentIP, err := svc.ResolveServer(serverID)
	if err != nil {
		return mapImageServiceError(e, err)
	}

	// Dial agent gRPC.
	addr := agentIP + ":" + agentImagePort
	conn, err := grpc.NewClient(addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		slog.Error("failed to dial agent for image pull", "addr", addr, "error", err)
		return e.JSON(http.StatusBadGateway, map[string]string{"error": "cannot reach agent"})
	}
	defer func() {
		if closeErr := conn.Close(); closeErr != nil {
			slog.Warn("failed to close agent connection", "addr", addr, "error", closeErr)
		}
	}()

	client := agentpb.NewAgentImageServiceClient(conn)

	stream, err := client.PullImage(e.Request.Context(), &agentpb.ImagePullRequest{
		ImageRef: imageRef,
	})
	if err != nil {
		slog.Error("failed to open image pull stream", "error", err)
		return e.JSON(http.StatusBadGateway, map[string]string{"error": "failed to start image pull"})
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
		msg *agentpb.ImagePullProgress
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
				data, _ := json.Marshal(pullProgressEvent{Error: res.err.Error(), Done: true})
				_, _ = fmt.Fprintf(w, "event: error\ndata: %s\n\n", data)
				flusher.Flush()
				return nil
			}

			ev := pullProgressEvent{
				ID:      res.msg.GetId(),
				Status:  res.msg.GetStatus(),
				Current: res.msg.GetCurrent(),
				Total:   res.msg.GetTotal(),
				Done:    res.msg.GetDone(),
				Error:   res.msg.GetError(),
			}

			data, _ := json.Marshal(ev)
			_, _ = fmt.Fprintf(w, "event: progress\ndata: %s\n\n", data)
			flusher.Flush()

			if ev.Done {
				return nil
			}
		}
	}
}

// handleRemoveImage removes a Docker image via the agent.
func handleRemoveImage(e *core.RequestEvent, svc *services.ImageService) error {
	imageID := e.Request.PathValue("id")
	force := e.Request.URL.Query().Get("force") == "true"

	userID := ""
	if e.Auth != nil {
		userID = e.Auth.Id
	}

	resp, err := svc.RemoveImage(e.Request.Context(), imageID, force, userID, e.RealIP())
	if err != nil {
		return mapImageServiceError(e, err)
	}

	return e.JSON(http.StatusOK, resp)
}

// handlePruneImages prunes unused Docker images on a server.
func handlePruneImages(e *core.RequestEvent, svc *services.ImageService) error {
	var body struct {
		ServerID string `json:"server_id"`
		All      bool   `json:"all"`
	}
	if err := e.BindBody(&body); err != nil || body.ServerID == "" {
		return e.JSON(http.StatusBadRequest, map[string]string{"error": "server_id is required"})
	}

	userID := ""
	if e.Auth != nil {
		userID = e.Auth.Id
	}

	resp, err := svc.PruneImages(e.Request.Context(), body.ServerID, body.All, userID, e.RealIP())
	if err != nil {
		return mapImageServiceError(e, err)
	}

	return e.JSON(http.StatusOK, resp)
}

func mapImageServiceError(e *core.RequestEvent, err error) error {
	switch err {
	case services.ErrServerNotFound:
		return e.JSON(http.StatusNotFound, map[string]string{"error": "server not found"})
	case services.ErrServerOffline:
		return e.JSON(http.StatusServiceUnavailable, map[string]string{"error": "server is offline"})
	default:
		if err.Error() == "image not found" {
			return e.JSON(http.StatusNotFound, map[string]string{"error": "image not found"})
		}
		return e.JSON(http.StatusBadGateway, map[string]string{"error": err.Error()})
	}
}
