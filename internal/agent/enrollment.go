package agent

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"net"
	"os"
	"runtime"

	dockerclient "github.com/docker/docker/client"
	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/mem"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	agentpb "github.com/onflux-tech/dockeeper/proto/gen/dockeeper/agent/v1"
)

// Enroll performs first-time enrollment with the manager.
// It connects without mTLS (insecure transport) and uses the one-time token
// for authentication. On success it returns the enrollment response containing
// the CA cert, client cert and client key.
func Enroll(ctx context.Context, managerURL, token string) (*agentpb.EnrollResponse, error) {
	conn, err := grpc.NewClient(
		managerURL,
		grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
			InsecureSkipVerify: true, //nolint:gosec // first enrollment — no CA cert yet
		})),
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

	info := &agentpb.ServerInfo{
		Hostname:     hostname,
		Os:           runtime.GOOS,
		Arch:         runtime.GOARCH,
		AgentVersion: Version,
	}

	// Detect outbound IP by dialing a public address (no actual traffic is sent).
	if conn, err := net.Dial("udp4", "8.8.8.8:53"); err == nil {
		if addr, ok := conn.LocalAddr().(*net.UDPAddr); ok {
			info.IpAddress = addr.IP.String()
		}
		_ = conn.Close()
	}

	// Get Docker version from the daemon.
	cli, err := dockerclient.NewClientWithOpts(dockerclient.FromEnv, dockerclient.WithAPIVersionNegotiation())
	if err == nil {
		if _, pingErr := cli.Ping(context.Background()); pingErr == nil {
			info.Orchestrator = "docker"
			if sv, verErr := cli.ServerVersion(context.Background()); verErr == nil {
				info.DockerVersion = sv.Version
			}
		}
		_ = cli.Close()
	}
	info.Orchestrator = normalizeOrchestrator(info.Orchestrator)

	// CPU model and core count.
	if infos, cpuErr := cpu.Info(); cpuErr == nil && len(infos) > 0 {
		info.CpuModel = infos[0].ModelName
	}
	if cores, cpuErr := cpu.Counts(true); cpuErr == nil {
		info.CpuCores = uint32(cores)
	}

	// Total RAM.
	if vm, memErr := mem.VirtualMemory(); memErr == nil {
		info.MemTotal = vm.Total
	}

	return info, nil
}

func normalizeOrchestrator(label string) string {
	if label == "" {
		return "none"
	}
	return label
}
