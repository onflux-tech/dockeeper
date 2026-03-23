package store

import (
	"fmt"
	"log/slog"
	"math"
	"time"

	"github.com/pocketbase/pocketbase/core"

	"github.com/jkdesenvolvedor/dockeeper/internal/shared/constants"
)

// TerminalStore manages terminal_sessions records.
type TerminalStore struct {
	app core.App
}

// NewTerminalStore creates a TerminalStore.
func NewTerminalStore(app core.App) *TerminalStore {
	return &TerminalStore{app: app}
}

// CreateSession creates a new terminal_sessions record and returns its ID.
func (s *TerminalStore) CreateSession(containerID, serverID, userID, command string) (string, error) {
	col, err := s.app.FindCachedCollectionByNameOrId(constants.CollTerminalSessions)
	if err != nil {
		return "", fmt.Errorf("terminal_sessions collection: %w", err)
	}

	record := core.NewRecord(col)
	record.Set("container", containerID)
	record.Set("server", serverID)
	record.Set("user", userID)
	record.Set("started_at", time.Now().UTC().Format("2006-01-02 15:04:05.000Z"))

	if err := s.app.Save(record); err != nil {
		return "", fmt.Errorf("create terminal session: %w", err)
	}

	return record.Id, nil
}

// EndSession sets ended_at and duration on a terminal session.
func (s *TerminalStore) EndSession(sessionID string) {
	record, err := s.app.FindRecordById(constants.CollTerminalSessions, sessionID)
	if err != nil {
		slog.Warn("terminal session not found for close", "session_id", sessionID, "error", err)
		return
	}

	now := time.Now().UTC()
	record.Set("ended_at", now.Format("2006-01-02 15:04:05.000Z"))

	startedAt := record.GetDateTime("started_at").Time()
	if !startedAt.IsZero() {
		dur := int(math.Round(now.Sub(startedAt).Seconds()))
		record.Set("duration", dur)
	}

	if err := s.app.Save(record); err != nil {
		slog.Warn("failed to end terminal session", "session_id", sessionID, "error", err)
	}
}
