package collector

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/volume"
	dockerclient "github.com/docker/docker/client"
	"google.golang.org/protobuf/types/known/timestamppb"

	agentpb "github.com/jkdesenvolvedor/dockeeper/proto/gen/dockeeper/agent/v1"
)

// ContainerMetrics holds per-container metrics extracted from Docker stats.
type ContainerMetrics struct {
	ContainerID string
	CPUPercent  float64
	MemUsed     uint64
	MemLimit    uint64
	NetRx       uint64
	NetTx       uint64
	BlockRead   uint64
	BlockWrite  uint64
	PIDs        uint32
}

// OrchestratorClient is the interface for collecting resource data from a
// container runtime (Docker, Podman, Kubernetes).
type OrchestratorClient interface {
	ListContainers(ctx context.Context) ([]*agentpb.ContainerInfo, error)
	ListImages(ctx context.Context) ([]*agentpb.ImageInfo, error)
	ListNetworks(ctx context.Context) ([]*agentpb.NetworkInfo, error)
	ListVolumes(ctx context.Context) ([]*agentpb.VolumeInfo, error)
	ListContainerStats(ctx context.Context) ([]ContainerMetrics, error)
	Ping(ctx context.Context) error
	Close() error
}

// DockerClient implements OrchestratorClient using the Docker SDK.
type DockerClient struct {
	cli *dockerclient.Client
}

// NewDockerClient creates a Docker client from environment settings.
func NewDockerClient() (*DockerClient, error) {
	cli, err := dockerclient.NewClientWithOpts(
		dockerclient.FromEnv,
		dockerclient.WithAPIVersionNegotiation(),
	)
	if err != nil {
		return nil, fmt.Errorf("create docker client: %w", err)
	}
	return &DockerClient{cli: cli}, nil
}

func (d *DockerClient) Ping(ctx context.Context) error {
	_, err := d.cli.Ping(ctx)
	return err
}

func (d *DockerClient) Close() error {
	return d.cli.Close()
}

func (d *DockerClient) ListContainers(ctx context.Context) ([]*agentpb.ContainerInfo, error) {
	containers, err := d.cli.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		return nil, fmt.Errorf("docker container list: %w", err)
	}

	result := make([]*agentpb.ContainerInfo, 0, len(containers))
	for _, c := range containers {
		info, err := d.inspectContainer(ctx, c.ID)
		if err != nil {
			slog.Warn("failed to inspect container, using summary", "id", c.ID[:12], "error", err)
			info = containerFromSummary(&c)
		}
		result = append(result, info)
	}
	return result, nil
}

func (d *DockerClient) inspectContainer(ctx context.Context, id string) (*agentpb.ContainerInfo, error) {
	inspect, err := d.cli.ContainerInspect(ctx, id)
	if err != nil {
		return nil, err
	}

	portsJSON := marshalJSON(inspect.NetworkSettings.Ports)
	labelsJSON := marshalJSON(inspect.Config.Labels)
	mountsJSON := marshalJSON(inspect.Mounts)
	networksJSON := marshalJSON(inspect.NetworkSettings.Networks)

	var limits *agentpb.ResourceLimits
	if inspect.HostConfig != nil {
		var pidsLimit uint32
		if inspect.HostConfig.PidsLimit != nil {
			pidsLimit = uint32(*inspect.HostConfig.PidsLimit)
		}
		limits = &agentpb.ResourceLimits{
			CpuLimit:  float64(inspect.HostConfig.NanoCPUs) / 1e9,
			MemLimit:  uint64(inspect.HostConfig.Memory),
			PidsLimit: pidsLimit,
		}
	}

	var restartPolicy string
	if inspect.HostConfig != nil {
		restartPolicy = string(inspect.HostConfig.RestartPolicy.Name)
	}

	createdAt := parseDockerTime(inspect.Created)
	startedAt := parseDockerTime(inspect.State.StartedAt)

	name := strings.TrimPrefix(inspect.Name, "/")

	return &agentpb.ContainerInfo{
		ContainerId:   inspect.ID[:12],
		Name:          name,
		Image:         inspect.Config.Image,
		Status:        inspect.State.Status,
		PortsJson:     portsJSON,
		LabelsJson:    labelsJSON,
		VolumesJson:   mountsJSON,
		NetworksJson:  networksJSON,
		ResourceLimits: limits,
		RestartPolicy: restartPolicy,
		Command:       strings.Join(inspect.Config.Cmd, " "),
		Entrypoint:    strings.Join(inspect.Config.Entrypoint, " "),
		CreatedAt:     createdAt,
		StartedAt:     startedAt,
	}, nil
}

func containerFromSummary(c *container.Summary) *agentpb.ContainerInfo {
	name := ""
	if len(c.Names) > 0 {
		name = strings.TrimPrefix(c.Names[0], "/")
	}
	return &agentpb.ContainerInfo{
		ContainerId: c.ID[:12],
		Name:        name,
		Image:       c.Image,
		Status:      c.State,
		PortsJson:   marshalJSON(c.Ports),
		LabelsJson:  marshalJSON(c.Labels),
		CreatedAt:   timestamppb.New(time.Unix(c.Created, 0)),
	}
}

func (d *DockerClient) ListImages(ctx context.Context) ([]*agentpb.ImageInfo, error) {
	images, err := d.cli.ImageList(ctx, image.ListOptions{All: true})
	if err != nil {
		return nil, fmt.Errorf("docker image list: %w", err)
	}

	// Build set of images in use by containers.
	usedImages := d.usedImageIDs(ctx)

	result := make([]*agentpb.ImageInfo, 0, len(images))
	for _, img := range images {
		imgID := strings.TrimPrefix(img.ID, "sha256:")
		if len(imgID) > 12 {
			imgID = imgID[:12]
		}
		result = append(result, &agentpb.ImageInfo{
			ImageId:   imgID,
			Tags:      img.RepoTags,
			Size:      uint64(img.Size),
			CreatedAt: timestamppb.New(time.Unix(img.Created, 0)),
			InUse:     usedImages[img.ID],
		})
	}
	return result, nil
}

func (d *DockerClient) usedImageIDs(ctx context.Context) map[string]bool {
	containers, err := d.cli.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		return nil
	}
	used := make(map[string]bool, len(containers))
	for _, c := range containers {
		used[c.ImageID] = true
	}
	return used
}

func (d *DockerClient) ListNetworks(ctx context.Context) ([]*agentpb.NetworkInfo, error) {
	networks, err := d.cli.NetworkList(ctx, network.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("docker network list: %w", err)
	}

	result := make([]*agentpb.NetworkInfo, 0, len(networks))
	for _, n := range networks {
		nID := n.ID
		if len(nID) > 12 {
			nID = nID[:12]
		}

		var subnet, gateway string
		if len(n.IPAM.Config) > 0 {
			subnet = n.IPAM.Config[0].Subnet
			gateway = n.IPAM.Config[0].Gateway
		}

		result = append(result, &agentpb.NetworkInfo{
			NetworkId:  nID,
			Name:       n.Name,
			Driver:     n.Driver,
			Scope:      n.Scope,
			Subnet:     subnet,
			Gateway:    gateway,
			Internal:   n.Internal,
			LabelsJson: marshalJSON(n.Labels),
		})
	}
	return result, nil
}

func (d *DockerClient) ListVolumes(ctx context.Context) ([]*agentpb.VolumeInfo, error) {
	resp, err := d.cli.VolumeList(ctx, volume.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("docker volume list: %w", err)
	}

	// Build set of volumes in use.
	usedVolumes := d.usedVolumeNames(ctx)

	result := make([]*agentpb.VolumeInfo, 0, len(resp.Volumes))
	for _, v := range resp.Volumes {
		result = append(result, &agentpb.VolumeInfo{
			Name:       v.Name,
			Driver:     v.Driver,
			Mountpoint: v.Mountpoint,
			LabelsJson: marshalJSON(v.Labels),
			InUse:      usedVolumes[v.Name],
		})
	}
	return result, nil
}

func (d *DockerClient) ListContainerStats(ctx context.Context) ([]ContainerMetrics, error) {
	containers, err := d.cli.ContainerList(ctx, container.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("docker container list for stats: %w", err)
	}

	result := make([]ContainerMetrics, 0, len(containers))
	for _, c := range containers {
		id := c.ID
		shortID := id
		if len(shortID) > 12 {
			shortID = shortID[:12]
		}

		statsResp, err := d.cli.ContainerStats(ctx, id, false)
		if err != nil {
			slog.Warn("failed to get container stats", "id", shortID, "error", err)
			continue
		}

		var stats container.StatsResponse
		if decErr := json.NewDecoder(statsResp.Body).Decode(&stats); decErr != nil {
			if closeErr := statsResp.Body.Close(); closeErr != nil {
				slog.Warn("failed to close stats body", "id", shortID, "error", closeErr)
			}
			slog.Warn("failed to decode container stats", "id", shortID, "error", decErr)
			continue
		}
		if closeErr := statsResp.Body.Close(); closeErr != nil {
			slog.Warn("failed to close stats body", "id", shortID, "error", closeErr)
		}

		m := ContainerMetrics{
			ContainerID: shortID,
			CPUPercent:  calculateCPUPercent(&stats),
			MemUsed:     stats.MemoryStats.Usage,
			MemLimit:    stats.MemoryStats.Limit,
			PIDs:        uint32(stats.PidsStats.Current),
		}

		for _, ns := range stats.Networks {
			m.NetRx += ns.RxBytes
			m.NetTx += ns.TxBytes
		}

		for _, entry := range stats.BlkioStats.IoServiceBytesRecursive {
			switch entry.Op {
			case "read", "Read":
				m.BlockRead += entry.Value
			case "write", "Write":
				m.BlockWrite += entry.Value
			}
		}

		result = append(result, m)
	}
	return result, nil
}

func calculateCPUPercent(stats *container.StatsResponse) float64 {
	cpuDelta := float64(stats.CPUStats.CPUUsage.TotalUsage - stats.PreCPUStats.CPUUsage.TotalUsage)
	systemDelta := float64(stats.CPUStats.SystemUsage - stats.PreCPUStats.SystemUsage)

	if systemDelta <= 0 || cpuDelta < 0 {
		return 0
	}

	cpus := float64(stats.CPUStats.OnlineCPUs)
	if cpus == 0 {
		cpus = float64(len(stats.CPUStats.CPUUsage.PercpuUsage))
	}
	if cpus == 0 {
		cpus = 1
	}

	pct := (cpuDelta / systemDelta) * cpus * 100.0
	if pct > 100*cpus {
		pct = 100 * cpus
	}
	return pct
}

func (d *DockerClient) usedVolumeNames(ctx context.Context) map[string]bool {
	containers, err := d.cli.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		return nil
	}
	used := make(map[string]bool)
	for _, c := range containers {
		for _, m := range c.Mounts {
			if m.Type == "volume" {
				used[m.Name] = true
			}
		}
	}
	return used
}

func marshalJSON(v any) string {
	data, err := json.Marshal(v)
	if err != nil {
		return "{}"
	}
	return string(data)
}

func parseDockerTime(s string) *timestamppb.Timestamp {
	if s == "" || s == "0001-01-01T00:00:00Z" {
		return nil
	}
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		return nil
	}
	return timestamppb.New(t)
}
