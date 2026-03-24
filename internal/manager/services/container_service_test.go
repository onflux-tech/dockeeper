package services

import (
	"context"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	agentpb "github.com/onflux-tech/dockeeper/proto/gen/dockeeper/agent/v1"
)

type testDispatcher struct {
	called chan *agentpb.ExecuteCommandRequest
}

func (d *testDispatcher) Dispatch(_ context.Context, _ string, req *agentpb.ExecuteCommandRequest) (*agentpb.ExecuteCommandResponse, error) {
	d.called <- req
	return &agentpb.ExecuteCommandResponse{Success: true}, nil
}

func TestIsDockeeperRuntimeContainer(t *testing.T) {
	rec := core.NewRecord(&core.Collection{})
	rec.Set("name", "dockeeper-agent")
	assert.True(t, isDockeeperRuntimeContainer(rec))

	rec.Set("name", "my-app")
	rec.Set("image", "ghcr.io/onflux-tech/dockeeper-manager:latest")
	assert.True(t, isDockeeperRuntimeContainer(rec))

	rec.Set("image", "nginx:latest")
	assert.False(t, isDockeeperRuntimeContainer(rec))
}

func TestScheduleInternalRestartDispatchesInBackground(t *testing.T) {
	dispatcher := &testDispatcher{called: make(chan *agentpb.ExecuteCommandRequest, 1)}
	svc := &ContainerService{dispatcher: dispatcher}

	container := core.NewRecord(&core.Collection{})
	container.Id = "rec1"
	container.Set("name", "dockeeper-agent")
	container.Set("container_id", "abc123")
	container.Set("server", "srv1")

	req := &agentpb.ExecuteCommandRequest{
		ServerId:    "srv1",
		ContainerId: "abc123",
		Type:        agentpb.CommandType_COMMAND_TYPE_RESTART,
		Params:      map[string]string{"timeout": "5"},
	}

	svc.scheduleInternalRestart("127.0.0.1", req, container, "u1", "127.0.0.1")

	select {
	case got := <-dispatcher.called:
		require.NotNil(t, got)
		assert.Equal(t, req.GetContainerId(), got.GetContainerId())
		assert.Equal(t, req.GetType(), got.GetType())
		assert.Equal(t, req.GetParams(), got.GetParams())
	case <-time.After(3 * time.Second):
		t.Fatal("expected scheduled restart to dispatch")
	}
}

func TestTimeoutForCommand(t *testing.T) {
	tests := []struct {
		name    string
		cmd     agentpb.CommandType
		wantSec int
	}{
		{"start uses default", agentpb.CommandType_COMMAND_TYPE_START, 30},
		{"stop uses default", agentpb.CommandType_COMMAND_TYPE_STOP, 30},
		{"restart uses default", agentpb.CommandType_COMMAND_TYPE_RESTART, 30},
		{"remove uses default", agentpb.CommandType_COMMAND_TYPE_REMOVE, 30},
		{"update uses extended", agentpb.CommandType_COMMAND_TYPE_UPDATE_IMAGE, 150},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := timeoutForCommand(tt.cmd)
			assert.Equal(t, tt.wantSec, int(got.Seconds()))
		})
	}
}

func TestSentinelErrors(t *testing.T) {
	assert.Equal(t, "container not found", ErrContainerNotFound.Error())
	assert.Equal(t, "server not found", ErrServerNotFound.Error())
	assert.Equal(t, "server is offline", ErrServerOffline.Error())
}
