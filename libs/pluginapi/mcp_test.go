package pluginapi_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/libs/pluginapi"
)

const waterPlant = "water_plant"

type wateringInput struct {
	Plant string `json:"plant" jsonschema:"required"`
	Litre int    `json:"litre"`
}

type wateringOutput struct {
	Watered string `json:"watered"`
}

// MCPHUB-FR-007: A plugin declares a tool, and the hub reads its shape from the
// contract alone. MCPHUB-DD-009: The contract names no type of the Model Context
// Protocol SDK, so NewTypedTool gives the plugin its own Go types with no import.
func TestNewTypedToolCarriesTheModelsOfItsTypeArguments(t *testing.T) {
	t.Parallel()

	tool := pluginapi.NewTypedTool(
		pluginapi.MCPToolMeta{
			Name:        waterPlant,
			Title:       "Water a plant",
			Description: "Record that a plant was watered.",
			Annotations: pluginapi.MCPToolAnnotations{IdempotentHint: true},
		},
		func(_ context.Context, input wateringInput) (wateringOutput, error) {
			return wateringOutput{Watered: input.Plant}, nil
		},
	)

	meta := tool.Meta()

	require.Equal(t, waterPlant, meta.Name)
	require.True(t, meta.Annotations.IdempotentHint)
	require.IsType(t, wateringInput{}, meta.InputModel)
	require.IsType(t, wateringOutput{}, meta.OutputModel)
}

// MCPHUB-FR-007: The handler receives the arguments of the call in its own type.
func TestNewTypedToolDecodesTheInputIntoItsOwnType(t *testing.T) {
	t.Parallel()

	tool := pluginapi.NewTypedTool(
		pluginapi.MCPToolMeta{Name: waterPlant},
		func(_ context.Context, input wateringInput) (wateringOutput, error) {
			require.Equal(t, 3, input.Litre)

			return wateringOutput{Watered: input.Plant}, nil
		},
	)

	output, err := tool.Handle(t.Context(), json.RawMessage(`{"plant":"basil","litre":3}`))

	require.NoError(t, err)
	require.Equal(t, wateringOutput{Watered: "basil"}, output)
}

// MCPHUB-FR-007: A document that the handler cannot read is a failure of the
// call, not a panic of the hub.
func TestNewTypedToolReportsAnInputItCannotRead(t *testing.T) {
	t.Parallel()

	tool := pluginapi.NewTypedTool(
		pluginapi.MCPToolMeta{Name: waterPlant},
		func(_ context.Context, _ wateringInput) (wateringOutput, error) {
			t.Error("the handler must not run")

			return wateringOutput{}, nil
		},
	)

	output, err := tool.Handle(t.Context(), json.RawMessage(`{"litre":"three"}`))

	require.Error(t, err)
	require.Nil(t, output)
}

// MCPHUB-FR-013: The hub recognizes ErrSettingsAbsent from the handler of a tool,
// so the error of the plugin reaches the adapter unchanged.
func TestNewTypedToolPassesTheErrorOfTheHandler(t *testing.T) {
	t.Parallel()

	tool := pluginapi.NewTypedTool(
		pluginapi.MCPToolMeta{Name: waterPlant},
		func(_ context.Context, _ wateringInput) (wateringOutput, error) {
			return wateringOutput{}, pluginapi.ErrSettingsAbsent
		},
	)

	_, err := tool.Handle(t.Context(), json.RawMessage(`{"plant":"basil"}`))

	require.ErrorIs(t, err, pluginapi.ErrSettingsAbsent)
}
