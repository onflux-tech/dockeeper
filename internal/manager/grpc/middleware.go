package grpc

import (
	"context"
	"crypto/tls"
	"log/slog"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

type contextKey string

const serverIDKey contextKey = "server_id"

// enrollmentFullMethod is the gRPC full method name for the Enroll RPC,
// which does not require mTLS client certificates.
const enrollmentFullMethod = "/dockeeper.agent.v1.AgentEnrollmentService/Enroll"

// ServerIDFromContext extracts the server_id set by the mTLS auth interceptor.
func ServerIDFromContext(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(serverIDKey).(string)
	return v, ok && v != ""
}

// LoggingUnaryInterceptor logs method, duration and status for every unary RPC.
func LoggingUnaryInterceptor(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
	start := time.Now()
	resp, err := handler(ctx, req)
	duration := time.Since(start)

	code := codes.OK
	if err != nil {
		code = status.Code(err)
	}

	slog.Info("grpc unary call",
		"method", info.FullMethod,
		"code", code.String(),
		"duration", duration,
	)
	return resp, err
}

// LoggingStreamInterceptor logs method and duration for streaming RPCs.
func LoggingStreamInterceptor(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	start := time.Now()
	err := handler(srv, ss)
	duration := time.Since(start)

	code := codes.OK
	if err != nil {
		code = status.Code(err)
	}

	slog.Info("grpc stream call",
		"method", info.FullMethod,
		"code", code.String(),
		"duration", duration,
	)
	return err
}

// RecoveryUnaryInterceptor catches panics in unary handlers and returns Internal.
func RecoveryUnaryInterceptor(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("grpc panic recovered", "method", info.FullMethod, "panic", r)
			err = status.Errorf(codes.Internal, "internal error")
		}
	}()
	return handler(ctx, req)
}

// RecoveryStreamInterceptor catches panics in stream handlers and returns Internal.
func RecoveryStreamInterceptor(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) (err error) {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("grpc panic recovered", "method", info.FullMethod, "panic", r)
			err = status.Errorf(codes.Internal, "internal error")
		}
	}()
	return handler(srv, ss)
}

// MTLSUnaryInterceptor extracts the server_id (CN) from the client's mTLS cert
// and stores it in the context. Enroll RPCs are exempt.
func MTLSUnaryInterceptor(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
	if info.FullMethod == enrollmentFullMethod {
		return handler(ctx, req)
	}

	serverID, err := extractServerID(ctx)
	if err != nil {
		return nil, err
	}

	ctx = context.WithValue(ctx, serverIDKey, serverID)
	return handler(ctx, req)
}

// MTLSStreamInterceptor extracts the server_id from the client's mTLS cert
// for streaming RPCs.
func MTLSStreamInterceptor(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	serverID, err := extractServerID(ss.Context())
	if err != nil {
		return err
	}

	wrapped := &serverStreamWithContext{
		ServerStream: ss,
		ctx:          context.WithValue(ss.Context(), serverIDKey, serverID),
	}
	return handler(srv, wrapped)
}

func extractServerID(ctx context.Context) (string, error) {
	p, ok := peer.FromContext(ctx)
	if !ok {
		return "", status.Error(codes.Unauthenticated, "no peer info")
	}

	tlsInfo, ok := p.AuthInfo.(credentials.TLSInfo)
	if !ok {
		return "", status.Error(codes.Unauthenticated, "no TLS info")
	}

	if len(tlsInfo.State.VerifiedChains) == 0 || len(tlsInfo.State.VerifiedChains[0]) == 0 {
		return "", status.Error(codes.Unauthenticated, "no client certificate")
	}

	cn := tlsInfo.State.VerifiedChains[0][0].Subject.CommonName
	if cn == "" {
		return "", status.Error(codes.Unauthenticated, "empty CN in client certificate")
	}

	return cn, nil
}

// serverStreamWithContext wraps a grpc.ServerStream to override Context().
type serverStreamWithContext struct {
	grpc.ServerStream
	ctx context.Context
}

func (s *serverStreamWithContext) Context() context.Context {
	return s.ctx
}

// ChainUnaryInterceptors chains unary interceptors with TLS check before auth.
func ChainUnaryInterceptors(tlsCfg *tls.Config) []grpc.UnaryServerInterceptor {
	_ = tlsCfg // reserved for future use
	return []grpc.UnaryServerInterceptor{
		RecoveryUnaryInterceptor,
		LoggingUnaryInterceptor,
		MTLSUnaryInterceptor,
	}
}

// ChainStreamInterceptors chains stream interceptors.
func ChainStreamInterceptors(tlsCfg *tls.Config) []grpc.StreamServerInterceptor {
	_ = tlsCfg
	return []grpc.StreamServerInterceptor{
		RecoveryStreamInterceptor,
		LoggingStreamInterceptor,
		MTLSStreamInterceptor,
	}
}
