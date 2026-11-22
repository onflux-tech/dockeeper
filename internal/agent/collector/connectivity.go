package collector

import (
	"context"
	"log/slog"
	"net"
	"strings"
	"time"
)

type ConnectivityReport struct {
	InternetOK    bool
	InternetMS    uint32
	DNSOK         bool
	DNSMS         uint32
	GatewayOK     bool
	GatewayMS     uint32
	GatewayMethod string
	Error         string
	CollectedAt   time.Time
}

type ConnectivityProbe func(ctx context.Context) (bool, uint32)

type ConnectivityGatewayProbe func(ctx context.Context) (bool, uint32, string)

type ConnectivityCollector struct {
	logger   *slog.Logger
	internet ConnectivityProbe
	dns      ConnectivityProbe
	gateway  ConnectivityGatewayProbe
}

type ConnectivityOption func(*ConnectivityCollector)

func WithInternetProbe(probe ConnectivityProbe) ConnectivityOption {
	return func(c *ConnectivityCollector) {
		c.internet = probe
	}
}

func WithDNSProbe(probe ConnectivityProbe) ConnectivityOption {
	return func(c *ConnectivityCollector) {
		c.dns = probe
	}
}

func WithGatewayProbe(probe ConnectivityGatewayProbe) ConnectivityOption {
	return func(c *ConnectivityCollector) {
		c.gateway = probe
	}
}

func NewConnectivityCollector(logger *slog.Logger, opts ...ConnectivityOption) *ConnectivityCollector {
	c := &ConnectivityCollector{
		logger:   logger,
		internet: defaultInternetProbe,
		dns:      defaultDNSProbe,
		gateway:  defaultGatewayProbe,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

func (c *ConnectivityCollector) Collect(ctx context.Context) *ConnectivityReport {
	internetOK, internetMS := c.internet(ctx)
	dnsOK, dnsMS := c.dns(ctx)
	gatewayOK, gatewayMS, gatewayMethod := c.gateway(ctx)

	report := &ConnectivityReport{
		InternetOK:    internetOK,
		InternetMS:    internetMS,
		DNSOK:         dnsOK,
		DNSMS:         dnsMS,
		GatewayOK:     gatewayOK,
		GatewayMS:     gatewayMS,
		GatewayMethod: gatewayMethod,
		CollectedAt:   time.Now().UTC(),
	}

	failures := make([]string, 0, 3)
	if !internetOK {
		failures = append(failures, "internet")
	}
	if !dnsOK {
		failures = append(failures, "dns")
	}
	if !gatewayOK {
		failures = append(failures, "gateway")
	}
	if len(failures) > 0 {
		report.Error = "probe failed: " + strings.Join(failures, ", ")
	}

	return report
}

func (c *ConnectivityCollector) Run(ctx context.Context, interval time.Duration, report func(*ConnectivityReport) error) {
	var previous *ConnectivityReport
	for {
		current := c.Collect(ctx)
		if err := report(current); err != nil {
			c.logger.Warn("connectivity report failed", "error", err)
		}
		if previous == nil {
			c.logger.Info("connectivity checked",
				"internet", current.InternetOK, "dns", current.DNSOK,
				"gateway", current.GatewayOK, "gateway_method", current.GatewayMethod)
		} else if connectivityChanged(previous, current) {
			c.logger.Info("connectivity state changed",
				"internet", current.InternetOK, "dns", current.DNSOK,
				"gateway", current.GatewayOK, "gateway_method", current.GatewayMethod)
		}
		previous = current

		select {
		case <-ctx.Done():
			return
		case <-time.After(interval):
		}
	}
}

func connectivityChanged(previous, current *ConnectivityReport) bool {
	return previous.InternetOK != current.InternetOK ||
		previous.DNSOK != current.DNSOK ||
		previous.GatewayOK != current.GatewayOK ||
		previous.GatewayMethod != current.GatewayMethod
}

func defaultInternetProbe(ctx context.Context) (bool, uint32) {
	started := time.Now()
	conn, err := (&net.Dialer{Timeout: 3 * time.Second}).DialContext(ctx, "tcp", "1.1.1.1:443")
	elapsed := uint32(time.Since(started).Milliseconds())
	if err != nil {
		return false, elapsed
	}
	_ = conn.Close()
	return true, elapsed
}

func defaultDNSProbe(ctx context.Context) (bool, uint32) {
	started := time.Now()
	lookupCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	_, err := net.DefaultResolver.LookupHost(lookupCtx, "one.one.one.one")
	return err == nil, uint32(time.Since(started).Milliseconds())
}
