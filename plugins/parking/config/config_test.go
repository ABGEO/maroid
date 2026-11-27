package config_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/libs/pluginconfig"
	"github.com/abgeo/maroid/plugins/parking/config"
)

// PARKMAP-SC-022: The map source has a default, and the time limit stays within 4 seconds.
func TestMapConfigDefaultsAndLimits(t *testing.T) {
	t.Parallel()

	cfg := new(config.Config)
	require.NoError(t, pluginconfig.DecodeAndValidateConfig(map[string]any{}, cfg))

	assert.Equal(t, "https://tile.openstreetmap.org/{z}/{x}/{y}.png", cfg.Map.TileURL)
	assert.Equal(t, "© OpenStreetMap contributors", cfg.Map.Attribution)
	assert.Equal(t, 19, cfg.Map.MaxZoom)
	assert.Equal(t, 3*time.Second, cfg.Map.Timeout)

	slow := new(config.Config)
	err := pluginconfig.DecodeAndValidateConfig(
		map[string]any{"map": map[string]any{"timeout": "5s"}},
		slow,
	)
	require.Error(t, err)
}
