package executor

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"strings"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/errdefs"
	"github.com/docker/go-connections/nat"
)

// ContainerSpec mirrors the shared model for agent-side deserialization.
type ContainerSpec struct {
	Name          string            `json:"name"`
	Image         string            `json:"image"`
	Ports         []PortMapping     `json:"ports"`
	Env           map[string]string `json:"env"`
	Volumes       []VolumeMount     `json:"volumes"`
	Labels        map[string]string `json:"labels"`
	Network       string            `json:"network"`
	CPULimit      float64           `json:"cpu_limit"`
	MemLimit      int64             `json:"mem_limit"`
	PidsLimit     int64             `json:"pids_limit"`
	RestartPolicy string            `json:"restart_policy"`
	Command       []string          `json:"command"`
	Entrypoint    []string          `json:"entrypoint"`
	AutoStart     bool              `json:"auto_start"`
}

// PortMapping defines a host-to-container port binding.
type PortMapping struct {
	HostPort      string `json:"host_port"`
	ContainerPort string `json:"container_port"`
	Protocol      string `json:"protocol"`
}

// VolumeMount defines a volume or bind mount.
type VolumeMount struct {
	Source   string `json:"source"`
	Target   string `json:"target"`
	ReadOnly bool   `json:"read_only"`
}

// ParseContainerSpec deserializes a ContainerSpec from JSON.
func ParseContainerSpec(specJSON string) (*ContainerSpec, error) {
	var spec ContainerSpec
	if err := json.Unmarshal([]byte(specJSON), &spec); err != nil {
		return nil, fmt.Errorf("parse container spec: %w", err)
	}
	return &spec, nil
}

// Create creates a new Docker container from the given spec.
// If the image is not found locally, it will attempt to pull it automatically.
func (e *DockerExecutor) Create(ctx context.Context, spec *ContainerSpec) Result {
	ctx, cancel := context.WithTimeout(ctx, 2*defaultTimeout) // extra time for possible image pull
	defer cancel()

	// Build Docker configs.
	config, hostConfig, networkConfig, err := buildDockerConfigs(spec)
	if err != nil {
		return Result{Error: fmt.Sprintf("invalid container spec: %s", err.Error())}
	}

	// Attempt creation.
	resp, err := e.cli.ContainerCreate(ctx, config, hostConfig, networkConfig, nil, spec.Name)
	if err != nil {
		// Auto-pull image if not found.
		if errdefs.IsNotFound(err) {
			slog.Info("image not found locally, pulling", "image", spec.Image)
			pullResult := e.PullImage(ctx, spec.Image)
			if pullResult.Error != "" {
				return Result{Error: fmt.Sprintf("auto-pull failed for %s: %s", spec.Image, pullResult.Error)}
			}

			// Retry creation after pull.
			resp, err = e.cli.ContainerCreate(ctx, config, hostConfig, networkConfig, nil, spec.Name)
			if err != nil {
				return mapCreateError(err, spec.Name)
			}
		} else {
			return mapCreateError(err, spec.Name)
		}
	}

	containerID := resp.ID
	if len(containerID) > 12 {
		containerID = containerID[:12]
	}

	// Auto-start if requested (default behavior).
	if spec.AutoStart {
		if startErr := e.cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); startErr != nil {
			return Result{
				Success: true,
				Output:  fmt.Sprintf("container %s created but failed to start: %s", containerID, startErr.Error()),
			}
		}
		return Result{Success: true, Output: fmt.Sprintf("container %s created and started", containerID)}
	}

	return Result{Success: true, Output: fmt.Sprintf("container %s created", containerID)}
}

// PullImage pulls a Docker image by reference.
func (e *DockerExecutor) PullImage(ctx context.Context, imageRef string) Result {
	ctx, cancel := context.WithTimeout(ctx, 5*defaultTimeout) // image pull can take a while
	defer cancel()

	reader, err := e.cli.ImagePull(ctx, imageRef, image.PullOptions{})
	if err != nil {
		return Result{Error: fmt.Sprintf("pull image %s: %s", imageRef, err.Error())}
	}
	defer func() {
		if closeErr := reader.Close(); closeErr != nil {
			slog.Warn("failed to close image pull reader", "image", imageRef, "error", closeErr)
		}
	}()

	// Drain the reader to complete the pull.
	if _, err := io.Copy(io.Discard, reader); err != nil {
		return Result{Error: fmt.Sprintf("pull image %s stream: %s", imageRef, err.Error())}
	}

	return Result{Success: true, Output: fmt.Sprintf("image %s pulled", imageRef)}
}

// buildDockerConfigs converts a ContainerSpec into Docker SDK config structs.
func buildDockerConfigs(spec *ContainerSpec) (*container.Config, *container.HostConfig, *network.NetworkingConfig, error) {
	// Build exposed ports and port bindings.
	exposedPorts := nat.PortSet{}
	portBindings := nat.PortMap{}

	for _, p := range spec.Ports {
		proto := strings.ToLower(p.Protocol)
		if proto == "" {
			proto = "tcp"
		}
		port, err := nat.NewPort(proto, p.ContainerPort)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("invalid port %s/%s: %w", p.ContainerPort, proto, err)
		}
		exposedPorts[port] = struct{}{}
		if p.HostPort != "" {
			portBindings[port] = []nat.PortBinding{
				{HostPort: p.HostPort},
			}
		}
	}

	// Build env slice.
	var envSlice []string
	for k, v := range spec.Env {
		envSlice = append(envSlice, k+"="+v)
	}

	// Container config.
	config := &container.Config{
		Image:        spec.Image,
		Env:          envSlice,
		Labels:       spec.Labels,
		ExposedPorts: exposedPorts,
	}

	if len(spec.Command) > 0 {
		config.Cmd = spec.Command
	}
	if len(spec.Entrypoint) > 0 {
		config.Entrypoint = spec.Entrypoint
	}

	// Build binds for volumes.
	var binds []string
	for _, v := range spec.Volumes {
		bind := v.Source + ":" + v.Target
		if v.ReadOnly {
			bind += ":ro"
		}
		binds = append(binds, bind)
	}

	// Restart policy.
	restartPolicy := container.RestartPolicy{}
	if spec.RestartPolicy != "" {
		restartPolicy.Name = container.RestartPolicyMode(spec.RestartPolicy)
	}

	// Host config with resource limits.
	hostConfig := &container.HostConfig{
		PortBindings:  portBindings,
		Binds:         binds,
		RestartPolicy: restartPolicy,
		Resources: container.Resources{
			NanoCPUs: int64(spec.CPULimit * 1e9),
			Memory:   spec.MemLimit,
		},
	}

	if spec.PidsLimit > 0 {
		hostConfig.PidsLimit = &spec.PidsLimit
	}

	// Network config.
	var networkConfig *network.NetworkingConfig
	if spec.Network != "" {
		networkConfig = &network.NetworkingConfig{
			EndpointsConfig: map[string]*network.EndpointSettings{
				spec.Network: {},
			},
		}
	}

	return config, hostConfig, networkConfig, nil
}

// mapCreateError translates Docker creation errors into Result.
func mapCreateError(err error, name string) Result {
	if errdefs.IsConflict(err) {
		return Result{Error: fmt.Sprintf("container name %q already in use", name)}
	}
	if errdefs.IsNotFound(err) {
		return Result{Error: fmt.Sprintf("image not found for container %q", name)}
	}
	return Result{Error: fmt.Sprintf("create container %q: %s", name, err.Error())}
}
