package executor

import (
	"context"
	"io"
)

// Result holds the outcome of a container command execution.
type Result struct {
	Success bool
	Output  string
	Error   string
}

// ImageRemoveResult holds the outcome of removing a Docker image.
type ImageRemoveResult struct {
	Success  bool
	Error    string
	Deleted  []string
	Untagged []string
}

// ImagePruneResult holds the outcome of pruning unused Docker images.
type ImagePruneResult struct {
	Success        bool
	Error          string
	Deleted        []string
	SpaceReclaimed uint64
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
	UpdateImage(ctx context.Context, containerID string) Result
	PullImageStream(ctx context.Context, imageRef string) (io.ReadCloser, error)
	RemoveImage(ctx context.Context, imageID string, force bool, pruneChildren bool) ImageRemoveResult
	PruneImages(ctx context.Context, all bool) ImagePruneResult
	CreateNetwork(ctx context.Context, opts NetworkCreateOpts) (string, error)
	RemoveNetwork(ctx context.Context, networkID string) error
	ConnectContainer(ctx context.Context, networkID, containerID string) error
	DisconnectContainer(ctx context.Context, networkID, containerID string, force bool) error
	CreateVolume(ctx context.Context, name, driver string, labels map[string]string) (string, error)
	RemoveVolume(ctx context.Context, name string, force bool) error
	InspectVolume(ctx context.Context, name string) (*VolumeInspectResult, error)
	BackupVolume(ctx context.Context, mountpoint string, s3 S3Config, progress func(BackupProgressInfo)) error
	RestoreVolume(ctx context.Context, mountpoint string, s3 S3Config, progress func(BackupProgressInfo)) error
}

// NetworkCreateOpts holds options for creating a Docker network.
type NetworkCreateOpts struct {
	Name     string
	Driver   string
	Subnet   string
	Gateway  string
	Internal bool
	Labels   map[string]string
}

// VolumeInspectResult holds the outcome of inspecting a Docker volume.
type VolumeInspectResult struct {
	Name       string
	Driver     string
	Mountpoint string
	SizeBytes  int64
	Labels     map[string]string
	CreatedAt  string
}

// S3Config holds S3-compatible storage configuration for backup operations.
type S3Config struct {
	Endpoint  string
	Bucket    string
	Key       string
	AccessKey string
	SecretKey string
	Region    string
	UseSSL    bool
}

// BackupProgressInfo holds progress information for backup/restore operations.
type BackupProgressInfo struct {
	Status         string
	Message        string
	BytesProcessed int64
	TotalBytes     int64
}
