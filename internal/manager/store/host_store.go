package store

import (
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/pocketbase/pocketbase/core"

	"github.com/onflux-tech/dockeeper/internal/shared/constants"
	agentpb "github.com/onflux-tech/dockeeper/proto/gen/dockeeper/agent/v1"
)

type HostStore struct {
	app core.App
}

func NewHostStore(app core.App) *HostStore {
	return &HostStore{app: app}
}

func (s *HostStore) UpsertInventory(serverID string, apps []*agentpb.HostApp, processes []*agentpb.HostProcess, ports []*agentpb.HostPort) (int, error) {
	var count int
	err := s.app.RunInTransaction(func(txApp core.App) error {
		var err error
		count, err = upsertHostInventory(txApp, serverID, apps, processes, ports)
		return err
	})
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (s *HostStore) SaveAppMetrics(serverID string, apps []*agentpb.HostApp, at time.Time) error {
	if len(apps) == 0 {
		return nil
	}
	if at.IsZero() {
		at = time.Now().UTC()
	}

	col, err := s.app.FindCachedCollectionByNameOrId(constants.CollHostAppMetricsRaw)
	if err != nil {
		return fmt.Errorf("find collection %s: %w", constants.CollHostAppMetricsRaw, err)
	}

	for _, app := range apps {
		record := core.NewRecord(col)
		record.Set("server", serverID)
		record.Set("app", app.GetName())
		record.Set("source", app.GetSource())
		record.Set("timestamp", at.UTC())
		record.Set("cpu_percent", app.GetCpuPercent())
		record.Set("mem_bytes", app.GetMemBytes())
		record.Set("restarts", app.GetRestarts())
		if err := s.app.Save(record); err != nil {
			slog.Error("save host app metrics failed",
				"server_id", serverID, "app", app.GetName(), "error", err)
		}
	}

	return nil
}

func upsertHostInventory(app core.App, serverID string, apps []*agentpb.HostApp, processes []*agentpb.HostProcess, ports []*agentpb.HostPort) (int, error) {
	appsCol, err := app.FindCachedCollectionByNameOrId(constants.CollHostApps)
	if err != nil {
		return 0, fmt.Errorf("find collection %s: %w", constants.CollHostApps, err)
	}
	processesCol, err := app.FindCachedCollectionByNameOrId(constants.CollHostProcesses)
	if err != nil {
		return 0, fmt.Errorf("find collection %s: %w", constants.CollHostProcesses, err)
	}

	portsByPID := make(map[uint32][]uint32, len(ports))
	for _, port := range ports {
		portsByPID[port.GetPid()] = append(portsByPID[port.GetPid()], port.GetPort())
	}

	now := time.Now().UTC()
	seenApps := make(map[string]bool, len(apps))
	count := 0

	for _, hostApp := range apps {
		key := hostApp.GetSource() + "\x00" + hostApp.GetName()
		if seenApps[key] {
			continue
		}
		seenApps[key] = true

		record, err := findHostApp(app, serverID, hostApp.GetSource(), hostApp.GetName())
		if err != nil {
			record = core.NewRecord(appsCol)
			record.Set("server", serverID)
			record.Set("source", hostApp.GetSource())
			record.Set("name", hostApp.GetName())
		}

		record.Set("unit", hostApp.GetUnit())
		record.Set("pm_id", hostApp.GetPmId())
		record.Set("status", hostApp.GetStatus())
		record.Set("pids", hostApp.GetPids())
		record.Set("ports", hostAppPorts(hostApp, portsByPID))
		record.Set("cpu_percent", hostApp.GetCpuPercent())
		record.Set("mem_bytes", hostApp.GetMemBytes())
		record.Set("disk_read_bytes", hostApp.GetDiskReadBytes())
		record.Set("disk_write_bytes", hostApp.GetDiskWriteBytes())
		record.Set("restarts", hostApp.GetRestarts())
		record.Set("uptime_seconds", hostApp.GetUptimeSeconds())
		record.Set("user", hostApp.GetUser())
		record.Set("cwd", hostApp.GetCwd())
		record.Set("exec_path", hostApp.GetExecPath())
		record.Set("log_path", hostApp.GetLogPath())
		record.Set("last_seen", now)

		if err := app.Save(record); err != nil {
			slog.Error("upsert host app failed",
				"server_id", serverID, "source", hostApp.GetSource(), "name", hostApp.GetName(), "error", err)
			continue
		}
		count++
	}

	existingApps, err := app.FindRecordsByFilter(constants.CollHostApps, "server = {:server}", "", 0, 0, map[string]any{"server": serverID})
	if err != nil {
		return 0, fmt.Errorf("list existing host apps: %w", err)
	}
	for _, record := range existingApps {
		key := record.GetString("source") + "\x00" + record.GetString("name")
		if seenApps[key] {
			continue
		}
		if err := app.Delete(record); err != nil {
			slog.Error("prune host app failed", "server_id", serverID, "record_id", record.Id, "error", err)
		}
	}

	seenPIDs := make(map[string]bool, len(processes))
	for _, process := range processes {
		key := strconv.FormatUint(uint64(process.GetPid()), 10)
		if seenPIDs[key] {
			continue
		}
		seenPIDs[key] = true

		record, err := findHostProcess(app, serverID, process.GetPid())
		if err != nil {
			record = core.NewRecord(processesCol)
			record.Set("server", serverID)
			record.Set("pid", process.GetPid())
		}

		record.Set("name", process.GetName())
		record.Set("app", process.GetApp())
		record.Set("cpu_percent", process.GetCpuPercent())
		record.Set("mem_bytes", process.GetMemBytes())
		record.Set("threads", process.GetThreads())
		record.Set("last_seen", now)

		if err := app.Save(record); err != nil {
			slog.Error("upsert host process failed",
				"server_id", serverID, "pid", process.GetPid(), "error", err)
		}
	}

	existingProcesses, err := app.FindRecordsByFilter(constants.CollHostProcesses, "server = {:server}", "", 0, 0, map[string]any{"server": serverID})
	if err != nil {
		return 0, fmt.Errorf("list existing host processes: %w", err)
	}
	for _, record := range existingProcesses {
		key := strconv.Itoa(record.GetInt("pid"))
		if seenPIDs[key] {
			continue
		}
		if err := app.Delete(record); err != nil {
			slog.Error("prune host process failed", "server_id", serverID, "record_id", record.Id, "error", err)
		}
	}

	return count, nil
}

func findHostApp(app core.App, serverID, source, name string) (*core.Record, error) {
	records, err := app.FindRecordsByFilter(
		constants.CollHostApps,
		"server = {:server} && source = {:source} && name = {:name}",
		"", 1, 0,
		map[string]any{"server": serverID, "source": source, "name": name},
	)
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return nil, fmt.Errorf("host app not found")
	}
	return records[0], nil
}

func findHostProcess(app core.App, serverID string, pid uint32) (*core.Record, error) {
	records, err := app.FindRecordsByFilter(
		constants.CollHostProcesses,
		"server = {:server} && pid = {:pid}",
		"", 1, 0,
		map[string]any{"server": serverID, "pid": pid},
	)
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return nil, fmt.Errorf("host process not found")
	}
	return records[0], nil
}

func hostAppPorts(app *agentpb.HostApp, portsByPID map[uint32][]uint32) []uint32 {
	ports := app.GetPorts()
	if len(ports) == 0 {
		for _, pid := range app.GetPids() {
			ports = append(ports, portsByPID[pid]...)
		}
	}

	unique := make([]uint32, 0, len(ports))
	seen := make(map[uint32]bool, len(ports))
	for _, port := range ports {
		if seen[port] {
			continue
		}
		seen[port] = true
		unique = append(unique, port)
	}
	return unique
}
