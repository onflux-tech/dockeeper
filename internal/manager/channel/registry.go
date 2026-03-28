package channel

import (
	"log/slog"
	"sync"
	"time"

	"github.com/pocketbase/pocketbase/core"

	"github.com/onflux-tech/dockeeper/internal/manager/events"
	"github.com/onflux-tech/dockeeper/internal/shared/constants"
)

// Registry tracks connected agents and their active channels.
// Thread-safe for concurrent access from gRPC handlers and REST routes.
type Registry struct {
	mu       sync.RWMutex
	channels map[string]*AgentChannel // keyed by serverID
	app      core.App
	bus      *events.Bus
}

// NewRegistry creates a new channel registry.
func NewRegistry(app core.App, bus *events.Bus) *Registry {
	return &Registry{
		channels: make(map[string]*AgentChannel),
		app:      app,
		bus:      bus,
	}
}

// Register adds or replaces the channel for a server.
// If an older channel exists it is closed first.
func (r *Registry) Register(serverID string, ch *AgentChannel) {
	r.mu.Lock()
	old, existed := r.channels[serverID]
	r.channels[serverID] = ch
	r.mu.Unlock()

	if existed {
		slog.Warn("replacing existing channel for agent", "server_id", serverID)
		old.Close()
	}

	r.setServerStatus(serverID, "online")

	slog.Info("agent channel registered", "server_id", serverID)

	if r.bus != nil {
		serverName := r.serverName(serverID)
		r.bus.Publish(events.Event{
			Type:       events.ServerOnline,
			Severity:   events.SeverityInfo,
			ServerID:   serverID,
			ServerName: serverName,
			Timestamp:  time.Now(),
		})
	}
}

// Unregister removes the channel for a server and marks it offline.
func (r *Registry) Unregister(serverID string) {
	r.mu.Lock()
	ch, existed := r.channels[serverID]
	delete(r.channels, serverID)
	r.mu.Unlock()

	if !existed {
		return
	}

	ch.Close()
	r.setServerStatus(serverID, "offline")

	slog.Info("agent channel unregistered", "server_id", serverID)

	if r.bus != nil {
		serverName := r.serverName(serverID)
		r.bus.Publish(events.Event{
			Type:       events.ServerOffline,
			Severity:   events.SeverityCritical,
			ServerID:   serverID,
			ServerName: serverName,
			Timestamp:  time.Now(),
		})
	}
}

// Get returns the active channel for a server, or nil if offline.
func (r *Registry) Get(serverID string) (*AgentChannel, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	ch, ok := r.channels[serverID]
	return ch, ok
}

// IsOnline returns true if the server has an active command channel.
func (r *Registry) IsOnline(serverID string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.channels[serverID]
	return ok
}

// List returns all currently connected server IDs.
func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	ids := make([]string, 0, len(r.channels))
	for id := range r.channels {
		ids = append(ids, id)
	}
	return ids
}

// Count returns the number of connected agents.
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.channels)
}

// setServerStatus updates the server record status and last_seen in PocketBase.
func (r *Registry) setServerStatus(serverID, status string) {
	record, err := r.app.FindRecordById(constants.CollServers, serverID)
	if err != nil {
		slog.Warn("channel registry: failed to find server", "server_id", serverID, "error", err)
		return
	}

	record.Set("status", status)
	record.Set("last_seen", time.Now().UTC().Format(time.RFC3339))

	if err := r.app.Save(record); err != nil {
		slog.Warn("channel registry: failed to update server status", "server_id", serverID, "status", status, "error", err)
	}
}

// serverName fetches the server name for event publishing.
func (r *Registry) serverName(serverID string) string {
	record, err := r.app.FindRecordById(constants.CollServers, serverID)
	if err != nil {
		return serverID
	}
	name := record.GetString("name")
	if name == "" {
		return record.GetString("hostname")
	}
	return name
}
