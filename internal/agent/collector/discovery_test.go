package collector

import (
	"errors"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"

	agentpb "github.com/onflux-tech/dockeeper/proto/gen/dockeeper/agent/v1"
)

func TestChangedDetectsNewData(t *testing.T) {
	d := &DiscoveryCollector{
		hashes: make(map[string][32]byte),
	}

	req := &agentpb.ReportContainersRequest{ServerId: "srv1"}

	// First call should return true (new data).
	assert.True(t, d.changed("containers", req))

	// Same data should return false.
	assert.False(t, d.changed("containers", req))
}

func TestChangedDetectsUpdate(t *testing.T) {
	d := &DiscoveryCollector{
		hashes: make(map[string][32]byte),
	}

	req1 := &agentpb.ReportContainersRequest{ServerId: "srv1"}
	d.changed("containers", req1)

	// Different data should return true.
	req2 := &agentpb.ReportContainersRequest{ServerId: "srv2"}
	assert.True(t, d.changed("containers", req2))
}

func TestChangedDifferentKeys(t *testing.T) {
	d := &DiscoveryCollector{
		hashes: make(map[string][32]byte),
	}

	req := &agentpb.ReportContainersRequest{ServerId: "srv1"}

	d.changed("containers", req)

	// Different key should be independent.
	assert.True(t, d.changed("images", req))
}

func TestHandleRPCErrorAllowsRetry(t *testing.T) {
	d := &DiscoveryCollector{
		hashes: make(map[string][32]byte),
		logger: slog.Default(),
		gate:   &ConnectionGate{},
	}

	req := &agentpb.ReportContainersRequest{ServerId: "srv1"}
	assert.True(t, d.changed("containers", req))

	d.handleRPCError("containers", errors.New("transport closed"))

	assert.True(t, d.changed("containers", req))
}
