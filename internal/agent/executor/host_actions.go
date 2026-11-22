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

	var (
		code   int
		output string
		err    error
	)
	if source == "systemd" {
		code, output, err = e.systemdRunner(ctx, []string{action, name})
	} else if app.User == "" {
		code, output, err = 1, "", fmt.Errorf("%w: %s", ErrHostAppOwnerUnknown, name)
	} else {
		code, output, err = e.pm2Runner(ctx, []string{action, name, "--no-color"}, app.User)
	}

	if e.logger != nil {
		if err != nil {
			e.logger.Warn("host app action failed", "source", source, "name", name, "action", action, "exit_code", code, "error", err)
		} else {
			e.logger.Info("host app action completed", "source", source, "name", name, "action", action, "exit_code", code)
		}
	}

	return code, output, err
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
