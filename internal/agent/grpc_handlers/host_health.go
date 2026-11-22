package grpc_handlers

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/onflux-tech/dockeeper/internal/agent/collector"
	agentpb "github.com/onflux-tech/dockeeper/proto/gen/dockeeper/agent/v1"
)

const (
	defaultHostCheckTimeout = 5 * time.Second
	maxHostCheckTimeout     = 30 * time.Second
	maxHostCheckRedirects   = 3
	maxHostCheckBodyBytes   = 1 << 20
)

type HostHealthHandler struct {
	logger        *slog.Logger
	inventory     func() []collector.HostApp
	systemdActive func(ctx context.Context, unit string) (bool, error)
	httpClient    *http.Client
	insecureHTTP  *http.Client
}

type HostHealthOption func(*HostHealthHandler)

func WithHostInventory(inventory func() []collector.HostApp) HostHealthOption {
	return func(h *HostHealthHandler) {
		h.inventory = inventory
	}
}

func WithSystemdActive(active func(ctx context.Context, unit string) (bool, error)) HostHealthOption {
	return func(h *HostHealthHandler) {
		h.systemdActive = active
	}
}

func NewHostHealthHandler(logger *slog.Logger, opts ...HostHealthOption) *HostHealthHandler {
	h := &HostHealthHandler{
		logger:        logger,
		systemdActive: defaultSystemdActive,
		httpClient:    newHostCheckClient(false),
		insecureHTTP:  newHostCheckClient(true),
	}
	for _, opt := range opts {
		opt(h)
	}
	return h
}

func newHostCheckClient(skipVerify bool) *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: skipVerify},
			IdleConnTimeout: 90 * time.Second,
			MaxIdleConns:    10,
		},
		CheckRedirect: func(_ *http.Request, via []*http.Request) error {
			if len(via) >= maxHostCheckRedirects {
				return fmt.Errorf("too many redirects")
			}
			return nil
		},
	}
}

func (h *HostHealthHandler) Execute(ctx context.Context, req *agentpb.HostHealthCheckCommand) *agentpb.HostHealthCheckResult {
	switch req.GetCheckType() {
	case "", "http":
		return h.executeHTTP(ctx, req)
	case "tcp":
		return h.executeTCP(ctx, req)
	case "process":
		return h.executeProcess(ctx, req)
	default:
		return hostCheckFailure(fmt.Errorf("unsupported check type: %s", req.GetCheckType()))
	}
}

func (h *HostHealthHandler) executeHTTP(ctx context.Context, req *agentpb.HostHealthCheckCommand) *agentpb.HostHealthCheckResult {
	host := req.GetHost()
	if host == "" {
		host = "127.0.0.1"
	}
	if err := validateHostLiteral(host); err != nil {
		return hostCheckFailure(err)
	}

	scheme := req.GetScheme()
	if scheme == "" {
		scheme = "http"
	}
	if scheme != "http" && scheme != "https" {
		return hostCheckFailure(fmt.Errorf("invalid scheme: %s", scheme))
	}

	method := strings.ToUpper(req.GetMethod())
	if method == "" {
		method = http.MethodGet
	}
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodPost:
	default:
		return hostCheckFailure(fmt.Errorf("invalid method: %s", method))
	}

	port := req.GetPort()
	if port == 0 {
		if scheme == "https" {
			port = 443
		} else {
			port = 80
		}
	}
	if port > 65535 {
		return hostCheckFailure(fmt.Errorf("invalid port: %d", port))
	}

	path := req.GetPath()
	if path == "" {
		path = "/"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	timeout := hostCheckTimeout(req.GetTimeoutMs())
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	target := fmt.Sprintf("%s://%s:%d%s", scheme, host, port, path)
	request, err := http.NewRequestWithContext(ctx, method, target, nil)
	if err != nil {
		return hostCheckFailure(fmt.Errorf("build request: %w", err))
	}
	for key, value := range req.GetHeaders() {
		request.Header.Set(key, value)
	}

	client := h.httpClient
	if req.GetTlsSkipVerify() {
		client = h.insecureHTTP
	}

	started := time.Now()
	response, err := client.Do(request)
	if err != nil {
		result := hostCheckFailure(fmt.Errorf("request failed: %w", err))
		result.ResponseTimeMs = elapsedMS(started)
		return result
	}
	defer func() { _ = response.Body.Close() }()

	body, _ := io.ReadAll(io.LimitReader(response.Body, maxHostCheckBodyBytes))
	result := &agentpb.HostHealthCheckResult{
		Success:        true,
		StatusCode:     int32(response.StatusCode),
		ResponseTimeMs: elapsedMS(started),
	}

	if expected := int(req.GetExpectedStatus()); expected > 0 && response.StatusCode != expected {
		result.Success = false
		result.Error = fmt.Sprintf("expected status %d, got %d", expected, response.StatusCode)
	}
	if result.Success {
		if expectedBody := req.GetExpectedBody(); expectedBody != "" && !strings.Contains(string(body), expectedBody) {
			result.Success = false
			result.Error = "expected body not found"
		}
	}

	return result
}

func (h *HostHealthHandler) executeTCP(ctx context.Context, req *agentpb.HostHealthCheckCommand) *agentpb.HostHealthCheckResult {
	host := req.GetHost()
	if host == "" {
		host = "127.0.0.1"
	}
	if err := validateHostLiteral(host); err != nil {
		return hostCheckFailure(err)
	}
	if req.GetPort() == 0 || req.GetPort() > 65535 {
		return hostCheckFailure(fmt.Errorf("invalid port: %d", req.GetPort()))
	}

	timeout := hostCheckTimeout(req.GetTimeoutMs())
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	started := time.Now()
	conn, err := (&net.Dialer{Timeout: timeout}).DialContext(ctx, "tcp", net.JoinHostPort(host, strconv.Itoa(int(req.GetPort()))))
	if err != nil {
		result := hostCheckFailure(fmt.Errorf("dial failed: %w", err))
		result.ResponseTimeMs = elapsedMS(started)
		return result
	}
	_ = conn.Close()

	return &agentpb.HostHealthCheckResult{
		Success:        true,
		ResponseTimeMs: elapsedMS(started),
	}
}

func (h *HostHealthHandler) executeProcess(ctx context.Context, req *agentpb.HostHealthCheckCommand) *agentpb.HostHealthCheckResult {
	app, found := h.findApp(req.GetSource(), req.GetName())
	if !found {
		return hostCheckFailure(fmt.Errorf("app not found in inventory: %s/%s", req.GetSource(), req.GetName()))
	}

	if app.Source == "systemd" {
		active, err := h.systemdActive(ctx, app.Name)
		if err != nil {
			return hostCheckFailure(fmt.Errorf("systemd status: %w", err))
		}
		if !active {
			return hostCheckFailure(fmt.Errorf("unit is not active: %s", app.Name))
		}
		return &agentpb.HostHealthCheckResult{Success: true}
	}

	if app.Status != "running" {
		return hostCheckFailure(fmt.Errorf("app is %s: %s", app.Status, app.Name))
	}
	return &agentpb.HostHealthCheckResult{Success: true}
}

func (h *HostHealthHandler) findApp(source, name string) (collector.HostApp, bool) {
	if h.inventory == nil {
		return collector.HostApp{}, false
	}
	for _, app := range h.inventory() {
		if app.Source == source && app.Name == name {
			return app, true
		}
	}
	return collector.HostApp{}, false
}

func hostCheckFailure(err error) *agentpb.HostHealthCheckResult {
	return &agentpb.HostHealthCheckResult{Success: false, Error: err.Error()}
}

func hostCheckTimeout(ms uint32) time.Duration {
	if ms == 0 {
		return defaultHostCheckTimeout
	}
	timeout := time.Duration(ms) * time.Millisecond
	if timeout > maxHostCheckTimeout {
		return maxHostCheckTimeout
	}
	return timeout
}

func elapsedMS(started time.Time) uint32 {
	elapsed := time.Since(started).Milliseconds()
	if elapsed < 0 {
		return 0
	}
	return uint32(elapsed)
}

func validateHostLiteral(host string) error {
	if host == "" || strings.ContainsAny(host, "/@\\ \t\r\n") {
		return fmt.Errorf("invalid host: %q", host)
	}
	if net.ParseIP(host) != nil {
		return nil
	}
	for _, char := range host {
		switch {
		case char >= 'a' && char <= 'z', char >= 'A' && char <= 'Z', char >= '0' && char <= '9', char == '.', char == '-', char == '_':
		default:
			return fmt.Errorf("invalid host: %q", host)
		}
	}
	return nil
}
