package settings

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/abgeo/maroid/apps/hub/internal/database"
	"github.com/abgeo/maroid/apps/hub/internal/domain/errs"
	"github.com/abgeo/maroid/apps/hub/internal/model"
	"github.com/abgeo/maroid/apps/hub/internal/repository"
	"github.com/abgeo/maroid/apps/hub/internal/secret"
	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/libs/rest"
)

// SecretMask stands for a secret that the row holds. Read returns it in place of every
// stored secret, and Save reads it as the instruction to keep the stored value.
const SecretMask = "******"

// Service reads and writes the settings that one user stores for one plugin.
type Service interface {
	Declares(pluginID string) bool
	Schema(pluginID string) (json.RawMessage, error)
	SecretFields(pluginID string) ([]string, error)
	ChangedSecrets(pluginID string, input map[string]any) ([]string, error)
	Read(ctx context.Context, pluginID string) (map[string]any, time.Time, error)
	Settings(ctx context.Context, pluginID *pluginapi.PluginID) (map[string]any, error)
	Save(ctx context.Context, pluginID string, input map[string]any, ifMatch *time.Time) error
}

// SchemaSource gives the settings schema of one plugin.
// registry.SettingsRegistry satisfies it.
type SchemaSource interface {
	Get(pluginID string) (*Schema, bool)
}

// Manager is the default implementation of Service.
type Manager struct {
	db      *sqlx.DB
	schemas SchemaSource
	cipher  secret.Cipher
}

var (
	_ Service                    = (*Manager)(nil)
	_ pluginapi.SettingsProvider = (*Manager)(nil)
)

// NewManager creates a new Manager instance.
func NewManager(db *sqlx.DB, schemas SchemaSource, cipher secret.Cipher) *Manager {
	return &Manager{
		db:      db,
		schemas: schemas,
		cipher:  cipher,
	}
}

// Declares reports whether the plugin declares a settings schema.
func (m *Manager) Declares(pluginID string) bool {
	_, found := m.schemas.Get(pluginID)

	return found
}

// Schema returns the settings schema that the plugin declares.
func (m *Manager) Schema(pluginID string) (json.RawMessage, error) {
	schema, err := m.schemaOf(pluginID)
	if err != nil {
		return nil, err
	}

	return schema.Document, nil
}

// SecretFields names each secret field that the plugin declares, in one order.
func (m *Manager) SecretFields(pluginID string) ([]string, error) {
	schema, err := m.schemaOf(pluginID)
	if err != nil {
		return nil, err
	}

	fields := make([]string, 0, len(schema.Kinds))

	for key, kind := range schema.Kinds {
		if kind == model.FieldKindSecret {
			fields = append(fields, key)
		}
	}

	slices.Sort(fields)

	return fields, nil
}

// ChangedSecrets names each secret field that the input changes, in one order.
// A value that is the mask keeps the stored secret, so it changes nothing. A
// value that is null or empty removes the stored secret, and that is a change.
func (m *Manager) ChangedSecrets(pluginID string, input map[string]any) ([]string, error) {
	schema, err := m.schemaOf(pluginID)
	if err != nil {
		return nil, err
	}

	changed := make([]string, 0, len(input))

	for key, value := range input {
		kind := schema.Kinds[key]
		if kind != model.FieldKindSecret || keepsSecret(kind, value) {
			continue
		}

		changed = append(changed, key)
	}

	slices.Sort(changed)

	return changed, nil
}

// Read returns the settings of the acting user, for the person who stored them.
// The second answer is the moment of the last write, which an entity tag names.
// It is the zero time when no row exists.
func (m *Manager) Read(
	ctx context.Context,
	pluginID string,
) (map[string]any, time.Time, error) {
	schema, err := m.schemaOf(pluginID)
	if err != nil {
		return nil, time.Time{}, err
	}

	stored, version, err := m.stored(ctx, pluginID)
	if err != nil {
		return nil, time.Time{}, err
	}

	values := make(map[string]any, len(schema.Kinds))

	for key, kind := range schema.Kinds {
		entry, held := stored[key]

		if kind == model.FieldKindSecret {
			values[key] = maskOf(held)

			continue
		}

		if held {
			values[key] = entry.Value
		}
	}

	return values, version, nil
}

// Settings returns the settings of the acting user, for the plugin that reads them.
func (m *Manager) Settings(
	ctx context.Context,
	pluginID *pluginapi.PluginID,
) (map[string]any, error) {
	id := pluginID.String()

	schema, err := m.schemaOf(id)
	if err != nil {
		return nil, err
	}

	stored, _, err := m.stored(ctx, id)
	if err != nil {
		return nil, err
	}

	for key := range schema.Required {
		if _, held := stored[key]; !held {
			return nil, fmt.Errorf(
				"the field %q holds no value: %w",
				key,
				pluginapi.ErrSettingsAbsent,
			)
		}
	}

	return m.reveal(ctx, schema, stored)
}

// Save stores the settings of the acting user for the plugin.
// A non-nil ifMatch refuses a write to a record that changed after the client
// read it.
func (m *Manager) Save(
	ctx context.Context,
	pluginID string,
	input map[string]any,
	ifMatch *time.Time,
) error {
	schema, err := m.schemaOf(pluginID)
	if err != nil {
		return err
	}

	if err = Validate(schema, input); err != nil {
		return err
	}

	err = database.WithUserTx(ctx, m.db, func(tx *sqlx.Tx) error {
		settingsRepo := repository.NewPluginSettings(tx)

		current, readErr := settingsRepo.Get(ctx, pluginID)
		if readErr != nil {
			return fmt.Errorf("reading the stored settings: %w", readErr)
		}

		fields, mergeErr := m.merge(ctx, schema, input, storedFields(current))
		if mergeErr != nil {
			return mergeErr
		}

		return settingsRepo.Upsert(ctx, pluginID, fields, ifMatch)
	})
	if err != nil {
		if errors.Is(err, rest.ErrModified) {
			return rest.ErrModified
		}

		return fmt.Errorf("saving the settings: %w", err)
	}

	return nil
}

// merge builds the row from the input and from the fields that the row already holds.
func (m *Manager) merge(
	ctx context.Context,
	schema *Schema,
	input map[string]any,
	stored model.Fields,
) (model.Fields, error) {
	fields := make(model.Fields, len(schema.Kinds))

	for key, kind := range schema.Kinds {
		entry, held, err := m.resolve(ctx, kind, input, stored, key)
		if err != nil {
			return nil, fmt.Errorf("protecting the field %q: %w", key, err)
		}

		if held {
			fields[key] = entry
		}
	}

	if missing := missingFields(schema, fields); len(missing) > 0 {
		return nil, &InvalidError{Fields: sortedFailures(missing)}
	}

	return fields, nil
}

// resolve returns the entry of one field, and whether the row holds it at all.
func (m *Manager) resolve(
	ctx context.Context,
	kind model.FieldKind,
	input map[string]any,
	stored model.Fields,
	key string,
) (model.SettingEntry, bool, error) {
	value, given := input[key]

	if !given || keepsSecret(kind, value) {
		entry, held := stored[key]

		return entry, held, nil
	}

	if isEmpty(value) {
		return model.SettingEntry{}, false, nil
	}

	entry, err := m.protect(ctx, kind, value)
	if err != nil {
		return model.SettingEntry{}, false, err
	}

	return entry, true, nil
}

// missingFields names each required field that the merged row does not hold.
func missingFields(schema *Schema, fields model.Fields) map[string]string {
	missing := make(map[string]string)

	for key := range schema.Required {
		if _, held := fields[key]; !held {
			missing[Pointer([]string{key})] = reasonRequired
		}
	}

	return missing
}

// protect returns the entry that the row holds for one field.
func (m *Manager) protect(
	ctx context.Context,
	kind model.FieldKind,
	value any,
) (model.SettingEntry, error) {
	if kind != model.FieldKindSecret {
		return model.SettingEntry{Kind: kind, Value: value}, nil
	}

	plaintext, ok := value.(string)
	if !ok {
		return model.SettingEntry{}, fmt.Errorf(
			"%w: a secret field holds a string",
			errs.ErrInvalidSettingsModel,
		)
	}

	ciphertext, err := m.cipher.Encrypt(ctx, m.keyOf(ctx), plaintext)
	if err != nil {
		return model.SettingEntry{}, fmt.Errorf("protecting the value: %w", err)
	}

	return model.SettingEntry{Kind: model.FieldKindSecret, Value: ciphertext}, nil
}

// reveal returns the plaintext of every declared field.
func (m *Manager) reveal(
	ctx context.Context,
	schema *Schema,
	stored model.Fields,
) (map[string]any, error) {
	values := make(map[string]any, len(schema.Kinds))

	for key, kind := range schema.Kinds {
		entry, held := stored[key]
		if !held {
			continue
		}

		if kind != model.FieldKindSecret {
			values[key] = entry.Value

			continue
		}

		ciphertext, ok := entry.Value.(string)
		if !ok {
			return nil, fmt.Errorf(
				"the field %q holds no protected value: %w",
				key,
				errs.ErrProtectionUnavailable,
			)
		}

		plaintext, err := m.cipher.Decrypt(ctx, m.keyOf(ctx), ciphertext)
		if err != nil {
			return nil, fmt.Errorf("reading the protected field %q: %w", key, err)
		}

		values[key] = plaintext
	}

	return values, nil
}

// keyOf names the key that protects the secrets of the acting user.
func (m *Manager) keyOf(ctx context.Context) secret.Key {
	return secret.UserKey(pluginapi.ActingUserFromContext(ctx))
}

func (m *Manager) schemaOf(pluginID string) (*Schema, error) {
	schema, found := m.schemas.Get(pluginID)
	if !found {
		return nil, fmt.Errorf("%w: %s", errs.ErrSettingsSchemaNotFound, pluginID)
	}

	return schema, nil
}

func (m *Manager) stored(
	ctx context.Context,
	pluginID string,
) (model.Fields, time.Time, error) {
	var entity *model.PluginSettings

	err := database.WithUserTx(ctx, m.db, func(tx *sqlx.Tx) error {
		var readErr error

		entity, readErr = repository.NewPluginSettings(tx).Get(ctx, pluginID)
		if readErr != nil {
			return fmt.Errorf("reading the stored settings: %w", readErr)
		}

		return nil
	})
	if err != nil {
		return nil, time.Time{}, fmt.Errorf("reading the settings of the acting user: %w", err)
	}

	var version time.Time
	if entity != nil {
		version = entity.UpdatedAt
	}

	return storedFields(entity), version, nil
}

func storedFields(entity *model.PluginSettings) model.Fields {
	if entity == nil {
		return model.Fields{}
	}

	return entity.Fields
}

// maskOf reports a secret field to the person who stored it.
func maskOf(held bool) string {
	if !held {
		return ""
	}

	return SecretMask
}

// keepsSecret reports the value that a client read and sent back without a change.
// It leaves the stored entry alone, exactly as an absent field does.
func keepsSecret(kind model.FieldKind, value any) bool {
	return kind == model.FieldKindSecret && value == SecretMask
}

// isEmpty reports a value that removes the stored entry of its field.
func isEmpty(value any) bool {
	if value == nil {
		return true
	}

	text, ok := value.(string)

	return ok && text == ""
}
