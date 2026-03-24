package executor

import "context"

// Result holds the outcome of a container command execution.
type Result struct {
	Success bool
	Output  string
	Error   string
}

// Executor defines the contract for a container command executor.
type Executor interface {
	Start(ctx context.Context, containerID string) Result
	Stop(ctx context.Context, containerID string, timeout *int) Result
	Restart(ctx context.Context, containerID string, timeout *int) Result
	Remove(ctx context.Context, containerID string, force bool) Result
	Rename(ctx context.Context, containerID string, newName string) Result
	Pause(ctx context.Context, containerID string) Result
	Unpause(ctx context.Context, containerID string) Result
	Create(ctx context.Context, spec *ContainerSpec) Result
	PullImage(ctx context.Context, imageRef string) Result
	UpdateResources(ctx context.Context, containerID string, update *ResourceUpdate) Result
	RecreateWithEnv(ctx context.Context, containerID string, newEnv []string) Result
}
