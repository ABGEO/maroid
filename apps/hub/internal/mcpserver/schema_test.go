package mcpserver //nolint:testpackage // It reads inferSchema, which stays unexported.

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/apps/hub/internal/domain/errs"
)

type wateringInput struct {
	Plant string `json:"plant" jsonschema:"required"`
	Litre int    `json:"litre"`
}

// MCPHUB-SC-008: A tool of a plugin reaches an MCP client with the shape that the
// plugin declared. MCPHUB-DD-010: The reflector of the hub carries no value limit
// of the settings form.
func TestInferSchemaReflectsTheModelOfATool(t *testing.T) {
	t.Parallel()

	document, err := inferSchema(wateringInput{})

	require.NoError(t, err)

	var shape struct {
		Type string `json:"type"`
		// The member names are the keywords of JSON Schema, not names of the hub.
		Properties map[string]struct {
			Type      string `json:"type"`
			MaxLength *int   `json:"maxLength"`
		} `json:"properties"`
		Required []string `json:"required"`
		ID       string   `json:"$id"`
	}

	decodeErr := json.Unmarshal(document, &shape)
	require.NoError(t, decodeErr, string(document))

	require.Equal(t, "object", shape.Type)
	require.Equal(t, "string", shape.Properties["plant"].Type)
	require.Equal(t, "integer", shape.Properties["litre"].Type)
	require.Equal(t, []string{"plant"}, shape.Required)
	require.Nil(t, shape.Properties["plant"].MaxLength, "a tool argument carries no value limit")
	require.Empty(t, shape.ID, "no identifier of the document names a package of the hub")
}

// MCPHUB-FR-007: A plugin that declares a model the hub cannot reflect fails the
// load, and the failure names the reason.
func TestInferSchemaRefusesAModelThatIsNotAStruct(t *testing.T) {
	t.Parallel()

	_, err := inferSchema("not a struct")

	require.ErrorIs(t, err, errs.ErrInvalidMCPToolModel)
}
