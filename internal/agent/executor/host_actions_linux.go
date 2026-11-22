//go:build linux

package executor

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strconv"
	"syscall"

	"github.com/coreos/go-systemd/v22/dbus"
)

func defaultHostActionRunners() (SystemdRunner, PM2Runner) {
	return runSystemdAction, runPM2Action
}

func runSystemdAction(ctx context.Context, args []string) (int, string, error) {
	if conn, err := dbus.NewSystemConnectionContext(ctx); err == nil {
		defer conn.Close()

		var dbusErr error
		switch args[0] {
		case "start":
			_, dbusErr = conn.StartUnitContext(ctx, args[1], "replace", nil)
		case "stop":
			_, dbusErr = conn.StopUnitContext(ctx, args[1], "replace", nil)
		case "restart":
			_, dbusErr = conn.RestartUnitContext(ctx, args[1], "replace", nil)
		}
		if dbusErr == nil {
			return 0, "", nil
		}
	}
	return runHostCommand(ctx, "systemctl", args, os.Environ(), nil)
}

func runPM2Action(ctx context.Context, args []string, username string) (int, string, error) {
	account, err := user.Lookup(username)
	if err != nil {
		return 1, "", fmt.Errorf("lookup pm2 owner %s: %w", username, err)
	}
	uid, err := strconv.ParseUint(account.Uid, 10, 32)
	if err != nil {
		return 1, "", fmt.Errorf("parse uid for %s: %w", username, err)
	}
	gid, err := strconv.ParseUint(account.Gid, 10, 32)
	if err != nil {
		return 1, "", fmt.Errorf("parse gid for %s: %w", username, err)
	}

	path, err := exec.LookPath("pm2")
	if err != nil {
		path, err = pm2Executable()
		if err != nil {
			return 1, "", err
		}
	}

	env := []string{
		"PATH=" + os.Getenv("PATH"),
		"HOME=" + account.HomeDir,
		"PM2_HOME=" + filepath.Join(account.HomeDir, ".pm2"),
		"USER=" + username,
		"LOGNAME=" + username,
	}
	procAttr := &syscall.SysProcAttr{Credential: &syscall.Credential{Uid: uint32(uid), Gid: uint32(gid)}}

	return runHostCommand(ctx, path, args, env, procAttr)
}

func runHostCommand(ctx context.Context, path string, args, env []string, procAttr *syscall.SysProcAttr) (int, string, error) {
	command := exec.CommandContext(ctx, path, args...)
	command.Env = env
	command.SysProcAttr = procAttr

	var output bytes.Buffer
	command.Stdout = &output
	command.Stderr = &output

	err := command.Run()
	exitCode := 0
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = 1
		}
	}
	return exitCode, output.String(), err
}

func pm2Executable() (string, error) {
	for _, candidate := range []string{"/usr/local/bin/pm2", "/usr/bin/pm2", "/opt/homebrew/bin/pm2"} {
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("pm2 binary not found")
}
