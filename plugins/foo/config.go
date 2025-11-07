package main

type Config struct {
	APIKey string `mapstructure:"api_key" validate:"required"`
}
