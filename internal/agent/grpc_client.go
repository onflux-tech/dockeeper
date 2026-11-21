package agent

import (
	"crypto/tls"
	"fmt"
	"log/slog"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	agentpb "github.com/onflux-tech/dockeeper/proto/gen/dockeeper/agent/v1"
)

// GRPCClient wraps a gRPC connection to the manager and exposes typed service
// clients for every agent service.
type GRPCClient struct {
	conn       *grpc.ClientConn
	enrollment agentpb.AgentEnrollmentServiceClient
	metrics    agentpb.AgentMetricsServiceClient
	discovery  agentpb.AgentDiscoveryServiceClient
	commands   agentpb.AgentCommandsServiceClient
	exec       agentpb.AgentExecServiceClient
	logs       agentpb.AgentLogsServiceClient
	traefik    agentpb.AgentTraefikServiceClient
	cmdChannel agentpb.AgentCommandChannelServiceClient
	dataStream agentpb.AgentDataStreamServiceClient
	execStream agentpb.AgentExecStreamServiceClient
}

// NewGRPCClient dials the manager with mTLS and returns a connected client.
func NewGRPCClient(addr string, tlsCfg *tls.Config) (*GRPCClient, error) {
	conn, err := grpc.NewClient(
		addr,
		grpc.WithTransportCredentials(credentials.NewTLS(tlsCfg)),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(16*1024*1024), // 16MB for binary update chunks
			grpc.MaxCallSendMsgSize(16*1024*1024), // 16MB for large uploads
		),
	)
	if err != nil {
		return nil, fmt.Errorf("dial manager: %w", err)
	}

	slog.Info("gRPC connection established", "addr", addr)

	return &GRPCClient{
		conn:       conn,
		enrollment: agentpb.NewAgentEnrollmentServiceClient(conn),
		metrics:    agentpb.NewAgentMetricsServiceClient(conn),
		discovery:  agentpb.NewAgentDiscoveryServiceClient(conn),
		commands:   agentpb.NewAgentCommandsServiceClient(conn),
		exec:       agentpb.NewAgentExecServiceClient(conn),
		logs:       agentpb.NewAgentLogsServiceClient(conn),
		traefik:    agentpb.NewAgentTraefikServiceClient(conn),
		cmdChannel: agentpb.NewAgentCommandChannelServiceClient(conn),
		dataStream: agentpb.NewAgentDataStreamServiceClient(conn),
		execStream: agentpb.NewAgentExecStreamServiceClient(conn),
	}, nil
}

// Close terminates the underlying gRPC connection.
func (c *GRPCClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

func (c *GRPCClient) Enrollment() agentpb.AgentEnrollmentServiceClient { return c.enrollment }
func (c *GRPCClient) Metrics() agentpb.AgentMetricsServiceClient       { return c.metrics }
func (c *GRPCClient) Discovery() agentpb.AgentDiscoveryServiceClient   { return c.discovery }
func (c *GRPCClient) Commands() agentpb.AgentCommandsServiceClient     { return c.commands }
func (c *GRPCClient) Exec() agentpb.AgentExecServiceClient             { return c.exec }
func (c *GRPCClient) Logs() agentpb.AgentLogsServiceClient             { return c.logs }
func (c *GRPCClient) Traefik() agentpb.AgentTraefikServiceClient       { return c.traefik }
func (c *GRPCClient) Conn() *grpc.ClientConn                           { return c.conn }
