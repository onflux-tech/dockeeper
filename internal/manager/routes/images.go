package routes

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/pocketbase/pocketbase/core"

	"github.com/onflux-tech/dockeeper/internal/manager/channel"
	"github.com/onflux-tech/dockeeper/internal/manager/middleware"
	"github.com/onflux-tech/dockeeper/internal/manager/services"
	"github.com/onflux-tech/dockeeper/internal/shared/constants"
	"github.com/onflux-tech/dockeeper/internal/shared/security"
	agentpb "github.com/onflux-tech/dockeeper/proto/gen/dockeeper/agent/v1"
)

// RegisterImageRoutes adds image management endpoints.
func RegisterImageRoutes(se *core.ServeEvent, app core.App, svc *services.ImageService, registry *channel.Registry, rbac *middleware.RBAC) {
	images := se.Router.Group("/api/dk/images")

	images.GET("", func(e *core.RequestEvent) error {
		return handleListImages(e, app)
	}).BindFunc(rbac.RequireCapability(constants.CapImagesView))

	images.GET("/pull", func(e *core.RequestEvent) error {
		return handlePullImageSSE(e, svc, registry)
	}).BindFunc(rbac.RequireCapability(constants.CapImagesManage))

	images.DELETE("/{id}", func(e *core.RequestEvent) error {
		return handleRemoveImage(e, svc)
	}).BindFunc(rbac.RequireCapability(constants.CapImagesManage))

	images.POST("/prune", func(e *core.RequestEvent) error {
		return handlePruneImages(e, svc)
	}).BindFunc(rbac.RequireCapability(constants.CapImagesManage))
}

// handleListImages returns images from PocketBase, optionally filtered by server.
func handleListImages(e *core.RequestEvent, app core.App) error {
	serverID := e.Request.URL.Query().Get("server_id")

	filter := ""
	if serverID != "" {
		filter = fmt.Sprintf("server = '%s'", security.SanitizeFilterParam(serverID))
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

// handlePullImageSSE streams image pull progress via SSE using the reverse tunnel DataStream.
func handlePullImageSSE(e *core.RequestEvent, svc *services.ImageService, registry *channel.Registry) error {
	serverID := e.Request.URL.Query().Get("server_id")
	imageRef := e.Request.URL.Query().Get("image")

	if serverID == "" || imageRef == "" {
		return e.JSON(http.StatusBadRequest, map[string]string{"error": "server_id and image query params are required"})
	}

	if err := security.ValidateDockerImageRef(imageRef); err != nil {
		return e.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	if _, err := svc.ResolveServer(serverID); err != nil {
		return mapImageServiceError(e, err)
	}

	// Get channel from registry.
	ch, ok := registry.Get(serverID)
	if !ok {
		return e.JSON(http.StatusBadGateway, map[string]string{"error": "agent not connected"})
	}

	// Signal a DataStream for image pull.
	streamID := channel.NewRequestID()
	signal := &agentpb.OpenDataStreamSignal{
		StreamId:   streamID,
		StreamType: agentpb.DataStreamType_DATA_STREAM_TYPE_IMAGE_PULL,
		Params:     &agentpb.OpenDataStreamSignal_PullImage{PullImage: &agentpb.ImagePullRequest{ImageRef: imageRef}},
	}

	ctx := e.Request.Context()
	streamCh, err := ch.SignalOpenDataStream(ctx, signal)
	if err != nil {
		slog.Error("failed to signal image pull data stream", "error", err)
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

	// Wait for the stream to arrive.
	var stream agentpb.AgentDataStreamService_OpenDataStreamServer
	select {
	case <-ctx.Done():
		return nil
	case s, chanOK := <-streamCh:
		if !chanOK {
			data, _ := json.Marshal(pullProgressEvent{Error: "data stream channel closed", Done: true})
			_, _ = fmt.Fprintf(w, "event: error\ndata: %s\n\n", data)
			flusher.Flush()
			return nil
		}
		stream = s
	}

	// Read DataStream messages and forward as SSE.
	type recvResult struct {
		msg *agentpb.OpenDataStreamRequest
		err error
	}
	recvCh := make(chan recvResult, 1)

	go func() {
		for {
			msg, recvErr := stream.Recv()
			recvCh <- recvResult{msg: msg, err: recvErr}
			if recvErr != nil {
				return
			}
		}
	}()

	for {
		select {
		case <-ctx.Done():
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
				data, _ := json.Marshal(pullProgressEvent{Error: res.err.Error(), Done: true})
				_, _ = fmt.Fprintf(w, "event: error\ndata: %s\n\n", data)
				flusher.Flush()
				return nil
			}

			var ev pullProgressEvent

			if progress := res.msg.GetProgress(); progress != nil {
				ev.ID = progress.GetId()
				ev.Status = progress.GetStatus()
				ev.Current = progress.GetCurrent()
				ev.Total = progress.GetTotal()
			}

			if complete := res.msg.GetComplete(); complete != nil {
				ev.Done = true
			}

			if errPayload := res.msg.GetError(); errPayload != nil {
				ev.Done = true
				ev.Error = errPayload.GetMessage()
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
