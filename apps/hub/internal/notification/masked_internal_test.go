package notification

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/apps/hub/internal/model"
	"github.com/abgeo/maroid/libs/pluginapi"
)

// optionalSecretKind declares a required secret and an optional one.
type optionalSecretKind struct{}

type optionalSecretFields struct {
	Token  string `json:"token"  jsonschema:"title=Token,format=password,writeOnly=true,required"`
	Backup string `json:"backup" jsonschema:"title=Backup,format=password,writeOnly=true"`
}

type plainDestination struct {
	Address string `json:"address" jsonschema:"title=Address,required"`
}

func (optionalSecretKind) ID() string            { return "probe" }
func (optionalSecretKind) Name() string          { return "Probe" }
func (optionalSecretKind) ProviderModel() any    { return optionalSecretFields{} }
func (optionalSecretKind) DestinationModel() any { return plainDestination{} }

func (optionalSecretKind) Verify(context.Context, map[string]any) error { return nil }

func (optionalSecretKind) Send(
	context.Context,
	map[string]any,
	map[string]any,
	pluginapi.Message,
) error {
	return nil
}

// NOTIF-SC-004: A reader sees the mask for a held secret, and the empty string for a
// declared secret that holds no value. NOTIF-DD-006.
func TestMaskedTellsAHeldSecretFromAnAbsentOne(t *testing.T) {
	t.Parallel()

	kinds := NewKindRegistry()
	require.NoError(t, kinds.Register(optionalSecretKind{}))

	manager := &Manager{kinds: kinds}
	masked := manager.masked(&model.NotificationProvider{
		Kind:   "probe",
		Fields: model.Fields{"token": {Kind: model.FieldKindSecret, Value: "protected"}},
	})

	require.Equal(t, model.Fields{
		"token":  {Kind: model.FieldKindSecret, Value: "******"},
		"backup": {Kind: model.FieldKindSecret, Value: ""},
	}, masked)
}
