package executor

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/onflux-tech/dockeeper/internal/agent/collector"
)

var (
	ErrUnsupportedHostSource = errors.New("unsupported host app source")
	ErrUnsupportedHostAction = errors.New("unsupported host app action")
	ErrHostAppNotFound       = errors.New("host app not found in inventory")
	ErrHostAppOwnerUnknown   = errors.New("host app owner unknown")
)

type SystemdRunner func(ctx context.Context, args []string) (int, string, error)

type PM2Runner func(ctx context.Context, args []string, username string) (int, string, error)

type HostActionOption func(*HostActionExecutor)

type HostActionExecutor struct {
	logger        *slog.Logger
	inventory     func() []collector.HostApp
	systemdRunner SystemdRunner
	pm2Runner     PM2Runner
}

func WithInventory(inventory func() []collector.HostApp) HostActionOption {
	return func(e *HostActionExecutor) {
		e.inventory = inventory
	}
}

func WithSystemdRunner(runner SystemdRunner) HostActionOption {
	return func(e *HostActionExecutor) {
		e.systemdRunner = runner
	}
}

func WithPM2Runner(runner PM2Runner) HostActionOption {
	return func(e *HostActionExecutor) {
		e.pm2Runner = runner
	}
}

func NewHostActionExecutor(logger *slog.Logger, opts ...HostActionOption) *HostActionExecutor {
	e := &HostActionExecutor{logger: logger}
	for _, opt := range opts {
		opt(e)
	}
	if e.systemdRunner == nil || e.pm2Runner == nil {
		systemdRunner, pm2Runner := defaultHostActionRunners()
		if e.systemdRunner == nil {
			e.systemdRunner = systemdRunner
		}
		if e.pm2Runner == nil {
			e.pm2Runner = pm2Runner
		}
	}
	return e
}

func (e *HostActionExecutor) Execute(ctx context.Context, source, name, action string) (int, string, error) {
	if source != "systemd" && source != "pm2" {
		return 1, "", ErrUnsupportedHostSource
	}
	switch action {
	case "start", "stop", "restart":
	default:
		return 1, "", ErrUnsupportedHostAction
	}

	app, found := e.findApp(source, name)
	if !found {
		return 1, "", fmt.Errorf("%w: %s/%s", ErrHostAppNotFound, source, name)
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	if source == "systemd" {
		return e.systemdRunner(ctx, []string{action, name})
	}

	if app.User == "" {
		return 1, "", fmt.Errorf("%w: %s", ErrHostAppOwnerUnknown, name)
	}
	return e.pm2Runner(ctx, []string{action, name, "--no-color"}, app.User)
}

func (e *HostActionExecutor) findApp(source, name string) (collector.HostApp, bool) {
	if e.inventory == nil {
		return collector.HostApp{}, false
	}
	for _, app := range e.inventory() {
		if app.Source == source && app.Name == name {
			return app, true
		}
	}
	return collector.HostApp{}, false
}
