package executor

import (
	"context"
	"fmt"
	"io"

	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
)

// PullImageStream starts pulling a Docker image and returns the raw progress
// reader. The caller is responsible for closing the reader and decoding the
// JSON progress lines (Docker ImagePull output format).
func (e *DockerExecutor) PullImageStream(ctx context.Context, imageRef string) (io.ReadCloser, error) {
	reader, err := e.cli.ImagePull(ctx, imageRef, image.PullOptions{})
	if err != nil {
		return nil, fmt.Errorf("pull image %s: %w", imageRef, err)
	}
	return reader, nil
}

// RemoveImage removes a Docker image by ID or reference.
func (e *DockerExecutor) RemoveImage(ctx context.Context, imageID string, force bool, pruneChildren bool) ImageRemoveResult {
	ctx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	items, err := e.cli.ImageRemove(ctx, imageID, image.RemoveOptions{
		Force:         force,
		PruneChildren: pruneChildren,
	})
	if err != nil {
		return ImageRemoveResult{Error: fmt.Sprintf("remove image %s: %s", imageID, err.Error())}
	}

	var deleted, untagged []string
	for _, item := range items {
		if item.Deleted != "" {
			deleted = append(deleted, item.Deleted)
		}
		if item.Untagged != "" {
			untagged = append(untagged, item.Untagged)
		}
	}

	return ImageRemoveResult{
		Success:  true,
		Deleted:  deleted,
		Untagged: untagged,
	}
}

// PruneImages removes unused Docker images.
// If all is true, removes all unused images (not just dangling).
func (e *DockerExecutor) PruneImages(ctx context.Context, all bool) ImagePruneResult {
	ctx, cancel := context.WithTimeout(ctx, 2*defaultTimeout)
	defer cancel()

	pruneFilter := filters.NewArgs()
	if !all {
		pruneFilter.Add("dangling", "true")
	}

	report, err := e.cli.ImagesPrune(ctx, pruneFilter)
	if err != nil {
		return ImagePruneResult{Error: fmt.Sprintf("prune images: %s", err.Error())}
	}

	var deleted []string
	for _, item := range report.ImagesDeleted {
		if item.Deleted != "" {
			deleted = append(deleted, item.Deleted)
		}
		if item.Untagged != "" {
			deleted = append(deleted, item.Untagged)
		}
	}

	return ImagePruneResult{
		Success:        true,
		Deleted:        deleted,
		SpaceReclaimed: report.SpaceReclaimed,
	}
}
