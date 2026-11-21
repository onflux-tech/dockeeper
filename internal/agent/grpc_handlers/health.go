package grpc_handlers

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/docker/docker/client"

	agentpb "github.com/onflux-tech/dockeeper/proto/gen/dockeeper/agent/v1"
)

// HealthHandler implements AgentHealthServiceServer.
type HealthHandler struct {
	agentpb.UnimplementedAgentHealthServiceServer
	logger *slog.Logger
}

// NewHealthHandler creates a HealthHandler.
func NewHealthHandler(logger *slog.Logger) *HealthHandler {
	return &HealthHandler{logger: logger}
}

// ExecuteHealthCheck resolves the container IP and makes an HTTP GET.
func (h *HealthHandler) ExecuteHealthCheck(ctx context.Context, req *agentpb.ExecuteHealthCheckRequest) (*agentpb.ExecuteHealthCheckResponse, error) {
	containerIP, err := resolveContainerIP(ctx, req.GetContainerId())
	if err != nil {
		return &agentpb.ExecuteHealthCheckResponse{
			Success: false,
			Error:   fmt.Sprintf("resolve IP: %v", err),
		}, nil
	}

	urlPath := req.GetUrlPath()
	if urlPath == "" {
		urlPath = "/"
	}
	if !strings.HasPrefix(urlPath, "/") {
		urlPath = "/" + urlPath
	}

	url := fmt.Sprintf("http://%s:%d%s", containerIP, req.GetPort(), urlPath)

	timeoutMs := req.GetTimeoutMs()
	if timeoutMs <= 0 {
		timeoutMs = 5000
	}

	httpClient := &http.Client{
		Timeout: time.Duration(timeoutMs) * time.Millisecond,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return fmt.Errorf("too many redirects")
			}
			return nil
		},
	}

	start := time.Now()
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return &agentpb.ExecuteHealthCheckResponse{
			Success: false,
			Error:   fmt.Sprintf("build request: %v", err),
		}, nil
	}

	resp, err := httpClient.Do(httpReq)
	elapsed := time.Since(start).Milliseconds()

	if err != nil {
		// Distinguish network errors for clearer diagnostics.
		errMsg := err.Error()
		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			errMsg = "timeout: " + errMsg
		}
		return &agentpb.ExecuteHealthCheckResponse{
			Success:        false,
			ResponseTimeMs: elapsed,
			Error:          errMsg,
		}, nil
	}
	defer func() { _ = resp.Body.Close() }()

	expectedStatus := int(req.GetExpectedStatus())
	if expectedStatus == 0 {
		expectedStatus = 200
	}

	success := resp.StatusCode == expectedStatus
	var errStr string
	if !success {
		errStr = fmt.Sprintf("expected %d, got %d", expectedStatus, resp.StatusCode)
	}

	return &agentpb.ExecuteHealthCheckResponse{
		StatusCode:     int32(resp.StatusCode),
		ResponseTimeMs: elapsed,
		Success:        success,
		Error:          errStr,
	}, nil
}

// resolveContainerIP gets the container's IP from the first attached network.
func resolveContainerIP(ctx context.Context, containerID string) (string, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return "", fmt.Errorf("docker client: %w", err)
	}
	defer func() { _ = cli.Close() }()

	info, err := cli.ContainerInspect(ctx, containerID)
	if err != nil {
		return "", fmt.Errorf("inspect %s: %w", containerID, err)
	}

	if info.State != nil && info.State.Status != "running" {
		return "", fmt.Errorf("container not running (status: %s)", info.State.Status)
	}

	if info.NetworkSettings != nil {
		for _, net := range info.NetworkSettings.Networks {
			if net.IPAddress != "" {
				return net.IPAddress, nil
			}
		}
	}

	return "", fmt.Errorf("no IP address found for container %s", containerID)
}
