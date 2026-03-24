package collector

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"google.golang.org/protobuf/proto"

	agentpb "github.com/jkdesenvolvedor/dockeeper/proto/gen/dockeeper/agent/v1"
)

const traefikInterval = 60 * time.Second

// traefikRouter represents a router from the Traefik API v2.
type traefikRouter struct {
	Name        string   `json:"name"`
	Rule        string   `json:"rule"`
	Service     string   `json:"service"`
	EntryPoints []string `json:"entryPoints"`
	TLS         *struct{} `json:"tls,omitempty"`
	Middlewares []string `json:"middlewares"`
	Provider    string   `json:"provider"`
	Status      string   `json:"status"`
}

// TraefikCollector periodically reads Traefik routes and reports them to the
// manager via gRPC. It first tries the Traefik API, then falls back to
// Docker labels. If Traefik is not detected, it silently does nothing.
type TraefikCollector struct {
	orch       OrchestratorClient
	client     agentpb.AgentTraefikServiceClient
	serverID   string
	logger     *slog.Logger
	lastHash   [32]byte
	hashSet    bool
	apiURL     string // cached Traefik API base URL
	detected   bool
	checkedAt  time.Time
}

// NewTraefikCollector creates a new Traefik collector.
func NewTraefikCollector(
	orch OrchestratorClient,
	client agentpb.AgentTraefikServiceClient,
	serverID string,
	logger *slog.Logger,
) *TraefikCollector {
	return &TraefikCollector{
		orch:     orch,
		client:   client,
		serverID: serverID,
		logger:   logger,
	}
}

// Run starts the periodic collection loop. It blocks until ctx is cancelled.
func (t *TraefikCollector) Run(ctx context.Context) {
	t.collect(ctx)

	ticker := time.NewTicker(traefikInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			t.collect(ctx)
		}
	}
}

func (t *TraefikCollector) collect(ctx context.Context) {
	// Re-check for Traefik every 5 minutes.
	if time.Since(t.checkedAt) > 5*time.Minute {
		t.detectTraefik(ctx)
		t.checkedAt = time.Now()
	}

	if !t.detected {
		return
	}

	routes, err := t.fetchRoutes(ctx)
	if err != nil {
		t.logger.Debug("traefik: fetch routes failed", "error", err)
		return
	}

	req := &agentpb.ReportTraefikRoutesRequest{
		ServerId: t.serverID,
		Routes:   routes,
	}

	if !t.changed(req) {
		return
	}

	resp, err := t.client.ReportTraefikRoutes(ctx, req)
	if err != nil {
		t.logger.Error("traefik: report routes failed", "error", err)
		return
	}
	t.logger.Info("traefik: routes reported", "count", resp.GetReceivedCount())
}

// detectTraefik checks if a Traefik container is running via Docker.
func (t *TraefikCollector) detectTraefik(ctx context.Context) {
	containers, err := t.orch.ListContainers(ctx)
	if err != nil {
		t.detected = false
		return
	}

	for _, c := range containers {
		img := strings.ToLower(c.GetImage())
		if strings.Contains(img, "traefik") {
			t.detected = true
			// Try to find the API port from container info.
			t.apiURL = t.resolveAPIURL(c)
			t.logger.Info("traefik: detected", "container", c.GetName(), "api", t.apiURL)
			return
		}
	}

	t.detected = false
	t.apiURL = ""
}

// resolveAPIURL attempts to determine the Traefik API URL from container ports.
func (t *TraefikCollector) resolveAPIURL(c *agentpb.ContainerInfo) string {
	// Try to parse ports from JSON to find 8080 mapping.
	if portsJSON := c.GetPortsJson(); portsJSON != "" {
		var ports []map[string]any
		if err := json.Unmarshal([]byte(portsJSON), &ports); err == nil {
			for _, p := range ports {
				cp, _ := p["container_port"].(float64)
				hp, _ := p["host_port"].(float64)
				if int(cp) == 8080 && int(hp) != 0 {
					return fmt.Sprintf("http://127.0.0.1:%d", int(hp))
				}
			}
		}
	}
	// Default: Traefik API typically on 8080.
	return "http://127.0.0.1:8080"
}

// fetchRoutes tries the Traefik API first, then falls back to Docker labels.
func (t *TraefikCollector) fetchRoutes(ctx context.Context) ([]*agentpb.TraefikRouteInfo, error) {
	routes, err := t.fetchFromAPI(ctx)
	if err == nil {
		return routes, nil
	}

	t.logger.Debug("traefik: API unavailable, trying Docker labels", "error", err)
	return t.fetchFromLabels(ctx)
}

// fetchFromAPI reads routers from the Traefik API v2.
func (t *TraefikCollector) fetchFromAPI(ctx context.Context) ([]*agentpb.TraefikRouteInfo, error) {
	url := t.apiURL + "/api/http/routers"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("traefik API returned %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 5<<20)) // 5 MB limit
	if err != nil {
		return nil, err
	}

	var routers []traefikRouter
	if err := json.Unmarshal(body, &routers); err != nil {
		return nil, fmt.Errorf("parse routers: %w", err)
	}

	routes := make([]*agentpb.TraefikRouteInfo, 0, len(routers))
	for _, r := range routers {
		routes = append(routes, &agentpb.TraefikRouteInfo{
			Name:        r.Name,
			Rule:        r.Rule,
			Service:     r.Service,
			Entrypoints: r.EntryPoints,
			Tls:         r.TLS != nil,
			Middlewares: r.Middlewares,
			Provider:    r.Provider,
			Status:      r.Status,
		})
	}

	return routes, nil
}

// fetchFromLabels reads Traefik routing info from Docker container labels.
func (t *TraefikCollector) fetchFromLabels(ctx context.Context) ([]*agentpb.TraefikRouteInfo, error) {
	containers, err := t.orch.ListContainers(ctx)
	if err != nil {
		return nil, err
	}

	routeMap := make(map[string]*agentpb.TraefikRouteInfo)

	for _, c := range containers {
		labelsJSON := c.GetLabelsJson()
		if labelsJSON == "" {
			continue
		}

		var labels map[string]string
		if err := json.Unmarshal([]byte(labelsJSON), &labels); err != nil {
			continue
		}

		// Parse labels like: traefik.http.routers.<name>.rule = Host(`...`)
		for key, val := range labels {
			if !strings.HasPrefix(key, "traefik.http.routers.") {
				continue
			}

			parts := strings.SplitN(key, ".", 5)
			if len(parts) < 5 {
				continue
			}
			routerName := parts[3]
			field := parts[4]

			route, ok := routeMap[routerName]
			if !ok {
				route = &agentpb.TraefikRouteInfo{
					Name:     routerName,
					Provider: "docker",
					Status:   "enabled",
				}
				routeMap[routerName] = route
			}

			switch field {
			case "rule":
				route.Rule = val
			case "service":
				route.Service = val
			case "entrypoints":
				route.Entrypoints = strings.Split(val, ",")
			case "tls":
				route.Tls = val == "true" || val == ""
			case "middlewares":
				route.Middlewares = strings.Split(val, ",")
			}
		}
	}

	routes := make([]*agentpb.TraefikRouteInfo, 0, len(routeMap))
	for _, r := range routeMap {
		if r.Rule != "" {
			routes = append(routes, r)
		}
	}

	return routes, nil
}

// changed returns true if the data has changed since last report.
func (t *TraefikCollector) changed(msg proto.Message) bool {
	data, err := proto.Marshal(msg)
	if err != nil {
		return true
	}

	h := sha256.Sum256(data)
	if t.hashSet && t.lastHash == h {
		return false
	}
	t.lastHash = h
	t.hashSet = true
	return true
}
