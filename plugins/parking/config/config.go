// Package config defines the configuration schema for the plugin.
package config

import "time"

// CronSchedule holds the schedule of each cron job of the plugin.
type CronSchedule struct {
	HistoryCollector string `default:"15 * * * *" mapstructure:"history_collector" validate:"cron"`
}

// MapConfig holds the map source that draws the parking lots near a member.
//
//nolint:lll // the tags of one field share one line.
type MapConfig struct {
	TileURL     string        `default:"https://tile.openstreetmap.org/{z}/{x}/{y}.png" mapstructure:"tile_url"    validate:"required,url,contains={z},contains={x},contains={y}"`
	Attribution string        `default:"© OpenStreetMap contributors"                   mapstructure:"attribution" validate:"required"`
	MaxZoom     int           `default:"19"                                             mapstructure:"max_zoom"    validate:"min=15,max=22"`
	Timeout     time.Duration `default:"3s"                                             mapstructure:"timeout"     validate:"gt=0,lte=4s"`
}

// Config represents the root plugin configuration.
//
//nolint:lll // the tags of one field share one line.
type Config struct {
	BaseURL        string        `default:"https://api.municipal.gov.ge" mapstructure:"base_url"`
	RequestTimeout time.Duration `default:"15s"                          mapstructure:"request_timeout" validate:"gt=0"`
	CollectorPages int           `default:"5"                            mapstructure:"collector_pages" validate:"min=1,max=20"`
	CronSchedule   CronSchedule  `mapstructure:"cron_schedule"`
	Map            MapConfig     `mapstructure:"map"`
}

// UserSettings represents the fields that one user fills for the plugin: the
// credentials of their parking account.
//
//nolint:lll // the tags of one field share one line.
type UserSettings struct {
	Username string `json:"username" jsonschema:"title=Username,required"                                jsonschema_extras:"x-maroid-scope=user"`
	Password string `json:"password" jsonschema:"title=Password,format=password,writeOnly=true,required" jsonschema_extras:"x-maroid-scope=user"`
}
