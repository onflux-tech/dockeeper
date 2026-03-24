package services

import (
	"fmt"

	"github.com/pocketbase/pocketbase/core"

	"github.com/onflux-tech/dockeeper/internal/manager/store"
	"github.com/onflux-tech/dockeeper/internal/shared/constants"
)

const agentExecPort = "50052"

// SessionInfo holds the details needed to establish a WS↔gRPC bridge.
type SessionInfo struct {
	SessionID         string
	AgentAddr         string
	DockerContainerID string
	ServerID          string
}

// TerminalService handles terminal session lifecycle.
type TerminalService struct {
	app        core.App
	termStore  *store.TerminalStore
	auditStore *store.AuditStore
}

// NewTerminalService creates a TerminalService.
func NewTerminalService(app core.App, termStore *store.TerminalStore, auditStore *store.AuditStore) *TerminalService {
	return &TerminalService{
		app:        app,
		termStore:  termStore,
		auditStore: auditStore,
	}
}

// OpenSession resolves the container, validates the server, creates a session
// record, logs the audit event, and returns connection details.
func (s *TerminalService) OpenSession(containerID, userID, command, ip string) (*SessionInfo, error) {
	// 1. Find container record.
	container, err := s.app.FindRecordById(constants.CollContainers, containerID)
	if err != nil {
		return nil, ErrContainerNotFound
	}

	// 2. Resolve server.
	serverID := container.GetString("server")
	server, err := s.app.FindRecordById(constants.CollServers, serverID)
	if err != nil {
		return nil, ErrServerNotFound
	}

	// 3. Check server is online.
	if server.GetString("status") != "online" {
		return nil, ErrServerOffline
	}

	agentAddr := server.GetString("ip_address") + ":" + agentExecPort
	dockerContainerID := container.GetString("container_id")

	// 4. Create terminal session record.
	sessionID, err := s.termStore.CreateSession(containerID, serverID, userID, command)
	if err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}

	// 5. Audit log.
	s.auditStore.Log("terminal.open", userID, "container", containerID, ip, map[string]any{
		"session_id":     sessionID,
		"container_name": container.GetString("name"),
		"server_id":      serverID,
		"command":         command,
	})

	return &SessionInfo{
		SessionID:         sessionID,
		AgentAddr:         agentAddr,
		DockerContainerID: dockerContainerID,
		ServerID:          serverID,
	}, nil
}

// CloseSession ends a terminal session and logs the audit event.
func (s *TerminalService) CloseSession(sessionID, userID, containerID, ip string) {
	s.termStore.EndSession(sessionID)

	s.auditStore.Log("terminal.close", userID, "container", containerID, ip, map[string]any{
		"session_id": sessionID,
	})
}
