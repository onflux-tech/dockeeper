package collector

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v4/host"
)

type HostInventoryState struct {
	mu        sync.RWMutex
	inventory *HostInventory
}

func NewHostInventoryState() *HostInventoryState {
	return &HostInventoryState{}
}

func (s *HostInventoryState) Set(inventory *HostInventory) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.inventory = inventory
}

func (s *HostInventoryState) Apps() []HostApp {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.inventory == nil {
		return nil
	}
	return s.inventory.Apps
}

type HostApp struct {
	Source         string
	Name           string
	Unit           string
	PMID           string
	Status         string
	PIDs           []uint32
	Ports          []uint32
	CPUPercent     float64
	MemBytes       uint64
	DiskReadBytes  uint64
	DiskWriteBytes uint64
	Restarts       uint32
	UptimeSeconds  uint64
	User           string
	Cwd            string
	ExecPath       string
	LogPath        string
}

type HostProcess struct {
	PID        uint32
	Name       string
	App        string
	CPUPercent float64
	MemBytes   uint64
	Threads    uint32
}

type HostPort struct {
	Port     uint32
	Protocol string
	Address  string
	PID      uint32
	App      string
}

type HostInventory struct {
	Apps          []HostApp
	Processes     []HostProcess
	Ports         []HostPort
	UptimeSeconds uint64
	CollectedAt   time.Time
}

type systemdSource interface {
	ListUnits(ctx context.Context) ([]SystemdUnit, error)
}

type pm2Source interface {
	ListApps(ctx context.Context) ([]PM2App, error)
}

type processSource interface {
	ListProcesses(ctx context.Context) ([]ProcessInfo, error)
	ListListeningPorts(ctx context.Context) ([]ListeningPort, error)
}

type HostInventoryCollector struct {
	logger    *slog.Logger
	systemd   systemdSource
	pm2       pm2Source
	processes processSource
}

func NewHostInventoryCollector(logger *slog.Logger) *HostInventoryCollector {
	return &HostInventoryCollector{
		logger:    logger,
		systemd:   &systemdCollector{logger: logger, cgroupRoot: defaultCgroupRoot},
		pm2:       &pm2Collector{},
		processes: &procfsCollector{logger: logger},
	}
}

func (c *HostInventoryCollector) Collect(ctx context.Context) (*HostInventory, error) {
	units, err := c.systemd.ListUnits(ctx)
	if err != nil {
		return nil, fmt.Errorf("list systemd units: %w", err)
	}

	pm2Apps, err := c.pm2.ListApps(ctx)
	if err != nil {
		return nil, fmt.Errorf("list pm2 apps: %w", err)
	}

	processes, err := c.processes.ListProcesses(ctx)
	if err != nil {
		return nil, fmt.Errorf("list host processes: %w", err)
	}

	ports, err := c.processes.ListListeningPorts(ctx)
	if err != nil {
		return nil, fmt.Errorf("list host listening ports: %w", err)
	}

	uptime, err := host.UptimeWithContext(ctx)
	if err != nil {
		c.logger.Warn("host uptime unavailable", "error", err)
	}

	return mergeHostInventory(units, pm2Apps, processes, ports, uptime, time.Now()), nil
}

func (c *HostInventoryCollector) Run(ctx context.Context, interval time.Duration, report func(*HostInventory) error) {
	c.reportOnce(ctx, report)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.reportOnce(ctx, report)
		}
	}
}

func (c *HostInventoryCollector) reportOnce(ctx context.Context, report func(*HostInventory) error) {
	inventory, err := c.Collect(ctx)
	if err != nil {
		c.logger.Warn("host inventory collection failed", "error", err)
		return
	}
	if err := report(inventory); err != nil {
		c.logger.Warn("host inventory report failed", "error", err)
	}
}

func mergeHostInventory(units []SystemdUnit, pm2Apps []PM2App, processes []ProcessInfo, ports []ListeningPort, uptime uint64, now time.Time) *HostInventory {
	apps := make([]HostApp, 0, len(pm2Apps)+len(units))
	pidToApp := make(map[uint32]int)

	for _, app := range pm2Apps {
		entry := HostApp{
			Source:        "pm2",
			Name:          app.Name,
			PMID:          app.PMID,
			Status:        normalizePM2Status(app.Status),
			PIDs:          app.PIDs,
			CPUPercent:    app.CPU,
			MemBytes:      app.Mem,
			Restarts:      app.Restarts,
			UptimeSeconds: app.UptimeSeconds,
			User:          app.User,
			Cwd:           app.Cwd,
			ExecPath:      app.ExecPath,
			LogPath:       app.LogPath,
		}
		apps = append(apps, entry)
		for _, pid := range app.PIDs {
			pidToApp[pid] = len(apps) - 1
		}
	}

	for _, unit := range units {
		if isPM2Unit(unit) {
			continue
		}
		entry := HostApp{
			Source:        "systemd",
			Name:          unit.Name,
			Unit:          unit.Name,
			Status:        normalizeSystemdStatus(unit.Active),
			CPUPercent:    unit.CPUPercent,
			MemBytes:      unit.MemBytes,
			UptimeSeconds: unit.UptimeSeconds,
			User:          unit.User,
		}
		if unit.MainPID > 0 {
			entry.PIDs = []uint32{unit.MainPID}
		}
		apps = append(apps, entry)
		if unit.MainPID > 0 {
			pidToApp[unit.MainPID] = len(apps) - 1
		}
	}

	processByPID := make(map[uint32]ProcessInfo, len(processes))
	for _, process := range processes {
		processByPID[process.PID] = process
	}

	for _, port := range ports {
		if index, ok := pidToApp[port.PID]; ok {
			apps[index].Ports = appendUniquePort(apps[index].Ports, port.Port)
		}
	}

	seenPortPID := make(map[uint32]bool)
	processAppByName := make(map[string]int)
	for _, port := range ports {
		if _, ok := pidToApp[port.PID]; ok {
			continue
		}
		if seenPortPID[port.PID] {
			continue
		}
		process, ok := processByPID[port.PID]
		if !ok {
			continue
		}
		seenPortPID[port.PID] = true
		if index, exists := processAppByName[process.Name]; exists {
			apps[index].PIDs = append(apps[index].PIDs, process.PID)
			apps[index].CPUPercent += process.CPU
			apps[index].MemBytes += process.Mem
			apps[index].Ports = appendUniquePort(apps[index].Ports, port.Port)
			pidToApp[port.PID] = index
			continue
		}
		entry := HostApp{
			Source:     "process",
			Name:       process.Name,
			Status:     "running",
			PIDs:       []uint32{port.PID},
			CPUPercent: process.CPU,
			MemBytes:   process.Mem,
			User:       process.User,
			Cwd:        process.Cwd,
			ExecPath:   process.ExecPath,
		}
		apps = append(apps, entry)
		index := len(apps) - 1
		processAppByName[process.Name] = index
		pidToApp[port.PID] = index
		apps[index].Ports = appendUniquePort(apps[index].Ports, port.Port)
	}

	for index := range apps {
		if len(apps[index].PIDs) == 0 {
			continue
		}
		process, ok := processByPID[apps[index].PIDs[0]]
		if !ok {
			continue
		}
		if apps[index].CPUPercent == 0 {
			apps[index].CPUPercent = process.CPU
		}
		if apps[index].MemBytes == 0 {
			apps[index].MemBytes = process.Mem
		}
		if apps[index].Cwd == "" {
			apps[index].Cwd = process.Cwd
		}
		if apps[index].ExecPath == "" {
			apps[index].ExecPath = process.ExecPath
		}
	}

	processEntries := make([]HostProcess, 0, len(processes))
	for _, process := range processes {
		entry := HostProcess{
			PID:        process.PID,
			Name:       process.Name,
			CPUPercent: process.CPU,
			MemBytes:   process.Mem,
			Threads:    process.Threads,
		}
		if index, ok := pidToApp[process.PID]; ok {
			entry.App = apps[index].Name
		}
		processEntries = append(processEntries, entry)
	}
	sort.SliceStable(processEntries, func(i, j int) bool {
		return processEntries[i].CPUPercent > processEntries[j].CPUPercent
	})
	if len(processEntries) > 50 {
		processEntries = processEntries[:50]
	}

	portEntries := make([]HostPort, 0, len(ports))
	for _, port := range ports {
		entry := HostPort{
			Port:     port.Port,
			Protocol: port.Protocol,
			Address:  port.Address,
			PID:      port.PID,
		}
		if index, ok := pidToApp[port.PID]; ok {
			entry.App = apps[index].Name
		}
		portEntries = append(portEntries, entry)
	}
	sort.SliceStable(portEntries, func(i, j int) bool {
		return portEntries[i].Port < portEntries[j].Port
	})

	return &HostInventory{
		Apps:          apps,
		Processes:     processEntries,
		Ports:         portEntries,
		UptimeSeconds: uptime,
		CollectedAt:   now,
	}
}

func appendUniquePort(ports []uint32, port uint32) []uint32 {
	for _, existing := range ports {
		if existing == port {
			return ports
		}
	}
	return append(ports, port)
}

func normalizePM2Status(status string) string {
	switch strings.ToLower(status) {
	case "online", "launching":
		return "running"
	case "stopped", "stopping":
		return "stopped"
	case "errored", "error":
		return "failed"
	default:
		return "unknown"
	}
}

func normalizeSystemdStatus(active string) string {
	switch active {
	case "active", "activating", "reloading":
		return "running"
	case "inactive", "deactivating":
		return "stopped"
	case "failed":
		return "failed"
	default:
		return "unknown"
	}
}

func isPM2Unit(unit SystemdUnit) bool {
	return strings.HasPrefix(unit.Name, "pm2-") || strings.Contains(strings.ToLower(unit.ExecPath), "pm2")
}
