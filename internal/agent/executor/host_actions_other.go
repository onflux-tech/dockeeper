//go:build !linux

package executor

import (
	"context"
	"errors"
)

func defaultHostActionRunners() (SystemdRunner, PM2Runner) {
	systemd := func(context.Context, []string) (int, string, error) {
		return 1, "", errors.New("host app actions require linux")
	}
	pm2 := func(context.Context, []string, string) (int, string, error) {
		return 1, "", errors.New("host app actions require linux")
	}
	return systemd, pm2
}
