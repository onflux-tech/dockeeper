package collector

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/shirou/gopsutil/v4/net"
	"github.com/shirou/gopsutil/v4/process"
)

type ProcessInfo struct {
	PID      uint32
	Name     string
	CPU      float64
	Mem      uint64
	Threads  uint32
	User     string
	Cwd      string
	ExecPath string
}

type ListeningPort struct {
	Port     uint32
	Protocol string
	Address  string
	PID      uint32
}

type procfsCollector struct {
	logger *slog.Logger
}

func (c *procfsCollector) ListProcesses(ctx context.Context) ([]ProcessInfo, error) {
	processes, err := process.ProcessesWithContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("list processes: %w", err)
	}

	infos := make([]ProcessInfo, 0, len(processes))
	for _, item := range processes {
		name, err := item.NameWithContext(ctx)
		if err != nil || name == "" {
			continue
		}

		info := ProcessInfo{
			PID:  uint32(item.Pid),
			Name: name,
		}
		if cpu, err := item.CPUPercentWithContext(ctx); err == nil {
			info.CPU = cpu
		}
		if memory, err := item.MemoryInfoWithContext(ctx); err == nil && memory != nil {
			info.Mem = memory.RSS
		}
		if threads, err := item.NumThreadsWithContext(ctx); err == nil {
			info.Threads = uint32(threads)
		}
		if username, err := item.UsernameWithContext(ctx); err == nil {
			info.User = username
		}
		if cwd, err := item.CwdWithContext(ctx); err == nil {
			info.Cwd = cwd
		}
		if executable, err := item.ExeWithContext(ctx); err == nil {
			info.ExecPath = executable
		}

		infos = append(infos, info)
	}

	return infos, nil
}

func (c *procfsCollector) ListListeningPorts(ctx context.Context) ([]ListeningPort, error) {
	ports := make([]ListeningPort, 0)
	seen := make(map[string]bool)
	var lastErr error
	succeeded := false

	for _, kind := range []string{"tcp", "tcp6"} {
		connections, err := net.ConnectionsWithContext(ctx, kind)
		if err != nil {
			lastErr = err
			continue
		}
		succeeded = true

		for _, connection := range connections {
			if connection.Status != "LISTEN" || connection.Pid <= 0 {
				continue
			}
			key := fmt.Sprintf("%s:%d:%d", connection.Laddr.IP, connection.Laddr.Port, connection.Pid)
			if seen[key] {
				continue
			}
			seen[key] = true

			ports = append(ports, ListeningPort{
				Port:     connection.Laddr.Port,
				Protocol: strings.TrimSuffix(kind, "6"),
				Address:  connection.Laddr.IP,
				PID:      uint32(connection.Pid),
			})
		}
	}

	if !succeeded && lastErr != nil {
		return nil, fmt.Errorf("list listening ports: %w", lastErr)
	}

	return ports, nil
}
