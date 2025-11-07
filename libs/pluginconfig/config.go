// Package pluginconfig provides structures and utilities for managing plugin configurations.
package pluginconfig

import (
	"fmt"

	"github.com/go-playground/validator/v10"
	"github.com/go-viper/mapstructure/v2"
	"github.com/mcuadros/go-defaults"
)

// Config represents the basic configuration for a plugin.
type Config struct {
	Path    string `validate:"required,filepath"`
	Enabled bool   `default:"true"`
	Config  map[string]any
}

// DecodeAndValidateConfig decodes a generic configuration into a strongly typed
// plugin configuration struct, applies default values, and validates it.
// Returns an error if decoding or validation fails.
func DecodeAndValidateConfig(cfg any, pluginConfig any) error {
	validate := validator.New()

	defaults.SetDefaults(pluginConfig)

	if err := mapstructure.Decode(cfg, &pluginConfig); err != nil {
		return fmt.Errorf("failed to decode config: %w", err)
	}

	if err := validate.Struct(pluginConfig); err != nil {
		return fmt.Errorf("failed to validate config: %w", err)
	}

	return nil
}
