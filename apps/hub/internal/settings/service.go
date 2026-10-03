package settings

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/abgeo/maroid/apps/hub/internal/database"
	"github.com/abgeo/maroid/apps/hub/internal/domain/errs"
	"github.com/abgeo/maroid/apps/hub/internal/model"
	"github.com/abgeo/maroid/apps/hub/internal/repository"
	"github.com/abgeo/maroid/apps/hub/internal/secret"
	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/libs/rest/precondition"
)

// SecretMask stands for a secret that the row holds. Read returns it in place of every
// stored secret, and Save reads it as the instruction to keep the stored value.
const SecretMask = "******"

// Service reads and writes the settings that one workspace and one user store for
// one plugin.
type Service interface {
	Declares(pluginID string) bool
	Schema(pluginID string) (json.RawMessage, error)
	SecretFields(pluginID string) ([]string, error)
	ChangedSecrets(pluginID string, input map[string]any) ([]string, error)
	Read(ctx context.Context, pluginID string) (map[string]any, time.Time, error)
	Settings(ctx context.Context, pluginID *pluginapi.PluginID) (map[string]any, error)
	Save(
		ctx context.Context,
		pluginID string,
		input map[string]any,
		ifMatch *time.Time,
	) (time.Time, error)
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

// Read returns the settings of the acting workspace and of the acting user, for a
// member who reads them. The second answer is the moment of the last write to
// either row, which an entity tag names. It is the zero time when no row exists.
func (m *Manager) Read(
	ctx context.Context,
	pluginID string,
) (map[string]any, time.Time, error) {
	schema, err := m.schemaOf(pluginID)
	if err != nil {
		return nil, time.Time{}, err
	}

	stored, err := m.stored(ctx, pluginID)
	if err != nil {
		return nil, time.Time{}, err
	}

	values := make(map[string]any, len(schema.Kinds))

	for key, kind := range schema.Kinds {
		entry, held := stored.entry(schema, key)

		if kind == model.FieldKindSecret {
			values[key] = maskOf(held)

			continue
		}

		if held {
			values[key] = entry.Value
		}
	}

	return values, stored.version, nil
}

// Settings returns the settings of the acting workspace and of the acting user, for
// the plugin that reads them. A run with no acting user reads no field of a user.
func (m *Manager) Settings(
	ctx context.Context,
	pluginID *pluginapi.PluginID,
) (map[string]any, error) {
	id := pluginID.String()

	schema, err := m.schemaOf(id)
	if err != nil {
		return nil, err
	}

	stored, err := m.stored(ctx, id)
	if err != nil {
		return nil, err
	}

	for key := range schema.Required {
		if _, held := stored.entry(schema, key); !held {
			return nil, fmt.Errorf(
				"the field %q holds no value: %w",
				key,
				pluginapi.ErrSettingsAbsent,
			)
		}
	}

	return m.reveal(ctx, schema, stored)
}

// Save stores the settings of the acting workspace and of the acting user for the
// plugin, and answers the moment of the write. Each field reaches the row of its
// scope, and a row that the save leaves as it was is not written. A non-nil ifMatch
// refuses a write to a record that changed after the client read it.
func (m *Manager) Save(
	ctx context.Context,
	pluginID string,
	input map[string]any,
	ifMatch *time.Time,
) (time.Time, error) {
	schema, err := m.schemaOf(pluginID)
	if err != nil {
		return time.Time{}, err
	}

	if err = Validate(schema, input); err != nil {
		return time.Time{}, err
	}

	var written time.Time

	err = database.WithScopeTx(ctx, m.db, func(tx *sqlx.Tx) error {
		var saveErr error

		written, saveErr = m.saveIn(
			ctx,
			repository.NewPluginSettings(tx),
			schema,
			pluginID,
			input,
			ifMatch,
		)

		return saveErr
	})
	if err != nil {
		if errors.Is(err, precondition.ErrModified) {
			return time.Time{}, precondition.ErrModified
		}

		return time.Time{}, fmt.Errorf("saving the settings: %w", err)
	}

	return written, nil
}

// saveIn holds the row of each scope, checks the version that the client read, and
// writes each row that the save changes. It answers the version after the save.
func (m *Manager) saveIn(
	ctx context.Context,
	settingsRepo repository.PluginSettingsRepository,
	schema *Schema,
	pluginID string,
	input map[string]any,
	ifMatch *time.Time,
) (time.Time, error) {
	current, err := readScopes(ctx, settingsRepo.GetForUpdate, pluginID)
	if err != nil {
		return time.Time{}, err
	}

	if ifMatch != nil && !current.version.Equal(*ifMatch) {
		return time.Time{}, precondition.ErrModified
	}

	fields, err := m.merge(ctx, schema, input, current)
	if err != nil {
		return time.Time{}, err
	}

	written := current.version

	for _, scope := range settingScopes() {
		if reflect.DeepEqual(fields[scope], current.fields[scope]) {
			continue
		}

		moment, upsertErr := settingsRepo.Upsert(ctx, scope, pluginID, fields[scope])
		if upsertErr != nil {
			return time.Time{}, fmt.Errorf("storing the settings: %w", upsertErr)
		}

		written = later(written, moment)
	}

	return written, nil
}

// merge builds the row of each scope from the input and from the fields that the
// rows already hold.
func (m *Manager) merge(
	ctx context.Context,
	schema *Schema,
	input map[string]any,
	stored *scopedSettings,
) (map[model.SettingScope]model.Fields, error) {
	fields := make(map[model.SettingScope]model.Fields, len(settingScopes()))
	for _, scope := range settingScopes() {
		fields[scope] = model.Fields{}
	}

	for key, kind := range schema.Kinds {
		scope := schema.Scopes[key]

		entry, held, err := m.resolve(ctx, scope, kind, input, stored.fields[scope], key)
		if err != nil {
			return nil, fmt.Errorf("protecting the field %q: %w", key, err)
		}

		if held {
			fields[scope][key] = entry
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
	scope model.SettingScope,
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

	entry, err := m.protect(ctx, keyOf(ctx, scope), kind, value)
	if err != nil {
		return model.SettingEntry{}, false, err
	}

	return entry, true, nil
}

// missingFields names each required field that the merged rows do not hold.
func missingFields(
	schema *Schema,
	fields map[model.SettingScope]model.Fields,
) map[string]string {
	missing := make(map[string]string)

	for key := range schema.Required {
		if _, held := fields[schema.Scopes[key]][key]; !held {
			missing[Pointer([]string{key})] = reasonRequired
		}
	}

	return missing
}

// protect returns the entry that the row holds for one field.
func (m *Manager) protect(
	ctx context.Context,
	key secret.Key,
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

	ciphertext, err := m.cipher.Encrypt(ctx, key, plaintext)
	if err != nil {
		return model.SettingEntry{}, fmt.Errorf("protecting the value: %w", err)
	}

	return model.SettingEntry{Kind: model.FieldKindSecret, Value: ciphertext}, nil
}

// reveal returns the plaintext of every declared field.
func (m *Manager) reveal(
	ctx context.Context,
	schema *Schema,
	stored *scopedSettings,
) (map[string]any, error) {
	values := make(map[string]any, len(schema.Kinds))

	for key, kind := range schema.Kinds {
		entry, held := stored.entry(schema, key)
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

		plaintext, err := m.cipher.Decrypt(ctx, keyOf(ctx, schema.Scopes[key]), ciphertext)
		if err != nil {
			return nil, fmt.Errorf("reading the protected field %q: %w", key, err)
		}

		values[key] = plaintext
	}

	return values, nil
}

// keyOf names the key that protects the secrets of the scope: the key of the acting
// workspace, or the key of the acting user.
func keyOf(ctx context.Context, scope model.SettingScope) secret.Key {
	if scope == model.SettingScopeUser {
		return secret.UserKey(pluginapi.ActingUserFromContext(ctx))
	}

	return secret.WorkspaceKey(pluginapi.ActingWorkspaceFromContext(ctx))
}

func (m *Manager) schemaOf(pluginID string) (*Schema, error) {
	schema, found := m.schemas.Get(pluginID)
	if !found {
		return nil, fmt.Errorf("%w: %s", errs.ErrSettingsSchemaNotFound, pluginID)
	}

	return schema, nil
}

func (m *Manager) stored(ctx context.Context, pluginID string) (*scopedSettings, error) {
	var stored *scopedSettings

	err := database.WithScopeTx(ctx, m.db, func(tx *sqlx.Tx) error {
		var readErr error

		stored, readErr = readScopes(ctx, repository.NewPluginSettings(tx).Get, pluginID)

		return readErr
	})
	if err != nil {
		return nil, fmt.Errorf("reading the settings of the acting scope: %w", err)
	}

	return stored, nil
}

// settingScopes returns every scope, in the order that a save writes them.
func settingScopes() []model.SettingScope {
	return []model.SettingScope{model.SettingScopeWorkspace, model.SettingScopeUser}
}

// scopedSettings holds the stored fields of each scope, and the moment of the last
// write to either row.
type scopedSettings struct {
	fields  map[model.SettingScope]model.Fields
	version time.Time
}

// entry returns the stored entry of one field, from the row of its scope.
func (s *scopedSettings) entry(schema *Schema, key string) (model.SettingEntry, bool) {
	entry, held := s.fields[schema.Scopes[key]][key]

	return entry, held
}

// readScopes reads the row of each scope through the given read.
func readScopes(
	ctx context.Context,
	read func(context.Context, model.SettingScope, string) (*model.PluginSettings, error),
	pluginID string,
) (*scopedSettings, error) {
	stored := &scopedSettings{
		fields: make(map[model.SettingScope]model.Fields, len(settingScopes())),
	}

	for _, scope := range settingScopes() {
		entity, err := read(ctx, scope, pluginID)
		if err != nil {
			return nil, fmt.Errorf("reading the stored settings: %w", err)
		}

		stored.fields[scope] = storedFields(entity)

		if entity != nil {
			stored.version = later(stored.version, entity.UpdatedAt)
		}
	}

	return stored, nil
}

func later(first time.Time, second time.Time) time.Time {
	if second.After(first) {
		return second
	}

	return first
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
