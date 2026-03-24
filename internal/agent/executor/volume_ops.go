package executor

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types/volume"
)

// CreateVolume creates a Docker volume with the given name, driver and labels.
func (e *DockerExecutor) CreateVolume(ctx context.Context, name, driver string, labels map[string]string) (string, error) {
	opts := volume.CreateOptions{
		Name:   name,
		Driver: driver,
		Labels: labels,
	}

	vol, err := e.cli.VolumeCreate(ctx, opts)
	if err != nil {
		return "", fmt.Errorf("volume create %q: %w", name, err)
	}

	return vol.Name, nil
}

// RemoveVolume removes a Docker volume by name.
func (e *DockerExecutor) RemoveVolume(ctx context.Context, name string, force bool) error {
	if err := e.cli.VolumeRemove(ctx, name, force); err != nil {
		return fmt.Errorf("volume remove %s: %w", name, err)
	}
	return nil
}

// InspectVolume inspects a Docker volume and returns its details.
func (e *DockerExecutor) InspectVolume(ctx context.Context, name string) (*VolumeInspectResult, error) {
	vol, err := e.cli.VolumeInspect(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("volume inspect %s: %w", name, err)
	}

	var sizeBytes int64
	if vol.UsageData != nil {
		sizeBytes = vol.UsageData.Size
	}

	return &VolumeInspectResult{
		Name:       vol.Name,
		Driver:     vol.Driver,
		Mountpoint: vol.Mountpoint,
		SizeBytes:  sizeBytes,
		Labels:     vol.Labels,
		CreatedAt:  vol.CreatedAt,
	}, nil
}
