package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/pocketbase/pocketbase/core"

	"github.com/onflux-tech/dockeeper/internal/manager/store"
	agentpb "github.com/onflux-tech/dockeeper/proto/gen/dockeeper/agent/v1"
)

// ActionResult describes the outcome of executing an action token.
type ActionTokenResult struct {
	Action  string `json:"action"`
	Message string `json:"message"`
}

// ActionTokenService manages single-use notification action tokens.
type ActionTokenService struct {
	store        *store.ActionTokenStore
	containerSvc *ContainerService
	app          core.App
}

// NewActionTokenService creates an ActionTokenService.
func NewActionTokenService(app core.App, tokenStore *store.ActionTokenStore, containerSvc *ContainerService) *ActionTokenService {
	return &ActionTokenService{
		store:        tokenStore,
		containerSvc: containerSvc,
		app:          app,
	}
}

// DefaultTTL for action tokens.
const DefaultActionTokenTTL = 1 * time.Hour

// Create generates a new action token and returns the token string.
func (s *ActionTokenService) Create(action string, params map[string]string, createdBy string) (string, error) {
	return s.store.CreateToken(action, params, DefaultActionTokenTTL, createdBy)
}

// CreateWithTTL generates a new action token with a custom TTL.
func (s *ActionTokenService) CreateWithTTL(action string, params map[string]string, ttl time.Duration, createdBy string) (string, error) {
	return s.store.CreateToken(action, params, ttl, createdBy)
}

// Execute validates and executes an action token.
func (s *ActionTokenService) Execute(token string) (*ActionTokenResult, error) {
	rec, err := s.store.GetByToken(token)
	if err != nil {
		return nil, fmt.Errorf("token not found")
	}

	// Check if already used
	if rec.GetBool("used") {
		return nil, fmt.Errorf("token already used")
	}

	// Check expiry
	expiresAt := rec.GetDateTime("expires_at")
	if !expiresAt.IsZero() && expiresAt.Time().Before(time.Now().UTC()) {
		return nil, fmt.Errorf("token expired")
	}

	// Parse params
	paramsRaw := rec.GetString("params")
	var params map[string]string
	if err := json.Unmarshal([]byte(paramsRaw), &params); err != nil {
		params = make(map[string]string)
	}

	action := rec.GetString("action")

	// Execute the action
	result, execErr := s.executeAction(action, params)

	// Mark token as used regardless of execution result
	if markErr := s.store.MarkUsed(rec); markErr != nil {
		slog.Error("failed to mark action token used", "token", token[:8], "error", markErr)
	}

	if execErr != nil {
		return nil, fmt.Errorf("action failed: %w", execErr)
	}

	return result, nil
}

// executeAction dispatches to the appropriate handler based on action type.
func (s *ActionTokenService) executeAction(action string, params map[string]string) (*ActionTokenResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	switch action {
	case "restart_container":
		containerID := params["container_id"]
		if containerID == "" {
			return nil, fmt.Errorf("missing container_id")
		}
		_, err := s.containerSvc.ExecuteAction(ctx, containerID, agentpb.CommandType_COMMAND_TYPE_RESTART, nil, "action_token", "")
		if err != nil {
			return nil, err
		}
		return &ActionTokenResult{
			Action:  action,
			Message: "Container restarted successfully",
		}, nil

	case "ack_alert":
		ruleID := params["rule_id"]
		if ruleID == "" {
			return nil, fmt.Errorf("missing rule_id")
		}
		// Acknowledge: just mark the alert rule as acknowledged (no specific action)
		return &ActionTokenResult{
			Action:  action,
			Message: "Alert acknowledged",
		}, nil

	case "silence_alert":
		ruleID := params["rule_id"]
		if ruleID == "" {
			return nil, fmt.Errorf("missing rule_id")
		}
		// Silence for 4 hours: update the rule's silenced_until field
		rec, err := s.app.FindRecordById("alert_rules", ruleID)
		if err != nil {
			return nil, fmt.Errorf("alert rule not found")
		}
		silencedUntil := time.Now().UTC().Add(4 * time.Hour).Format(time.RFC3339)
		rec.Set("silenced_until", silencedUntil)
		if err := s.app.Save(rec); err != nil {
			return nil, fmt.Errorf("failed to silence alert: %w", err)
		}
		return &ActionTokenResult{
			Action:  action,
			Message: "Alert silenced for 4 hours",
		}, nil

	case "approve_update":
		return &ActionTokenResult{
			Action:  action,
			Message: "Update approved",
		}, nil

	default:
		return nil, fmt.Errorf("unknown action: %s", action)
	}
}

// Cleanup removes expired action tokens. Called by the daily cleanup job.
func (s *ActionTokenService) Cleanup() {
	deleted, err := s.store.DeleteExpired()
	if err != nil {
		slog.Error("action token cleanup failed", "error", err)
		return
	}
	if deleted > 0 {
		slog.Info("action tokens cleaned up", "deleted", deleted)
	}
}
