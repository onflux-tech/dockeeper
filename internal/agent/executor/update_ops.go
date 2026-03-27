package executor

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/go-connections/nat"
)

// ContainerSnapshot captures the full config of a container for rollback.
type ContainerSnapshot struct {
	Name          string
	Image         string
	WasRunning    bool
	Config        *container.Config
	HostConfig    *container.HostConfig
	NetworkConfig *network.NetworkingConfig
}

// CheckImageDigest inspects the current image digest for a given image reference.
func (e *DockerExecutor) CheckImageDigest(ctx context.Context, imageRef string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	inspect, err := e.cli.ImageInspect(ctx, imageRef)
	if err != nil {
		return "", fmt.Errorf("inspect image %s: %w", imageRef, err)
	}

	// RepoDigests contains image@sha256:... entries
	for _, d := range inspect.RepoDigests {
		if idx := strings.Index(d, "@"); idx >= 0 {
			return d[idx+1:], nil
		}
	}

	return inspect.ID, nil
}

// PullImageWithAuth pulls an image with optional registry auth.
func (e *DockerExecutor) PullImageWithAuth(ctx context.Context, imageRef, registryAuth string) (io.ReadCloser, error) {
	opts := image.PullOptions{}
	if registryAuth != "" {
		opts.RegistryAuth = registryAuth
	}
	return e.cli.ImagePull(ctx, imageRef, opts)
}

// SnapshotContainer captures the full config of a container for later rollback.
func (e *DockerExecutor) SnapshotContainer(ctx context.Context, containerID string) (*ContainerSnapshot, error) {
	ctx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	inspect, err := e.cli.ContainerInspect(ctx, containerID)
	if err != nil {
		return nil, fmt.Errorf("inspect container %s: %w", containerID, err)
	}

	snap := &ContainerSnapshot{
		Name:       strings.TrimPrefix(inspect.Name, "/"),
		Image:      inspect.Config.Image,
		WasRunning: inspect.State != nil && inspect.State.Running,
		Config:     inspect.Config,
		HostConfig: inspect.HostConfig,
	}

	if inspect.Config.ExposedPorts == nil {
		snap.Config.ExposedPorts = nat.PortSet{}
	}

	if inspect.NetworkSettings != nil && len(inspect.NetworkSettings.Networks) > 0 {
		endpoints := make(map[string]*network.EndpointSettings)
		for name, ep := range inspect.NetworkSettings.Networks {
			endpoints[name] = &network.EndpointSettings{
				IPAMConfig: ep.IPAMConfig,
				Aliases:    ep.Aliases,
				NetworkID:  ep.NetworkID,
				IPAddress:  ep.IPAddress,
				MacAddress: ep.MacAddress,
			}
		}
		snap.NetworkConfig = &network.NetworkingConfig{EndpointsConfig: endpoints}
	}

	return snap, nil
}

// ReplaceContainer stops and removes the old container, then creates a new one
// with the given image while preserving the snapshot config.
func (e *DockerExecutor) ReplaceContainer(ctx context.Context, containerID, newImage string, snap *ContainerSnapshot) (string, error) {
	// Stop
	if snap.WasRunning {
		if err := e.cli.ContainerStop(ctx, containerID, container.StopOptions{}); err != nil {
			return "", fmt.Errorf("stop container: %w", err)
		}
	}

	// Remove
	if err := e.cli.ContainerRemove(ctx, containerID, container.RemoveOptions{}); err != nil {
		if snap.WasRunning {
			_ = e.cli.ContainerStart(ctx, containerID, container.StartOptions{})
		}
		return "", fmt.Errorf("remove container: %w", err)
	}

	// Create new with updated image
	config := *snap.Config
	config.Image = newImage

	resp, err := e.cli.ContainerCreate(ctx, &config, snap.HostConfig, snap.NetworkConfig, nil, snap.Name)
	if err != nil {
		return "", fmt.Errorf("create container: %w", err)
	}

	// Start
	if snap.WasRunning {
		if err := e.cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
			return resp.ID, fmt.Errorf("start container: %w", err)
		}
	}

	return resp.ID, nil
}

// RollbackContainer restores a container from a snapshot.
func (e *DockerExecutor) RollbackContainer(ctx context.Context, currentID string, snap *ContainerSnapshot) (string, error) {
	// Try to stop and remove the failed new container
	_ = e.cli.ContainerStop(ctx, currentID, container.StopOptions{})
	_ = e.cli.ContainerRemove(ctx, currentID, container.RemoveOptions{Force: true})

	// Recreate from snapshot
	resp, err := e.cli.ContainerCreate(ctx, snap.Config, snap.HostConfig, snap.NetworkConfig, nil, snap.Name)
	if err != nil {
		return "", fmt.Errorf("rollback create: %w", err)
	}

	if snap.WasRunning {
		if err := e.cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
			return resp.ID, fmt.Errorf("rollback start: %w", err)
		}
	}

	return resp.ID, nil
}

// WaitHealthy polls the container status for the given duration, returning
// nil as soon as the container is running. An error is returned if the
// container exits or the timeout is reached.
func (e *DockerExecutor) WaitHealthy(ctx context.Context, containerID string, timeout time.Duration) error {
	deadline := time.After(timeout)
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-deadline:
			return fmt.Errorf("health check timed out after %s", timeout)
		case <-ticker.C:
			inspect, err := e.cli.ContainerInspect(ctx, containerID)
			if err != nil {
				return fmt.Errorf("inspect during health check: %w", err)
			}
			if inspect.State == nil {
				continue
			}
			if inspect.State.Running {
				// If container has a healthcheck, verify it
				if inspect.State.Health != nil {
					if inspect.State.Health.Status == "healthy" {
						return nil
					}
					if inspect.State.Health.Status == "unhealthy" {
						return fmt.Errorf("container is unhealthy")
					}
					continue // still starting
				}
				return nil // no healthcheck, running is good enough
			}
			if !inspect.State.Running && inspect.State.ExitCode != 0 {
				return fmt.Errorf("container exited with code %d", inspect.State.ExitCode)
			}
		}
	}
}

// BuildRegistryAuth creates a base64 Docker auth config from username/password.
func BuildRegistryAuth(username, password string) string {
	auth := map[string]string{"username": username, "password": password}
	j, _ := json.Marshal(auth)
	return base64.URLEncoding.EncodeToString(j)
}

// LogProgress is a helper that reads Docker pull progress and reports back.
func LogProgress(reader io.ReadCloser, report func(msg string, progress float32)) {
	defer reader.Close()
	dec := json.NewDecoder(reader)
	for {
		var evt struct {
			Status   string `json:"status"`
			ID       string `json:"id"`
			Progress string `json:"progress"`
		}
		if err := dec.Decode(&evt); err != nil {
			return
		}
		msg := evt.Status
		if evt.ID != "" {
			msg = fmt.Sprintf("%s: %s", evt.ID, evt.Status)
		}
		report(msg, 0)
	}
}
