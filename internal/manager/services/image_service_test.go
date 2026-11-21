package services

import (
	"testing"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	agentpb "github.com/onflux-tech/dockeeper/proto/gen/dockeeper/agent/v1"
)

func newImageServiceTestApp(t *testing.T) (*ImageService, core.App) {
	t.Helper()

	app, err := tests.NewTestApp()
	require.NoError(t, err)
	t.Cleanup(app.Cleanup)

	images := core.NewBaseCollection("images")
	images.Fields.Add(
		&core.TextField{Name: "server"},
		&core.TextField{Name: "image_id"},
		&core.JSONField{Name: "tags"},
	)
	require.NoError(t, app.Save(images))

	return NewImageService(app, nil, nil), app
}

func seedImage(t *testing.T, app core.App, serverID, imageID string) {
	t.Helper()

	col, err := app.FindCachedCollectionByNameOrId("images")
	require.NoError(t, err)

	rec := core.NewRecord(col)
	rec.Set("server", serverID)
	rec.Set("image_id", imageID)
	rec.Set("tags", []string{"repo:tag"})
	require.NoError(t, app.Save(rec))
}

func TestDeletePrunedImagesRemovesOnlyMatchingServerAndImage(t *testing.T) {
	svc, app := newImageServiceTestApp(t)
	seedImage(t, app, "srv1", "87c02fd58f47")
	seedImage(t, app, "srv1", "aaaaaaaaaaaa")
	seedImage(t, app, "srv2", "87c02fd58f47")

	removed := svc.deletePrunedImages("srv1", []string{
		"sha256:87c02fd58f47ffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
		"nginx:latest",
	})
	assert.Equal(t, 1, removed)

	records, err := app.FindRecordsByFilter("images", "server = {:sid}", "", 0, 0, map[string]any{"sid": "srv1"})
	require.NoError(t, err)
	require.Len(t, records, 1)
	assert.Equal(t, "aaaaaaaaaaaa", records[0].GetString("image_id"))

	others, err := app.FindRecordsByFilter("images", "server = {:sid}", "", 0, 0, map[string]any{"sid": "srv2"})
	require.NoError(t, err)
	assert.Len(t, others, 1)
}

func TestCleanupPrunedImagesSkipsFailedPrune(t *testing.T) {
	svc, app := newImageServiceTestApp(t)
	seedImage(t, app, "srv1", "87c02fd58f47")

	removed := svc.cleanupPrunedImages("srv1", &agentpb.ImagePruneResponse{
		Success: false,
		Deleted: []string{"sha256:87c02fd58f47ffffffffffffffffffffffffffffffffffffffffffffffffffffffff"},
	})
	assert.Equal(t, 0, removed)

	records, err := app.FindRecordsByFilter("images", "server = {:sid}", "", 0, 0, map[string]any{"sid": "srv1"})
	require.NoError(t, err)
	assert.Len(t, records, 1)
}

func TestCleanupPrunedImagesRemovesRecordsOnSuccess(t *testing.T) {
	svc, app := newImageServiceTestApp(t)
	seedImage(t, app, "srv1", "87c02fd58f47")

	removed := svc.cleanupPrunedImages("srv1", &agentpb.ImagePruneResponse{
		Success: true,
		Deleted: []string{"sha256:87c02fd58f47ffffffffffffffffffffffffffffffffffffffffffffffffffffffff"},
	})
	assert.Equal(t, 1, removed)

	records, err := app.FindRecordsByFilter("images", "server = {:sid}", "", 0, 0, map[string]any{"sid": "srv1"})
	require.NoError(t, err)
	assert.Len(t, records, 0)
}

func TestDeletePrunedImagesIgnoresEmptyList(t *testing.T) {
	svc, app := newImageServiceTestApp(t)
	seedImage(t, app, "srv1", "87c02fd58f47")

	assert.Equal(t, 0, svc.deletePrunedImages("srv1", nil))

	records, err := app.FindRecordsByFilter("images", "server = {:sid}", "", 0, 0, map[string]any{"sid": "srv1"})
	require.NoError(t, err)
	assert.Len(t, records, 1)
}
