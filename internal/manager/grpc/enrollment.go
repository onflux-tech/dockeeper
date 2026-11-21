package grpc

import (
	"context"
	"crypto/subtle"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/pocketbase/pocketbase/core"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/onflux-tech/dockeeper/internal/manager/events"
	"github.com/onflux-tech/dockeeper/internal/manager/services"
	"github.com/onflux-tech/dockeeper/internal/shared/constants"
	agentpb "github.com/onflux-tech/dockeeper/proto/gen/dockeeper/agent/v1"
)

// EnrollmentHandler implements the AgentEnrollmentService gRPC server.
type EnrollmentHandler struct {
	agentpb.UnimplementedAgentEnrollmentServiceServer
	app core.App
	ca  *services.CertAuthority
	bus *events.Bus
}

// NewEnrollmentHandler creates a new enrollment handler.
func NewEnrollmentHandler(app core.App, ca *services.CertAuthority, bus *events.Bus) *EnrollmentHandler {
	return &EnrollmentHandler{app: app, ca: ca, bus: bus}
}

// Enroll validates the enrollment token, creates or updates a server record in PocketBase,
// signs a client certificate and returns it along with the CA cert.
func (h *EnrollmentHandler) Enroll(ctx context.Context, req *agentpb.EnrollRequest) (*agentpb.EnrollResponse, error) {
	if req.GetToken() == "" {
		return nil, status.Error(codes.InvalidArgument, "enrollment token is required")
	}

	info := req.GetInfo()
	if info == nil {
		return nil, status.Error(codes.InvalidArgument, "server info is required")
	}

	// Try per-server token first (provisioned via wizard).
	server, err := h.findProvisionedServer(req.GetToken())
	if err != nil {
		// Fall back to legacy global token.
		if err := h.validateToken(req.GetToken()); err != nil {
			slog.Warn("enrollment rejected: invalid token", "error", err)
			return nil, status.Error(codes.Unauthenticated, "invalid enrollment token")
		}

		// Reuse an existing server with the same hostname to avoid duplicates
		// on re-enrollment with the legacy global token.
		server, err = h.serverForEnrollment(info)
		if err != nil {
			slog.Error("failed to resolve server record", "error", err)
			return nil, status.Error(codes.Internal, "internal error")
		}
	}

	// Update server record with agent info.
	// Prefer the IP reported by the agent (outbound-detected), fall back to gRPC peer.
	ipAddress := info.GetIpAddress()
	if ipAddress == "" {
		if peerIP, ok := peerHostFromContext(ctx); ok {
			ipAddress = peerIP
		}
	}

	server.Set("hostname", info.GetHostname())
	server.Set("ip_address", ipAddress)
	server.Set("os", info.GetOs())
	server.Set("arch", info.GetArch())
	server.Set("docker_version", info.GetDockerVersion())
	server.Set("agent_version", info.GetAgentVersion())
	server.Set("orchestrator", info.GetOrchestrator())
	server.Set("status", "online")
	server.Set("last_seen", time.Now().UTC())
	server.Set("enrolled_at", time.Now().UTC())
	server.Set("enrollment_token", "") // clear token after use
	server.Set("cpu_model", info.GetCpuModel())
	server.Set("cpu_cores", info.GetCpuCores())
	server.Set("mem_total", info.GetMemTotal())

	if err := h.app.Save(server); err != nil {
		slog.Error("failed to save server record", "error", err)
		return nil, status.Error(codes.Internal, "failed to register server")
	}

	// Sign agent certificate (CN = server.Id for mTLS identification).
	certPEM, keyPEM, expiresAt, err := h.ca.SignAgentCert(server.Id)
	if err != nil {
		slog.Error("failed to sign agent cert", "error", err, "server_id", server.Id)
		return nil, status.Error(codes.Internal, "failed to generate certificate")
	}

	// Update cert_expires_at on the server record.
	server.Set("cert_expires_at", expiresAt.UTC())
	if err := h.app.Save(server); err != nil {
		slog.Error("failed to update cert expiry", "error", err, "server_id", server.Id)
	}

	slog.Info("agent enrolled",
		"server_id", server.Id,
		"hostname", info.GetHostname(),
		"cert_expires", expiresAt.Format(time.RFC3339),
	)

	if h.bus != nil {
		h.bus.Publish(events.Event{
			Type:       events.ServerEnrolled,
			Severity:   events.SeverityFor(events.ServerEnrolled),
			ServerID:   server.Id,
			ServerName: info.GetHostname(),
			Details: map[string]any{
				"ip_address":     ipAddress,
				"os":             info.GetOs(),
				"arch":           info.GetArch(),
				"docker_version": info.GetDockerVersion(),
				"agent_version":  info.GetAgentVersion(),
			},
			Timestamp: time.Now().UTC(),
		})
	}

	return &agentpb.EnrollResponse{
		ServerId:      server.Id,
		CaCert:        h.ca.CACertPEM(),
		ClientCert:    certPEM,
		ClientKey:     keyPEM,
		CertExpiresAt: timestamppb.New(expiresAt),
	}, nil
}

// RenewCert issues a new client certificate for an already-enrolled agent.
func (h *EnrollmentHandler) RenewCert(ctx context.Context, req *agentpb.RenewCertRequest) (*agentpb.RenewCertResponse, error) {
	// Always use the mTLS identity as the authoritative server ID.
	mtlsID, ok := ServerIDFromContext(ctx)
	if !ok || mtlsID == "" {
		return nil, status.Error(codes.Unauthenticated, "mTLS identity required")
	}

	serverID := req.GetServerId()
	if serverID == "" {
		serverID = mtlsID
	}

	// Prevent impersonation: the requested server_id MUST match the mTLS CN.
	if serverID != mtlsID {
		slog.Warn("RenewCert: server_id mismatch with mTLS CN",
			"requested", serverID, "mtls_cn", mtlsID)
		return nil, status.Error(codes.PermissionDenied, "server_id does not match certificate identity")
	}

	// Verify server exists.
	server, err := h.app.FindRecordById(constants.CollServers, serverID)
	if err != nil {
		return nil, status.Error(codes.NotFound, "server not found")
	}
	syncServerAddressFromContext(h.app, ctx, serverID)

	// Sign new certificate.
	certPEM, keyPEM, expiresAt, err := h.ca.SignAgentCert(serverID)
	if err != nil {
		slog.Error("failed to sign renewal cert", "error", err, "server_id", serverID)
		return nil, status.Error(codes.Internal, "failed to generate certificate")
	}

	// Update cert_expires_at and last_seen.
	server.Set("cert_expires_at", expiresAt.UTC())
	server.Set("last_seen", time.Now().UTC())
	if err := h.app.Save(server); err != nil {
		slog.Error("failed to update server after cert renewal", "error", err, "server_id", serverID)
	}

	slog.Info("agent cert renewed", "server_id", serverID, "cert_expires", expiresAt.Format(time.RFC3339))

	return &agentpb.RenewCertResponse{
		ClientCert: certPEM,
		ClientKey:  keyPEM,
		ExpiresAt:  timestamppb.New(expiresAt),
	}, nil
}

// validateToken checks the enrollment token against the stored setting.
func (h *EnrollmentHandler) validateToken(token string) error {
	// Look for enrollment_token in the settings collection.
	records, err := h.app.FindRecordsByFilter(
		constants.CollSettings,
		"key = 'enrollment_token'",
		"", 1, 0,
	)
	if err != nil || len(records) == 0 {
		// No token configured — reject enrollment instead of auto-creating.
		return fmt.Errorf("no enrollment token configured")
	}

	stored := strings.Trim(records[0].GetString("value"), "\"")
	// Constant-time comparison to prevent timing attacks.
	if subtle.ConstantTimeCompare([]byte(stored), []byte(token)) != 1 {
		return fmt.Errorf("token mismatch")
	}
	return nil
}

// findProvisionedServer looks for a pending server record that matches the given per-server token.
func (h *EnrollmentHandler) findProvisionedServer(token string) (*core.Record, error) {
	records, err := h.app.FindRecordsByFilter(
		constants.CollServers,
		"enrollment_token = {:token} && status = 'pending'",
		"", 1, 0,
		map[string]any{"token": token},
	)
	if err != nil || len(records) == 0 {
		return nil, fmt.Errorf("no provisioned server found for token")
	}
	return records[0], nil
}

func (h *EnrollmentHandler) serverForEnrollment(info *agentpb.ServerInfo) (*core.Record, error) {
	if hostname := info.GetHostname(); hostname != "" {
		records, err := h.app.FindRecordsByFilter(
			constants.CollServers,
			"hostname = {:hostname}",
			"-last_seen", 1, 0,
			map[string]any{"hostname": hostname},
		)
		if err == nil && len(records) > 0 {
			return records[0], nil
		}
	}

	col, err := h.app.FindCachedCollectionByNameOrId(constants.CollServers)
	if err != nil {
		return nil, fmt.Errorf("find collection %s: %w", constants.CollServers, err)
	}
	server := core.NewRecord(col)
	server.Set("name", info.GetHostname())
	return server, nil
}
