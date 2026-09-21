// Plugin for working with Tbilisi parking service. It allows users to start and stop parking sessions,
// check their balance and parking status, and receive notifications about their parking sessions.
package main

import (
	"fmt"
	"log/slog"

	"github.com/abgeo/maroid/libs/pluginapi"
	telegramconversationapi "github.com/abgeo/maroid/libs/pluginapi/telegram/conversation"
	"github.com/abgeo/maroid/libs/pluginconfig"
	"github.com/abgeo/maroid/plugins/parking/config"
	"github.com/abgeo/maroid/plugins/parking/service"
	"github.com/abgeo/maroid/plugins/parking/telegram/command"
	"github.com/abgeo/maroid/plugins/parking/telegram/conversation"
)

// pluginID names the plugin. The constructor needs it before Meta exists.
//
//nolint:gochecknoglobals
var pluginID = pluginapi.ParsePluginID("dev.maroid.parking")

type ParkingPlugin struct {
	config                     *config.Config
	logger                     *slog.Logger
	telegramBot                pluginapi.TelegramBot
	telegramConversationEngine telegramconversationapi.Engine
	apiClientSvc               service.APIClientService
}

var (
	_ pluginapi.Plugin                     = (*ParkingPlugin)(nil)
	_ pluginapi.ConfigurablePlugin         = (*ParkingPlugin)(nil)
	_ pluginapi.TelegramCommandPlugin      = (*ParkingPlugin)(nil)
	_ pluginapi.TelegramConversationPlugin = (*ParkingPlugin)(nil)
)

// New creates a plugin instance.
//
//nolint:gochecknoglobals
var New pluginapi.Constructor = func(host pluginapi.Host, cfg map[string]any) (pluginapi.Plugin, error) {
	pluginConfig := new(config.Config)
	if err := pluginconfig.DecodeAndValidateConfig(cfg, pluginConfig); err != nil {
		return nil, fmt.Errorf("validating config: %w", err)
	}

	telegramBot, err := host.TelegramBot()
	if err != nil {
		return nil, fmt.Errorf("getting host telegram bot instance: %w", err)
	}

	settingsProvider, err := host.Settings()
	if err != nil {
		return nil, fmt.Errorf("getting host settings provider: %w", err)
	}

	settings := pluginapi.NewPluginSettings(settingsProvider, pluginID)

	plg := &ParkingPlugin{
		config:                     pluginConfig,
		telegramBot:                telegramBot,
		telegramConversationEngine: host.TelegramConversationEngine(),
		apiClientSvc:               service.NewAPIClient(pluginConfig, settings),
	}

	plg.logger = host.Logger().With(
		slog.String("plugin", plg.Meta().ID.String()),
		slog.String("plugin_version", plg.Meta().Version),
		slog.String("plugin_api_version", plg.Meta().APIVersion),
	)

	return plg, nil
}

func (p *ParkingPlugin) Meta() pluginapi.Metadata {
	return pluginapi.Metadata{
		ID:         pluginID,
		Version:    "0.1.0",
		APIVersion: pluginapi.APIVersion,
	}
}

func (p *ParkingPlugin) SettingsModel() (any, error) {
	return config.UserSettings{}, nil
}

func (p *ParkingPlugin) TelegramCommands() ([]pluginapi.TelegramCommand, error) {
	return []pluginapi.TelegramCommand{
		command.NewParking(p.telegramConversationEngine, p.apiClientSvc),
		command.NewBalance(p.apiClientSvc),
		command.NewStatus(p.apiClientSvc),
		command.NewStop(p.apiClientSvc),
	}, nil
}

func (p *ParkingPlugin) TelegramConversations() ([]telegramconversationapi.Conversation, error) {
	return []telegramconversationapi.Conversation{
		conversation.NewParkingConversation(p.telegramBot, p.apiClientSvc),
	}, nil
}
