//go:build linux

package collector

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

func discoverPM2Daemons() ([]pm2Daemon, error) {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil, err
	}

	daemons := make([]pm2Daemon, 0)
	seenUID := make(map[uint32]bool)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		pid, err := strconv.ParseUint(entry.Name(), 10, 32)
		if err != nil {
			continue
		}

		cmdline, err := os.ReadFile(filepath.Join("/proc", entry.Name(), "cmdline"))
		if err != nil {
			continue
		}
		command := strings.ReplaceAll(string(bytes.TrimRight(cmdline, "\x00")), "\x00", " ")
		lower := strings.ToLower(command)
		if !strings.Contains(lower, "pm2") || !strings.Contains(lower, "daemon") {
			continue
		}

		uid, gid, username, home, err := procCredentials(entry.Name())
		if err != nil || seenUID[uid] {
			continue
		}
		seenUID[uid] = true

		daemons = append(daemons, pm2Daemon{
			PID:     uint32(pid),
			UID:     uid,
			GID:     gid,
			User:    username,
			Home:    home,
			PM2Home: filepath.Join(home, ".pm2"),
		})
	}

	return daemons, nil
}

func procCredentials(pid string) (uint32, uint32, string, string, error) {
	status, err := os.ReadFile(filepath.Join("/proc", pid, "status"))
	if err != nil {
		return 0, 0, "", "", err
	}

	var uid, gid uint32
	for _, line := range strings.Split(string(status), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		switch fields[0] {
		case "Uid:":
			value, err := strconv.ParseUint(fields[1], 10, 32)
			if err != nil {
				return 0, 0, "", "", err
			}
			uid = uint32(value)
		case "Gid:":
			value, err := strconv.ParseUint(fields[1], 10, 32)
			if err != nil {
				return 0, 0, "", "", err
			}
			gid = uint32(value)
		}
	}

	account, err := user.LookupId(strconv.FormatUint(uint64(uid), 10))
	if err != nil {
		return uid, gid, strconv.FormatUint(uint64(uid), 10), "/", nil
	}

	return uid, gid, account.Username, account.HomeDir, nil
}

func runPM2Jlist(ctx context.Context, daemon pm2Daemon) ([]byte, error) {
	path, err := exec.LookPath("pm2")
	if err != nil {
		path, err = pm2Binary()
		if err != nil {
			return nil, err
		}
	}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	command := exec.CommandContext(ctx, path, "jlist")
	command.Env = append(os.Environ(), "HOME="+daemon.Home, "PM2_HOME="+daemon.PM2Home)
	command.SysProcAttr = &syscall.SysProcAttr{
		Credential: &syscall.Credential{Uid: daemon.UID, Gid: daemon.GID},
	}

	var stdout, stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	if err := command.Run(); err != nil {
		return nil, fmt.Errorf("pm2 jlist: %w: %s", err, strings.TrimSpace(stderr.String()))
	}

	return stdout.Bytes(), nil
}

func pm2Binary() (string, error) {
	for _, candidate := range []string{"/usr/local/bin/pm2", "/usr/bin/pm2", "/opt/homebrew/bin/pm2"} {
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("pm2 binary not found")
}
