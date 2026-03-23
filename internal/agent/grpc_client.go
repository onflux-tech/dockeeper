package agent

import (
	"crypto/tls"
	"fmt"
	"log/slog"

	agentpb "github.com/jkdesenvolvedor/dockeeper/proto/gen/dockeeper/agent/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
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
}

// NewGRPCClient dials the manager with mTLS and returns a connected client.
func NewGRPCClient(addr string, tlsCfg *tls.Config) (*GRPCClient, error) {
	conn, err := grpc.NewClient(
		addr,
		grpc.WithTransportCredentials(credentials.NewTLS(tlsCfg)),
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
	}, nil
}

// Close terminates the underlying gRPC connection.
func (c *GRPCClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

func (c *GRPCClient) Enrollment() agentpb.AgentEnrollmentServiceClient  { return c.enrollment }
func (c *GRPCClient) Metrics() agentpb.AgentMetricsServiceClient        { return c.metrics }
func (c *GRPCClient) Discovery() agentpb.AgentDiscoveryServiceClient    { return c.discovery }
func (c *GRPCClient) Commands() agentpb.AgentCommandsServiceClient      { return c.commands }
func (c *GRPCClient) Exec() agentpb.AgentExecServiceClient              { return c.exec }
func (c *GRPCClient) Logs() agentpb.AgentLogsServiceClient              { return c.logs }
