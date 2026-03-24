package services

import (
	"testing"

	"github.com/stretchr/testify/assert"

	agentpb "github.com/onflux-tech/dockeeper/proto/gen/dockeeper/agent/v1"
)

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
