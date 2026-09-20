package mcpserver

import (
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/invopop/jsonschema"

	"github.com/abgeo/maroid/apps/hub/internal/domain/errs"
)

// inferSchema reflects the JSON schema of one model.
func inferSchema(model any) (json.RawMessage, error) {
	if err := requireStruct(model); err != nil {
		return nil, err
	}

	reflector := &jsonschema.Reflector{
		ExpandedStruct:             true,
		DoNotReference:             true,
		RequiredFromJSONSchemaTags: true,
		Anonymous:                  true,
	}

	raw, err := json.Marshal(reflector.Reflect(model))
	if err != nil {
		return nil, fmt.Errorf("encoding the schema of the tool: %w", err)
	}

	return raw, nil
}

// requireStruct refuses a model that reflects to anything but a JSON object.
// mcp.AddTool panics on such a schema, and a plugin must fail its own load
// instead of stopping the hub.
func requireStruct(model any) error {
	if model == nil {
		return fmt.Errorf("%w: it is nil", errs.ErrInvalidMCPToolModel)
	}

	declared := reflect.TypeOf(model)
	for declared.Kind() == reflect.Pointer {
		declared = declared.Elem()
	}

	if declared.Kind() != reflect.Struct {
		return fmt.Errorf(
			"%w: the model is a %s, not a struct",
			errs.ErrInvalidMCPToolModel,
			declared.Kind(),
		)
	}

	return nil
}
