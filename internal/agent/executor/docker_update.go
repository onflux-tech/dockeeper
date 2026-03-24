package executor

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/go-connections/nat"
)

// ResourceUpdate holds the resource limits to apply via Docker ContainerUpdate.
type ResourceUpdate struct {
	CPULimit  float64 `json:"cpu_limit"`  // CPU cores (e.g. 0.5)
	MemLimit  int64   `json:"mem_limit"`  // bytes
	PidsLimit int64   `json:"pids_limit"`
}

// ParseResourceUpdate deserializes a ResourceUpdate from JSON.
func ParseResourceUpdate(data string) (*ResourceUpdate, error) {
	var ru ResourceUpdate
	if err := json.Unmarshal([]byte(data), &ru); err != nil {
		return nil, fmt.Errorf("parse resource update: %w", err)
	}
	return &ru, nil
}

// UpdateResources applies resource limits to a running container without restart.
func (e *DockerExecutor) UpdateResources(ctx context.Context, containerID string, update *ResourceUpdate) Result {
	ctx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	resources := container.Resources{
		NanoCPUs: int64(update.CPULimit * 1e9),
		Memory:   update.MemLimit,
	}

	if update.PidsLimit > 0 {
		resources.PidsLimit = &update.PidsLimit
	}

	_, err := e.cli.ContainerUpdate(ctx, containerID, container.UpdateConfig{
		Resources: resources,
	})
	if err != nil {
		return errResult(err, containerID)
	}

	return Result{
		Success: true,
		Output:  fmt.Sprintf("container %s resources updated", containerID),
	}
}

// RecreateWithEnv recreates a container with new environment variables.
// Docker does not support hot-updating env vars, so the container is stopped,
// removed, recreated with the same config but updated env, and started again.
func (e *DockerExecutor) RecreateWithEnv(ctx context.Context, containerID string, newEnv []string) Result {
	ctx, cancel := context.WithTimeout(ctx, 3*defaultTimeout)
	defer cancel()

	// 1. Inspect the existing container to capture its full config.
	inspect, err := e.cli.ContainerInspect(ctx, containerID)
	if err != nil {
		return errResult(err, containerID)
	}

	wasRunning := inspect.State != nil && inspect.State.Running
	containerName := strings.TrimPrefix(inspect.Name, "/")

	// 2. Stop the container if running.
	if wasRunning {
		if stopErr := e.cli.ContainerStop(ctx, containerID, container.StopOptions{}); stopErr != nil {
			return Result{Error: fmt.Sprintf("stop before recreate: %s", stopErr.Error())}
		}
	}

	// 3. Remove the old container.
	if removeErr := e.cli.ContainerRemove(ctx, containerID, container.RemoveOptions{}); removeErr != nil {
		// Try to restart if removal fails.
		if wasRunning {
			_ = e.cli.ContainerStart(ctx, containerID, container.StartOptions{})
		}
		return Result{Error: fmt.Sprintf("remove before recreate: %s", removeErr.Error())}
	}

	slog.Info("old container removed for env recreate",
		"container_id", containerID,
		"name", containerName,
	)

	// 4. Build new config from the old one but with updated env.
	config := inspect.Config
	config.Env = newEnv

	// Rebuild host config.
	hostConfig := inspect.HostConfig

	// Rebuild port bindings from the inspected config.
	if config.ExposedPorts == nil {
		config.ExposedPorts = nat.PortSet{}
	}

	// Rebuild network config.
	var networkConfig *network.NetworkingConfig
	if inspect.NetworkSettings != nil && len(inspect.NetworkSettings.Networks) > 0 {
		endpoints := make(map[string]*network.EndpointSettings)
		for name, ep := range inspect.NetworkSettings.Networks {
			endpoints[name] = &network.EndpointSettings{
				IPAMConfig:  ep.IPAMConfig,
				Aliases:     ep.Aliases,
				NetworkID:   ep.NetworkID,
				IPAddress:   ep.IPAddress,
				MacAddress:  ep.MacAddress,
			}
		}
		networkConfig = &network.NetworkingConfig{EndpointsConfig: endpoints}
	}

	// 5. Create the new container with the same name.
	resp, createErr := e.cli.ContainerCreate(ctx, config, hostConfig, networkConfig, nil, containerName)
	if createErr != nil {
		return Result{Error: fmt.Sprintf("recreate container %q: %s", containerName, createErr.Error())}
	}

	newID := resp.ID
	if len(newID) > 12 {
		newID = newID[:12]
	}

	// 6. Start the container if it was previously running.
	if wasRunning {
		if startErr := e.cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); startErr != nil {
			return Result{
				Success: true,
				Output:  fmt.Sprintf("container %s recreated but failed to start: %s", newID, startErr.Error()),
			}
		}
	}

	slog.Info("container recreated with new env",
		"old_id", containerID,
		"new_id", newID,
		"name", containerName,
	)

	return Result{
		Success: true,
		Output:  fmt.Sprintf("container %s recreated with updated environment (new ID: %s)", containerName, newID),
	}
}
