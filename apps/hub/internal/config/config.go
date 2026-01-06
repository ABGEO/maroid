// Package config provides functionality for loading, validating, and managing
// configuration settings for the Maroid application. It supports environment
// variables, YAML files, and default values.
package config

import (
	"errors"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/mcuadros/go-defaults"
	"github.com/spf13/viper"

	"github.com/abgeo/maroid/libs/notifier"
	"github.com/abgeo/maroid/libs/pluginconfig"
)

// @todo: move config objects to packages.

// Logger defines logging configuration parameters.
type Logger struct {
	Level  string `default:"info" validate:"oneof=debug info warn error"`
	Format string `default:"json" validate:"oneof=text json"`
}

// Database defines database connection parameters.
type Database struct {
	Host     string `validate:"required"`
	Port     string `validate:"min=1,max=65535"`
	User     string `validate:"required"`
	Password string `validate:"required"`
	Database string `validate:"required"`
}

// DSN builds and returns the connection string for the database.
func (c *Database) DSN() string {
	return fmt.Sprintf(
		"pgx://%s:%s@%s/%s",
		c.User,
		c.Password,
		net.JoinHostPort(c.Host, c.Port),
		c.Database,
	)
}

// Server defines HTTP server configuration parameters.
type Server struct {
	Hostname   string `validate:"fqdn"`
	ListenAddr string `validate:"ip"              default:"0.0.0.0" mapstructure:"address"`
	Port       string `validate:"min=1,max=65535" default:"8000"`

	ReadTimeout       time.Duration `default:"15s"  mapstructure:"read_timeout"`
	ReadHeaderTimeout time.Duration `default:"5s"   mapstructure:"read_header_timeout"`
	WriteTimeout      time.Duration `default:"15s"  mapstructure:"write_timeout"`
	IdleTimeout       time.Duration `default:"120s" mapstructure:"idle_timeout"`
}

// Address returns the full server address in host:port format.
func (c *Server) Address() string {
	return net.JoinHostPort(c.ListenAddr, c.Port)
}

// Telegram defines Telegram integration configuration parameters.
type Telegram struct {
	Token   string `validate:"required"`
	Debug   bool   `                    default:"false"`
	Setup   bool   `                    default:"true"`
	Webhook struct {
		Path            string   `default:"/telegram/webhook"`
		AllowedNetworks []string `mapstructure:"allowed_networks" validate:"required"`
		AllowedUsers    []int64  `mapstructure:"allowed_users"    validate:"required"`
	}
}

// Config represents the main application configuration.
type Config struct {
	Env string `default:"prod" validate:"oneof=dev prod"`

	Logger   Logger
	Database Database
	Server   Server
	Telegram Telegram
	Notifier notifier.Config
	Plugins  []pluginconfig.Config
}

// New loads configuration from the given file path or environment variables.
// It applies default values, validates the result, and returns a fully
// initialized Config instance.
func New(cfgFile string) (*Config, error) {
	cfg := new(Config)
	viperInstance := viper.NewWithOptions(
		viper.EnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_")),
	)

	defaults.SetDefaults(cfg)

	err := setConfigFile(cfgFile, viperInstance)
	if err != nil {
		return nil, err
	}

	viperInstance.SetEnvPrefix("MAROID")
	viperInstance.AutomaticEnv()

	if err = viperInstance.ReadInConfig(); err != nil {
		var configFileNotFoundError viper.ConfigFileNotFoundError
		if !errors.As(err, &configFileNotFoundError) {
			return nil, fmt.Errorf("failed to read config: %w", err)
		}
	}

	if err = viperInstance.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	validate := validator.New()
	if err = validate.Struct(cfg); err != nil {
		return nil, fmt.Errorf("failed to validate config: %w", err)
	}

	return cfg, nil
}

func setConfigFile(cfgFile string, viperInstance *viper.Viper) error {
	if cfgFile != "" {
		viperInstance.SetConfigFile(cfgFile)

		return nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to detect home directory: %w", err)
	}

	viperInstance.AddConfigPath(".")
	viperInstance.AddConfigPath(home + "/.maroid")
	viperInstance.SetConfigName("config")
	viperInstance.SetConfigType("yaml")

	return nil
}
