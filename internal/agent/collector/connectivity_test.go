package collector

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestConnectivityCollectorReportsProbeResults(t *testing.T) {
	c := NewConnectivityCollector(slog.Default(),
		WithInternetProbe(func(context.Context) (bool, uint32) { return true, 24 }),
		WithDNSProbe(func(context.Context) (bool, uint32) { return true, 12 }),
		WithGatewayProbe(func(context.Context) (bool, uint32, string) { return true, 1, "icmp" }),
	)

	report := c.Collect(context.Background())

	require.True(t, report.InternetOK)
	require.Equal(t, uint32(24), report.InternetMS)
	require.True(t, report.DNSOK)
	require.Equal(t, uint32(12), report.DNSMS)
	require.True(t, report.GatewayOK)
	require.Equal(t, "icmp", report.GatewayMethod)
	require.Empty(t, report.Error)
	require.False(t, report.CollectedAt.IsZero())
}

func TestConnectivityCollectorCarriesErrors(t *testing.T) {
	c := NewConnectivityCollector(slog.Default(),
		WithInternetProbe(func(context.Context) (bool, uint32) { return false, 0 }),
		WithDNSProbe(func(context.Context) (bool, uint32) { return false, 0 }),
		WithGatewayProbe(func(context.Context) (bool, uint32, string) { return false, 0, "none" }),
	)

	report := c.Collect(context.Background())

	require.False(t, report.InternetOK)
	require.False(t, report.DNSOK)
	require.False(t, report.GatewayOK)
	require.Equal(t, "none", report.GatewayMethod)
	require.Contains(t, report.Error, "internet")
	require.Contains(t, report.Error, "dns")
	require.Contains(t, report.Error, "gateway")
}

func TestConnectivityCollectorRunReportsEveryCycle(t *testing.T) {
	collector := NewConnectivityCollector(slog.Default(),
		WithInternetProbe(func(context.Context) (bool, uint32) { return true, 5 }),
		WithDNSProbe(func(context.Context) (bool, uint32) { return true, 4 }),
		WithGatewayProbe(func(context.Context) (bool, uint32, string) { return true, 1, "tcp" }),
	)

	ctx, cancel := context.WithCancel(context.Background())
	reports := make(chan *ConnectivityReport, 2)
	go collector.Run(ctx, 10*time.Millisecond, func(report *ConnectivityReport) error {
		reports <- report
		return nil
	})

	first := <-reports
	second := <-reports
	cancel()

	require.NotNil(t, first)
	require.NotNil(t, second)
	require.Equal(t, "tcp", second.GatewayMethod)
}
