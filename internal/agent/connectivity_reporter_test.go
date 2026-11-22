package agent

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/onflux-tech/dockeeper/internal/agent/collector"
)

func TestConnectivityReporterMapsReport(t *testing.T) {
	fake := &fakeHostClient{}
	reporter := NewConnectivityReporter(fake, "srv1", slog.Default())

	err := reporter.Report(context.Background(), &collector.ConnectivityReport{
		InternetOK:    true,
		InternetMS:    24,
		DNSOK:         true,
		DNSMS:         9,
		GatewayOK:     false,
		GatewayMS:     0,
		GatewayMethod: "none",
		Error:         "probe failed: gateway",
		CollectedAt:   time.Unix(1700000000, 0).UTC(),
	})
	require.NoError(t, err)
	require.Equal(t, "srv1", fake.connReq.GetServerId())
	require.True(t, fake.connReq.GetInternetOk())
	require.Equal(t, uint32(24), fake.connReq.GetInternetMs())
	require.True(t, fake.connReq.GetDnsOk())
	require.False(t, fake.connReq.GetGatewayOk())
	require.Equal(t, "none", fake.connReq.GetGatewayMethod())
	require.Equal(t, "probe failed: gateway", fake.connReq.GetError())
	require.Equal(t, int64(1700000000), fake.connReq.GetCollectedAt().GetSeconds())
}
