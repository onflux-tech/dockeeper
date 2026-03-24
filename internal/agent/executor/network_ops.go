package executor

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types/network"
)

// CreateNetwork creates a Docker network with the given options.
func (e *DockerExecutor) CreateNetwork(ctx context.Context, opts NetworkCreateOpts) (string, error) {
	ipamConfig := []network.IPAMConfig{}
	if opts.Subnet != "" {
		cfg := network.IPAMConfig{Subnet: opts.Subnet}
		if opts.Gateway != "" {
			cfg.Gateway = opts.Gateway
		}
		ipamConfig = append(ipamConfig, cfg)
	}

	createOpts := network.CreateOptions{
		Driver:   opts.Driver,
		Internal: opts.Internal,
		Labels:   opts.Labels,
	}
	if len(ipamConfig) > 0 {
		createOpts.IPAM = &network.IPAM{
			Config: ipamConfig,
		}
	}

	resp, err := e.cli.NetworkCreate(ctx, opts.Name, createOpts)
	if err != nil {
		return "", fmt.Errorf("network create %q: %w", opts.Name, err)
	}

	return resp.ID, nil
}

// RemoveNetwork removes a Docker network by ID.
func (e *DockerExecutor) RemoveNetwork(ctx context.Context, networkID string) error {
	if err := e.cli.NetworkRemove(ctx, networkID); err != nil {
		return fmt.Errorf("network remove %s: %w", networkID, err)
	}
	return nil
}

// ConnectContainer connects a container to a network.
func (e *DockerExecutor) ConnectContainer(ctx context.Context, networkID, containerID string) error {
	if err := e.cli.NetworkConnect(ctx, networkID, containerID, nil); err != nil {
		return fmt.Errorf("network connect %s to %s: %w", containerID, networkID, err)
	}
	return nil
}

// DisconnectContainer disconnects a container from a network.
func (e *DockerExecutor) DisconnectContainer(ctx context.Context, networkID, containerID string, force bool) error {
	if err := e.cli.NetworkDisconnect(ctx, networkID, containerID, force); err != nil {
		return fmt.Errorf("network disconnect %s from %s: %w", containerID, networkID, err)
	}
	return nil
}
