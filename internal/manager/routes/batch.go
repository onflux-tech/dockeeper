package routes

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/pocketbase/pocketbase/core"

	"github.com/onflux-tech/dockeeper/internal/manager/middleware"
	"github.com/onflux-tech/dockeeper/internal/manager/services"
	"github.com/onflux-tech/dockeeper/internal/manager/store"
	"github.com/onflux-tech/dockeeper/internal/shared/constants"
)

// RegisterBatchRoutes registers all /api/dk/batch/* routes.
func RegisterBatchRoutes(se *core.ServeEvent, runner *services.BatchJobRunner, batchStore *store.BatchStore, tagSvc *services.TagService, rbac *middleware.RBAC) {
	group := se.Router.Group("/api/dk/batch")

	group.POST("", handleBatchSubmit(runner, tagSvc)).BindFunc(rbac.RequireCapability(constants.CapContainersManage))
	group.GET("", handleBatchList(batchStore)).BindFunc(rbac.RequireCapability(constants.CapContainersView))
	group.GET("/{id}", handleBatchGet(batchStore)).BindFunc(rbac.RequireCapability(constants.CapContainersView))
	group.GET("/{id}/progress", handleBatchProgress(runner, batchStore)).BindFunc(rbac.RequireCapability(constants.CapContainersView))
	group.POST("/{id}/cancel", handleBatchCancel(runner)).BindFunc(rbac.RequireCapability(constants.CapContainersManage))
}

func handleBatchSubmit(runner *services.BatchJobRunner, tagSvc *services.TagService) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		var body struct {
			Action       string   `json:"action"`
			ContainerIDs []string `json:"container_ids"`
			TagFilter    string   `json:"tag_filter"`
		}
		if err := e.BindBody(&body); err != nil {
			return e.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		}

		if body.Action == "" {
			return e.JSON(http.StatusBadRequest, map[string]string{"error": "action is required"})
		}

		// Resolve tag filter to container IDs if provided.
		containerIDs := body.ContainerIDs
		if body.TagFilter != "" && len(containerIDs) == 0 {
			ids, err := tagSvc.GetResourcesByTag(body.TagFilter, "container")
			if err != nil {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": "failed to resolve tag filter"})
			}
			containerIDs = ids
		}

		if len(containerIDs) == 0 {
			return e.JSON(http.StatusBadRequest, map[string]string{"error": "no containers specified"})
		}

		jobID, err := runner.Submit(body.Action, e.Auth.Id, containerIDs)
		if err != nil {
			code := http.StatusInternalServerError
			if _, ok := err.(*services.BatchError); ok {
				code = http.StatusBadRequest
			}
			return e.JSON(code, map[string]string{"error": err.Error()})
		}

		return e.JSON(http.StatusAccepted, map[string]string{
			"job_id": jobID,
			"status": "pending",
		})
	}
}

func handleBatchGet(batchStore *store.BatchStore) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		id := e.Request.PathValue("id")
		if id == "" {
			return e.JSON(http.StatusBadRequest, map[string]string{"error": "id is required"})
		}

		rec, err := batchStore.GetJob(id)
		if err != nil {
			return e.JSON(http.StatusNotFound, map[string]string{"error": "batch job not found"})
		}

		items, _ := batchStore.GetItems(rec)
		return e.JSON(http.StatusOK, map[string]any{
			"id":           rec.Id,
			"action":       rec.GetString("action"),
			"status":       rec.GetString("status"),
			"total":        rec.GetInt("total"),
			"succeeded":    rec.GetInt("succeeded"),
			"failed":       rec.GetInt("failed"),
			"cancelled":    rec.GetInt("cancelled"),
			"created_by":   rec.GetString("created_by"),
			"started_at":   rec.GetString("started_at"),
			"completed_at": rec.GetString("completed_at"),
			"items":        items,
			"created":      rec.GetString("created"),
		})
	}
}

func handleBatchList(batchStore *store.BatchStore) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		q := e.Request.URL.Query()
		status := q.Get("status")
		page, _ := strconv.Atoi(q.Get("page"))
		perPage, _ := strconv.Atoi(q.Get("per_page"))
		if page < 1 {
			page = 1
		}
		if perPage < 1 || perPage > 100 {
			perPage = 50
		}

		records, err := batchStore.ListJobs(status, page, perPage)
		if err != nil {
			return e.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}

		jobs := make([]map[string]any, 0, len(records))
		for _, rec := range records {
			jobs = append(jobs, map[string]any{
				"id":           rec.Id,
				"action":       rec.GetString("action"),
				"status":       rec.GetString("status"),
				"total":        rec.GetInt("total"),
				"succeeded":    rec.GetInt("succeeded"),
				"failed":       rec.GetInt("failed"),
				"cancelled":    rec.GetInt("cancelled"),
				"created_by":   rec.GetString("created_by"),
				"started_at":   rec.GetString("started_at"),
				"completed_at": rec.GetString("completed_at"),
				"created":      rec.GetString("created"),
			})
		}

		return e.JSON(http.StatusOK, map[string]any{
			"items": jobs,
			"page":  page,
		})
	}
}

func handleBatchProgress(runner *services.BatchJobRunner, batchStore *store.BatchStore) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		id := e.Request.PathValue("id")
		if id == "" {
			return e.JSON(http.StatusBadRequest, map[string]string{"error": "id is required"})
		}

		// Verify job exists.
		rec, err := batchStore.GetJob(id)
		if err != nil {
			return e.JSON(http.StatusNotFound, map[string]string{"error": "batch job not found"})
		}

		// If job is already completed, return current state.
		status := rec.GetString("status")
		if status == "completed" || status == "cancelled" {
			items, _ := batchStore.GetItems(rec)
			data, _ := json.Marshal(map[string]any{
				"job_id":    id,
				"status":    status,
				"total":     rec.GetInt("total"),
				"succeeded": rec.GetInt("succeeded"),
				"failed":    rec.GetInt("failed"),
				"cancelled": rec.GetInt("cancelled"),
				"items":     items,
			})

			w := e.Response
			w.Header().Set("Content-Type", "text/event-stream")
			w.Header().Set("Cache-Control", "no-cache")
			w.Header().Set("Connection", "keep-alive")
			w.Header().Set("X-Accel-Buffering", "no")
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, "event: complete\ndata: %s\n\n", data)
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
			return nil
		}

		// Set up SSE streaming.
		w := e.Response
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("X-Accel-Buffering", "no")
		w.WriteHeader(http.StatusOK)

		flusher, ok := w.(http.Flusher)
		if !ok {
			return e.JSON(http.StatusInternalServerError, map[string]string{"error": "streaming not supported"})
		}

		// Disable write deadline for long-lived SSE.
		rc := http.NewResponseController(w)
		_ = rc.SetWriteDeadline(time.Time{})

		// Send initial connected event.
		fmt.Fprintf(w, "event: connected\ndata: {\"job_id\":\"%s\"}\n\n", id)
		flusher.Flush()

		ctx := e.Request.Context()
		ch := runner.Subscribe(id)
		defer runner.Unsubscribe(id, ch)

		keepalive := time.NewTicker(15 * time.Second)
		defer keepalive.Stop()

		eventID := 0
		for {
			select {
			case <-ctx.Done():
				return nil
			case <-keepalive.C:
				fmt.Fprint(w, ": ping\n\n")
				flusher.Flush()
			case progress, open := <-ch:
				if !open {
					return nil
				}
				eventID++
				data, _ := json.Marshal(progress)
				eventName := "progress"
				if progress.JobStatus == "completed" || progress.JobStatus == "cancelled" {
					eventName = "complete"
				}
				fmt.Fprintf(w, "id: %d\nevent: %s\ndata: %s\n\n", eventID, eventName, data)
				flusher.Flush()

				if eventName == "complete" {
					return nil
				}
			}
		}
	}
}

func handleBatchCancel(runner *services.BatchJobRunner) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		id := e.Request.PathValue("id")
		if id == "" {
			return e.JSON(http.StatusBadRequest, map[string]string{"error": "id is required"})
		}

		if err := runner.Cancel(id); err != nil {
			return e.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}

		return e.JSON(http.StatusOK, map[string]string{
			"status": "cancelling",
		})
	}
}
