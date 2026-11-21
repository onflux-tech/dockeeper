package store

import (
	"testing"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	agentpb "github.com/onflux-tech/dockeeper/proto/gen/dockeeper/agent/v1"
)

func newTestResourceStore(t *testing.T) *ResourceStore {
	t.Helper()

	app, err := tests.NewTestApp()
	require.NoError(t, err)
	t.Cleanup(app.Cleanup)

	images := core.NewBaseCollection("images")
	images.Fields.Add(
		&core.TextField{Name: "server"},
		&core.TextField{Name: "image_id"},
		&core.JSONField{Name: "tags"},
		&core.NumberField{Name: "size"},
		&core.BoolField{Name: "in_use"},
		&core.DateField{Name: "image_created_at"},
	)
	require.NoError(t, app.Save(images))

	networks := core.NewBaseCollection("networks")
	networks.Fields.Add(
		&core.TextField{Name: "server"},
		&core.TextField{Name: "network_id"},
		&core.TextField{Name: "name"},
		&core.TextField{Name: "driver"},
		&core.TextField{Name: "scope"},
		&core.TextField{Name: "subnet"},
		&core.TextField{Name: "gateway"},
		&core.BoolField{Name: "internal"},
		&core.JSONField{Name: "labels"},
	)
	require.NoError(t, app.Save(networks))

	volumes := core.NewBaseCollection("volumes")
	volumes.Fields.Add(
		&core.TextField{Name: "server"},
		&core.TextField{Name: "name"},
		&core.TextField{Name: "driver"},
		&core.TextField{Name: "mountpoint"},
		&core.NumberField{Name: "size"},
		&core.JSONField{Name: "labels"},
		&core.BoolField{Name: "in_use"},
	)
	require.NoError(t, app.Save(volumes))

	return NewResourceStore(app, nil)
}

func TestUpsertImagesDeletesStale(t *testing.T) {
	s := newTestResourceStore(t)

	_, err := s.UpsertImages("srv1", []*agentpb.ImageInfo{{ImageId: "a"}, {ImageId: "b"}})
	require.NoError(t, err)
	_, err = s.UpsertImages("srv2", []*agentpb.ImageInfo{{ImageId: "c"}})
	require.NoError(t, err)

	count, err := s.UpsertImages("srv1", []*agentpb.ImageInfo{{ImageId: "a"}})
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	records, err := s.app.FindRecordsByFilter("images", "server = {:sid}", "", 0, 0, map[string]any{"sid": "srv1"})
	require.NoError(t, err)
	require.Len(t, records, 1)
	assert.Equal(t, "a", records[0].GetString("image_id"))

	others, err := s.app.FindRecordsByFilter("images", "server = {:sid}", "", 0, 0, map[string]any{"sid": "srv2"})
	require.NoError(t, err)
	assert.Len(t, others, 1)
}

func TestUpsertNetworksDeletesStale(t *testing.T) {
	s := newTestResourceStore(t)

	_, err := s.UpsertNetworks("srv1", []*agentpb.NetworkInfo{{NetworkId: "a", Name: "net-a"}, {NetworkId: "b", Name: "net-b"}})
	require.NoError(t, err)

	count, err := s.UpsertNetworks("srv1", []*agentpb.NetworkInfo{{NetworkId: "a", Name: "net-a"}})
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	records, err := s.app.FindRecordsByFilter("networks", "server = {:sid}", "", 0, 0, map[string]any{"sid": "srv1"})
	require.NoError(t, err)
	require.Len(t, records, 1)
	assert.Equal(t, "a", records[0].GetString("network_id"))
}

func TestUpsertVolumesDeletesStale(t *testing.T) {
	s := newTestResourceStore(t)

	_, err := s.UpsertVolumes("srv1", []*agentpb.VolumeInfo{{Name: "vol-a"}, {Name: "vol-b"}})
	require.NoError(t, err)

	count, err := s.UpsertVolumes("srv1", []*agentpb.VolumeInfo{{Name: "vol-a"}})
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	records, err := s.app.FindRecordsByFilter("volumes", "server = {:sid}", "", 0, 0, map[string]any{"sid": "srv1"})
	require.NoError(t, err)
	require.Len(t, records, 1)
	assert.Equal(t, "vol-a", records[0].GetString("name"))
}
