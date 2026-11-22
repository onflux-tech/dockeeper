package routes

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestProxyRouteInstanceFilter(t *testing.T) {
	filter, params := proxyRouteInstanceFilter([]string{"inst1", "inst2"})
	require.Equal(t, "(instance = {:instance0} || instance = {:instance1})", filter)
	require.Equal(t, map[string]any{"instance0": "inst1", "instance1": "inst2"}, params)
}

func TestProxyRouteInstanceFilterSingle(t *testing.T) {
	filter, params := proxyRouteInstanceFilter([]string{"inst1"})
	require.Equal(t, "(instance = {:instance0})", filter)
	require.Equal(t, map[string]any{"instance0": "inst1"}, params)
}

func TestProxyCountFilterTranslatesOperators(t *testing.T) {
	filter := proxyCountFilter("server = {:server} && (instance = {:instance0} || instance = {:instance1})")
	require.Equal(t, "server = {:server} AND (instance = {:instance0} OR instance = {:instance1})", filter)
}
