package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/pocketbase/pocketbase/core"

	"github.com/onflux-tech/dockeeper/internal/manager/channel"
	"github.com/onflux-tech/dockeeper/internal/manager/events"
	"github.com/onflux-tech/dockeeper/internal/shared/constants"
	agentpb "github.com/onflux-tech/dockeeper/proto/gen/dockeeper/agent/v1"
)

const (
	updateSagaTimeout       = 5 * time.Minute
	healthVerifyTimeout     = 60 * time.Second
	maxConcurrentPerHost    = 2
	defaultCheckSchedule    = "0 */6 * * *"
)

// UpdateOrchestrator coordinates container image update checks and saga execution.
type UpdateOrchestrator struct {
	app             core.App
	registryAdapter *RegistryAdapter
	registry        *channel.Registry
	bus             *events.Bus

	// Per-host semaphores to limit concurrent updates.
	hostSem sync.Map // serverID → chan struct{}
}

// NewUpdateOrchestrator creates a new orchestrator.
func NewUpdateOrchestrator(
	app core.App,
	registryAdapter *RegistryAdapter,
	channelReg *channel.Registry,
	bus *events.Bus,
) *UpdateOrchestrator {
	return &UpdateOrchestrator{
		app:             app,
		registryAdapter: registryAdapter,
		registry:        channelReg,
		bus:             bus,
	}
}

// App returns the underlying core.App for use in route handlers.
func (o *UpdateOrchestrator) App() core.App {
	return o.app
}

// CheckResult describes one container's update status after a check.
type CheckResult struct {
	ContainerID   string `json:"container_id"`
	ContainerName string `json:"container_name"`
	ServerID      string `json:"server_id"`
	Image         string `json:"image"`
	CurrentDigest string `json:"current_digest,omitempty"`
	RemoteDigest  string `json:"remote_digest,omitempty"`
	UpdateAvail   bool   `json:"update_available"`
	Error         string `json:"error,omitempty"`
}

// CheckForUpdates iterates all containers (with update policies), queries the
// registry for new digests, and returns a list of results.
func (o *UpdateOrchestrator) CheckForUpdates(ctx context.Context) ([]CheckResult, error) {
	containers, err := o.app.FindRecordsByFilter(
		constants.CollContainers,
		"status = 'running'",
		"", 0, 0,
	)
	if err != nil {
		return nil, fmt.Errorf("list containers: %w", err)
	}

	var (
		mu      sync.Mutex
		results []CheckResult
		wg      sync.WaitGroup
		sem     = make(chan struct{}, 10) // max 10 parallel registry checks
	)

	for _, c := range containers {
		imageRef := c.GetString("image")
		if imageRef == "" {
			continue
		}

		// Skip pinned images.
		ref := ParseImageRef(imageRef)
		if ref.Digest != "" {
			continue
		}

		wg.Add(1)
		go func(rec *core.Record) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			result := CheckResult{
				ContainerID:   rec.Id,
				ContainerName: rec.GetString("name"),
				ServerID:      rec.GetString("server"),
				Image:         rec.GetString("image"),
			}

			remoteDigest, err := o.registryAdapter.GetLatestDigest(ctx, rec.GetString("image"))
			if err != nil {
				result.Error = err.Error()
				mu.Lock()
				results = append(results, result)
				mu.Unlock()
				return
			}

			result.RemoteDigest = remoteDigest

			// Get current digest from agent.
			currentDigest := rec.GetString("update_digest")
			result.CurrentDigest = currentDigest

			if remoteDigest != "" && remoteDigest != currentDigest {
				result.UpdateAvail = true
			}

			// Update container record.
			rec.Set("update_digest", remoteDigest)
			rec.Set("last_update_check", time.Now().UTC())
			if err := o.app.Save(rec); err != nil {
				slog.Error("save update_digest failed", "container", rec.Id, "error", err)
			}

			// Publish event if update is available.
			if result.UpdateAvail && o.bus != nil {
				serverName := o.resolveServerName(result.ServerID)
				o.bus.Publish(events.Event{
					Type:          events.ImageUpdateAvailable,
					Severity:      events.SeverityInfo,
					ServerID:      result.ServerID,
					ServerName:    serverName,
					ContainerID:   rec.Id,
					ContainerName: result.ContainerName,
					Details: map[string]any{
						"image":          result.Image,
						"current_digest": currentDigest,
						"remote_digest":  remoteDigest,
					},
					Timestamp: time.Now().UTC(),
				})
			}

			mu.Lock()
			results = append(results, result)
			mu.Unlock()
		}(c)
	}

	wg.Wait()
	return results, nil
}

// ApplyPendingUpdates finds containers with mode=auto that have a new digest
// and triggers updates for them.
func (o *UpdateOrchestrator) ApplyPendingUpdates(ctx context.Context) {
	policies, err := o.app.FindRecordsByFilter(
		constants.CollUpdatePolicies,
		"enabled = true && mode = 'auto'",
		"", 0, 0,
	)
	if err != nil {
		slog.Error("list auto-update policies failed", "error", err)
		return
	}

	for _, pol := range policies {
		containerID := pol.GetString("container")
		if containerID == "" {
			continue
		}

		container, err := o.app.FindRecordById(constants.CollContainers, containerID)
		if err != nil {
			continue
		}

		// Check if update is available.
		updateDigest := container.GetString("update_digest")
		currentImage := container.GetString("image")
		if updateDigest == "" || currentImage == "" {
			continue
		}

		// Skip if container belongs to a compose stack.
		if container.GetString("compose_project") != "" {
			slog.Info("skipping compose-managed container for auto-update",
				"container", containerID,
				"project", container.GetString("compose_project"),
			)
			continue
		}

		// Check container still running.
		if container.GetString("status") != "running" {
			continue
		}

		go func(cID, sID string) {
			sagaCtx, cancel := context.WithTimeout(context.Background(), updateSagaTimeout)
			defer cancel()

			if err := o.ExecuteUpdate(sagaCtx, cID); err != nil {
				slog.Error("auto-update failed", "container", cID, "error", err)
			}
		}(containerID, container.GetString("server"))
	}
}

// ExecuteUpdate performs the full update saga for a single container.
func (o *UpdateOrchestrator) ExecuteUpdate(ctx context.Context, containerID string) error {
	container, err := o.app.FindRecordById(constants.CollContainers, containerID)
	if err != nil {
		return fmt.Errorf("find container: %w", err)
	}

	serverID := container.GetString("server")
	server, err := o.app.FindRecordById(constants.CollServers, serverID)
	if err != nil {
		return fmt.Errorf("find server: %w", err)
	}

	if server.GetString("status") != "online" {
		return fmt.Errorf("server %s is offline", serverID)
	}

	// Acquire per-host semaphore.
	sem := o.getHostSem(serverID)
	select {
	case sem <- struct{}{}:
		defer func() { <-sem }()
	case <-ctx.Done():
		return fmt.Errorf("timeout waiting for host semaphore: %w", ctx.Err())
	}

	dockerContainerID := container.GetString("container_id")
	containerName := container.GetString("name")
	oldImage := container.GetString("image")
	newImage := oldImage // same ref, but with new digest after pull
	serverName := server.GetString("hostname")

	// Resolve registry auth.
	registryAuth := o.resolveRegistryAuth(oldImage)

	// Create update_history record.
	historyRec, err := o.createHistoryRecord(containerID, serverID, oldImage, newImage)
	if err != nil {
		return fmt.Errorf("create history: %w", err)
	}

	// Get the agent's active channel.
	ch, ok := o.registry.Get(serverID)
	if !ok {
		o.failHistory(historyRec, "no active channel for agent")
		return fmt.Errorf("no active channel for server %s", serverID)
	}

	// Signal the agent to open a DataStream for the container update.
	o.updateHistoryStatus(historyRec, "pulling")

	streamID := channel.NewRequestID()
	signal := &agentpb.OpenDataStreamSignal{
		StreamId:   streamID,
		StreamType: agentpb.DataStreamType_DATA_STREAM_TYPE_CONTAINER_UPDATE,
		Params: &agentpb.OpenDataStreamSignal_ContainerUpdate{
			ContainerUpdate: &agentpb.ExecuteContainerUpdateRequest{
				ContainerId:  dockerContainerID,
				NewImage:     newImage,
				RegistryAuth: registryAuth,
			},
		},
	}

	streamCh, err := ch.SignalOpenDataStream(ctx, signal)
	if err != nil {
		o.failHistory(historyRec, fmt.Sprintf("signal data stream: %s", err))
		o.publishUpdateEvent(events.ContainerUpdateFailed, serverID, serverName, containerID, containerName, oldImage, newImage, err.Error())
		return fmt.Errorf("signal update stream: %w", err)
	}

	// Wait for the agent to open the DataStream.
	var stream agentpb.AgentDataStreamService_OpenDataStreamServer
	select {
	case <-ctx.Done():
		o.failHistory(historyRec, "timeout waiting for data stream")
		return fmt.Errorf("timeout waiting for data stream: %w", ctx.Err())
	case s, ok := <-streamCh:
		if !ok {
			o.failHistory(historyRec, "data stream channel closed")
			return fmt.Errorf("update data stream channel closed")
		}
		stream = s
	}

	// Consume progress stream from the agent.
	var lastStep string
	for {
		msg, recvErr := stream.Recv()
		if recvErr != nil {
			if lastStep == "" {
				o.failHistory(historyRec, fmt.Sprintf("stream error: %s", recvErr))
				o.publishUpdateEvent(events.ContainerUpdateFailed, serverID, serverName, containerID, containerName, oldImage, newImage, recvErr.Error())
				return fmt.Errorf("update stream: %w", recvErr)
			}
			break
		}

		if text := msg.GetText(); text != nil {
			line := text.GetLine()
			step := extractUpdateStep(line)
			if step != "" {
				lastStep = step
			}

			slog.Debug("update progress",
				"container", containerID,
				"step", lastStep,
				"line", line,
			)

			// Map agent steps to history status.
			switch lastStep {
			case "pulling":
				o.updateHistoryStatus(historyRec, "pulling")
			case "snapshot", "stopping", "creating", "starting", "replacing":
				o.updateHistoryStatus(historyRec, "replacing")
			case "verifying":
				o.updateHistoryStatus(historyRec, "verifying")
			case "done":
				o.completeHistory(historyRec, "success")
				o.publishUpdateEvent(events.ContainerUpdateCompleted, serverID, serverName, containerID, containerName, oldImage, newImage, "")
				return nil
			case "rolledback":
				o.completeHistory(historyRec, "rolledback")
				o.publishUpdateEvent(events.ContainerUpdateRolledback, serverID, serverName, containerID, containerName, oldImage, newImage, line)
				return fmt.Errorf("update rolled back: %s", line)
			case "failed":
				o.failHistory(historyRec, line)
				o.publishUpdateEvent(events.ContainerUpdateFailed, serverID, serverName, containerID, containerName, oldImage, newImage, line)
				return fmt.Errorf("update failed: %s", line)
			}
		}

		if complete := msg.GetComplete(); complete != nil {
			o.completeHistory(historyRec, "success")
			o.publishUpdateEvent(events.ContainerUpdateCompleted, serverID, serverName, containerID, containerName, oldImage, newImage, "")
			return nil
		}

		if errPayload := msg.GetError(); errPayload != nil {
			errMsg := errPayload.GetMessage()
			o.failHistory(historyRec, errMsg)
			o.publishUpdateEvent(events.ContainerUpdateFailed, serverID, serverName, containerID, containerName, oldImage, newImage, errMsg)
			return fmt.Errorf("update failed: %s", errMsg)
		}
	}

	// If stream ended cleanly without "done", treat as failure.
	if lastStep == "" {
		o.failHistory(historyRec, "update stream ended without result")
		return fmt.Errorf("update stream ended without result")
	}

	return nil
}

// RunCheckAndApply is the cron job entry point: check + auto-apply.
func (o *UpdateOrchestrator) RunCheckAndApply() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	results, err := o.CheckForUpdates(ctx)
	if err != nil {
		slog.Error("update check failed", "error", err)
		return
	}

	updatesAvailable := 0
	for _, r := range results {
		if r.UpdateAvail {
			updatesAvailable++
		}
	}

	slog.Info("update check completed",
		"total_checked", len(results),
		"updates_available", updatesAvailable,
	)

	if updatesAvailable > 0 {
		o.ApplyPendingUpdates(ctx)
	}
}

// ─── Helpers ────────────────────────────────────────────

func (o *UpdateOrchestrator) getHostSem(serverID string) chan struct{} {
	val, loaded := o.hostSem.LoadOrStore(serverID, make(chan struct{}, maxConcurrentPerHost))
	if !loaded {
		return val.(chan struct{})
	}
	return val.(chan struct{})
}

func (o *UpdateOrchestrator) resolveServerName(serverID string) string {
	rec, err := o.app.FindRecordById(constants.CollServers, serverID)
	if err != nil {
		return ""
	}
	return rec.GetString("hostname")
}

func (o *UpdateOrchestrator) resolveRegistryAuth(imageRef string) string {
	ref := ParseImageRef(imageRef)

	o.registryAdapter.mu.RLock()
	cred, ok := o.registryAdapter.credentials[ref.Registry]
	o.registryAdapter.mu.RUnlock()

	if !ok || cred.Username == "" {
		return ""
	}

	authJSON, _ := json.Marshal(map[string]string{
		"username": cred.Username,
		"password": cred.Password,
	})
	return string(authJSON)
}

func (o *UpdateOrchestrator) createHistoryRecord(containerID, serverID, oldImage, newImage string) (*core.Record, error) {
	col, err := o.app.FindCachedCollectionByNameOrId(constants.CollUpdateHistory)
	if err != nil {
		return nil, err
	}

	rec := core.NewRecord(col)
	rec.Set("container", containerID)
	rec.Set("server", serverID)
	rec.Set("old_image", oldImage)
	rec.Set("new_image", newImage)
	rec.Set("status", "pending")
	rec.Set("started_at", time.Now().UTC())

	if err := o.app.Save(rec); err != nil {
		return nil, err
	}
	return rec, nil
}

func (o *UpdateOrchestrator) updateHistoryStatus(rec *core.Record, status string) {
	rec.Set("status", status)
	if err := o.app.Save(rec); err != nil {
		slog.Error("update history status failed", "id", rec.Id, "status", status, "error", err)
	}
}

func (o *UpdateOrchestrator) completeHistory(rec *core.Record, status string) {
	rec.Set("status", status)
	rec.Set("completed_at", time.Now().UTC())
	if err := o.app.Save(rec); err != nil {
		slog.Error("complete history failed", "id", rec.Id, "status", status, "error", err)
	}
}

func (o *UpdateOrchestrator) failHistory(rec *core.Record, errMsg string) {
	rec.Set("status", "failed")
	rec.Set("error", errMsg)
	rec.Set("completed_at", time.Now().UTC())
	if err := o.app.Save(rec); err != nil {
		slog.Error("fail history failed", "id", rec.Id, "error", err)
	}
}

func (o *UpdateOrchestrator) publishUpdateEvent(
	eventType events.EventType,
	serverID, serverName, containerID, containerName, oldImage, newImage, errMsg string,
) {
	if o.bus == nil {
		return
	}

	details := map[string]any{
		"old_image": oldImage,
		"new_image": newImage,
	}
	if errMsg != "" {
		details["error"] = errMsg
	}

	o.bus.Publish(events.Event{
		Type:          eventType,
		Severity:      events.SeverityFor(eventType),
		ServerID:      serverID,
		ServerName:    serverName,
		ContainerID:   containerID,
		ContainerName: containerName,
		Details:       details,
		Timestamp:     time.Now().UTC(),
	})
}

// extractUpdateStep parses the step name from a DataStreamText line
// in the format "[step] message (progress%)".
func extractUpdateStep(line string) string {
	if len(line) > 2 && line[0] == '[' {
		if idx := strings.IndexByte(line, ']'); idx > 0 {
			return line[1:idx]
		}
	}
	return ""
}
