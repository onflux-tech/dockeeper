package services

import (
	"log/slog"
	"time"

	"github.com/pocketbase/pocketbase/core"

	"github.com/onflux-tech/dockeeper/internal/shared/constants"
)

// BlackoutService manages blackout windows for alert suppression.
type BlackoutService struct {
	app core.App
}

// NewBlackoutService creates a BlackoutService.
func NewBlackoutService(app core.App) *BlackoutService {
	return &BlackoutService{app: app}
}

// IsBlackedOut checks if the given server is currently in a blackout window.
func (s *BlackoutService) IsBlackedOut(serverID string) bool {
	now := time.Now().UTC()

	windows, err := s.app.FindRecordsByFilter(
		constants.CollBlackoutWindows,
		"enabled = true",
		"", 0, 0,
	)
	if err != nil {
		return false
	}

	for _, w := range windows {
		if !s.matchesServer(w, serverID) {
			continue
		}

		if w.GetBool("recurring") {
			if s.isInRecurringWindow(w, now) {
				return true
			}
		} else {
			start := w.GetDateTime("start_time").Time()
			end := w.GetDateTime("end_time").Time()
			if now.After(start) && now.Before(end) {
				return true
			}
		}
	}

	return false
}

// matchesServer checks if a blackout window applies to the given server.
func (s *BlackoutService) matchesServer(w *core.Record, serverID string) bool {
	raw := w.Get("server_ids")
	if raw == nil {
		return true // empty = all servers
	}

	switch v := raw.(type) {
	case []any:
		if len(v) == 0 {
			return true
		}
		for _, item := range v {
			if id, ok := item.(string); ok && id == serverID {
				return true
			}
		}
		return false
	case []string:
		if len(v) == 0 {
			return true
		}
		for _, id := range v {
			if id == serverID {
				return true
			}
		}
		return false
	default:
		return true
	}
}

// isInRecurringWindow checks if 'now' falls within a recurring blackout window.
// Uses simple day-of-week + hour/minute matching from the reference start_time.
func (s *BlackoutService) isInRecurringWindow(w *core.Record, now time.Time) bool {
	refStart := w.GetDateTime("start_time").Time()
	durationMin := w.GetInt("duration_minutes")
	if durationMin <= 0 {
		durationMin = 60 // default 1 hour
	}

	// Match day-of-week and time.
	if refStart.Weekday() != now.Weekday() {
		return false
	}

	// Build today's window start using ref time.
	todayStart := time.Date(now.Year(), now.Month(), now.Day(),
		refStart.Hour(), refStart.Minute(), 0, 0, time.UTC)
	todayEnd := todayStart.Add(time.Duration(durationMin) * time.Minute)

	if now.After(todayStart) && now.Before(todayEnd) {
		slog.Debug("blackout: in recurring window", "window", w.Id)
		return true
	}

	return false
}

// ListWindows returns all blackout windows.
func (s *BlackoutService) ListWindows() ([]*core.Record, error) {
	return s.app.FindRecordsByFilter(constants.CollBlackoutWindows, "", "-created", 200, 0)
}

// CreateWindow creates a new blackout window.
func (s *BlackoutService) CreateWindow(fields map[string]any) (string, error) {
	col, err := s.app.FindCollectionByNameOrId(constants.CollBlackoutWindows)
	if err != nil {
		return "", err
	}

	rec := core.NewRecord(col)
	for k, v := range fields {
		rec.Set(k, v)
	}
	if err := s.app.Save(rec); err != nil {
		return "", err
	}
	return rec.Id, nil
}

// UpdateWindow updates a blackout window.
func (s *BlackoutService) UpdateWindow(id string, fields map[string]any) error {
	rec, err := s.app.FindRecordById(constants.CollBlackoutWindows, id)
	if err != nil {
		return err
	}
	for k, v := range fields {
		rec.Set(k, v)
	}
	return s.app.Save(rec)
}

// DeleteWindow deletes a blackout window.
func (s *BlackoutService) DeleteWindow(id string) error {
	rec, err := s.app.FindRecordById(constants.CollBlackoutWindows, id)
	if err != nil {
		return err
	}
	return s.app.Delete(rec)
}
