package services

import (
	"container/heap"
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/pocketbase/pocketbase/core"

	"github.com/onflux-tech/dockeeper/internal/manager/channel"
	"github.com/onflux-tech/dockeeper/internal/manager/events"
	"github.com/onflux-tech/dockeeper/internal/manager/store"
	"github.com/onflux-tech/dockeeper/internal/shared/constants"
	agentpb "github.com/onflux-tech/dockeeper/proto/gen/dockeeper/agent/v1"
)

// ── Min-Heap for scheduled checks ───────────────────────

type scheduledCheck struct {
	checkID   string
	nextCheck time.Time
	index     int
}

type checkHeap []*scheduledCheck

func (h checkHeap) Len() int            { return len(h) }
func (h checkHeap) Less(i, j int) bool   { return h[i].nextCheck.Before(h[j].nextCheck) }
func (h checkHeap) Swap(i, j int)        { h[i], h[j] = h[j], h[i]; h[i].index = i; h[j].index = j }
func (h *checkHeap) Push(x any)          { item := x.(*scheduledCheck); item.index = len(*h); *h = append(*h, item) }
func (h *checkHeap) Pop() any            { old := *h; n := len(old); item := old[n-1]; old[n-1] = nil; item.index = -1; *h = old[:n-1]; return item }

// ── HealthScheduler ─────────────────────────────────────

// HealthScheduler runs health checks using a min-heap priority queue.
type HealthScheduler struct {
	app        core.App
	store      *store.HealthStore
	registry   *channel.Registry
	bus        *events.Bus
	heap       checkHeap
	heapMu     sync.Mutex
	restarts   map[string][]time.Time // checkID → recent restart times
	restartsMu sync.Mutex
	cancel     context.CancelFunc
	done       chan struct{}
}

// NewHealthScheduler creates a new HealthScheduler.
func NewHealthScheduler(app core.App, healthStore *store.HealthStore, registry *channel.Registry, bus *events.Bus) *HealthScheduler {
	return &HealthScheduler{
		app:      app,
		store:    healthStore,
		registry: registry,
		bus:      bus,
		restarts: make(map[string][]time.Time),
		done:     make(chan struct{}),
	}
}

// Start loads all enabled checks and begins the scheduling loop.
func (s *HealthScheduler) Start() {
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel

	s.ReloadChecks()

	go s.loop(ctx)
	slog.Info("health scheduler started")
}

// Stop gracefully stops the scheduler.
func (s *HealthScheduler) Stop() {
	if s.cancel != nil {
		s.cancel()
	}
	<-s.done
	slog.Info("health scheduler stopped")
}

// ReloadChecks reloads all enabled checks from the DB into the heap.
func (s *HealthScheduler) ReloadChecks() {
	records, err := s.store.GetEnabledChecks()
	if err != nil {
		slog.Error("health scheduler: reload checks failed", "error", err)
		return
	}

	s.heapMu.Lock()
	defer s.heapMu.Unlock()

	s.heap = make(checkHeap, 0, len(records))
	for _, rec := range records {
		nextCheck := rec.GetDateTime("next_check_at").Time()
		if nextCheck.IsZero() {
			nextCheck = time.Now()
		}
		heap.Push(&s.heap, &scheduledCheck{
			checkID:   rec.Id,
			nextCheck: nextCheck,
		})
	}

	slog.Info("health scheduler: loaded checks", "count", len(s.heap))
}

// AddCheck adds a check to the heap by ID.
func (s *HealthScheduler) AddCheck(checkID string) {
	s.heapMu.Lock()
	defer s.heapMu.Unlock()

	heap.Push(&s.heap, &scheduledCheck{
		checkID:   checkID,
		nextCheck: time.Now(),
	})
}

// RemoveCheck removes a check from the heap by ID.
func (s *HealthScheduler) RemoveCheck(checkID string) {
	s.heapMu.Lock()
	defer s.heapMu.Unlock()

	for i, item := range s.heap {
		if item.checkID == checkID {
			heap.Remove(&s.heap, i)
			return
		}
	}
}

// TriggerCheck triggers an immediate check by moving it to the front of the heap.
func (s *HealthScheduler) TriggerCheck(checkID string) {
	s.heapMu.Lock()
	defer s.heapMu.Unlock()

	for _, item := range s.heap {
		if item.checkID == checkID {
			item.nextCheck = time.Now()
			heap.Fix(&s.heap, item.index)
			return
		}
	}
	// If not in heap (maybe disabled), add it.
	heap.Push(&s.heap, &scheduledCheck{
		checkID:   checkID,
		nextCheck: time.Now(),
	})
}

// loop is the main scheduling loop.
func (s *HealthScheduler) loop(ctx context.Context) {
	defer close(s.done)

	for {
		s.heapMu.Lock()
		if len(s.heap) == 0 {
			s.heapMu.Unlock()
			select {
			case <-ctx.Done():
				return
			case <-time.After(5 * time.Second):
				continue
			}
		}

		nextTime := s.heap[0].nextCheck
		s.heapMu.Unlock()

		waitDuration := time.Until(nextTime)
		if waitDuration > 0 {
			select {
			case <-ctx.Done():
				return
			case <-time.After(waitDuration):
			}
		}

		// Pop and execute the check.
		s.heapMu.Lock()
		if len(s.heap) == 0 {
			s.heapMu.Unlock()
			continue
		}
		item := heap.Pop(&s.heap).(*scheduledCheck)
		s.heapMu.Unlock()

		go s.executeCheck(ctx, item.checkID)

		select {
		case <-ctx.Done():
			return
		default:
		}
	}
}

func (s *HealthScheduler) executeCheck(ctx context.Context, checkID string) {
	rec, err := s.store.GetCheckByID(checkID)
	if err != nil {
		slog.Warn("health check not found, skipping", "check_id", checkID)
		return
	}

	if !rec.GetBool("enabled") {
		return
	}

	serverID := rec.GetString("server")
	containerID := rec.GetString("container")

	// Verify container is running.
	containerRec, err := s.app.FindRecordById(constants.CollContainers, containerID)
	if err != nil {
		slog.Warn("health check: container not found", "check_id", checkID, "container_id", containerID)
		s.reschedule(checkID, rec.GetInt("interval_seconds"))
		return
	}

	if containerRec.GetString("status") != "running" {
		slog.Debug("health check: container not running, skipping", "check_id", checkID)
		s.reschedule(checkID, rec.GetInt("interval_seconds"))
		return
	}

	// Get channel for the agent.
	ch, ok := s.registry.Get(serverID)
	if !ok {
		slog.Warn("health check: no active channel for server", "check_id", checkID, "server_id", serverID)
		s.handleFailure(ctx, rec, checkID, 0, 0, "no active channel for server")
		s.reschedule(checkID, rec.GetInt("interval_seconds"))
		return
	}

	timeoutSec := rec.GetInt("timeout_seconds")
	if timeoutSec <= 0 {
		timeoutSec = 5
	}

	grpcCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutSec+5)*time.Second)
	defer cancel()

	cmd := &agentpb.ChannelCommand{
		RequestId: channel.NewRequestID(),
		Priority:  3, // LOW
		Command: &agentpb.ChannelCommand_HealthCheck{
			HealthCheck: &agentpb.ExecuteHealthCheckRequest{
				ContainerId:    containerRec.GetString("container_id"),
				UrlPath:        rec.GetString("url_path"),
				Port:           int32(rec.GetInt("port")),
				TimeoutMs:      int32(timeoutSec * 1000),
				ExpectedStatus: int32(rec.GetInt("expected_status")),
			},
		},
	}

	result, err := ch.SendCommand(grpcCtx, cmd)
	now := time.Now().UTC()

	if err != nil {
		s.handleFailure(ctx, rec, checkID, 0, 0, fmt.Sprintf("channel command error: %s", err.Error()))
		s.reschedule(checkID, rec.GetInt("interval_seconds"))
		return
	}

	resp := result.GetHealthCheck()
	if resp == nil {
		if cmdErr := result.GetError(); cmdErr != nil {
			s.handleFailure(ctx, rec, checkID, 0, 0, fmt.Sprintf("agent error: %s", cmdErr.GetMessage()))
		} else {
			s.handleFailure(ctx, rec, checkID, 0, 0, "unexpected command result type")
		}
		s.reschedule(checkID, rec.GetInt("interval_seconds"))
		return
	}

	// Log the result.
	_ = s.store.LogCheckResult(checkID, int(resp.GetStatusCode()), resp.GetResponseTimeMs(), resp.GetSuccess(), resp.GetError(), now)

	if resp.GetSuccess() {
		s.handleSuccess(rec, checkID)
	} else {
		s.handleFailure(ctx, rec, checkID, int(resp.GetStatusCode()), resp.GetResponseTimeMs(), resp.GetError())
	}

	s.reschedule(checkID, rec.GetInt("interval_seconds"))
}

func (s *HealthScheduler) handleSuccess(rec *core.Record, checkID string) {
	consecutiveFailures := rec.GetInt("consecutive_failures")
	successThreshold := rec.GetInt("success_threshold")
	if successThreshold <= 0 {
		successThreshold = 1
	}
	currentStatus := rec.GetString("status")

	now := time.Now().UTC()
	interval := rec.GetInt("interval_seconds")
	if interval <= 0 {
		interval = 30
	}
	nextCheck := now.Add(time.Duration(interval) * time.Second)

	if consecutiveFailures > 0 || currentStatus != "healthy" {
		// Transition to healthy after reaching success threshold.
		_ = s.store.UpdateCheckStatus(checkID, "healthy", 0, now, nextCheck)

		if currentStatus == "unhealthy" {
			s.bus.Publish(events.Event{
				Type:        events.HealthRecovered,
				Severity:    events.SeverityInfo,
				ServerID:    rec.GetString("server"),
				ContainerID: rec.GetString("container"),
				Details:     map[string]any{"check_name": rec.GetString("name")},
				Timestamp:   now,
			})
		}
	} else {
		_ = s.store.UpdateCheckStatus(checkID, "healthy", 0, now, nextCheck)
	}
}

func (s *HealthScheduler) handleFailure(ctx context.Context, rec *core.Record, checkID string, statusCode int, responseTimeMs int64, errMsg string) {
	consecutiveFailures := rec.GetInt("consecutive_failures") + 1
	failureThreshold := rec.GetInt("failure_threshold")
	if failureThreshold <= 0 {
		failureThreshold = 3
	}

	now := time.Now().UTC()
	interval := rec.GetInt("interval_seconds")
	if interval <= 0 {
		interval = 30
	}
	nextCheck := now.Add(time.Duration(interval) * time.Second)

	_ = s.store.LogCheckResult(checkID, statusCode, responseTimeMs, false, errMsg, now)

	if consecutiveFailures >= failureThreshold {
		_ = s.store.UpdateCheckStatus(checkID, "unhealthy", consecutiveFailures, now, nextCheck)

		s.bus.Publish(events.Event{
			Type:        events.HealthDegraded,
			Severity:    events.SeverityWarning,
			ServerID:    rec.GetString("server"),
			ContainerID: rec.GetString("container"),
			Details: map[string]any{
				"check_name":           rec.GetString("name"),
				"consecutive_failures": consecutiveFailures,
				"error":                errMsg,
			},
			Timestamp: now,
		})

		// Auto-restart logic.
		if rec.GetBool("auto_restart") {
			s.tryRestart(ctx, rec, checkID, consecutiveFailures)
		}
	} else {
		_ = s.store.UpdateCheckStatus(checkID, rec.GetString("status"), consecutiveFailures, now, nextCheck)
	}
}

func (s *HealthScheduler) tryRestart(ctx context.Context, rec *core.Record, checkID string, consecutiveFailures int) {
	maxRestarts := rec.GetInt("max_restarts")
	if maxRestarts <= 0 {
		maxRestarts = 3
	}
	windowMinutes := rec.GetInt("restart_window_minutes")
	if windowMinutes <= 0 {
		windowMinutes = 60
	}

	now := time.Now().UTC()
	windowStart := now.Add(-time.Duration(windowMinutes) * time.Minute)

	// Count recent restarts.
	s.restartsMu.Lock()
	restartTimes := s.restarts[checkID]
	// Prune old entries.
	var recentRestarts []time.Time
	for _, t := range restartTimes {
		if t.After(windowStart) {
			recentRestarts = append(recentRestarts, t)
		}
	}
	s.restarts[checkID] = recentRestarts
	restartsInWindow := len(recentRestarts)
	s.restartsMu.Unlock()

	if restartsInWindow >= maxRestarts {
		// Give up — disable the check.
		_ = s.store.UpdateCheck(checkID, map[string]any{"enabled": false})

		s.bus.Publish(events.Event{
			Type:        events.HealthGaveUp,
			Severity:    events.SeverityCritical,
			ServerID:    rec.GetString("server"),
			ContainerID: rec.GetString("container"),
			Details: map[string]any{
				"check_name":     rec.GetString("name"),
				"restarts":       restartsInWindow,
				"max_restarts":   maxRestarts,
				"window_minutes": windowMinutes,
			},
			Timestamp: now,
		})
		slog.Warn("health check: gave up after max restarts",
			"check_id", checkID,
			"restarts", restartsInWindow,
		)
		return
	}

	// Restart via channel command.
	ch, ok := s.registry.Get(rec.GetString("server"))
	if !ok {
		slog.Warn("health check: cannot restart, no active channel", "check_id", checkID, "server_id", rec.GetString("server"))
		return
	}

	containerRec, err := s.app.FindRecordById(constants.CollContainers, rec.GetString("container"))
	if err != nil {
		slog.Warn("health check: cannot restart, container not found", "check_id", checkID)
		return
	}

	grpcCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	cmd := &agentpb.ChannelCommand{
		RequestId: channel.NewRequestID(),
		Priority:  1, // HIGH
		Command: &agentpb.ChannelCommand_ExecuteCommand{
			ExecuteCommand: &agentpb.ExecuteCommandRequest{
				ServerId:    rec.GetString("server"),
				ContainerId: containerRec.GetString("container_id"),
				Type:        agentpb.CommandType_COMMAND_TYPE_RESTART,
			},
		},
	}

	_, err = ch.SendCommand(grpcCtx, cmd)
	if err != nil {
		slog.Warn("health check: restart failed", "check_id", checkID, "error", err)
		return
	}

	// Record the restart.
	s.restartsMu.Lock()
	s.restarts[checkID] = append(s.restarts[checkID], now)
	s.restartsMu.Unlock()

	slog.Info("health check: restarted container", "check_id", checkID, "container", rec.GetString("container"))

	s.bus.Publish(events.Event{
		Type:        events.HealthRestart,
		Severity:    events.SeverityWarning,
		ServerID:    rec.GetString("server"),
		ContainerID: rec.GetString("container"),
		Details: map[string]any{
			"check_name":    rec.GetString("name"),
			"restart_count": restartsInWindow + 1,
		},
		Timestamp: now,
	})
}

func (s *HealthScheduler) reschedule(checkID string, intervalSeconds int) {
	if intervalSeconds <= 0 {
		intervalSeconds = 30
	}
	nextCheck := time.Now().Add(time.Duration(intervalSeconds) * time.Second)

	s.heapMu.Lock()
	defer s.heapMu.Unlock()

	heap.Push(&s.heap, &scheduledCheck{
		checkID:   checkID,
		nextCheck: nextCheck,
	})
}

