package collector

import (
	"context"
	"crypto/sha256"
	"log/slog"
	"sort"
	"strings"
	"sync"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	agentpb "github.com/onflux-tech/dockeeper/proto/gen/dockeeper/agent/v1"
)

const (
	discoveryInterval         = 30 * time.Second
	discoveryFullSyncInterval = 10 * time.Minute
	discoveryEventRetry       = 30 * time.Second
)

// DiscoveryCollector periodically collects container, image, network and volume
// data from the orchestrator and sends it to the manager via gRPC. It uses
// hash-based diff detection to avoid sending unchanged data.
type DiscoveryCollector struct {
	orch         OrchestratorClient
	client       agentpb.AgentDiscoveryServiceClient
	serverID     string
	logger       *slog.Logger
	gate         *ConnectionGate
	hashes       map[string][32]byte
	mu           sync.Mutex
	containers   map[string]*agentpb.ContainerInfo
	dirty        map[string]struct{}
	removed      map[string]struct{}
	lastFullSync time.Time
	onServerGone func() // called when the manager reports the server was deleted
}

// NewDiscoveryCollector creates a new discovery collector.
func NewDiscoveryCollector(
	orch OrchestratorClient,
	client agentpb.AgentDiscoveryServiceClient,
	serverID string,
	gate *ConnectionGate,
	logger *slog.Logger,
) *DiscoveryCollector {
	return &DiscoveryCollector{
		orch:       orch,
		client:     client,
		serverID:   serverID,
		logger:     logger,
		gate:       gate,
		hashes:     make(map[string][32]byte),
		containers: make(map[string]*agentpb.ContainerInfo),
		dirty:      make(map[string]struct{}),
		removed:    make(map[string]struct{}),
	}
}

// SetServerGoneCallback sets the function called when a discovery report returns
// NotFound, indicating the server record was deleted from the manager.
func (d *DiscoveryCollector) SetServerGoneCallback(fn func()) {
	d.onServerGone = fn
}

// Run starts the periodic discovery loop. It blocks until ctx is cancelled.
func (d *DiscoveryCollector) Run(ctx context.Context) {
	go d.watchContainerEvents(ctx)

	// Collect immediately on start.
	d.collect(ctx)

	ticker := time.NewTicker(discoveryInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			d.collect(ctx)
		}
	}
}

func (d *DiscoveryCollector) collect(ctx context.Context) {
	if !d.gate.Allow() {
		return
	}

	d.reportContainers(ctx)
	d.reportImages(ctx)
	d.reportNetworks(ctx)
	d.reportVolumes(ctx)
}

func (d *DiscoveryCollector) refreshContainers(ctx context.Context) error {
	d.mu.Lock()
	full := d.lastFullSync.IsZero() || time.Since(d.lastFullSync) >= discoveryFullSyncInterval
	d.mu.Unlock()

	if full {
		return d.fullSync(ctx)
	}
	d.incrementalSync(ctx)
	return nil
}

func (d *DiscoveryCollector) fullSync(ctx context.Context) error {
	d.mu.Lock()
	d.dirty = make(map[string]struct{})
	d.mu.Unlock()

	containers, err := d.orch.ListContainers(ctx)
	if err != nil {
		return err
	}

	cache := make(map[string]*agentpb.ContainerInfo, len(containers))
	for _, c := range containers {
		cache[c.GetContainerId()] = c
	}

	d.mu.Lock()
	for id := range d.removed {
		delete(cache, id)
	}
	d.removed = make(map[string]struct{})
	d.containers = cache
	d.lastFullSync = time.Now()
	d.mu.Unlock()
	return nil
}

func (d *DiscoveryCollector) incrementalSync(ctx context.Context) {
	d.mu.Lock()
	ids := make([]string, 0, len(d.dirty))
	for id := range d.dirty {
		ids = append(ids, id)
	}
	d.dirty = make(map[string]struct{})
	d.mu.Unlock()

	for _, id := range ids {
		info, err := d.orch.InspectContainer(ctx, id)
		if err != nil {
			d.logger.Debug("discovery: inspect changed container failed", "container", id, "error", err)
			d.mu.Lock()
			if _, removed := d.removed[id]; !removed {
				d.dirty[id] = struct{}{}
			}
			d.mu.Unlock()
			continue
		}
		if info == nil {
			continue
		}

		d.mu.Lock()
		if _, removed := d.removed[id]; !removed {
			d.containers[id] = info
		}
		d.mu.Unlock()
	}
}

func (d *DiscoveryCollector) containerSnapshot() []*agentpb.ContainerInfo {
	d.mu.Lock()
	defer d.mu.Unlock()

	containers := make([]*agentpb.ContainerInfo, 0, len(d.containers))
	for _, c := range d.containers {
		containers = append(containers, c)
	}
	sort.Slice(containers, func(i, j int) bool {
		return containers[i].GetContainerId() < containers[j].GetContainerId()
	})
	return containers
}

func (d *DiscoveryCollector) handleContainerEvent(ev ContainerEvent) {
	id := containerEventID(ev.ContainerID)
	action := strings.ToLower(ev.Action)
	if strings.HasPrefix(action, "health_status") {
		action = "health_status"
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	switch action {
	case "destroy":
		delete(d.containers, id)
		delete(d.dirty, id)
		d.removed[id] = struct{}{}
	case "create", "start", "die", "update", "rename", "health_status", "stop", "kill", "pause", "unpause":
		delete(d.removed, id)
		d.dirty[id] = struct{}{}
	}
}

func (d *DiscoveryCollector) watchContainerEvents(ctx context.Context) {
	for {
		err := d.consumeContainerEvents(ctx)
		if ctx.Err() != nil {
			return
		}
		if err != nil {
			d.logger.Warn("discovery: container events stream failed", "error", err)
		}

		select {
		case <-ctx.Done():
			return
		case <-time.After(discoveryEventRetry):
		}
	}
}

func (d *DiscoveryCollector) consumeContainerEvents(ctx context.Context) error {
	events, errs := d.orch.WatchContainerEvents(ctx)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case ev, ok := <-events:
			if !ok {
				return nil
			}
			d.handleContainerEvent(ev)
		case err, ok := <-errs:
			if !ok {
				return nil
			}
			if err != nil {
				return err
			}
		}
	}
}

func containerEventID(id string) string {
	if len(id) > 12 {
		return id[:12]
	}
	return id
}

func (d *DiscoveryCollector) reportContainers(ctx context.Context) {
	if err := d.refreshContainers(ctx); err != nil {
		d.logger.Error("discovery: list containers failed", "error", err)
		return
	}

	req := &agentpb.ReportContainersRequest{
		ServerId:   d.serverID,
		Containers: d.containerSnapshot(),
	}

	if !d.changed("containers", req) {
		return
	}

	resp, err := d.client.ReportContainers(ctx, req)
	if err != nil {
		d.handleRPCError("containers", err)
		return
	}
	d.gate.Succeeded()
	d.logger.Info("discovery: containers reported", "count", resp.GetReceivedCount())
}

func (d *DiscoveryCollector) reportImages(ctx context.Context) {
	images, err := d.orch.ListImages(ctx)
	if err != nil {
		d.logger.Error("discovery: list images failed", "error", err)
		return
	}

	req := &agentpb.ReportImagesRequest{
		ServerId: d.serverID,
		Images:   images,
	}

	if !d.changed("images", req) {
		return
	}

	resp, err := d.client.ReportImages(ctx, req)
	if err != nil {
		d.handleRPCError("images", err)
		return
	}
	d.gate.Succeeded()
	d.logger.Info("discovery: images reported", "count", resp.GetReceivedCount())
}

func (d *DiscoveryCollector) reportNetworks(ctx context.Context) {
	networks, err := d.orch.ListNetworks(ctx)
	if err != nil {
		d.logger.Error("discovery: list networks failed", "error", err)
		return
	}

	req := &agentpb.ReportNetworksRequest{
		ServerId: d.serverID,
		Networks: networks,
	}

	if !d.changed("networks", req) {
		return
	}

	resp, err := d.client.ReportNetworks(ctx, req)
	if err != nil {
		d.handleRPCError("networks", err)
		return
	}
	d.gate.Succeeded()
	d.logger.Info("discovery: networks reported", "count", resp.GetReceivedCount())
}

func (d *DiscoveryCollector) reportVolumes(ctx context.Context) {
	volumes, err := d.orch.ListVolumes(ctx)
	if err != nil {
		d.logger.Error("discovery: list volumes failed", "error", err)
		return
	}

	req := &agentpb.ReportVolumesRequest{
		ServerId: d.serverID,
		Volumes:  volumes,
	}

	if !d.changed("volumes", req) {
		return
	}

	resp, err := d.client.ReportVolumes(ctx, req)
	if err != nil {
		d.handleRPCError("volumes", err)
		return
	}
	d.gate.Succeeded()
	d.logger.Info("discovery: volumes reported", "count", resp.GetReceivedCount())
}

// handleRPCError logs the error and triggers the server-gone callback when
// the manager returned NotFound (server deleted) or when the error indicates
// an mTLS/certificate failure (e.g. the manager CA was rotated or the DB was
// recreated). In both cases the agent clears its enrollment and re-enrolls.
func (d *DiscoveryCollector) handleRPCError(resource string, err error) {
	d.gate.Failed()
	delete(d.hashes, resource)
	if status.Code(err) == codes.NotFound && d.onServerGone != nil {
		d.logger.Warn("discovery: server not found on manager, triggering re-enrollment",
			"resource", resource)
		d.onServerGone()
		return
	}
	if IsAuthError(err) && d.onServerGone != nil {
		d.logger.Warn("discovery: mTLS authentication failed, certificates may be invalid — triggering re-enrollment",
			"resource", resource, "error", err)
		d.onServerGone()
		return
	}
	d.logger.Error("discovery: report failed", "resource", resource, "error", err)
}

// changed returns true if the proto message has changed since last check.
func (d *DiscoveryCollector) changed(key string, msg proto.Message) bool {
	data, err := proto.Marshal(msg)
	if err != nil {
		d.logger.Warn("discovery: failed to marshal for diff", "key", key, "error", err)
		return true // send on error to be safe
	}

	h := sha256.Sum256(data)
	prev, exists := d.hashes[key]
	if exists && prev == h {
		return false
	}
	d.hashes[key] = h
	return true
}
