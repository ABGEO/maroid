package settings

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/invopop/jsonschema"
	jsonvalidate "github.com/santhosh-tekuri/jsonschema/v6"

	"github.com/abgeo/maroid/apps/hub/internal/domain/errs"
	"github.com/abgeo/maroid/apps/hub/internal/model"
)

// MaxValueLength is the longest value that one field accepts.
const MaxValueLength = 4096

// secretFormat marks a property whose value Maroid returns to nobody.
const secretFormat = "password"

// resourceName names the document inside the compiler. No request reads it.
const resourceName = "settings.json"

// Schema is the settings schema of one plugin, in each form that the hub needs.
type Schema struct {
	// Document is the JSON Schema that the schema route serves.
	Document json.RawMessage
	// Compiled validates a save.
	Compiled *jsonvalidate.Schema
	// Kinds names the kind of each field, keyed by the field key.
	Kinds map[string]model.FieldKind
	// Required holds the key of each field that must hold a value.
	Required map[string]struct{}
}

// Infer builds the schema of a plugin from the model that the plugin declares.
func Infer(settingsModel any) (*Schema, error) {
	if err := requireStruct(settingsModel); err != nil {
		return nil, err
	}

	reflector := &jsonschema.Reflector{
		ExpandedStruct:             true,
		DoNotReference:             true,
		RequiredFromJSONSchemaTags: true,
	}

	document := reflector.Reflect(settingsModel)
	applyValueLimit(document)

	raw, err := json.Marshal(document)
	if err != nil {
		return nil, fmt.Errorf("encoding the settings schema: %w", err)
	}

	compiled, err := compile(withoutRequired(raw))
	if err != nil {
		return nil, err
	}

	return &Schema{
		Document: raw,
		Compiled: compiled,
		Kinds:    kinds(document),
		Required: required(document),
	}, nil
}

func requireStruct(settingsModel any) error {
	if settingsModel == nil {
		return fmt.Errorf("%w: it is nil", errs.ErrInvalidSettingsModel)
	}

	declared := reflect.TypeOf(settingsModel)
	for declared.Kind() == reflect.Pointer {
		declared = declared.Elem()
	}

	if declared.Kind() != reflect.Struct {
		return fmt.Errorf("%w: it is a %s", errs.ErrInvalidSettingsModel, declared.Kind())
	}

	return nil
}

// withoutRequired removes the required array from the document.
func withoutRequired(raw json.RawMessage) json.RawMessage {
	var document map[string]any

	if err := json.Unmarshal(raw, &document); err != nil {
		return raw
	}

	delete(document, "required")

	stripped, err := json.Marshal(document)
	if err != nil {
		return raw
	}

	return stripped
}

func compile(raw json.RawMessage) (*jsonvalidate.Schema, error) {
	document, err := jsonvalidate.UnmarshalJSON(bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("reading the settings schema: %w", err)
	}

	compiler := jsonvalidate.NewCompiler()
	if err = compiler.AddResource(resourceName, document); err != nil {
		return nil, fmt.Errorf("adding the settings schema: %w", err)
	}

	compiled, err := compiler.Compile(resourceName)
	if err != nil {
		return nil, fmt.Errorf("compiling the settings schema: %w", err)
	}

	return compiled, nil
}

func applyValueLimit(document *jsonschema.Schema) {
	limit := uint64(MaxValueLength)

	for pair := document.Properties.Oldest(); pair != nil; pair = pair.Next() {
		if pair.Value.Type == "string" && pair.Value.MaxLength == nil {
			pair.Value.MaxLength = &limit
		}
	}
}

func kinds(document *jsonschema.Schema) map[string]model.FieldKind {
	found := make(map[string]model.FieldKind, document.Properties.Len())

	for pair := document.Properties.Oldest(); pair != nil; pair = pair.Next() {
		found[pair.Key] = kindOf(pair.Value)
	}

	return found
}

func kindOf(property *jsonschema.Schema) model.FieldKind {
	switch {
	case property.WriteOnly || property.Format == secretFormat:
		return model.FieldKindSecret
	case len(property.Enum) > 0:
		return model.FieldKindChoice
	case property.Type == "boolean":
		return model.FieldKindSwitch
	default:
		return model.FieldKindText
	}
}

func required(document *jsonschema.Schema) map[string]struct{} {
	found := make(map[string]struct{}, len(document.Required))
	for _, key := range document.Required {
		found[key] = struct{}{}
	}

	return found
}
