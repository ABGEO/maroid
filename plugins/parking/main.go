// Plugin for working with Tbilisi parking service. It allows users to start and stop parking sessions,
// check their balance and parking status, and receive notifications about their parking sessions.
package main

import (
	"fmt"
	"io/fs"
	"log/slog"

	"github.com/abgeo/maroid/libs/pluginapi"
	telegramconversationapi "github.com/abgeo/maroid/libs/pluginapi/telegram/conversation"
	"github.com/abgeo/maroid/libs/pluginconfig"
	"github.com/abgeo/maroid/plugins/parking/config"
	"github.com/abgeo/maroid/plugins/parking/handler"
	"github.com/abgeo/maroid/plugins/parking/permission"
	"github.com/abgeo/maroid/plugins/parking/service"
	"github.com/abgeo/maroid/plugins/parking/telegram/command"
	"github.com/abgeo/maroid/plugins/parking/telegram/conversation"
	"github.com/abgeo/maroid/plugins/parking/widget"
)

// Meta names the plugin. The hub reads it before it calls New.
//
//nolint:gochecknoglobals
var Meta = pluginapi.Metadata{
	ID:          pluginapi.ParsePluginID("dev.maroid.parking"),
	Name:        "Tbilisi Parking",
	Description: "Start and stop parking sessions, and check the balance.",
	Version:     "0.1.0",
	APIVersion:  pluginapi.APIVersion,
}

type ParkingPlugin struct {
	config                     *config.Config
	logger                     *slog.Logger
	telegramBot                pluginapi.TelegramBot
	telegramConversationEngine telegramconversationapi.Engine
	apiClientSvc               service.APIClientService
	connections                service.ConnectionService
	answers                    *command.Notice
}

var (
	_ pluginapi.ConfigurablePlugin         = (*ParkingPlugin)(nil)
	_ pluginapi.TelegramCommandPlugin      = (*ParkingPlugin)(nil)
	_ pluginapi.TelegramConversationPlugin = (*ParkingPlugin)(nil)
	_ pluginapi.PermissionPlugin           = (*ParkingPlugin)(nil)
	_ pluginapi.RoutePlugin                = (*ParkingPlugin)(nil)
	_ pluginapi.UIPlugin                   = (*ParkingPlugin)(nil)
	_ pluginapi.WidgetPlugin               = (*ParkingPlugin)(nil)
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

	settings, err := host.Settings()
	if err != nil {
		return nil, fmt.Errorf("getting host settings: %w", err)
	}

	pages, err := host.Pages()
	if err != nil {
		return nil, fmt.Errorf("getting host page linker: %w", err)
	}

	secrets, err := host.Secrets()
	if err != nil {
		return nil, fmt.Errorf("getting host secrets: %w", err)
	}

	apiClient := service.NewAPIClient(pluginConfig, secrets)

	return &ParkingPlugin{
		config:                     pluginConfig,
		logger:                     host.Logger(),
		telegramBot:                telegramBot,
		telegramConversationEngine: host.TelegramConversationEngine(),
		apiClientSvc:               apiClient,
		connections:                service.NewConnections(apiClient, settings, secrets),
		answers:                    command.NewNotice(pages),
	}, nil
}

func (p *ParkingPlugin) SettingsModel() (any, error) {
	return config.UserSettings{}, nil
}

func (p *ParkingPlugin) Permissions() ([]pluginapi.Permission, error) {
	return []pluginapi.Permission{
		{
			Name:        permission.SessionsRead,
			Description: "Read the active session and the balance",
			Lowest:      pluginapi.RoleViewer,
		},
		{
			Name:        permission.SessionsWrite,
			Description: "Start and stop a parking session",
			Lowest:      pluginapi.RoleEditor,
		},
		{
			Name:        permission.AccountConnect,
			Description: "Read, connect, and disconnect your own parking account",
			Lowest:      pluginapi.RoleViewer,
		},
	}, nil
}

// UIManifest returns the account page of the remote and its assets.
func (p *ParkingPlugin) UIManifest() (*pluginapi.UIManifest, error) {
	assets, err := fs.Sub(uiAssets, "ui/dist")
	if err != nil {
		return nil, fmt.Errorf("getting UI manifest FS: %w", err)
	}

	return &pluginapi.UIManifest{
		Routes: []pluginapi.UIRoute{
			{Path: "/account", Label: "Account"},
		},
		Assets: assets,
	}, nil
}

// Widgets returns the balance of the parking account, for the home page.
func (p *ParkingPlugin) Widgets() ([]pluginapi.Widget, error) {
	return []pluginapi.Widget{widget.NewBalance(p.connections)}, nil
}

// Routes returns the HTTP routes of the connection page.
func (p *ParkingPlugin) Routes() ([]pluginapi.Route, error) {
	return handler.NewConnectionHandler(p.logger, p.connections).Routes(), nil
}

func (p *ParkingPlugin) TelegramCommands() ([]pluginapi.TelegramCommand, error) {
	return []pluginapi.TelegramCommand{
		command.NewParking(p.telegramConversationEngine, p.apiClientSvc, p.answers),
		command.NewBalance(p.apiClientSvc, p.answers),
		command.NewStatus(p.apiClientSvc, p.answers),
		command.NewStop(p.apiClientSvc, p.answers),
	}, nil
}

func (p *ParkingPlugin) TelegramConversations() ([]telegramconversationapi.Conversation, error) {
	return []telegramconversationapi.Conversation{
		conversation.NewParkingConversation(p.telegramBot, p.apiClientSvc, p.answers),
	}, nil
}
