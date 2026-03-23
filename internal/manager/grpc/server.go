package grpc

import (
	"fmt"
	"log/slog"
	"net"

	"github.com/pocketbase/pocketbase"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/jkdesenvolvedor/dockeeper/internal/manager/services"
	agentpb "github.com/jkdesenvolvedor/dockeeper/proto/gen/dockeeper/agent/v1"
)

// Server wraps the gRPC server for agent communication.
type Server struct {
	grpcServer *grpc.Server
	addr       string
}

// NewServer creates a configured gRPC server with TLS, interceptors and
// the enrollment service registered.
func NewServer(app *pocketbase.PocketBase, ca *services.CertAuthority, addr string) (*Server, error) {
	tlsCfg, err := ca.ServerTLSConfig()
	if err != nil {
		return nil, fmt.Errorf("server TLS config: %w", err)
	}

	gs := grpc.NewServer(
		grpc.Creds(credentials.NewTLS(tlsCfg)),
		grpc.ChainUnaryInterceptor(ChainUnaryInterceptors(tlsCfg)...),
		grpc.ChainStreamInterceptor(ChainStreamInterceptors(tlsCfg)...),
	)

	// Register services.
	enrollment := NewEnrollmentHandler(app, ca)
	agentpb.RegisterAgentEnrollmentServiceServer(gs, enrollment)

	return &Server{grpcServer: gs, addr: addr}, nil
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
