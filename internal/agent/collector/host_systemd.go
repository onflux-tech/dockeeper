package collector

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/coreos/go-systemd/v22/dbus"
)

const defaultCgroupRoot = "/sys/fs/cgroup"

type SystemdUnit struct {
	Name          string
	Description   string
	Active        string
	SubState      string
	MainPID       uint32
	User          string
	ExecPath      string
	CPUPercent    float64
	MemBytes      uint64
	UptimeSeconds uint64
}

type systemdCollector struct {
	logger     *slog.Logger
	cgroupRoot string
}

func (c *systemdCollector) ListUnits(ctx context.Context) ([]SystemdUnit, error) {
	units, err := c.listViaDBus(ctx)
	if err == nil {
		return units, nil
	}

	c.logger.Warn("systemd dbus unavailable, falling back to systemctl", "error", err)
	return c.listViaSystemctl(ctx)
}

func (c *systemdCollector) listViaDBus(ctx context.Context) ([]SystemdUnit, error) {
	conn, err := dbus.NewSystemConnectionContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("connect to systemd dbus: %w", err)
	}
	defer conn.Close()

	statuses, err := conn.ListUnitsContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("list systemd units: %w", err)
	}

	units := make([]SystemdUnit, 0, len(statuses))
	for _, status := range statuses {
		if !strings.HasSuffix(status.Name, ".service") {
			continue
		}

		unit := SystemdUnit{
			Name:        status.Name,
			Description: status.Description,
			Active:      status.ActiveState,
			SubState:    status.SubState,
		}

		properties, err := conn.GetUnitTypePropertiesContext(ctx, status.Name, "Service")
		if err == nil {
			unit.MainPID = toUint32(properties["MainPID"])
			unit.User = toString(properties["User"])
			if entered := toUint64(properties["ActiveEnterTimestamp"]); entered > 0 {
				activeSince := time.UnixMicro(int64(entered))
				if uptime := time.Since(activeSince); uptime > 0 {
					unit.UptimeSeconds = uint64(uptime.Seconds())
				}
			}
		}

		c.readCgroupUsage(&unit)
		units = append(units, unit)
	}

	return units, nil
}

func (c *systemdCollector) listViaSystemctl(ctx context.Context) ([]SystemdUnit, error) {
	output, err := exec.CommandContext(ctx, "systemctl", "list-units", "--type=service", "--all", "--output=json", "--no-pager").Output()
	if err != nil {
		return nil, fmt.Errorf("systemctl list-units: %w", err)
	}

	var entries []struct {
		Unit        string `json:"unit"`
		Active      string `json:"active"`
		Sub         string `json:"sub"`
		Description string `json:"description"`
	}
	if err := json.Unmarshal(output, &entries); err != nil {
		return nil, fmt.Errorf("parse systemctl list-units output: %w", err)
	}

	names := make([]string, 0, len(entries))
	units := make([]SystemdUnit, 0, len(entries))
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Unit, ".service") {
			continue
		}
		units = append(units, SystemdUnit{
			Name:        entry.Unit,
			Description: entry.Description,
			Active:      entry.Active,
			SubState:    entry.Sub,
		})
		names = append(names, entry.Unit)
	}

	properties := systemctlProperties(ctx, names)
	for index := range units {
		values := properties[units[index].Name]
		units[index].MainPID = parseMainPID(values["MainPID"])
		units[index].User = values["User"]
		c.readCgroupUsage(&units[index])
	}

	return units, nil
}

func (c *systemdCollector) readCgroupUsage(unit *SystemdUnit) {
	dir := filepath.Join(c.cgroupRoot, "system.slice", unit.Name)
	cpuUsec, memBytes, err := readUnitUsage(dir)
	if err != nil {
		return
	}
	unit.MemBytes = memBytes
	if unit.UptimeSeconds > 0 {
		unit.CPUPercent = float64(cpuUsec) / (float64(unit.UptimeSeconds) * 1e6) * 100
	}
}

func readUnitUsage(dir string) (uint64, uint64, error) {
	cpuStat, err := os.ReadFile(filepath.Join(dir, "cpu.stat"))
	if err != nil {
		return 0, 0, err
	}

	var cpuUsec uint64
	for _, line := range strings.Split(string(cpuStat), "\n") {
		fields := strings.Fields(line)
		if len(fields) != 2 || fields[0] != "usage_usec" {
			continue
		}
		value, err := strconv.ParseUint(fields[1], 10, 64)
		if err != nil {
			return 0, 0, fmt.Errorf("parse cpu usage: %w", err)
		}
		cpuUsec = value
	}

	memoryCurrent, err := os.ReadFile(filepath.Join(dir, "memory.current"))
	if err != nil {
		return 0, 0, err
	}
	memBytes, err := strconv.ParseUint(strings.TrimSpace(string(memoryCurrent)), 10, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("parse memory usage: %w", err)
	}

	return cpuUsec, memBytes, nil
}

func systemctlProperties(ctx context.Context, names []string) map[string]map[string]string {
	properties := make(map[string]map[string]string)
	if len(names) == 0 {
		return properties
	}

	args := append([]string{"show", "--property=Id,MainPID,User", "--no-pager"}, names...)
	output, err := exec.CommandContext(ctx, "systemctl", args...).Output()
	if err != nil {
		return properties
	}

	var current map[string]string
	for _, line := range strings.Split(string(output), "\n") {
		if strings.TrimSpace(line) == "" {
			current = nil
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		if key == "Id" {
			current = make(map[string]string)
			properties[value] = current
			continue
		}
		if current != nil {
			current[key] = value
		}
	}

	return properties
}

func parseMainPID(value string) uint32 {
	pid, err := strconv.ParseUint(value, 10, 32)
	if err != nil {
		return 0
	}
	return uint32(pid)
}

func toUint32(value any) uint32 {
	switch typed := value.(type) {
	case uint32:
		return typed
	case uint64:
		return uint32(typed)
	case int32:
		return uint32(typed)
	case int:
		return uint32(typed)
	default:
		return 0
	}
}

func toUint64(value any) uint64 {
	switch typed := value.(type) {
	case uint64:
		return typed
	case uint32:
		return uint64(typed)
	case int64:
		return uint64(typed)
	case int:
		return uint64(typed)
	default:
		return 0
	}
}

func toString(value any) string {
	if text, ok := value.(string); ok {
		return text
	}
	return ""
}
