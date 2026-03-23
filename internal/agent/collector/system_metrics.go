package collector

import (
	"context"
	"runtime"
	"sync"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/load"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/shirou/gopsutil/v4/net"
)

// SystemMetrics holds a single snapshot of host-level metrics.
type SystemMetrics struct {
	CPUPercent float64
	MemPercent float64
	MemUsed    uint64
	MemTotal   uint64
	DiskUsed   uint64
	DiskTotal  uint64
	NetRx      uint64 // bytes received since last sample
	NetTx      uint64 // bytes sent since last sample
	Load1m     float64
	Load5m     float64
	Load15m    float64
}

// SystemCollector collects host-level system metrics using gopsutil.
type SystemCollector struct {
	mu        sync.Mutex
	prevNetRx uint64
	prevNetTx uint64
	hasPrev   bool
}

// NewSystemCollector creates a SystemCollector.
func NewSystemCollector() *SystemCollector {
	return &SystemCollector{}
}

// Collect gathers a single snapshot of system metrics.
func (s *SystemCollector) Collect(ctx context.Context) (*SystemMetrics, error) {
	m := &SystemMetrics{}

	// CPU percent (aggregate across all cores, 0-length interval uses delta
	// from the previous call internally tracked by gopsutil).
	cpuPcts, err := cpu.PercentWithContext(ctx, 0, false)
	if err == nil && len(cpuPcts) > 0 {
		m.CPUPercent = cpuPcts[0]
	}

	// Memory.
	vm, err := mem.VirtualMemoryWithContext(ctx)
	if err == nil {
		m.MemPercent = vm.UsedPercent
		m.MemUsed = vm.Used
		m.MemTotal = vm.Total
	}

	// Disk (root partition on Linux, C: on Windows).
	root := "/"
	if runtime.GOOS == "windows" {
		root = "C:\\"
	}
	du, err := disk.UsageWithContext(ctx, root)
	if err == nil {
		m.DiskUsed = du.Used
		m.DiskTotal = du.Total
	}

	// Network I/O (all interfaces combined) — delta since last sample.
	counters, err := net.IOCountersWithContext(ctx, false)
	if err == nil && len(counters) > 0 {
		totalRx := counters[0].BytesRecv
		totalTx := counters[0].BytesSent

		s.mu.Lock()
		if s.hasPrev {
			m.NetRx = totalRx - s.prevNetRx
			m.NetTx = totalTx - s.prevNetTx
		}
		s.prevNetRx = totalRx
		s.prevNetTx = totalTx
		s.hasPrev = true
		s.mu.Unlock()
	}

	// Load average (Linux / macOS only; returns error on Windows).
	avg, err := load.AvgWithContext(ctx)
	if err == nil {
		m.Load1m = avg.Load1
		m.Load5m = avg.Load5
		m.Load15m = avg.Load15
	}

	return m, nil
}
