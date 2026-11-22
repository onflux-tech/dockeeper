package services

import (
	"context"
	"errors"
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

var (
	ErrHostAppNotFound    = errors.New("host app not found")
	ErrHostActionInFlight = errors.New("host app action already in flight")
	ErrHostActionFailed   = errors.New("host app action failed")
	ErrInvalidHostAction  = errors.New("invalid host app action")
)

const maxActionOutput = 2048

type HostCommandDispatcher interface {
	SendHostCommand(ctx context.Context, serverID string, command *agentpb.ChannelCommand) (*agentpb.ChannelCommandResult, error)
}

type RegistryHostDispatcher struct {
	registry *channel.Registry
}

func NewRegistryHostDispatcher(registry *channel.Registry) *RegistryHostDispatcher {
	return &RegistryHostDispatcher{registry: registry}
}

func (d *RegistryHostDispatcher) SendHostCommand(ctx context.Context, serverID string, command *agentpb.ChannelCommand) (*agentpb.ChannelCommandResult, error) {
	agentChannel, ok := d.registry.Get(serverID)
	if !ok {
		return nil, ErrServerOffline
	}
	return agentChannel.SendCommand(ctx, command)
}

type HostActionService struct {
	app        core.App
	dispatcher HostCommandDispatcher
	store      *store.HostStore
	bus        *events.Bus
	audit      *store.AuditStore
	locks      sync.Map
}

func NewHostActionService(app core.App, dispatcher HostCommandDispatcher, bus *events.Bus, audit *store.AuditStore) *HostActionService {
	return &HostActionService{
		app:        app,
		dispatcher: dispatcher,
		store:      store.NewHostStore(app),
		bus:        bus,
		audit:      audit,
	}
}

func (s *HostActionService) Execute(ctx context.Context, serverID, source, name, action, userID, ip string) (string, error) {
	if source != "systemd" && source != "pm2" {
		return "", fmt.Errorf("%w: source %s", ErrInvalidHostAction, source)
	}
	switch action {
	case "start", "stop", "restart":
	default:
		return "", fmt.Errorf("%w: action %s", ErrInvalidHostAction, action)
	}
	if name == "" {
		return "", fmt.Errorf("%w: name is required", ErrInvalidHostAction)
	}

	record, err := s.app.FindFirstRecordByFilter(
		constants.CollHostApps,
		"server = {:server} && source = {:source} && name = {:name}",
		map[string]any{"server": serverID, "source": source, "name": name},
	)
	if err != nil {
		return "", fmt.Errorf("%w: %s/%s", ErrHostAppNotFound, source, name)
	}

	lockKey := serverID + "\x00" + source + "\x00" + name
	if _, loaded := s.locks.LoadOrStore(lockKey, struct{}{}); loaded {
		return "", fmt.Errorf("%w: %s/%s", ErrHostActionInFlight, source, name)
	}
	defer s.locks.Delete(lockKey)

	command := &agentpb.ChannelCommand{
		RequestId: channel.NewRequestID(),
		Priority:  1,
		Command: &agentpb.ChannelCommand_HostAppAction{
			HostAppAction: &agentpb.HostAppActionCommand{Source: source, Name: name, Action: action},
		},
	}

	commandCtx, cancel := context.WithTimeout(ctx, channel.DefaultCommandTimeout)
	defer cancel()

	result, err := s.dispatcher.SendHostCommand(commandCtx, serverID, command)
	if err != nil {
		s.logAction(record.Id, serverID, source, name, action, userID, ip, false, err.Error())
		return "", err
	}

	actionResult := result.GetHostAppAction()
	if actionResult == nil {
		message := result.GetError().GetMessage()
		if message == "" {
			message = "empty host action response"
		}
		s.logAction(record.Id, serverID, source, name, action, userID, ip, false, message)
		return "", fmt.Errorf("%w: %s", ErrHostActionFailed, message)
	}
	if !actionResult.GetSuccess() {
		message := actionResult.GetError()
		if message == "" {
			message = "host action failed"
		}
		s.logAction(record.Id, serverID, source, name, action, userID, ip, false, message)
		return "", fmt.Errorf("%w: %s", ErrHostActionFailed, message)
	}

	status := "running"
	if action == "stop" {
		status = "stopped"
	}
	if err := s.store.UpdateAppStatus(serverID, source, name, status); err != nil {
		slog.Error("host action: optimistic status update failed",
			"server_id", serverID, "source", source, "name", name, "error", err)
	}

	s.logAction(record.Id, serverID, source, name, action, userID, ip, true, actionResult.GetOutput())

	if s.bus != nil {
		s.bus.Publish(events.Event{
			Type:      events.HostAppAction,
			Severity:  events.SeverityFor(events.HostAppAction),
			ServerID:  serverID,
			Details:   map[string]any{"source": source, "name": name, "action": action, "success": true},
			Timestamp: time.Now().UTC(),
		})
	}

	return actionResult.GetOutput(), nil
}

func (s *HostActionService) logAction(targetID, serverID, source, name, action, userID, ip string, success bool, output string) {
	if s.audit == nil {
		return
	}
	s.audit.Log(
		"host_app."+action,
		userID,
		"host_app",
		targetID,
		ip,
		map[string]any{
			"server":  serverID,
			"source":  source,
			"name":    name,
			"action":  action,
			"success": success,
			"output":  truncateActionOutput(output),
		},
	)
}

func truncateActionOutput(output string) string {
	runes := []rune(output)
	if len(runes) <= maxActionOutput {
		return output
	}
	return string(runes[:maxActionOutput])
}
