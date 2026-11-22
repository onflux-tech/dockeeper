package store

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/pocketbase/pocketbase/core"

	"github.com/onflux-tech/dockeeper/internal/shared/constants"
	agentpb "github.com/onflux-tech/dockeeper/proto/gen/dockeeper/agent/v1"
)

type ConnectivityStore struct {
	app core.App
}

func NewConnectivityStore(app core.App) *ConnectivityStore {
	return &ConnectivityStore{app: app}
}

func (s *ConnectivityStore) Save(serverID string, req *agentpb.ReportConnectivityRequest) (string, error) {
	collectedAt := time.Now().UTC()
	if req.GetCollectedAt() != nil {
		collectedAt = req.GetCollectedAt().AsTime().UTC()
	}

	rawCol, err := s.app.FindCachedCollectionByNameOrId(constants.CollServerConnectivityRaw)
	if err != nil {
		return "", fmt.Errorf("find collection %s: %w", constants.CollServerConnectivityRaw, err)
	}

	raw := core.NewRecord(rawCol)
	raw.Set("server", serverID)
	raw.Set("timestamp", collectedAt)
	raw.Set("internet_ok", req.GetInternetOk())
	raw.Set("internet_ms", req.GetInternetMs())
	raw.Set("dns_ok", req.GetDnsOk())
	raw.Set("dns_ms", req.GetDnsMs())
	raw.Set("gateway_ok", req.GetGatewayOk())
	raw.Set("gateway_ms", req.GetGatewayMs())
	raw.Set("gateway_method", req.GetGatewayMethod())
	raw.Set("error", req.GetError())
	if err := s.app.Save(raw); err != nil {
		return "", fmt.Errorf("save connectivity raw: %w", err)
	}

	status := "degraded"
	if req.GetInternetOk() && req.GetDnsOk() && req.GetGatewayOk() {
		status = "healthy"
	}

	details := map[string]any{
		"internet": map[string]any{"ok": req.GetInternetOk(), "ms": req.GetInternetMs()},
		"dns":      map[string]any{"ok": req.GetDnsOk(), "ms": req.GetDnsMs()},
		"gateway":  map[string]any{"ok": req.GetGatewayOk(), "ms": req.GetGatewayMs(), "method": req.GetGatewayMethod()},
		"error":    req.GetError(),
	}
	if summary := s.httpSummary(serverID); summary != nil {
		details["http"] = summary
	}

	server, err := s.app.FindRecordById(constants.CollServers, serverID)
	if err != nil {
		return "", fmt.Errorf("find server %s: %w", serverID, err)
	}
	server.Set("connectivity_status", status)
	server.Set("connectivity_checked_at", collectedAt.Format(time.RFC3339))
	server.Set("connectivity_latency_ms", int(req.GetInternetMs())+int(req.GetDnsMs())+int(req.GetGatewayMs()))
	if encoded, err := json.Marshal(details); err == nil {
		server.Set("connectivity_details", encoded)
	}
	if err := s.app.Save(server); err != nil {
		return "", fmt.Errorf("update server connectivity: %w", err)
	}

	return status, nil
}

func (s *ConnectivityStore) httpSummary(serverID string) map[string]any {
	checks, err := s.app.FindRecordsByFilter(
		constants.CollHealthChecks,
		"server = {:server} && enabled = true && (check_type = 'http' || check_type = 'tcp')",
		"", 0, 0,
		map[string]any{"server": serverID},
	)
	if err != nil || len(checks) == 0 {
		return nil
	}

	var total, successes, latencySum int
	for _, check := range checks {
		logs, err := s.app.FindRecordsByFilter(
			constants.CollHealthCheckLogs,
			"health_check = {:check}",
			"-checked_at", 1, 0,
			map[string]any{"check": check.Id},
		)
		if err != nil || len(logs) == 0 {
			continue
		}
		total++
		if logs[0].GetBool("success") {
			successes++
		}
		latencySum += logs[0].GetInt("response_time_ms")
	}
	if total == 0 {
		return nil
	}

	return map[string]any{
		"ok":         successes,
		"total":      total,
		"latency_ms": latencySum / total,
	}
}
