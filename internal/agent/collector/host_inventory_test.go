package collector

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHostInventoryPrecedence(t *testing.T) {
	c := newFixtureCollector(
		[]SystemdUnit{{Name: "pm2-root.service", Active: "active"}},
		[]PM2App{{Name: "api-takeflow", Status: "online", PIDs: []uint32{4122}}},
		[]ProcessInfo{{PID: 4122, Name: "node", CPU: 8.2, Mem: 322 << 20, Threads: 11}},
		[]ListeningPort{{Port: 3000, PID: 4122, Protocol: "tcp", Address: "127.0.0.1"}},
	)

	inv, err := c.Collect(context.Background())
	require.NoError(t, err)
	require.Len(t, inv.Apps, 1)
	require.Equal(t, "api-takeflow", inv.Apps[0].Name)
	require.Equal(t, "pm2", inv.Apps[0].Source)
	require.Equal(t, []uint32{3000}, inv.Apps[0].Ports)
	require.Len(t, inv.Processes, 1)
	require.Equal(t, "api-takeflow", inv.Processes[0].App)
	require.Len(t, inv.Ports, 1)
	require.Equal(t, "api-takeflow", inv.Ports[0].App)
}

func TestParsePM2JlistFixture(t *testing.T) {
	apps, err := parsePM2Jlist([]byte(`[{"name":"api","pm_id":3,"pid":4122,"pm2_env":{"status":"online","pm_uptime":1},"monit":{"cpu":2.5,"memory":1024}}]`))
	require.NoError(t, err)
	require.Len(t, apps, 1)
	require.Equal(t, "api", apps[0].Name)
	require.Equal(t, "online", apps[0].Status)
	require.Equal(t, []uint32{4122}, apps[0].PIDs)
}

func TestUnitUsage(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nginx.service")
	require.NoError(t, os.MkdirAll(dir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "cpu.stat"), []byte("usage_usec 2500000\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "memory.current"), []byte("1048576\n"), 0o644))

	cpuUsec, memBytes, err := readUnitUsage(dir)
	require.NoError(t, err)
	require.Equal(t, uint64(2500000), cpuUsec)
	require.Equal(t, uint64(1048576), memBytes)
}

func TestHostInventoryProcessFailure(t *testing.T) {
	c := newFixtureCollector(nil, nil, nil, nil)
	c.processes = failingProcessSource{}

	_, err := c.Collect(context.Background())
	require.Error(t, err)
}

func TestHostInventorySystemdFailure(t *testing.T) {
	c := newFixtureCollector(nil, nil, nil, nil)
	c.systemd = fixtureSystemd{err: errors.New("dbus down")}

	_, err := c.Collect(context.Background())
	require.Error(t, err)
}

func TestHostInventoryPM2Failure(t *testing.T) {
	c := newFixtureCollector(nil, nil, nil, nil)
	c.pm2 = fixturePM2{err: errors.New("jlist failed")}

	_, err := c.Collect(context.Background())
	require.Error(t, err)
}

func TestHostInventoryPortsFailure(t *testing.T) {
	c := newFixtureCollector(nil, nil, nil, nil)
	c.processes = fixtureProcesses{portsErr: errors.New("proc net unavailable")}

	_, err := c.Collect(context.Background())
	require.Error(t, err)
}

func TestHostInventoryAggregatesSameNameProcesses(t *testing.T) {
	c := newFixtureCollector(
		nil,
		nil,
		[]ProcessInfo{
			{PID: 300, Name: "node", CPU: 1.5, Mem: 100 << 20, Threads: 6},
			{PID: 301, Name: "node", CPU: 2.5, Mem: 200 << 20, Threads: 8},
		},
		[]ListeningPort{
			{Port: 3000, PID: 300, Protocol: "tcp", Address: "0.0.0.0"},
			{Port: 8080, PID: 301, Protocol: "tcp", Address: "0.0.0.0"},
		},
	)

	inv, err := c.Collect(context.Background())
	require.NoError(t, err)
	require.Len(t, inv.Apps, 1)
	require.Equal(t, "process", inv.Apps[0].Source)
	require.Equal(t, "node", inv.Apps[0].Name)
	require.Equal(t, []uint32{300, 301}, inv.Apps[0].PIDs)
	require.Equal(t, []uint32{3000, 8080}, inv.Apps[0].Ports)
	require.Equal(t, 4.0, inv.Apps[0].CPUPercent)
}

type fixtureSystemd struct {
	units []SystemdUnit
	err   error
}

func (f fixtureSystemd) ListUnits(context.Context) ([]SystemdUnit, error) {
	return f.units, f.err
}

type fixturePM2 struct {
	apps []PM2App
	err  error
}

func (f fixturePM2) ListApps(context.Context) ([]PM2App, error) {
	return f.apps, f.err
}

type fixtureProcesses struct {
	procs    []ProcessInfo
	ports    []ListeningPort
	err      error
	portsErr error
}

func (f fixtureProcesses) ListProcesses(context.Context) ([]ProcessInfo, error) {
	return f.procs, f.err
}

func (f fixtureProcesses) ListListeningPorts(context.Context) ([]ListeningPort, error) {
	return f.ports, f.portsErr
}

type failingProcessSource struct{}

func (failingProcessSource) ListProcesses(context.Context) ([]ProcessInfo, error) {
	return nil, errors.New("proc unavailable")
}

func (failingProcessSource) ListListeningPorts(context.Context) ([]ListeningPort, error) {
	return nil, nil
}

func newFixtureCollector(units []SystemdUnit, apps []PM2App, procs []ProcessInfo, ports []ListeningPort) *HostInventoryCollector {
	return &HostInventoryCollector{
		logger:    slog.Default(),
		systemd:   fixtureSystemd{units: units},
		pm2:       fixturePM2{apps: apps},
		processes: fixtureProcesses{procs: procs, ports: ports},
	}
}
