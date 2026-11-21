package services

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	"github.com/pocketbase/pocketbase/core"

	"github.com/onflux-tech/dockeeper/internal/manager/events"
	"github.com/onflux-tech/dockeeper/internal/manager/store"
	"github.com/onflux-tech/dockeeper/internal/shared/constants"
	agentpb "github.com/onflux-tech/dockeeper/proto/gen/dockeeper/agent/v1"
)

const (
	maxBatchSize        = 500
	perHostConcurrency  = 5
	progressChannelSize = 100
	channelCleanupDelay = 60 * time.Second
)

// BatchProgress is sent to SSE consumers for each item state change.
type BatchProgress struct {
	JobID       string `json:"job_id"`
	ContainerID string `json:"container_id"`
	Status      string `json:"status"`
	Error       string `json:"error,omitempty"`
	Succeeded   int    `json:"succeeded"`
	Failed      int    `json:"failed"`
	Cancelled   int    `json:"cancelled"`
	Total       int    `json:"total"`
	JobStatus   string `json:"job_status"`
}

// BatchJobRunner executes batch operations with per-host concurrency control.
type BatchJobRunner struct {
	app          core.App
	containerSvc *ContainerService
	bus          *events.Bus
	store        *store.BatchStore

	// Per-job progress subscribers: jobID -> list of channels.
	subscribers sync.Map // map[string][]chan BatchProgress

	// Per-job cancel functions.
	cancels sync.Map // map[string]context.CancelFunc
}

// NewBatchJobRunner creates a new BatchJobRunner.
func NewBatchJobRunner(app core.App, containerSvc *ContainerService, bus *events.Bus, batchStore *store.BatchStore) *BatchJobRunner {
	return &BatchJobRunner{
		app:          app,
		containerSvc: containerSvc,
		bus:          bus,
		store:        batchStore,
	}
}

// Submit creates a batch job and starts async execution.
func (r *BatchJobRunner) Submit(action, createdBy string, containerIDs []string) (string, error) {
	if len(containerIDs) > maxBatchSize {
		return "", ErrBatchTooLarge
	}

	// Build items with server mapping.
	items := make([]store.BatchItem, 0, len(containerIDs))
	for _, cid := range containerIDs {
		rec, err := r.app.FindRecordById(constants.CollContainers, cid)
		if err != nil {
			slog.Warn("batch: container not found, skipping", "container_id", cid)
			continue
		}
		items = append(items, store.BatchItem{
			ContainerID: cid,
			ServerID:    rec.GetString("server"),
			Status:      "pending",
		})
	}

	if len(items) == 0 {
		return "", ErrBatchEmpty
	}

	jobRec, err := r.store.CreateJob(action, createdBy, items)
	if err != nil {
		return "", err
	}

	jobID := jobRec.Id

	// Publish batch.started event.
	r.bus.Publish(events.Event{
		Type:     events.BatchStarted,
		Severity: events.SeverityInfo,
		Details: map[string]any{
			"job_id": jobID,
			"action": action,
			"total":  len(items),
		},
		Timestamp: time.Now(),
	})

	// Start execution in background.
	ctx, cancel := context.WithCancel(context.Background())
	r.cancels.Store(jobID, cancel)
	go r.execute(ctx, jobID, action, items)

	return jobID, nil
}

// Cancel marks pending items as cancelled.
func (r *BatchJobRunner) Cancel(jobID string) error {
	if cancelFn, ok := r.cancels.Load(jobID); ok {
		cancelFn.(context.CancelFunc)()
	}
	return nil
}

// Subscribe returns a channel that receives progress updates for a job.
func (r *BatchJobRunner) Subscribe(jobID string) chan BatchProgress {
	ch := make(chan BatchProgress, progressChannelSize)
	val, _ := r.subscribers.LoadOrStore(jobID, []chan BatchProgress{})
	subs := val.([]chan BatchProgress)
	subs = append(subs, ch)
	r.subscribers.Store(jobID, subs)
	return ch
}

// Unsubscribe removes a subscriber channel.
func (r *BatchJobRunner) Unsubscribe(jobID string, ch chan BatchProgress) {
	val, ok := r.subscribers.Load(jobID)
	if !ok {
		return
	}
	subs := val.([]chan BatchProgress)
	for i, s := range subs {
		if s == ch {
			subs = append(subs[:i], subs[i+1:]...)
			break
		}
	}
	r.subscribers.Store(jobID, subs)
	close(ch)
}

// execute runs the batch job: groups items by server, executes with per-host concurrency.
func (r *BatchJobRunner) execute(ctx context.Context, jobID, action string, items []store.BatchItem) {
	defer func() {
		r.cancels.Delete(jobID)
		// Delayed cleanup of subscriber channels.
		time.AfterFunc(channelCleanupDelay, func() {
			r.cleanupSubscribers(jobID)
		})
	}()

	// Mark job as running.
	_ = r.store.UpdateJob(jobID, map[string]any{
		"status":     "running",
		"started_at": time.Now().UTC().Format(time.RFC3339),
	})

	// Group items by server.
	serverGroups := make(map[string][]store.BatchItem)
	for _, item := range items {
		serverGroups[item.ServerID] = append(serverGroups[item.ServerID], item)
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	succeeded, failed, cancelled := 0, 0, 0

	for serverID, group := range serverGroups {
		wg.Add(1)
		go func(sid string, groupItems []store.BatchItem) {
			defer wg.Done()
			sem := make(chan struct{}, perHostConcurrency)

			// Check if server is online.
			serverRec, err := r.app.FindRecordById(constants.CollServers, sid)
			serverOffline := err != nil || serverRec.GetString("status") != "online"

			var itemWg sync.WaitGroup
			for _, item := range groupItems {
				itemWg.Add(1)
				go func(itm store.BatchItem) {
					defer itemWg.Done()

					// Check cancellation.
					if ctx.Err() != nil {
						_ = r.store.UpdateItemStatus(jobID, itm.ContainerID, "cancelled", "", now())
						_ = r.store.IncrementCounter(jobID, "cancelled")
						mu.Lock()
						cancelled++
						mu.Unlock()
						r.notifyProgress(jobID, itm.ContainerID, "cancelled", "", succeeded, failed, cancelled, len(items), "running")
						return
					}

					// Server offline → fail immediately.
					if serverOffline {
						_ = r.store.UpdateItemStatus(jobID, itm.ContainerID, "failed", "server offline", now())
						_ = r.store.IncrementCounter(jobID, "failed")
						mu.Lock()
						failed++
						mu.Unlock()
						r.notifyProgress(jobID, itm.ContainerID, "failed", "server offline", succeeded, failed, cancelled, len(items), "running")
						return
					}

					// Acquire semaphore slot.
					sem <- struct{}{}
					defer func() { <-sem }()

					// Mark as running.
					_ = r.store.UpdateItemStatus(jobID, itm.ContainerID, "running", "", now())
					r.notifyProgress(jobID, itm.ContainerID, "running", "", succeeded, failed, cancelled, len(items), "running")

					// Execute action.
					execErr := r.executeItem(ctx, itm.ContainerID, action)

					if execErr != nil {
						_ = r.store.UpdateItemStatus(jobID, itm.ContainerID, "failed", execErr.Error(), now())
						_ = r.store.IncrementCounter(jobID, "failed")
						mu.Lock()
						failed++
						mu.Unlock()
						r.notifyProgress(jobID, itm.ContainerID, "failed", execErr.Error(), succeeded, failed, cancelled, len(items), "running")
					} else {
						_ = r.store.UpdateItemStatus(jobID, itm.ContainerID, "success", "", now())
						_ = r.store.IncrementCounter(jobID, "succeeded")
						mu.Lock()
						succeeded++
						mu.Unlock()
						r.notifyProgress(jobID, itm.ContainerID, "success", "", succeeded, failed, cancelled, len(items), "running")
					}
				}(item)
			}
			itemWg.Wait()
		}(serverID, group)
	}

	wg.Wait()

	// Determine final status.
	finalStatus := "completed"
	if cancelled == len(items) {
		finalStatus = "cancelled"
	}

	_ = r.store.UpdateJob(jobID, map[string]any{
		"status":       finalStatus,
		"completed_at": time.Now().UTC().Format(time.RFC3339),
	})

	// Final progress notification.
	r.notifyProgress(jobID, "", "", "", succeeded, failed, cancelled, len(items), finalStatus)

	// Publish completion event.
	eventType := events.BatchCompleted
	severity := events.SeverityInfo
	if failed > 0 {
		eventType = events.BatchFailed
		severity = events.SeverityWarning
	}
	r.bus.Publish(events.Event{
		Type:     eventType,
		Severity: severity,
		Details: map[string]any{
			"job_id":    jobID,
			"action":    action,
			"total":     len(items),
			"succeeded": succeeded,
			"failed":    failed,
			"cancelled": cancelled,
		},
		Timestamp: time.Now(),
	})
}

// executeItem dispatches a single container action.
func (r *BatchJobRunner) executeItem(ctx context.Context, containerID, action string) error {
	cmdType := mapBatchAction(action)
	if cmdType == agentpb.CommandType_COMMAND_TYPE_UNSPECIFIED {
		return ErrBatchInvalidAction
	}

	_, err := r.containerSvc.ExecuteAction(ctx, containerID, cmdType, nil, "batch", "batch")
	return err
}

// mapBatchAction maps a batch action string to a gRPC command type.
func mapBatchAction(action string) agentpb.CommandType {
	switch action {
	case "start":
		return agentpb.CommandType_COMMAND_TYPE_START
	case "stop":
		return agentpb.CommandType_COMMAND_TYPE_STOP
	case "restart":
		return agentpb.CommandType_COMMAND_TYPE_RESTART
	case "remove":
		return agentpb.CommandType_COMMAND_TYPE_REMOVE
	case "pause":
		return agentpb.CommandType_COMMAND_TYPE_PAUSE
	case "unpause":
		return agentpb.CommandType_COMMAND_TYPE_UNPAUSE
	default:
		return agentpb.CommandType_COMMAND_TYPE_UNSPECIFIED
	}
}

// notifyProgress sends a progress update to all subscribers.
func (r *BatchJobRunner) notifyProgress(jobID, containerID, status, errMsg string, succeeded, failed, cancelled, total int, jobStatus string) {
	progress := BatchProgress{
		JobID:       jobID,
		ContainerID: containerID,
		Status:      status,
		Error:       errMsg,
		Succeeded:   succeeded,
		Failed:      failed,
		Cancelled:   cancelled,
		Total:       total,
		JobStatus:   jobStatus,
	}

	val, ok := r.subscribers.Load(jobID)
	if !ok {
		return
	}
	subs := val.([]chan BatchProgress)
	data, _ := json.Marshal(progress)
	_ = data // serialization check

	for _, ch := range subs {
		select {
		case ch <- progress:
		default:
			// Drop if consumer is slow.
		}
	}
}

// cleanupSubscribers closes and removes all subscriber channels for a job.
func (r *BatchJobRunner) cleanupSubscribers(jobID string) {
	val, ok := r.subscribers.LoadAndDelete(jobID)
	if !ok {
		return
	}
	subs := val.([]chan BatchProgress)
	for _, ch := range subs {
		close(ch)
	}
}

func now() string {
	return time.Now().UTC().Format(time.RFC3339)
}

// Sentinel errors for batch operations.
var (
	ErrBatchTooLarge      = &BatchError{"batch exceeds maximum size of 500 items"}
	ErrBatchEmpty         = &BatchError{"no valid containers found for batch"}
	ErrBatchInvalidAction = &BatchError{"invalid batch action"}
)

// BatchError represents a batch operation error.
type BatchError struct {
	Message string
}

func (e *BatchError) Error() string {
	return e.Message
}
