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
	keyAccount  = "account_number"
	keyPeriod   = "period"
	keyNotify   = "notify"

	valueEmail    = "person@example.com"
	valuePassword = "hunter2"
	valuePeriod   = "month"
)

// probeModel is the model that the scenarios of spec-scenarios.md declare.
type probeModel struct {
	Email         string `json:"email"          jsonschema:"title=Email,required"`
	Password      string `json:"password"       jsonschema:"title=Password,format=password,writeOnly=true,required"`
	AccountNumber string `json:"account_number" jsonschema:"title=Account number"`
	Period        string `json:"period"         jsonschema:"title=Period,enum=month,enum=year"`
	Notify        bool   `json:"notify"         jsonschema:"title=Notify me"`
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
	}, schema.Kinds)

	require.Equal(t, map[string]struct{}{keyEmail: {}, keyPassword: {}}, schema.Required)
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
