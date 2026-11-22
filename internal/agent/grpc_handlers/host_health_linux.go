//go:build linux

package grpc_handlers

import (
	"context"
	"fmt"

	"github.com/coreos/go-systemd/v22/dbus"
)

func defaultSystemdActive(ctx context.Context, unit string) (bool, error) {
	conn, err := dbus.NewSystemConnectionContext(ctx)
	if err != nil {
		return false, fmt.Errorf("systemd connection: %w", err)
	}
	defer conn.Close()

	property, err := conn.GetUnitPropertyContext(ctx, unit, "ActiveState")
	if err != nil {
		return false, err
	}
	state, _ := property.Value.Value().(string)
	switch state {
	case "active", "activating", "reloading":
		return true, nil
	default:
		return false, nil
	}
}
