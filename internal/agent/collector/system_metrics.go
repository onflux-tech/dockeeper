package collector

import (
	"context"
	"runtime"
	"sync"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/host"
	"github.com/shirou/gopsutil/v4/load"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/shirou/gopsutil/v4/net"
	"github.com/shirou/gopsutil/v4/sensors"
)

// ignoredFSTypes lists pseudo-filesystems that should be excluded from
// partition monitoring. Only real block devices are reported.
var ignoredFSTypes = map[string]bool{
	"tmpfs":           true,
	"devtmpfs":        true,
	"squashfs":        true,
	"overlay":         true,
	"aufs":            true,
	"devfs":           true,
	"sysfs":           true,
	"proc":            true,
	"cgroup":          true,
	"cgroup2":         true,
	"nsfs":            true,
	"tracefs":         true,
	"debugfs":         true,
	"securityfs":      true,
	"pstore":          true,
	"bpf":             true,
	"mqueue":          true,
	"hugetlbfs":       true,
	"configfs":        true,
	"fusectl":         true,
	"fuse.lxcfs":      true,
	"efivarfs":        true,
	"autofs":          true,
	"binfmt_misc":     true,
	"ramfs":           true,
	"rootfs":          true,
	"rpc_pipefs":      true,
	"nfsd":            true,
	"fuse.gvfsd-fuse": true,
}

// TemperatureReading holds a single sensor temperature.
type TemperatureReading struct {
	Name    string
	Celsius float64
}

// DiskPartitionInfo holds usage data for a single disk partition.
type DiskPartitionInfo struct {
	Mountpoint string
	Device     string
	FSType     string
	Total      uint64
	Used       uint64
	Available  uint64
	Percent    float64
}

// SystemMetrics holds a single snapshot of host-level metrics.
type SystemMetrics struct {
	CPUPercent    float64
	MemPercent    float64
	MemUsed       uint64
	MemTotal      uint64
	MemAvailable  uint64
	DiskUsed      uint64
	DiskTotal     uint64
	NetRx         uint64 // bytes received since last sample
	NetTx         uint64 // bytes sent since last sample
	Load1m        float64
	Load5m        float64
	Load15m       float64
	DiskIORead    uint64 // bytes read since last sample (delta)
	DiskIOWrite   uint64 // bytes written since last sample (delta)
	SwapUsed      uint64
	SwapTotal     uint64
	Temperatures  []TemperatureReading
	Partitions    []DiskPartitionInfo
	CPUCores      uint32
	CPUModel      string
	UptimeSeconds uint64
}

// SystemCollector collects host-level system metrics using gopsutil.
type SystemCollector struct {
	mu            sync.Mutex
	prevNetRx     uint64
	prevNetTx     uint64
	prevDiskRead  uint64
	prevDiskWrite uint64
	hasPrev       bool

	// Static info — resolved once.
	cpuModel string
	cpuCores uint32
	infoOnce sync.Once
}

// NewSystemCollector creates a SystemCollector.
func NewSystemCollector() *SystemCollector {
	return &SystemCollector{}
}

// initStaticInfo gathers CPU model and core count once.
func (s *SystemCollector) initStaticInfo() {
	s.infoOnce.Do(func() {
		if infos, err := cpu.Info(); err == nil && len(infos) > 0 {
			s.cpuModel = infos[0].ModelName
		}
		if cores, err := cpu.Counts(true); err == nil {
			s.cpuCores = uint32(cores)
		}
	})
}

// Collect gathers a single snapshot of system metrics.
func (s *SystemCollector) Collect(ctx context.Context) (*SystemMetrics, error) {
	s.initStaticInfo()
	m := &SystemMetrics{
		CPUModel: s.cpuModel,
		CPUCores: s.cpuCores,
	}

	// CPU percent (aggregate across all cores).
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
		m.MemAvailable = vm.Available
	}

	// Swap.
	swap, err := mem.SwapMemoryWithContext(ctx)
	if err == nil {
		m.SwapUsed = swap.Used
		m.SwapTotal = swap.Total
	}

	// Multi-partition disk usage.
	m.Partitions = collectPartitions(ctx)

	// Keep root partition in top-level fields for backward compatibility.
	root := "/"
	if runtime.GOOS == "windows" {
		root = "C:\\"
	}
	for _, p := range m.Partitions {
		if p.Mountpoint == root {
			m.DiskUsed = p.Used
			m.DiskTotal = p.Total
			break
		}
	}
	// Fallback if root not in filtered partitions.
	if m.DiskTotal == 0 {
		if du, err := disk.UsageWithContext(ctx, root); err == nil {
			m.DiskUsed = du.Used
			m.DiskTotal = du.Total
		}
	}

	// Network I/O — delta since last sample.
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
		s.mu.Unlock()
	}

	// Disk I/O — delta since last sample.
	ioCounters, err := disk.IOCountersWithContext(ctx)
	if err == nil {
		var totalRead, totalWrite uint64
		for _, c := range ioCounters {
			totalRead += c.ReadBytes
			totalWrite += c.WriteBytes
		}
		s.mu.Lock()
		if s.hasPrev {
			m.DiskIORead = totalRead - s.prevDiskRead
			m.DiskIOWrite = totalWrite - s.prevDiskWrite
		}
		s.prevDiskRead = totalRead
		s.prevDiskWrite = totalWrite

		if !s.hasPrev {
			// First sample also sets hasPrev (network sets it above,
			// but if network fails we still need to track disk).
			s.hasPrev = true
		}
		s.mu.Unlock()
	}

	// Ensure hasPrev is set even if disk IO fails but network succeeded.
	s.mu.Lock()
	if !s.hasPrev {
		s.hasPrev = true
	}
	s.mu.Unlock()

	// Load average (Linux/macOS only).
	avg, err := load.AvgWithContext(ctx)
	if err == nil {
		m.Load1m = avg.Load1
		m.Load5m = avg.Load5
		m.Load15m = avg.Load15
	}

	// Temperature sensors (graceful on unsupported platforms).
	temps, err := sensors.TemperaturesWithContext(ctx)
	if err == nil {
		for _, t := range temps {
			if t.Temperature > 0 {
				m.Temperatures = append(m.Temperatures, TemperatureReading{
					Name:    t.SensorKey,
					Celsius: t.Temperature,
				})
			}
		}
	}

	// System uptime.
	if uptime, err := host.UptimeWithContext(ctx); err == nil {
		m.UptimeSeconds = uptime
	}

	return m, nil
}

// collectPartitions gathers usage info for all real (non-pseudo) partitions.
func collectPartitions(ctx context.Context) []DiskPartitionInfo {
	parts, err := disk.PartitionsWithContext(ctx, false)
	if err != nil {
		return nil
	}

	seen := make(map[string]bool)
	var result []DiskPartitionInfo

	for _, p := range parts {
		if ignoredFSTypes[p.Fstype] {
			continue
		}
		// Deduplicate by mountpoint (bind mounts can repeat).
		if seen[p.Mountpoint] {
			continue
		}

		usage, err := disk.UsageWithContext(ctx, p.Mountpoint)
		if err != nil || usage.Total == 0 {
			continue
		}

		seen[p.Mountpoint] = true
		result = append(result, DiskPartitionInfo{
			Mountpoint: p.Mountpoint,
			Device:     p.Device,
			FSType:     p.Fstype,
			Total:      usage.Total,
			Used:       usage.Used,
			Available:  usage.Free,
			Percent:    usage.UsedPercent,
		})
	}

	return result
}
