//go:build !linux

package collector

import (
	"context"
	"errors"
)

func discoverPM2Daemons() ([]pm2Daemon, error) {
	return nil, nil
}

func runPM2Jlist(ctx context.Context, daemon pm2Daemon) ([]byte, error) {
	return nil, errors.New("pm2 discovery requires linux")
}
