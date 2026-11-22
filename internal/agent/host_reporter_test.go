package agent

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	"github.com/onflux-tech/dockeeper/internal/agent/collector"
	agentpb "github.com/onflux-tech/dockeeper/proto/gen/dockeeper/agent/v1"
)

type fakeHostClient struct {
	req *agentpb.ReportHostInventoryRequest
	err error
}

func (f *fakeHostClient) ReportHostInventory(ctx context.Context, req *agentpb.ReportHostInventoryRequest, opts ...grpc.CallOption) (*agentpb.ReportHostInventoryResponse, error) {
	f.req = req
	if f.err != nil {
		return nil, f.err
	}
	return &agentpb.ReportHostInventoryResponse{ReceivedApps: int32(len(req.GetApps()))}, nil
}

func (f *fakeHostClient) ReportConnectivity(ctx context.Context, req *agentpb.ReportConnectivityRequest, opts ...grpc.CallOption) (*agentpb.ReportConnectivityResponse, error) {
	return &agentpb.ReportConnectivityResponse{}, nil
}

func TestHostReporterMapsInventory(t *testing.T) {
	fake := &fakeHostClient{}
	reporter := NewHostReporter(fake, "srv1", slog.Default())

	err := reporter.Report(context.Background(), &collector.HostInventory{
		Apps: []collector.HostApp{
			{Source: "systemd", Name: "nginx.service", Status: "running", Ports: []uint32{80}},
		},
		Processes: []collector.HostProcess{
			{PID: 42, Name: "nginx", App: "nginx.service", CPUPercent: 1.5},
		},
		Ports: []collector.HostPort{
			{Port: 80, Protocol: "tcp", Address: "0.0.0.0", PID: 42, App: "nginx.service"},
		},
		UptimeSeconds: 100,
		CollectedAt:   time.Unix(1700000000, 0).UTC(),
	})
	require.NoError(t, err)
	require.Equal(t, "srv1", fake.req.GetServerId())
	require.Equal(t, uint64(100), fake.req.GetUptimeSeconds())
	require.Equal(t, "nginx.service", fake.req.GetApps()[0].GetName())
	require.Equal(t, []uint32{80}, fake.req.GetApps()[0].GetPorts())
	require.Equal(t, uint32(42), fake.req.GetProcesses()[0].GetPid())
	require.Equal(t, "0.0.0.0", fake.req.GetPorts()[0].GetAddress())
	require.Equal(t, int64(1700000000), fake.req.GetCollectedAt().GetSeconds())
}
