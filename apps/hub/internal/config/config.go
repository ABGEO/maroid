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

type CORS struct {
	Enabled          bool          `default:"false" mapstructure:"enabled"           validate:"boolean"`
	AllowOrigins     []string      `default:"[*]"   mapstructure:"allow_origins"`
	AllowMethods     []string      `default:"[*]"   mapstructure:"allow_methods"`
	AllowHeaders     []string      `default:"[*]"   mapstructure:"allow_headers"`
	ExposeHeaders    []string      `default:"[*]"   mapstructure:"expose_headers"`
	AllowCredentials bool          `default:"false" mapstructure:"allow_credentials"`
	MaxAge           time.Duration `default:"12h"   mapstructure:"max_age"`
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

// Auth defines general authentication configuration parameters.
type Auth struct {
	AllowedRedirects []string `mapstructure:"allowed_redirects" validate:"required,min=1,dive,url"`
}

// JWT defines JWT authentication configuration parameters.
type JWT struct {
	Issuer      string        `default:"https://hub.maroid.dev" mapstructure:"issuer"       validate:"required,url"`
	PrivateKey  string        `                                 mapstructure:"private_key"  validate:"required"`
	PublicKey   string        `                                 mapstructure:"public_key"   validate:"required"`
	TokenExpiry time.Duration `default:"168h"                   mapstructure:"token_expiry"`
}

// OIDC defines OpenID Connect configuration parameters for authentication.
type OIDC struct {
	Issuer       string `default:"https://oauth.telegram.org" mapstructure:"issuer"        validate:"required,url"`
	ClientID     string `                                     mapstructure:"client_id"     validate:"required"`
	ClientSecret string `                                     mapstructure:"client_secret" validate:"required"`
	RedirectURI  string `                                     mapstructure:"redirect_uri"  validate:"required"`
}

// MQTT defines MQTT broker configuration parameters.
// All fields are optional; the broker is only required when MQTT subscriber plugins are loaded.
type MQTT struct {
	Broker            string
	User              string
	Password          string
	ClientIDPrefix    string        `default:"maroid" mapstructure:"client_id_prefix"`
	SharedGroup       string        `default:"maroid" mapstructure:"shared_group"`
	ConnectTimeout    time.Duration `default:"5s"     mapstructure:"connect_timeout"`
	DisconnectQuiesce uint          `default:"250"    mapstructure:"disconnect_quiesce"`
}

// Telegram defines Telegram integration configuration parameters.
type Telegram struct {
	Token        string  `validate:"required"`
	Debug        bool    `                    default:"false"`
	Setup        bool    `                    default:"true"`
	AllowedUsers []int64 `validate:"required"                 mapstructure:"allowed_users"`
	Webhook      struct {
		Path            string   `default:"/telegram/webhook"`
		AllowedNetworks []string `mapstructure:"allowed_networks" validate:"required"`
	}
}

// Config represents the main application configuration.
type Config struct {
	Env string `default:"prod" validate:"oneof=dev prod"`

	Logger   Logger
	Database Database
	Server   Server
	CORS     CORS
	JWT      JWT
	Auth     Auth
	OIDC     OIDC
	MQTT     MQTT
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
			return nil, fmt.Errorf("reading config: %w", err)
		}
	}

	if err = viperInstance.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("unmarshaling config: %w", err)
	}

	validate := validator.New()
	if err = validate.Struct(cfg); err != nil {
		return nil, fmt.Errorf("validating config: %w", err)
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
		return fmt.Errorf("detecting home directory: %w", err)
	}

	viperInstance.AddConfigPath(".")
	viperInstance.AddConfigPath(home + "/.maroid")
	viperInstance.SetConfigName("config")
	viperInstance.SetConfigType("yaml")

	return nil
}
