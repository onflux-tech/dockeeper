package grpc

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/onflux-tech/dockeeper/internal/manager/services"
	"github.com/onflux-tech/dockeeper/internal/shared/constants"
	agentpb "github.com/onflux-tech/dockeeper/proto/gen/dockeeper/agent/v1"
)

// EnrollmentHandler implements the AgentEnrollmentService gRPC server.
type EnrollmentHandler struct {
	agentpb.UnimplementedAgentEnrollmentServiceServer
	app *pocketbase.PocketBase
	ca  *services.CertAuthority
}

// NewEnrollmentHandler creates a new enrollment handler.
func NewEnrollmentHandler(app *pocketbase.PocketBase, ca *services.CertAuthority) *EnrollmentHandler {
	return &EnrollmentHandler{app: app, ca: ca}
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

		// Create new server record (legacy flow).
		col, colErr := h.app.FindCachedCollectionByNameOrId(constants.CollServers)
		if colErr != nil {
			slog.Error("failed to find servers collection", "error", colErr)
			return nil, status.Error(codes.Internal, "internal error")
		}
		server = core.NewRecord(col)
		server.Set("name", info.GetHostname())
	}

	// Update server record with agent info.
	server.Set("hostname", info.GetHostname())
	server.Set("ip_address", info.GetIpAddress())
	server.Set("os", info.GetOs())
	server.Set("arch", info.GetArch())
	server.Set("docker_version", info.GetDockerVersion())
	server.Set("agent_version", info.GetAgentVersion())
	server.Set("orchestrator", info.GetOrchestrator())
	server.Set("status", "online")
	server.Set("last_seen", time.Now().UTC())
	server.Set("enrolled_at", time.Now().UTC())
	server.Set("enrollment_token", "") // clear token after use

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
	serverID := req.GetServerId()
	if serverID == "" {
		// Try to get from mTLS context.
		if id, ok := ServerIDFromContext(ctx); ok {
			serverID = id
		} else {
			return nil, status.Error(codes.InvalidArgument, "server_id is required")
		}
	}

	// Verify server exists.
	server, err := h.app.FindRecordById(constants.CollServers, serverID)
	if err != nil {
		return nil, status.Error(codes.NotFound, "server not found")
	}

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
		// No token configured — generate one and save it for future use.
		return h.createAndValidateToken(token)
	}

	stored := strings.Trim(records[0].GetString("value"), "\"")
	if stored != token {
		return fmt.Errorf("token mismatch")
	}
	return nil
}

// createAndValidateToken stores the provided token as the enrollment token setting.
// This is used on first enrollment when no token is configured yet.
func (h *EnrollmentHandler) createAndValidateToken(token string) error {
	col, err := h.app.FindCachedCollectionByNameOrId(constants.CollSettings)
	if err != nil {
		return fmt.Errorf("find settings collection: %w", err)
	}

	record := core.NewRecord(col)
	record.Set("key", "enrollment_token")
	record.Set("value", token)

	if err := h.app.Save(record); err != nil {
		return fmt.Errorf("save enrollment token: %w", err)
	}

	slog.Info("enrollment token saved from first enrollment")
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
