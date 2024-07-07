package app

import (
	"os"
	"testing"
	"time"

	"github.com/FedeBP/pumoide/backend/internal/app"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadConfig(t *testing.T) {
	err := os.Setenv("CLIENT_TIMEOUT", "10s")
	require.NoError(t, err, "Failed to set CLIENT_TIMEOUT environment variable")

	err = os.Setenv("DIAL_TIMEOUT", "5s")
	require.NoError(t, err, "Failed to set DIAL_TIMEOUT environment variable")

	err = os.Setenv("MAX_IDLE_CONNS", "200")
	require.NoError(t, err, "Failed to set MAX_IDLE_CONNS environment variable")

	err = os.Setenv("DISABLE_COMPRESSION", "true")
	require.NoError(t, err, "Failed to set DISABLE_COMPRESSION environment variable")

	config := app.LoadConfig()

	assert.Equal(t, 10*time.Second, config.ClientTimeout)
	assert.Equal(t, 5*time.Second, config.DialTimeout)
	assert.Equal(t, 200, config.MaxIdleConns)
	assert.True(t, config.DisableCompression)

	assert.Equal(t, 30*time.Second, config.KeepAlive)
	assert.Equal(t, 90*time.Second, config.IdleConnTimeout)
	assert.Equal(t, 10*time.Second, config.TLSHandshakeTimeout)
	assert.Equal(t, 1*time.Second, config.ExpectContinueTimeout)
	assert.False(t, config.DisableKeepAlives)

	err = os.Unsetenv("CLIENT_TIMEOUT")
	require.NoError(t, err, "Failed to unset CLIENT_TIMEOUT environment variable")

	err = os.Unsetenv("DIAL_TIMEOUT")
	require.NoError(t, err, "Failed to unset DIAL_TIMEOUT environment variable")

	err = os.Unsetenv("MAX_IDLE_CONNS")
	require.NoError(t, err, "Failed to unset MAX_IDLE_CONNS environment variable")

	err = os.Unsetenv("DISABLE_COMPRESSION")
	require.NoError(t, err, "Failed to unset DISABLE_COMPRESSION environment variable")
}

func TestLoadConfig_DefaultValues(t *testing.T) {
	envVars := []string{"CLIENT_TIMEOUT", "DIAL_TIMEOUT", "MAX_IDLE_CONNS", "DISABLE_COMPRESSION"}

	for _, env := range envVars {
		err := os.Unsetenv(env)
		require.NoError(t, err, "Failed to unset %s environment variable", env)
	}

	config := app.LoadConfig()

	assert.Equal(t, time.Duration(0), config.ClientTimeout)
	assert.Equal(t, 30*time.Second, config.DialTimeout)
	assert.Equal(t, 100, config.MaxIdleConns)
	assert.False(t, config.DisableCompression)
}
