package collector

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPM2EnvDoesNotLeakAgentEnvironment(t *testing.T) {
	t.Setenv("DK_SECRET", "super-secret")

	env := pm2Env(pm2Daemon{
		User:    "deploy",
		Home:    "/home/deploy",
		PM2Home: "/home/deploy/.pm2",
	})

	require.NotContains(t, env, "DK_SECRET=super-secret")
	require.Contains(t, env, "HOME=/home/deploy")
	require.Contains(t, env, "PM2_HOME=/home/deploy/.pm2")
	require.Contains(t, env, "USER=deploy")
}
