package pluginconfig_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/libs/pluginconfig"
)

const (
	keyEmail   = "email"
	valueEmail = "person@example.com"
)

type userSettings struct {
	Email    string `json:"email"    validate:"required"`
	Password string `json:"password"`
	Period   string `default:"month" json:"period"`
}

// PSET-SC-010: A run reads the settings of the acting user. This part covers the
// decode, because the `json` tag names the field in the schema and in the struct.
func TestDecodeAndValidateSettingsReadsTheJSONTag(t *testing.T) {
	t.Parallel()

	target := new(userSettings)
	values := map[string]any{keyEmail: valueEmail, "password": "hunter2"}

	require.NoError(t, pluginconfig.DecodeAndValidateSettings(values, target))
	require.Equal(t, valueEmail, target.Email)
	require.Equal(t, "hunter2", target.Password)
	require.Equal(t, "month", target.Period, "a field the values omit keeps its default")
}

// PSET-SC-007: A value that does not match the declaration is rejected. The plugin
// applies its own validation to the values that the hub returns.
func TestDecodeAndValidateSettingsRejectsAMissingRequiredField(t *testing.T) {
	t.Parallel()

	err := pluginconfig.DecodeAndValidateSettings(map[string]any{}, new(userSettings))

	require.ErrorContains(t, err, "validating the settings")
}

// pluginSchedule mirrors the nested shape that a plugin declares for its jobs.
type pluginSchedule struct {
	Collector string `default:"0 10 1 * *" mapstructure:"collector"`
}

type pluginSettings struct {
	BaseURL  string         `default:"https://example.test" mapstructure:"base_url"`
	Email    string         `mapstructure:"email"           validate:"required"`
	Schedule pluginSchedule `mapstructure:"cron_schedule"`
}

// CFG-006: A plugin decodes its configuration with the `mapstructure` tag, and the
// struct declares the defaults and the validation.
func TestDecodeAndValidateConfigReadsTheMapstructureTag(t *testing.T) {
	t.Parallel()

	target := new(pluginSettings)
	cfg := map[string]any{
		keyEmail:        valueEmail,
		"cron_schedule": map[string]any{"collector": "*/5 * * * * *"},
	}

	require.NoError(t, pluginconfig.DecodeAndValidateConfig(cfg, target))
	require.Equal(t, valueEmail, target.Email)
	require.Equal(t, "https://example.test", target.BaseURL, "an absent key keeps its default")
	require.Equal(t, "*/5 * * * * *", target.Schedule.Collector, "a nested key decodes")
}

// CFG-003: A wrong configuration stops the load, not the first job.
func TestDecodeAndValidateConfigRejectsAMissingRequiredField(t *testing.T) {
	t.Parallel()

	err := pluginconfig.DecodeAndValidateConfig(map[string]any{}, new(pluginSettings))

	require.ErrorContains(t, err, "validating")
}

// A nested struct keeps the default that its own field declares.
func TestDecodeAndValidateConfigAppliesANestedDefault(t *testing.T) {
	t.Parallel()

	target := new(pluginSettings)

	require.NoError(t, pluginconfig.DecodeAndValidateConfig(
		map[string]any{keyEmail: valueEmail}, target,
	))
	require.Equal(t, "0 10 1 * *", target.Schedule.Collector)
}
