package agent

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"os"
	"runtime"

	agentpb "github.com/jkdesenvolvedor/dockeeper/proto/gen/dockeeper/agent/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

// Enroll performs first-time enrollment with the manager.
// It connects without mTLS (insecure transport) and uses the one-time token
// for authentication. On success it returns the enrollment response containing
// the CA cert, client cert and client key.
func Enroll(ctx context.Context, managerURL, token string) (*agentpb.EnrollResponse, error) {
	conn, err := grpc.NewClient(
		managerURL,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("dial manager for enrollment: %w", err)
	}
	defer func() {
		if cerr := conn.Close(); cerr != nil {
			slog.Error("failed to close enrollment connection", "error", cerr)
		}
	}()

	client := agentpb.NewAgentEnrollmentServiceClient(conn)

	info, err := collectServerInfo()
	if err != nil {
		return nil, fmt.Errorf("collect server info: %w", err)
	}

	resp, err := client.Enroll(ctx, &agentpb.EnrollRequest{
		Token: token,
		Info:  info,
	})
	if err != nil {
		return nil, fmt.Errorf("enrollment RPC: %w", err)
	}

	slog.Info("enrollment successful", "server_id", resp.GetServerId())
	return resp, nil
}

// RenewCert requests a new client certificate using an existing mTLS connection.
func RenewCert(ctx context.Context, managerURL, serverID string, tlsCfg *tls.Config) (*agentpb.RenewCertResponse, error) {
	conn, err := grpc.NewClient(
		managerURL,
		grpc.WithTransportCredentials(credentials.NewTLS(tlsCfg)),
	)
	if err != nil {
		return nil, fmt.Errorf("dial manager for cert renewal: %w", err)
	}
	defer func() {
		if cerr := conn.Close(); cerr != nil {
			slog.Error("failed to close renewal connection", "error", cerr)
		}
	}()

	client := agentpb.NewAgentEnrollmentServiceClient(conn)

	resp, err := client.RenewCert(ctx, &agentpb.RenewCertRequest{
		ServerId: serverID,
	})
	if err != nil {
		return nil, fmt.Errorf("renew cert RPC: %w", err)
	}

	slog.Info("certificate renewed", "server_id", serverID)
	return resp, nil
}

func collectServerInfo() (*agentpb.ServerInfo, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return nil, fmt.Errorf("get hostname: %w", err)
	}

	return &agentpb.ServerInfo{
		Hostname:      hostname,
		Os:            runtime.GOOS,
		Arch:          runtime.GOARCH,
		AgentVersion:  Version,
		Orchestrator:  "docker",
	}, nil
}
