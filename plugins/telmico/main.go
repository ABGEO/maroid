// Plugin for working with https://my.telmico.ge/
package main

import (
	"fmt"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/libs/pluginconfig"
	"github.com/abgeo/maroid/plugins/telmico/config"
)

// pluginID names the plugin. The constructor needs it before Meta exists.
//
//nolint:gochecknoglobals
var pluginID = pluginapi.ParsePluginID("dev.maroid.telmico")

// TelmicoPlugin connects the electricity account of a workspace, collects its
// transactions and its bills, and sends the bill.
type TelmicoPlugin struct {
	config *config.Config
}

var (
	_ pluginapi.Plugin             = (*TelmicoPlugin)(nil)
	_ pluginapi.ConfigurablePlugin = (*TelmicoPlugin)(nil)
)

// New creates a plugin instance.
//
//nolint:gochecknoglobals
var New pluginapi.Constructor = func(_ pluginapi.Host, cfg map[string]any) (pluginapi.Plugin, error) {
	pluginConfig := new(config.Config)
	if err := pluginconfig.DecodeAndValidateConfig(cfg, pluginConfig); err != nil {
		return nil, fmt.Errorf("validating config: %w", err)
	}

	return &TelmicoPlugin{config: pluginConfig}, nil
}

// Meta returns the plugin metadata.
func (p *TelmicoPlugin) Meta() pluginapi.Metadata {
	return pluginapi.Metadata{
		ID:          pluginID,
		Name:        "Telmico",
		Description: "Electricity bills and usage from Telmico.",
		Version:     "0.1.0",
		APIVersion:  pluginapi.APIVersion,
	}
}

// SettingsModel declares the Telmico account that a workspace stores.
func (p *TelmicoPlugin) SettingsModel() (any, error) {
	return config.WorkspaceSettings{}, nil
}
