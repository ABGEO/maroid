// Package depresolver provides dependency resolution utilities.
package depresolver

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sync"

	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"
	"github.com/mymmrac/telego"
	"github.com/robfig/cron/v3"

	"github.com/abgeo/maroid/apps/hub/internal/config"
	"github.com/abgeo/maroid/apps/hub/internal/logger"
	"github.com/abgeo/maroid/apps/hub/internal/migrator"
	pluginhost "github.com/abgeo/maroid/apps/hub/internal/plugin/host"
	pluginloader "github.com/abgeo/maroid/apps/hub/internal/plugin/loader"
	"github.com/abgeo/maroid/apps/hub/internal/registry"
	"github.com/abgeo/maroid/apps/hub/internal/telegram"
	"github.com/abgeo/maroid/libs/notifier/dispatcher"
	notifierregistry "github.com/abgeo/maroid/libs/notifier/registry"
)

// Resolver defines an interface for resolving shared dependencies.
//
//nolint:interfacebloat
type Resolver interface {
	Config() *config.Config
	Logger() *slog.Logger
	HTTPRouter() *chi.Mux
	HTTPServer() (*http.Server, error)
	CloseHTTPServer() error
	Database() (*sqlx.DB, error)
	CloseDatabase() error
	Migrator() (*migrator.Migrator, error)
	PluginHost() (*pluginhost.Host, error)
	PluginLoader() (*pluginloader.Loader, error)
	CommandRegistry() (*registry.CommandRegistry, error)
	CronRegistry() (*registry.CronRegistry, error)
	MigrationRegistry() (*registry.MigrationRegistry, error)
	TelegramCommandRegistry() (*registry.TelegramCommandRegistry, error)
	Cron() *cron.Cron
	NotifierRegistry() (*notifierregistry.SchemeRegistry, error)
	NotifierDispatcher() (*dispatcher.ChannelDispatcher, error)
	TelegramBot() (*telego.Bot, error)
	TelegramUpdatesHandler() (*telegram.ChannelHandler, error)
	Close(ctx context.Context) error
}

// Container is the default implementation of the Resolver interface.
// It holds and manages application-wide dependencies.
type Container struct {
	config *config.Config
	logger *slog.Logger

	cron struct {
		once     sync.Once
		instance *cron.Cron
	}

	database struct {
		mu       sync.Mutex
		once     sync.Once
		instance *sqlx.DB
	}

	migrator struct {
		mu       sync.Mutex
		once     sync.Once
		instance *migrator.Migrator
	}

	httpRouter struct {
		once     sync.Once
		instance *chi.Mux
	}

	httpServer struct {
		mu       sync.Mutex
		once     sync.Once
		instance *http.Server
	}

	pluginHost struct {
		mu       sync.Mutex
		once     sync.Once
		instance *pluginhost.Host
	}

	pluginLoader struct {
		mu       sync.Mutex
		once     sync.Once
		instance *pluginloader.Loader
	}

	commandRegistry struct {
		mu       sync.Mutex
		once     sync.Once
		instance *registry.CommandRegistry
	}

	cronRegistry struct {
		mu       sync.Mutex
		once     sync.Once
		instance *registry.CronRegistry
	}

	migrationRegistry struct {
		mu       sync.Mutex
		once     sync.Once
		instance *registry.MigrationRegistry
	}

	telegramCommandRegistry struct {
		mu       sync.Mutex
		once     sync.Once
		instance *registry.TelegramCommandRegistry
	}

	notifierRegistry struct {
		mu       sync.Mutex
		once     sync.Once
		instance *notifierregistry.SchemeRegistry
	}

	notifierDispatcher struct {
		mu       sync.Mutex
		once     sync.Once
		instance *dispatcher.ChannelDispatcher
	}

	telegramBot struct {
		mu       sync.Mutex
		once     sync.Once
		instance *telego.Bot
	}

	telegramUpdatesHandler struct {
		mu       sync.Mutex
		once     sync.Once
		instance *telegram.ChannelHandler
	}
}

var _ Resolver = (*Container)(nil)

// NewResolver creates a new dependency container initialized with the
// given configuration file.
func NewResolver() (*Container, error) {
	cfg, err := config.New("")
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	loggerInstance, err := logger.New(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize logger: %w", err)
	}

	return &Container{
		config: cfg,
		logger: loggerInstance,
	}, nil
}

// Config returns the loaded application configuration.
func (c *Container) Config() *config.Config {
	return c.config
}

// Logger returns the initialized slog logger instance.
func (c *Container) Logger() *slog.Logger {
	return c.logger
}

// Close gracefully shuts down managed dependencies.
func (c *Container) Close(_ context.Context) error {
	var errList []error

	errList = append(errList,
		c.CloseHTTPServer(),
		c.CloseDatabase(),
	)

	return errors.Join(errList...)
}
