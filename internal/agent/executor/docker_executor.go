package executor

import (
	"context"
	"fmt"
	"time"

	"github.com/containerd/errdefs"
	"github.com/docker/docker/api/types/container"
	dockerclient "github.com/docker/docker/client"
)

const defaultTimeout = 30 * time.Second

// DockerExecutor implements Executor using the Docker Engine SDK.
type DockerExecutor struct {
	cli *dockerclient.Client
}

// NewDockerExecutor creates a DockerExecutor from environment settings.
func NewDockerExecutor() (*DockerExecutor, error) {
	cli, err := dockerclient.NewClientWithOpts(
		dockerclient.FromEnv,
		dockerclient.WithAPIVersionNegotiation(),
	)
	if err != nil {
		return nil, fmt.Errorf("create docker client: %w", err)
	}
	return &DockerExecutor{cli: cli}, nil
}

// NewDockerExecutorFromClient creates a DockerExecutor from an existing client.
func NewDockerExecutorFromClient(cli *dockerclient.Client) *DockerExecutor {
	return &DockerExecutor{cli: cli}
}

func (e *DockerExecutor) Start(ctx context.Context, containerID string) Result {
	ctx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	if err := e.cli.ContainerStart(ctx, containerID, container.StartOptions{}); err != nil {
		return errResult(err, containerID)
	}
	return Result{Success: true, Output: fmt.Sprintf("container %s started", containerID)}
}

func (e *DockerExecutor) Stop(ctx context.Context, containerID string, timeout *int) Result {
	ctx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	opts := container.StopOptions{}
	if timeout != nil {
		opts.Timeout = timeout
	}

	if err := e.cli.ContainerStop(ctx, containerID, opts); err != nil {
		return errResult(err, containerID)
	}
	return Result{Success: true, Output: fmt.Sprintf("container %s stopped", containerID)}
}

func (e *DockerExecutor) Restart(ctx context.Context, containerID string, timeout *int) Result {
	ctx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	opts := container.StopOptions{}
	if timeout != nil {
		opts.Timeout = timeout
	}

	if err := e.cli.ContainerRestart(ctx, containerID, opts); err != nil {
		return errResult(err, containerID)
	}
	return Result{Success: true, Output: fmt.Sprintf("container %s restarted", containerID)}
}

func (e *DockerExecutor) Remove(ctx context.Context, containerID string, force bool) Result {
	ctx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	opts := container.RemoveOptions{Force: force}

	if err := e.cli.ContainerRemove(ctx, containerID, opts); err != nil {
		return errResult(err, containerID)
	}
	return Result{Success: true, Output: fmt.Sprintf("container %s removed", containerID)}
}

func (e *DockerExecutor) Rename(ctx context.Context, containerID string, newName string) Result {
	ctx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	if err := e.cli.ContainerRename(ctx, containerID, newName); err != nil {
		return errResult(err, containerID)
	}
	return Result{Success: true, Output: fmt.Sprintf("container %s renamed to %s", containerID, newName)}
}

func (e *DockerExecutor) Pause(ctx context.Context, containerID string) Result {
	ctx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	if err := e.cli.ContainerPause(ctx, containerID); err != nil {
		return errResult(err, containerID)
	}
	return Result{Success: true, Output: fmt.Sprintf("container %s paused", containerID)}
}

func (e *DockerExecutor) Unpause(ctx context.Context, containerID string) Result {
	ctx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	if err := e.cli.ContainerUnpause(ctx, containerID); err != nil {
		return errResult(err, containerID)
	}
	return Result{Success: true, Output: fmt.Sprintf("container %s unpaused", containerID)}
}

// errResult maps a Docker error to a Result with appropriate error message.
func errResult(err error, containerID string) Result {
	if errdefs.IsNotFound(err) {
		return Result{Error: fmt.Sprintf("container not found: %s", containerID)}
	}
	if errdefs.IsConflict(err) {
		return Result{Error: fmt.Sprintf("conflict for container %s: %s", containerID, err.Error())}
	}
	return Result{Error: fmt.Sprintf("operation failed on %s: %s", containerID, err.Error())}
}
