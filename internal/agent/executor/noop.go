package executor

import (
	"context"
	"errors"
	"io"
	"time"
)

var ErrNoContainerRuntime = errors.New("no container runtime available")

type NoopExecutor struct{}

func NewNoopExecutor() *NoopExecutor {
	return &NoopExecutor{}
}

func noopResult() Result {
	return Result{Error: ErrNoContainerRuntime.Error()}
}

func (e *NoopExecutor) Start(ctx context.Context, containerID string) Result {
	return noopResult()
}

func (e *NoopExecutor) Stop(ctx context.Context, containerID string, timeout *int) Result {
	return noopResult()
}

func (e *NoopExecutor) Restart(ctx context.Context, containerID string, timeout *int) Result {
	return noopResult()
}

func (e *NoopExecutor) Remove(ctx context.Context, containerID string, force bool) Result {
	return noopResult()
}

func (e *NoopExecutor) Rename(ctx context.Context, containerID string, newName string) Result {
	return noopResult()
}

func (e *NoopExecutor) Pause(ctx context.Context, containerID string) Result {
	return noopResult()
}

func (e *NoopExecutor) Unpause(ctx context.Context, containerID string) Result {
	return noopResult()
}

func (e *NoopExecutor) Create(ctx context.Context, spec *ContainerSpec) Result {
	return noopResult()
}

func (e *NoopExecutor) PullImage(ctx context.Context, imageRef string) Result {
	return noopResult()
}

func (e *NoopExecutor) UpdateResources(ctx context.Context, containerID string, update *ResourceUpdate) Result {
	return noopResult()
}

func (e *NoopExecutor) RecreateWithEnv(ctx context.Context, containerID string, newEnv []string) Result {
	return noopResult()
}

func (e *NoopExecutor) UpdateImage(ctx context.Context, containerID string) Result {
	return noopResult()
}

func (e *NoopExecutor) PullImageStream(ctx context.Context, imageRef string) (io.ReadCloser, error) {
	return nil, ErrNoContainerRuntime
}

func (e *NoopExecutor) RemoveImage(ctx context.Context, imageID string, force bool, pruneChildren bool) ImageRemoveResult {
	return ImageRemoveResult{Error: ErrNoContainerRuntime.Error()}
}

func (e *NoopExecutor) PruneImages(ctx context.Context, all bool) ImagePruneResult {
	return ImagePruneResult{Error: ErrNoContainerRuntime.Error()}
}

func (e *NoopExecutor) CreateNetwork(ctx context.Context, opts NetworkCreateOpts) (string, error) {
	return "", ErrNoContainerRuntime
}

func (e *NoopExecutor) RemoveNetwork(ctx context.Context, networkID string) error {
	return ErrNoContainerRuntime
}

func (e *NoopExecutor) ConnectContainer(ctx context.Context, networkID, containerID string) error {
	return ErrNoContainerRuntime
}

func (e *NoopExecutor) DisconnectContainer(ctx context.Context, networkID, containerID string, force bool) error {
	return ErrNoContainerRuntime
}

func (e *NoopExecutor) CreateVolume(ctx context.Context, name, driver string, labels map[string]string) (string, error) {
	return "", ErrNoContainerRuntime
}

func (e *NoopExecutor) RemoveVolume(ctx context.Context, name string, force bool) error {
	return ErrNoContainerRuntime
}

func (e *NoopExecutor) InspectVolume(ctx context.Context, name string) (*VolumeInspectResult, error) {
	return nil, ErrNoContainerRuntime
}

func (e *NoopExecutor) BackupVolume(ctx context.Context, mountpoint string, s3 S3Config, progress func(BackupProgressInfo)) error {
	return ErrNoContainerRuntime
}

func (e *NoopExecutor) RestoreVolume(ctx context.Context, mountpoint string, s3 S3Config, progress func(BackupProgressInfo)) error {
	return ErrNoContainerRuntime
}

func (e *NoopExecutor) Uninstall(ctx context.Context) Result {
	return noopResult()
}

func (e *NoopExecutor) CheckImageDigest(ctx context.Context, imageRef string) (string, error) {
	return "", ErrNoContainerRuntime
}

func (e *NoopExecutor) PullImageWithAuth(ctx context.Context, imageRef, registryAuth string) (io.ReadCloser, error) {
	return nil, ErrNoContainerRuntime
}

func (e *NoopExecutor) SnapshotContainer(ctx context.Context, containerID string) (*ContainerSnapshot, error) {
	return nil, ErrNoContainerRuntime
}

func (e *NoopExecutor) ReplaceContainer(ctx context.Context, containerID, newImage string, snap *ContainerSnapshot) (string, error) {
	return "", ErrNoContainerRuntime
}

func (e *NoopExecutor) RollbackContainer(ctx context.Context, currentID string, snap *ContainerSnapshot) (string, error) {
	return "", ErrNoContainerRuntime
}

func (e *NoopExecutor) WaitHealthy(ctx context.Context, containerID string, timeout time.Duration) error {
	return ErrNoContainerRuntime
}
