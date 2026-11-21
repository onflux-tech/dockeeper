package grpc

import (
	"fmt"
	"log/slog"
	"net"

	"github.com/pocketbase/pocketbase"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/onflux-tech/dockeeper/internal/manager/channel"
	"github.com/onflux-tech/dockeeper/internal/manager/events"
	"github.com/onflux-tech/dockeeper/internal/manager/services"
	"github.com/onflux-tech/dockeeper/internal/manager/store"
	agentpb "github.com/onflux-tech/dockeeper/proto/gen/dockeeper/agent/v1"
)

// Server wraps the gRPC server for agent communication.
type Server struct {
	grpcServer *grpc.Server
	addr       string
	Registry   *channel.Registry
}

// NewServer creates a configured gRPC server with TLS, interceptors and
// all agent services registered. The registry parameter is the shared
// channel registry that all manager services use to communicate with agents.
func NewServer(app *pocketbase.PocketBase, ca *services.CertAuthority, addr string, bus *events.Bus, registry *channel.Registry) (*Server, error) {
	tlsCfg, err := ca.ServerTLSConfig()
	if err != nil {
		return nil, fmt.Errorf("server TLS config: %w", err)
	}

	gs := grpc.NewServer(
		grpc.Creds(credentials.NewTLS(tlsCfg)),
		grpc.MaxRecvMsgSize(16*1024*1024), // 16MB for large agent messages
		grpc.MaxSendMsgSize(16*1024*1024), // 16MB for binary update chunks
		grpc.ChainUnaryInterceptor(ChainUnaryInterceptors(tlsCfg)...),
		grpc.ChainStreamInterceptor(ChainStreamInterceptors(tlsCfg)...),
	)

	// Register existing agent→manager services.
	enrollment := NewEnrollmentHandler(app, ca, bus)
	agentpb.RegisterAgentEnrollmentServiceServer(gs, enrollment)

	metricsStore := store.NewMetricsStore(app)
	resourceStore := store.NewResourceStore(app, bus)

	metricsHandler := NewMetricsHandler(app, metricsStore, bus)
	agentpb.RegisterAgentMetricsServiceServer(gs, metricsHandler)

	discoveryHandler := NewDiscoveryHandler(app, resourceStore, bus)
	agentpb.RegisterAgentDiscoveryServiceServer(gs, discoveryHandler)

	traefikStore := store.NewTraefikStore(app)
	traefikHandler := NewTraefikHandler(app, traefikStore, bus)
	agentpb.RegisterAgentTraefikServiceServer(gs, traefikHandler)

	hostStore := store.NewHostStore(app)
	hostHandler := NewHostHandler(app, hostStore, bus)
	agentpb.RegisterAgentHostServiceServer(gs, hostHandler)

	// Register reverse-tunnel services (agent connects to manager).
	cmdChannelHandler := NewCommandChannelHandler(registry)
	agentpb.RegisterAgentCommandChannelServiceServer(gs, cmdChannelHandler)

	dataStreamHandler := NewDataStreamHandler(registry)
	agentpb.RegisterAgentDataStreamServiceServer(gs, dataStreamHandler)

	execStreamHandler := NewExecStreamHandler(registry)
	agentpb.RegisterAgentExecStreamServiceServer(gs, execStreamHandler)

	return &Server{grpcServer: gs, addr: addr, Registry: registry}, nil
}

// Start begins listening and serving gRPC requests. This method blocks.
func (s *Server) Start() error {
	lis, err := net.Listen("tcp", s.addr)
	if err != nil {
		return fmt.Errorf("listen %s: %w", s.addr, err)
	}

	slog.Info("gRPC server listening", "addr", s.addr)
	return s.grpcServer.Serve(lis)
}

// Stop gracefully stops the gRPC server.
func (s *Server) Stop() {
	slog.Info("stopping gRPC server")
	s.grpcServer.GracefulStop()
}
