package settings_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/apps/hub/internal/domain/errs"
	"github.com/abgeo/maroid/apps/hub/internal/registry"
	"github.com/abgeo/maroid/apps/hub/internal/settings"
	"github.com/abgeo/maroid/libs/pluginapi"
)

const unknownPluginID = "dev.maroid.unknown"

// probeManager builds a manager that reaches no database and no protection
// service. Neither SecretFields nor ChangedSecrets reads one.
func probeManager(t *testing.T) *settings.Manager {
	t.Helper()

	schemas := registry.NewSettingsRegistry()
	require.NoError(t, schemas.Register(pluginapi.ParsePluginID(probeID), probeSchema(t)))

	return settings.NewManager(nil, schemas, nil)
}

// MCPHUB-SC-017: The report of a settings schema names each secret field, so an
// agent reads that fact from one member.
func TestSecretFieldsNamesEachSecretField(t *testing.T) {
	t.Parallel()

	fields, err := probeManager(t).SecretFields(probeID)

	require.NoError(t, err)
	require.Equal(t, []string{keyPassword}, fields)
}

// MCPHUB-SC-018: A plugin that declares no settings schema answers no report.
func TestSecretFieldsFailsForAPluginThatDeclaresNoSchema(t *testing.T) {
	t.Parallel()

	_, err := probeManager(t).SecretFields(unknownPluginID)

	require.ErrorIs(t, err, errs.ErrSettingsSchemaNotFound)
}

// MCPHUB-SC-021: A value that replaces a stored secret changes that field.
func TestChangedSecretsNamesAValueThatReplacesASecret(t *testing.T) {
	t.Parallel()

	changed, err := probeManager(t).ChangedSecrets(probeID, map[string]any{
		keyEmail:    valueEmail,
		keyPassword: valuePassword,
	})

	require.NoError(t, err)
	require.Equal(t, []string{keyPassword}, changed)
}

// MCPHUB-SC-022: The mask keeps the stored secret, so it changes no field.
func TestChangedSecretsNamesNoFieldForTheMask(t *testing.T) {
	t.Parallel()

	changed, err := probeManager(t).ChangedSecrets(probeID, map[string]any{
		keyEmail:    valueEmail,
		keyPassword: settings.SecretMask,
	})

	require.NoError(t, err)
	require.Empty(t, changed)
}

// MCPHUB-SC-023: A null removes the stored secret, and a removal changes it.
func TestChangedSecretsNamesARemoval(t *testing.T) {
	t.Parallel()

	changed, err := probeManager(t).ChangedSecrets(probeID, map[string]any{
		keyPassword: nil,
	})

	require.NoError(t, err)
	require.Equal(t, []string{keyPassword}, changed)
}

// MCPHUB-SC-021: A field that is no secret field changes no secret.
func TestChangedSecretsNamesNoFieldThatIsNoSecret(t *testing.T) {
	t.Parallel()

	changed, err := probeManager(t).ChangedSecrets(probeID, map[string]any{
		keyEmail:  valueEmail,
		keyPeriod: valuePeriod,
		keyNotify: true,
	})

	require.NoError(t, err)
	require.Empty(t, changed)
}
