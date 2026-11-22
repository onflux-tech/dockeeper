package collector

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"
)

type PM2App struct {
	Name          string
	PMID          string
	Status        string
	PIDs          []uint32
	CPU           float64
	Mem           uint64
	Restarts      uint32
	UptimeSeconds uint64
	User          string
	Cwd           string
	ExecPath      string
	LogPath       string
}

type pm2Daemon struct {
	PID     uint32
	UID     uint32
	GID     uint32
	User    string
	Home    string
	PM2Home string
}

type pm2Collector struct{}

func pm2Env(daemon pm2Daemon) []string {
	return []string{
		"PATH=" + os.Getenv("PATH"),
		"HOME=" + daemon.Home,
		"PM2_HOME=" + daemon.PM2Home,
		"USER=" + daemon.User,
		"LOGNAME=" + daemon.User,
	}
}

func (c *pm2Collector) ListApps(ctx context.Context) ([]PM2App, error) {
	daemons, err := discoverPM2Daemons()
	if err != nil {
		return nil, fmt.Errorf("discover pm2 daemons: %w", err)
	}

	apps := make([]PM2App, 0)
	for _, daemon := range daemons {
		output, err := runPM2Jlist(ctx, daemon)
		if err != nil {
			return nil, fmt.Errorf("pm2 jlist for %s: %w", daemon.User, err)
		}
		parsed, err := parsePM2Jlist(output)
		if err != nil {
			return nil, fmt.Errorf("parse pm2 jlist for %s: %w", daemon.User, err)
		}
		for index := range parsed {
			if parsed[index].User == "" {
				parsed[index].User = daemon.User
			}
		}
		apps = append(apps, parsed...)
	}

	return apps, nil
}

func parsePM2Jlist(data []byte) ([]PM2App, error) {
	var entries []struct {
		Name  string      `json:"name"`
		PID   json.Number `json:"pid"`
		PMID  json.Number `json:"pm_id"`
		Monit struct {
			Memory uint64  `json:"memory"`
			CPU    float64 `json:"cpu"`
		} `json:"monit"`
		Env struct {
			Status       string `json:"status"`
			Uptime       int64  `json:"pm_uptime"`
			RestartTime  uint32 `json:"restart_time"`
			Cwd          string `json:"pm_cwd"`
			ExecPath     string `json:"pm_exec_path"`
			OutLogPath   string `json:"pm_out_log_path"`
			ErrorLogPath string `json:"pm_err_log_path"`
			Username     string `json:"username"`
		} `json:"pm2_env"`
	}
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, fmt.Errorf("parse pm2 jlist: %w", err)
	}

	now := time.Now().UnixMilli()
	apps := make([]PM2App, 0, len(entries))
	for _, entry := range entries {
		app := PM2App{
			Name:     entry.Name,
			PMID:     entry.PMID.String(),
			Status:   entry.Env.Status,
			CPU:      entry.Monit.CPU,
			Mem:      entry.Monit.Memory,
			Restarts: entry.Env.RestartTime,
			User:     entry.Env.Username,
			Cwd:      entry.Env.Cwd,
			ExecPath: entry.Env.ExecPath,
			LogPath:  entry.Env.OutLogPath,
		}
		if app.LogPath == "" {
			app.LogPath = entry.Env.ErrorLogPath
		}
		if pid, err := entry.PID.Int64(); err == nil && pid > 0 {
			app.PIDs = []uint32{uint32(pid)}
		}
		if entry.Env.Uptime > 0 && now > entry.Env.Uptime {
			app.UptimeSeconds = uint64((now - entry.Env.Uptime) / 1000)
		}
		apps = append(apps, app)
	}

	return apps, nil
}
