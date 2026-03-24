package executor

import (
	"context"
	"log/slog"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

const (
	systemdServiceName = "dockeeper-agent"
	systemdUnitPath    = "/etc/systemd/system/dockeeper-agent.service"
	binaryPath         = "/usr/local/bin/dockeeper-agent"
	defaultDataDir     = "/var/lib/dockeeper-agent"
)

// Uninstall removes the agent from the host. For systemd installs it disables
// the service, removes the unit file, binary and data directory.
// The method returns success immediately — actual cleanup runs best-effort.
func (e *DockerExecutor) Uninstall(ctx context.Context) Result {
	if runtime.GOOS != "linux" {
		return Result{
			Success: true,
			Output:  "uninstall acknowledged (non-linux; manual cleanup required)",
		}
	}

	isSystemd := fileExists(systemdUnitPath)
	if !isSystemd {
		// Docker-based agent — can't remove itself. Manager handles container removal.
		return Result{
			Success: true,
			Output:  "uninstall acknowledged (docker mode; container cleanup handled by manager)",
		}
	}

	// Systemd mode — best-effort cleanup.
	var msgs []string

	if out, err := runCmd(ctx, "systemctl", "stop", systemdServiceName); err != nil {
		slog.Warn("uninstall: stop service", "error", err, "output", out)
	} else {
		msgs = append(msgs, "service stopped")
	}

	if out, err := runCmd(ctx, "systemctl", "disable", systemdServiceName); err != nil {
		slog.Warn("uninstall: disable service", "error", err, "output", out)
	} else {
		msgs = append(msgs, "service disabled")
	}

	if err := os.Remove(systemdUnitPath); err != nil && !os.IsNotExist(err) {
		slog.Warn("uninstall: remove unit file", "error", err)
	} else {
		msgs = append(msgs, "unit file removed")
	}

	if out, err := runCmd(ctx, "systemctl", "daemon-reload"); err != nil {
		slog.Warn("uninstall: daemon-reload", "error", err, "output", out)
	}

	if err := os.Remove(binaryPath); err != nil && !os.IsNotExist(err) {
		slog.Warn("uninstall: remove binary", "error", err)
	} else {
		msgs = append(msgs, "binary removed")
	}

	if err := os.RemoveAll(defaultDataDir); err != nil {
		slog.Warn("uninstall: remove data dir", "error", err)
	} else {
		msgs = append(msgs, "data directory removed")
	}

	slog.Info("agent uninstall completed", "actions", strings.Join(msgs, ", "))

	return Result{
		Success: true,
		Output:  "agent uninstalled: " + strings.Join(msgs, ", "),
	}
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func runCmd(ctx context.Context, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}
