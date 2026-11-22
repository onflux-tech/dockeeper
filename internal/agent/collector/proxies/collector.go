package proxies

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"google.golang.org/protobuf/proto"

	"github.com/onflux-tech/dockeeper/internal/agent/collector"
	agentpb "github.com/onflux-tech/dockeeper/proto/gen/dockeeper/agent/v1"
)

const (
	collectInterval = time.Minute
	commandTimeout  = 15 * time.Second
	httpTimeout     = 5 * time.Second
)

type Option func(*Collector)

type Collector struct {
	logger        *slog.Logger
	hostApps      func() []collector.HostApp
	runCommand    func(ctx context.Context, name string, args ...string) (string, error)
	httpClient    *http.Client
	traefikAPIURL func(ctx context.Context) string
}

func New(logger *slog.Logger, hostApps func() []collector.HostApp, opts ...Option) *Collector {
	c := &Collector{
		logger:     logger,
		hostApps:   hostApps,
		runCommand: runCommand,
		httpClient: &http.Client{Timeout: httpTimeout},
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

func WithTraefikAPIURL(fn func(ctx context.Context) string) Option {
	return func(c *Collector) {
		c.traefikAPIURL = fn
	}
}

func WithCommandRunner(runner func(ctx context.Context, name string, args ...string) (string, error)) Option {
	return func(c *Collector) {
		c.runCommand = runner
	}
}

func WithHTTPClient(client *http.Client) Option {
	return func(c *Collector) {
		c.httpClient = client
	}
}

func (c *Collector) Run(ctx context.Context, interval time.Duration, report func([]*agentpb.ProxyInstanceInfo) error) {
	if interval <= 0 {
		interval = collectInterval
	}

	var lastHash [32]byte
	hashSet := false
	collect := func() {
		instances, err := c.Collect(ctx)
		if err != nil {
			c.logger.Debug("proxies: collect failed", "error", err)
			return
		}
		if len(instances) == 0 {
			instances = nil
		}

		encoded, err := proto.Marshal(&agentpb.ReportProxyInventoryRequest{Instances: instances})
		if err != nil {
			return
		}
		sum := sha256.Sum256(encoded)
		if !hashSet || sum != lastHash {
			lastHash = sum
			hashSet = true
			for _, instance := range instances {
				c.logger.Info("proxies: instance detected", "kind", instance.GetKind(), "app", instance.GetApp(), "routes", len(instance.GetRoutes()))
			}
		}

		if report != nil {
			if err := report(instances); err != nil {
				c.logger.Warn("proxies: report failed", "error", err)
			}
		}
	}

	collect()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			collect()
		}
	}
}

func (c *Collector) Collect(ctx context.Context) ([]*agentpb.ProxyInstanceInfo, error) {
	var apps []collector.HostApp
	if c.hostApps != nil {
		apps = c.hostApps()
	}

	instances := make([]*agentpb.ProxyInstanceInfo, 0, 4)

	if app := findApp(apps, "nginx"); app != nil {
		instances = append(instances, c.collectNginx(ctx, *app))
	}
	if app := findApp(apps, "caddy"); app != nil {
		instances = append(instances, c.collectCaddy(ctx, *app))
	}
	if app := findApp(apps, "haproxy"); app != nil {
		instances = append(instances, c.collectHAProxy(ctx, *app))
	}
	if app := findApp(apps, "envoy"); app != nil {
		instances = append(instances, c.collectEnvoy(ctx, *app))
	}

	traefikApp := findApp(apps, "traefik")
	apiURL := ""
	if c.traefikAPIURL != nil {
		apiURL = c.traefikAPIURL(ctx)
	}
	if traefikApp != nil || apiURL != "" {
		if apiURL == "" {
			apiURL = "http://127.0.0.1:8080"
		}
		instances = append(instances, c.collectTraefik(ctx, traefikApp, apiURL))
	}

	if len(instances) == 0 {
		return nil, nil
	}
	return instances, nil
}

func findApp(apps []collector.HostApp, kind string) *collector.HostApp {
	for i := range apps {
		name := strings.ToLower(apps[i].Name)
		unit := strings.ToLower(apps[i].Unit)
		if strings.Contains(name, kind) || strings.Contains(unit, kind) {
			return &apps[i]
		}
	}
	return nil
}

func runCommand(ctx context.Context, name string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, commandTimeout)
	defer cancel()

	output, err := exec.CommandContext(ctx, name, args...).CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf("%s %s: %w", name, strings.Join(args, " "), err)
	}
	return string(output), nil
}

func (c *Collector) getJSON(ctx context.Context, url string, target any) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	response, err := c.httpClient.Do(request)
	if err != nil {
		return err
	}
	defer func() { _ = response.Body.Close() }()

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("unexpected status %d", response.StatusCode)
	}
	return json.NewDecoder(response.Body).Decode(target)
}

func instance(kind, source, app string) *agentpb.ProxyInstanceInfo {
	return &agentpb.ProxyInstanceInfo{
		Kind:   kind,
		Source: source,
		App:    app,
		Status: "healthy",
	}
}
