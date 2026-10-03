package settings_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/apps/hub/internal/model"
	"github.com/abgeo/maroid/apps/hub/internal/settings"
)

const (
	keyEmail    = "email"
	keyPassword = "password"
	keyAccount  = "accountNumber"
	keyPeriod   = "period"
	keyNotify   = "notify"
	keyPin      = "pin"

	valueEmail    = "person@example.com"
	valuePassword = "hunter2"
	valuePeriod   = "month"
)

// probeModel is the model that the scenarios of spec-scenarios.md declare.
//
//nolint:tagliatelle,lll // RES-003: a settings key, which the plugin author chose.
type probeModel struct {
	Email         string `json:"email"         jsonschema:"title=Email,required"`
	Password      string `json:"password"      jsonschema:"title=Password,format=password,writeOnly=true,required"`
	AccountNumber string `json:"accountNumber" jsonschema:"title=Account number"`
	Period        string `json:"period"        jsonschema:"title=Period,enum=month,enum=year"`
	Notify        bool   `json:"notify"        jsonschema:"title=Notify me"`
	Pin           string `json:"pin"           jsonschema:"title=PIN,format=password,writeOnly=true"               jsonschema_extras:"x-maroid-scope=user"`
}

// teamModel names a scope that Maroid does not know.
type teamModel struct {
	Name string `json:"name" jsonschema_extras:"x-maroid-scope=team"`
}

// PSET-SC-002: The inferred document carries the kind of every field.
func TestInferCarriesTheKindOfEveryField(t *testing.T) {
	t.Parallel()

	schema, err := settings.Infer(&probeModel{})
	require.NoError(t, err)

	var document map[string]any

	require.NoError(t, json.Unmarshal(schema.Document, &document))
	require.Equal(t, false, document["additionalProperties"])
	require.ElementsMatch(t, []any{keyEmail, keyPassword}, document["required"])

	properties, ok := document["properties"].(map[string]any)
	require.True(t, ok)

	password, ok := properties[keyPassword].(map[string]any)
	require.True(t, ok)
	require.Equal(t, keyPassword, password["format"])
	require.Equal(t, true, password["writeOnly"])

	period, ok := properties[keyPeriod].(map[string]any)
	require.True(t, ok)
	require.ElementsMatch(t, []any{valuePeriod, "year"}, period["enum"])

	notify, ok := properties[keyNotify].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "boolean", notify["type"])

	require.Equal(t, map[string]model.FieldKind{
		keyEmail:    model.FieldKindText,
		keyPassword: model.FieldKindSecret,
		keyAccount:  model.FieldKindText,
		keyPeriod:   model.FieldKindChoice,
		keyNotify:   model.FieldKindSwitch,
		keyPin:      model.FieldKindSecret,
	}, schema.Kinds)

	require.Equal(t, map[string]struct{}{keyEmail: {}, keyPassword: {}}, schema.Required)
}

// PSET-SC-023: A field names the user as its scope, the document keeps the name so the
// deck labels the field, and a field that names no scope belongs to the workspace.
func TestInferReadsTheScopeOfEveryField(t *testing.T) {
	t.Parallel()

	schema, err := settings.Infer(&probeModel{})
	require.NoError(t, err)

	require.Equal(t, map[string]model.SettingScope{
		keyEmail:    model.SettingScopeWorkspace,
		keyPassword: model.SettingScopeWorkspace,
		keyAccount:  model.SettingScopeWorkspace,
		keyPeriod:   model.SettingScopeWorkspace,
		keyNotify:   model.SettingScopeWorkspace,
		keyPin:      model.SettingScopeUser,
	}, schema.Scopes)

	var document map[string]any

	require.NoError(t, json.Unmarshal(schema.Document, &document))

	properties, ok := document["properties"].(map[string]any)
	require.True(t, ok)

	pin, ok := properties[keyPin].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "user", pin["x-maroid-scope"])
}

// PSET-FR-021: A field that names a scope Maroid does not know is rejected.
func TestInferRejectsAnUnknownScope(t *testing.T) {
	t.Parallel()

	schema, err := settings.Infer(&teamModel{})

	require.ErrorContains(t, err, "team")
	require.Nil(t, schema)
}

// PSET-SC-001: A model that is not a struct is rejected.
func TestInferRejectsAModelThatIsNotAStruct(t *testing.T) {
	t.Parallel()

	schema, err := settings.Infer("not a struct")

	require.ErrorContains(t, err, "settings model")
	require.Nil(t, schema)
}

// PSET-SC-007: A value longer than the limit is rejected, and the limit applies to
// every string field that declares none of its own.
func TestInferAppliesTheValueLimit(t *testing.T) {
	t.Parallel()

	schema, err := settings.Infer(&probeModel{})
	require.NoError(t, err)

	var document map[string]any

	require.NoError(t, json.Unmarshal(schema.Document, &document))

	properties, ok := document["properties"].(map[string]any)
	require.True(t, ok)

	email, ok := properties[keyEmail].(map[string]any)
	require.True(t, ok)
	require.InDelta(t, float64(settings.MaxValueLength), email["maxLength"], 0)
}
