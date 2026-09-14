// Package pluginconfig provides structures and utilities for managing plugin configurations.
package pluginconfig

import (
	"fmt"

	"github.com/go-playground/validator/v10"
	"github.com/go-viper/mapstructure/v2"
	"github.com/mcuadros/go-defaults"
)

const (
	// configTag names a field of the installation configuration. See CFG-006.
	configTag = "mapstructure"
	// settingsTag names a field of the settings of one user. See PSET-FR-012.
	settingsTag = "json"
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
	return decodeAndValidate(configTag, "config", cfg, pluginConfig)
}

// DecodeAndValidateSettings decodes the settings that one user stored into a strongly
// typed struct, applies the default values, and validates it.
// Returns an error if decoding or validation fails.
func DecodeAndValidateSettings(values map[string]any, target any) error {
	return decodeAndValidate(settingsTag, "settings", values, target)
}

// decodeAndValidate applies the defaults, decodes the source with the given tag, and
// validates the result.
func decodeAndValidate(tag string, subject string, source any, target any) error {
	defaults.SetDefaults(target)

	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		TagName: tag,
		Result:  target,
	})
	if err != nil {
		return fmt.Errorf("building the %s decoder: %w", subject, err)
	}

	if err = decoder.Decode(source); err != nil {
		return fmt.Errorf("decoding the %s: %w", subject, err)
	}

	if err = validator.New().Struct(target); err != nil {
		return fmt.Errorf("validating the %s: %w", subject, err)
	}

	return nil
}
