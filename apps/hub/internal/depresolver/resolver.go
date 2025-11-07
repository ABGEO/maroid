// Package depresolver provides dependency resolution utilities.
package depresolver

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"

	"github.com/jmoiron/sqlx"
	"github.com/robfig/cron/v3"

	"github.com/abgeo/maroid/apps/hub/internal/config"
	"github.com/abgeo/maroid/apps/hub/internal/logger"
)

// Resolver defines an interface for resolving shared dependencies.
type Resolver interface {
	Config() *config.Config
	Logger() *slog.Logger
	Database() (*sqlx.DB, error)
	CloseDatabase() error
	Cron() *cron.Cron
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
		c.CloseDatabase(),
	)

	return errors.Join(errList...)
}
