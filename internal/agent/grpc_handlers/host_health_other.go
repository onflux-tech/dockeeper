//go:build !linux

package grpc_handlers

import (
	"context"
	"errors"
)

func defaultSystemdActive(context.Context, string) (bool, error) {
	return false, errors.New("systemd status requires linux")
}
