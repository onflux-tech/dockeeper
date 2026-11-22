package executor

import (
	"context"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/onflux-tech/dockeeper/internal/agent/collector"
)

type fakeSystemdRunner struct {
	argv []string
	err  error
}

func (f *fakeSystemdRunner) run(ctx context.Context, args []string) (int, string, error) {
	f.argv = args
	if f.err != nil {
		return 1, "", f.err
	}
	return 0, "ok", nil
}

type fakePM2Runner struct {
	argv []string
	user string
	err  error
}

func (f *fakePM2Runner) run(ctx context.Context, args []string, username string) (int, string, error) {
	f.argv = args
	f.user = username
	if f.err != nil {
		return 1, "", f.err
	}
	return 0, "ok", nil
}

func inventoryWith(apps ...collector.HostApp) func() []collector.HostApp {
	return func() []collector.HostApp { return apps }
}

func TestHostActionRejectsUnknownSource(t *testing.T) {
	e := NewHostActionExecutor(slog.Default())

	_, _, err := e.Execute(context.Background(), "shell", "nginx", "restart")
	require.ErrorIs(t, err, ErrUnsupportedHostSource)
}

func TestHostActionRejectsUnknownAction(t *testing.T) {
	e := NewHostActionExecutor(slog.Default())

	_, _, err := e.Execute(context.Background(), "systemd", "nginx.service", "reload")
	require.ErrorIs(t, err, ErrUnsupportedHostAction)
}

func TestHostActionRejectsAppOutsideInventory(t *testing.T) {
	e := NewHostActionExecutor(slog.Default(), WithInventory(inventoryWith()))

	_, _, err := e.Execute(context.Background(), "systemd", "nginx.service", "restart")
	require.ErrorIs(t, err, ErrHostAppNotFound)
}

func TestHostActionSystemdUsesArgvNotShell(t *testing.T) {
	runner := &fakeSystemdRunner{}
	e := NewHostActionExecutor(slog.Default(),
		WithInventory(inventoryWith(collector.HostApp{Source: "systemd", Name: "nginx.service", Status: "running"})),
		WithSystemdRunner(runner.run),
	)

	code, output, err := e.Execute(context.Background(), "systemd", "nginx.service", "restart")
	require.NoError(t, err)
	require.Equal(t, 0, code)
	require.Equal(t, "ok", output)
	require.Equal(t, []string{"restart", "nginx.service"}, runner.argv)
}

func TestHostActionPM2UsesOwner(t *testing.T) {
	runner := &fakePM2Runner{}
	e := NewHostActionExecutor(slog.Default(),
		WithInventory(inventoryWith(collector.HostApp{Source: "pm2", Name: "api-takeflow", User: "deploy", Status: "running"})),
		WithPM2Runner(runner.run),
	)

	_, _, err := e.Execute(context.Background(), "pm2", "api-takeflow", "restart")
	require.NoError(t, err)
	require.Equal(t, "deploy", runner.user)
	require.Equal(t, []string{"restart", "api-takeflow", "--no-color"}, runner.argv)
}

func TestHostActionPM2FailsClosedWithoutOwner(t *testing.T) {
	e := NewHostActionExecutor(slog.Default(),
		WithInventory(inventoryWith(collector.HostApp{Source: "pm2", Name: "api-takeflow", Status: "running"})),
		WithPM2Runner((&fakePM2Runner{}).run),
	)

	_, _, err := e.Execute(context.Background(), "pm2", "api-takeflow", "stop")
	require.ErrorIs(t, err, ErrHostAppOwnerUnknown)
}
