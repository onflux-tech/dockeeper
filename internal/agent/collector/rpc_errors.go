package collector

import (
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// IsAuthError returns true when the gRPC error indicates a TLS/mTLS
// authentication failure (e.g. CA changed, certificate expired or revoked).
// This is used to detect stale enrollment and trigger automatic re-enrollment.
func IsAuthError(err error) bool {
	if err == nil {
		return false
	}

	code := status.Code(err)

	// Explicit auth rejection from the manager.
	if code == codes.Unauthenticated {
		return true
	}

	// Transport-level TLS failures surface as Unavailable with descriptive messages.
	if code == codes.Unavailable || code == codes.Internal {
		msg := strings.ToLower(err.Error())
		for _, keyword := range []string{
			"certificate",
			"x509",
			"tls:",
			"tls handshake",
			"authentication handshake",
			"bad certificate",
			"unknown authority",
			"cert",
		} {
			if strings.Contains(msg, keyword) {
				return true
			}
		}
	}

	return false
}
