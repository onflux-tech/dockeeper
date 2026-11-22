//go:build !linux

package collector

import "context"

func defaultGatewayProbe(context.Context) (bool, uint32, string) {
	return false, 0, "none"
}
