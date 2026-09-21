package tools

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"slices"
	"strings"

	"github.com/abgeo/maroid/apps/hub/internal/domain/errs"
	"github.com/abgeo/maroid/apps/hub/internal/settings"
)

var (
	errNoSettings      = errors.New("declares no settings")
	errInvalidSettings = errors.New("the settings do not match the schema")
	errSecretRefused   = errors.New("a secret field changes in the deck only")
	errSettingsRequest = errors.New("the settings request failed")
)

// settingsAccess holds what both settings tools read.
type settingsAccess struct {
	logger      *slog.Logger
	settingsSvc settings.Service
}

// stored reads the key of each secret field and the values of the acting user.
func (a *settingsAccess) stored(
	ctx context.Context,
	pluginID string,
) ([]string, map[string]any, error) {
	secretFields, err := a.settingsSvc.SecretFields(pluginID)
	if err != nil {
		return nil, nil, a.failure(ctx, pluginID, err)
	}

	values, err := a.settingsSvc.Read(ctx, pluginID)
	if err != nil {
		return nil, nil, a.failure(ctx, pluginID, err)
	}

	return secretFields, values, nil
}

// failure turns a failure of the settings service into the text that an agent
// reads.
func (a *settingsAccess) failure(ctx context.Context, pluginID string, err error) error {
	var invalid *settings.InvalidError

	switch {
	case errors.Is(err, errs.ErrSettingsSchemaNotFound):
		return fmt.Errorf("the plugin %s %w", pluginID, errNoSettings)
	case errors.As(err, &invalid):
		return fmt.Errorf("%w: %s", errInvalidSettings, fieldReasons(invalid.Fields))
	default:
		// The SDK packs the error of a tool into the result of the call, so no
		// middleware writes it. This line is the one report of the cause.
		a.logger.ErrorContext(ctx, "the settings tool failed", slog.Any("error", err))

		return errSettingsRequest
	}
}

// fieldReasons names each field that caused a rejection, with its reason. The
// order does not change between two calls, so an agent reads one text.
func fieldReasons(fields map[string]string) string {
	pairs := make([]string, 0, len(fields))

	for _, key := range slices.Sorted(maps.Keys(fields)) {
		pairs = append(pairs, key+": "+fields[key])
	}

	return strings.Join(pairs, ", ")
}
