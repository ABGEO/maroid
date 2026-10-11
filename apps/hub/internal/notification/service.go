package notification

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/abgeo/maroid/apps/hub/internal/database"
	"github.com/abgeo/maroid/apps/hub/internal/domain/errs"
	"github.com/abgeo/maroid/apps/hub/internal/model"
	"github.com/abgeo/maroid/apps/hub/internal/registry"
	"github.com/abgeo/maroid/apps/hub/internal/repository"
	"github.com/abgeo/maroid/apps/hub/internal/secret"
	"github.com/abgeo/maroid/apps/hub/internal/settings"
	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/libs/rest/precondition"
)

// MaxNameLength is the longest name of a notification provider, in characters.
const MaxNameLength = 64

const (
	pointerKind        = "/kind"
	pointerName        = "/name"
	pointerPluginID    = "/plugin_id"
	pointerTopicID     = "/topic_id"
	pointerProviderID  = "/provider_id"
	fieldsPrefix       = "/fields"
	destinationPrefix  = "/destination"
	detailRequired     = "the field is required"
	detailNeverChanges = "the member never changes after the record exists"
	testTitle          = "Maroid test message"
)

// FieldError reports one member of a body that the service refuses.
type FieldError struct {
	Pointer string
	Detail  string
}

// Error returns the member and the reason.
func (e *FieldError) Error() string {
	return e.Pointer + ": " + e.Detail
}

// ProviderInput carries the members of a body. A nil member is absent, so a change
// touches the named members only.
type ProviderInput struct {
	Kind    *string
	Name    *string
	Fields  map[string]any
	Enabled *bool
}

// RouteInput carries the members of a body. A nil member is absent.
type RouteInput struct {
	PluginID    *string
	TopicID     *string
	ProviderID  *string
	Destination map[string]any
	Enabled     *bool
}

// TopicView is one topic of a plugin that the acting workspace enables, with its routes.
type TopicView struct {
	PluginID string
	Topic    pluginapi.NotificationTopic
	Routes   []model.NotificationRoute
	Unrouted bool
}

// EnablementSource gives the plugins that the acting workspace enables.
// workspace.Enablements satisfies it.
type EnablementSource interface {
	Enabled(ctx context.Context) ([]model.Enablement, error)
}

// TopicCatalog gives the notification topics that the plugins declare.
// registry.NotificationTopicRegistry satisfies it.
type TopicCatalog interface {
	TopicSource
	Of(pluginID string) []registry.NotificationTopicEntry
}

// Service manages the notification providers and the notification routes of the
// acting workspace.
type Service interface {
	ProviderService
	RouteService
}

// ProviderService manages the notification providers of the acting workspace. A
// provider that it answers carries the mask in place of each secret.
type ProviderService interface {
	Kinds() []KindEntry
	Providers(ctx context.Context) ([]model.NotificationProvider, error)
	Provider(ctx context.Context, id string) (*model.NotificationProvider, error)
	CreateProvider(ctx context.Context, input ProviderInput) (*model.NotificationProvider, error)
	UpdateProvider(
		ctx context.Context,
		id string,
		input ProviderInput,
		ifMatch *time.Time,
	) (*model.NotificationProvider, error)
	DeleteProvider(ctx context.Context, id string) error
}

// RouteService manages the notification routes of the acting workspace.
type RouteService interface {
	Topics(ctx context.Context) ([]TopicView, error)
	Routes(ctx context.Context) ([]model.NotificationRoute, error)
	Route(ctx context.Context, id string) (*model.NotificationRoute, error)
	CreateRoute(ctx context.Context, input RouteInput) (*model.NotificationRoute, error)
	UpdateRoute(
		ctx context.Context,
		id string,
		input RouteInput,
		ifMatch *time.Time,
	) (*model.NotificationRoute, error)
	DeleteRoute(ctx context.Context, id string) error
	TestRoute(ctx context.Context, id string) error
}

// Manager is the default implementation of Service.
type Manager struct {
	db          *sqlx.DB
	kinds       *KindRegistry
	topics      TopicCatalog
	enablements EnablementSource
	cipher      secret.Cipher
}

var _ Service = (*Manager)(nil)

// NewManager creates a new Manager.
func NewManager(
	db *sqlx.DB,
	kinds *KindRegistry,
	topics TopicCatalog,
	enablements EnablementSource,
	cipher secret.Cipher,
) *Manager {
	return &Manager{db: db, kinds: kinds, topics: topics, enablements: enablements, cipher: cipher}
}

// Kinds answers every notification provider kind of the hub.
func (m *Manager) Kinds() []KindEntry {
	return m.kinds.All()
}

// Providers answers every notification provider of the acting workspace.
func (m *Manager) Providers(ctx context.Context) ([]model.NotificationProvider, error) {
	var providers []model.NotificationProvider

	err := database.WithScopeTx(ctx, m.db, func(tx *sqlx.Tx) error {
		var listErr error

		providers, listErr = repository.NewNotificationProvider(tx).List(ctx)

		return listErr //nolint:wrapcheck // the repository named the failure
	})
	if err != nil {
		return nil, fmt.Errorf("listing the notification providers: %w", err)
	}

	for index := range providers {
		providers[index].Fields = m.masked(&providers[index])
	}

	return providers, nil
}

// Provider answers one notification provider of the acting workspace.
func (m *Manager) Provider(ctx context.Context, id string) (*model.NotificationProvider, error) {
	provider, err := m.storedProvider(ctx, id)
	if err != nil {
		return nil, err
	}

	provider.Fields = m.masked(provider)

	return provider, nil
}

// CreateProvider checks the credential at the external messaging service, protects
// each secret, and stores the provider in the acting workspace.
func (m *Manager) CreateProvider(
	ctx context.Context,
	input ProviderInput,
) (*model.NotificationProvider, error) {
	entry, err := m.kindOf(input.Kind)
	if err != nil {
		return nil, err
	}

	name, err := nameOf(input.Name)
	if err != nil {
		return nil, err
	}

	values, err := checkValues(entry.Provider, input.Fields, nil, fieldsPrefix)
	if err != nil {
		return nil, err
	}

	if err = entry.Kind.Verify(ctx, values); err != nil {
		return nil, fmt.Errorf("checking the credential: %w", err)
	}

	fields, err := m.protect(ctx, entry.Provider, values)
	if err != nil {
		return nil, err
	}

	var created *model.NotificationProvider

	err = database.WithScopeTx(ctx, m.db, func(tx *sqlx.Tx) error {
		var createErr error

		created, createErr = repository.NewNotificationProvider(tx).
			Create(ctx, &model.NotificationProvider{
				Kind:    entry.Kind.ID(),
				Name:    name,
				Fields:  fields,
				Enabled: enabledOf(input.Enabled),
			})

		return createErr //nolint:wrapcheck // the repository named the failure
	})
	if err != nil {
		return nil, fmt.Errorf("storing the notification provider: %w", err)
	}

	created.Fields = m.masked(created)

	return created, nil
}

// UpdateProvider applies the named members to a provider. A secret that the input
// omits, or that holds the mask, keeps its stored value. A changed secret runs the
// check of the kind again.
func (m *Manager) UpdateProvider(
	ctx context.Context,
	id string,
	input ProviderInput,
	ifMatch *time.Time,
) (*model.NotificationProvider, error) {
	if input.Kind != nil {
		return nil, &FieldError{Pointer: pointerKind, Detail: detailNeverChanges}
	}

	stored, err := m.storedProvider(ctx, id)
	if err != nil {
		return nil, err
	}

	if ifMatch != nil && !stored.UpdatedAt.Equal(*ifMatch) {
		return nil, precondition.ErrModified
	}

	changed, err := m.changedProvider(ctx, stored, input)
	if err != nil {
		return nil, err
	}

	var written *model.NotificationProvider

	err = database.WithScopeTx(ctx, m.db, func(tx *sqlx.Tx) error {
		var updateErr error

		written, updateErr = repository.NewNotificationProvider(tx).
			Update(ctx, changed, stored.UpdatedAt)

		return updateErr //nolint:wrapcheck // the repository named the failure
	})
	if err != nil {
		return nil, fmt.Errorf("changing the notification provider: %w", err)
	}

	written.Fields = m.masked(written)

	return written, nil
}

// DeleteProvider removes a provider of the acting workspace and each of its routes.
func (m *Manager) DeleteProvider(ctx context.Context, id string) error {
	err := database.WithScopeTx(ctx, m.db, func(tx *sqlx.Tx) error {
		return repository.NewNotificationProvider(tx).Delete(ctx, id)
	})
	if err != nil {
		return fmt.Errorf("removing the notification provider: %w", err)
	}

	return nil
}

// Topics answers each topic of each plugin that the acting workspace enables, ordered
// by the plugin identifier and then by the declared order. A route whose plugin the
// workspace does not enable, or whose topic the plugin no longer declares, reaches no
// item, and its row stays.
func (m *Manager) Topics(ctx context.Context) ([]TopicView, error) {
	enabled, err := m.enabledPlugins(ctx)
	if err != nil {
		return nil, err
	}

	var (
		routes    []model.NotificationRoute
		providers []model.NotificationProvider
	)

	err = database.WithScopeTx(ctx, m.db, func(tx *sqlx.Tx) error {
		var readErr error

		if routes, readErr = repository.NewNotificationRoute(tx).List(ctx); readErr != nil {
			return readErr //nolint:wrapcheck // the repository named the failure
		}

		providers, readErr = repository.NewNotificationProvider(tx).List(ctx)

		return readErr //nolint:wrapcheck // the repository named the failure
	})
	if err != nil {
		return nil, fmt.Errorf("reading the notification routes: %w", err)
	}

	providerEnabled := make(map[string]bool, len(providers))
	for _, provider := range providers {
		providerEnabled[provider.ID] = provider.Enabled
	}

	views := []TopicView{}

	for _, pluginID := range enabled {
		for _, entry := range m.topics.Of(pluginID) {
			views = append(views, viewOf(pluginID, entry.Topic, routes, providerEnabled))
		}
	}

	return views, nil
}

// Routes answers the routes that Topics shows, in the order of the identifiers: those of
// an enabled plugin and of a topic that the plugin declares.
func (m *Manager) Routes(ctx context.Context) ([]model.NotificationRoute, error) {
	enabled, err := m.enabledPlugins(ctx)
	if err != nil {
		return nil, err
	}

	var routes []model.NotificationRoute

	err = database.WithScopeTx(ctx, m.db, func(tx *sqlx.Tx) error {
		var listErr error

		routes, listErr = repository.NewNotificationRoute(tx).List(ctx)

		return listErr //nolint:wrapcheck // the repository named the failure
	})
	if err != nil {
		return nil, fmt.Errorf("listing the notification routes: %w", err)
	}

	return slices.DeleteFunc(routes, func(route model.NotificationRoute) bool {
		_, declared := m.topics.Get(route.PluginID, route.TopicID)

		return !declared || !slices.Contains(enabled, route.PluginID)
	}), nil
}

// Route answers one route of the acting workspace.
func (m *Manager) Route(ctx context.Context, id string) (*model.NotificationRoute, error) {
	route, _, err := m.storedRoute(ctx, id)
	if err != nil {
		return nil, err
	}

	return route, nil
}

// viewOf gathers the routes of one topic. The topic is unrouted when no enabled route
// passes through an enabled provider.
func viewOf(
	pluginID string,
	topic pluginapi.NotificationTopic,
	routes []model.NotificationRoute,
	providerEnabled map[string]bool,
) TopicView {
	view := TopicView{PluginID: pluginID, Topic: topic, Routes: []model.NotificationRoute{}}

	for _, route := range routes {
		if route.PluginID == pluginID && route.TopicID == topic.ID {
			view.Routes = append(view.Routes, route)
		}
	}

	view.Unrouted = !slices.ContainsFunc(view.Routes, func(route model.NotificationRoute) bool {
		return route.Enabled && providerEnabled[route.ProviderID]
	})

	return view
}

// CreateRoute stores a route of a declared topic of an enabled plugin, through a
// provider of the acting workspace.
func (m *Manager) CreateRoute(
	ctx context.Context,
	input RouteInput,
) (*model.NotificationRoute, error) {
	pluginID, topicID, err := m.routeTopicOf(ctx, input)
	if err != nil {
		return nil, err
	}

	if input.ProviderID == nil || *input.ProviderID == "" {
		return nil, &FieldError{Pointer: pointerProviderID, Detail: detailRequired}
	}

	if uuid.Validate(*input.ProviderID) != nil {
		return nil, &FieldError{Pointer: pointerProviderID, Detail: "the value is not a UUID"}
	}

	provider, err := m.storedProvider(ctx, *input.ProviderID)
	if err != nil {
		return nil, providerFieldError(err)
	}

	destination, err := m.destinationOf(provider.Kind, input.Destination, nil)
	if err != nil {
		return nil, err
	}

	var created *model.NotificationRoute

	err = database.WithScopeTx(ctx, m.db, func(tx *sqlx.Tx) error {
		var createErr error

		created, createErr = repository.NewNotificationRoute(tx).
			Create(ctx, &model.NotificationRoute{
				ProviderID:  provider.ID,
				PluginID:    pluginID,
				TopicID:     topicID,
				Destination: destination,
				Enabled:     enabledOf(input.Enabled),
			})

		return createErr //nolint:wrapcheck // the repository named the failure
	})
	if err != nil {
		return nil, providerFieldError(fmt.Errorf("storing the notification route: %w", err))
	}

	return created, nil
}

// UpdateRoute merges the destination of the input into the stored one, and sets the
// enabled state. A null member of the destination removes that field.
func (m *Manager) UpdateRoute(
	ctx context.Context,
	id string,
	input RouteInput,
	ifMatch *time.Time,
) (*model.NotificationRoute, error) {
	if pointer := fixedRouteMember(input); pointer != "" {
		return nil, &FieldError{Pointer: pointer, Detail: detailNeverChanges}
	}

	stored, provider, err := m.storedRoute(ctx, id)
	if err != nil {
		return nil, err
	}

	if ifMatch != nil && !stored.UpdatedAt.Equal(*ifMatch) {
		return nil, precondition.ErrModified
	}

	changed := *stored

	if input.Destination != nil {
		changed.Destination, err = m.destinationOf(
			provider.Kind,
			input.Destination,
			stored.Destination,
		)
		if err != nil {
			return nil, err
		}
	}

	if input.Enabled != nil {
		changed.Enabled = *input.Enabled
	}

	var written *model.NotificationRoute

	err = database.WithScopeTx(ctx, m.db, func(tx *sqlx.Tx) error {
		var updateErr error

		written, updateErr = repository.NewNotificationRoute(tx).
			Update(ctx, &changed, stored.UpdatedAt)

		return updateErr //nolint:wrapcheck // the repository named the failure
	})
	if err != nil {
		return nil, fmt.Errorf("changing the notification route: %w", err)
	}

	return written, nil
}

// DeleteRoute removes a route of the acting workspace.
func (m *Manager) DeleteRoute(ctx context.Context, id string) error {
	err := database.WithScopeTx(ctx, m.db, func(tx *sqlx.Tx) error {
		return repository.NewNotificationRoute(tx).Delete(ctx, id)
	})
	if err != nil {
		return fmt.Errorf("removing the notification route: %w", err)
	}

	return nil
}

// TestRoute sends a fixed message through the route, whatever its enabled state and
// the enabled state of its provider. It stores nothing.
func (m *Manager) TestRoute(ctx context.Context, id string) error {
	route, provider, err := m.storedRoute(ctx, id)
	if err != nil {
		return err
	}

	entry, found := m.kinds.Get(provider.Kind)
	if !found {
		return fmt.Errorf(
			"%w: the kind %q is not registered",
			errs.ErrInvalidNotificationKind,
			provider.Kind,
		)
	}

	values, err := Reveal(ctx, m.cipher, pluginapi.ActingWorkspaceFromContext(ctx), provider.Fields)
	if err != nil {
		return err
	}

	topicName := route.TopicID
	if declared, held := m.topics.Get(route.PluginID, route.TopicID); held {
		topicName = declared.Topic.Name
	}

	msg := pluginapi.Message{
		Title: testTitle,
		Body:  "This message tests the route of " + topicName + " of " + route.PluginID + ".",
	}

	if err = entry.Kind.Send(ctx, values, route.Destination, msg); err != nil {
		return fmt.Errorf("sending the test message: %w", err)
	}

	return nil
}

func (m *Manager) kindOf(id *string) (KindEntry, error) {
	if id == nil || *id == "" {
		return KindEntry{}, &FieldError{Pointer: pointerKind, Detail: detailRequired}
	}

	entry, found := m.kinds.Get(*id)
	if !found {
		return KindEntry{}, &FieldError{Pointer: pointerKind, Detail: "the kind is unknown"}
	}

	return entry, nil
}

// changedProvider builds the row that a change writes, and checks a changed secret at
// the external messaging service.
func (m *Manager) changedProvider(
	ctx context.Context,
	stored *model.NotificationProvider,
	input ProviderInput,
) (*model.NotificationProvider, error) {
	changed := *stored

	if input.Name != nil {
		name, err := nameOf(input.Name)
		if err != nil {
			return nil, err
		}

		changed.Name = name
	}

	if input.Enabled != nil {
		changed.Enabled = *input.Enabled
	}

	if input.Fields == nil {
		return &changed, nil
	}

	fields, err := m.changedFields(ctx, stored, input.Fields)
	if err != nil {
		return nil, err
	}

	changed.Fields = fields

	return &changed, nil
}

// changedFields merges the input into the stored fields, and checks a changed secret at
// the external messaging service.
func (m *Manager) changedFields(
	ctx context.Context,
	stored *model.NotificationProvider,
	input map[string]any,
) (model.Fields, error) {
	entry, found := m.kinds.Get(stored.Kind)
	if !found {
		return nil, fmt.Errorf(
			"%w: the kind %q is not registered",
			errs.ErrInvalidNotificationKind,
			stored.Kind,
		)
	}

	current, err := Reveal(ctx, m.cipher, pluginapi.ActingWorkspaceFromContext(ctx), stored.Fields)
	if err != nil {
		return nil, err
	}

	values, err := checkValues(entry.Provider, input, current, fieldsPrefix)
	if err != nil {
		return nil, err
	}

	if secretChanged(entry.Provider, current, values) {
		if err = entry.Kind.Verify(ctx, values); err != nil {
			return nil, fmt.Errorf("checking the credential: %w", err)
		}
	}

	return m.protect(ctx, entry.Provider, values)
}

func (m *Manager) routeTopicOf(ctx context.Context, input RouteInput) (string, string, error) {
	if input.PluginID == nil || *input.PluginID == "" {
		return "", "", &FieldError{Pointer: pointerPluginID, Detail: detailRequired}
	}

	if input.TopicID == nil || *input.TopicID == "" {
		return "", "", &FieldError{Pointer: pointerTopicID, Detail: detailRequired}
	}

	enabled, err := m.enabledPlugins(ctx)
	if err != nil {
		return "", "", err
	}

	if !slices.Contains(enabled, *input.PluginID) {
		return "", "", &FieldError{
			Pointer: pointerPluginID,
			Detail:  "the workspace does not enable the plugin",
		}
	}

	if _, declared := m.topics.Get(*input.PluginID, *input.TopicID); !declared {
		return "", "", &FieldError{
			Pointer: pointerTopicID,
			Detail:  "the plugin does not declare the topic",
		}
	}

	return *input.PluginID, *input.TopicID, nil
}

// destinationOf merges the input into the stored destination and validates the
// result against the destination schema of the kind.
func (m *Manager) destinationOf(
	kindID string,
	input map[string]any,
	stored model.Destination,
) (model.Destination, error) {
	entry, found := m.kinds.Get(kindID)
	if !found {
		return nil, fmt.Errorf(
			"%w: the kind %q is not registered",
			errs.ErrInvalidNotificationKind,
			kindID,
		)
	}

	values, err := checkValues(entry.Destination, input, stored, destinationPrefix)
	if err != nil {
		return nil, err
	}

	return model.Destination(values), nil
}

func (m *Manager) enabledPlugins(ctx context.Context) ([]string, error) {
	enablements, err := m.enablements.Enabled(ctx)
	if err != nil {
		return nil, fmt.Errorf("reading the enabled plugins: %w", err)
	}

	plugins := make([]string, 0, len(enablements))
	for _, enablement := range enablements {
		plugins = append(plugins, enablement.PluginID)
	}

	slices.Sort(plugins)

	return plugins, nil
}

func (m *Manager) storedProvider(
	ctx context.Context,
	id string,
) (*model.NotificationProvider, error) {
	var provider *model.NotificationProvider

	err := database.WithScopeTx(ctx, m.db, func(tx *sqlx.Tx) error {
		var getErr error

		provider, getErr = repository.NewNotificationProvider(tx).Get(ctx, id)

		return getErr //nolint:wrapcheck // the repository named the failure
	})
	if err != nil {
		return nil, fmt.Errorf("reading the notification provider: %w", err)
	}

	return provider, nil
}

func (m *Manager) storedRoute(
	ctx context.Context,
	id string,
) (*model.NotificationRoute, *model.NotificationProvider, error) {
	var (
		route    *model.NotificationRoute
		provider *model.NotificationProvider
	)

	err := database.WithScopeTx(ctx, m.db, func(tx *sqlx.Tx) error {
		var getErr error

		if route, getErr = repository.NewNotificationRoute(tx).Get(ctx, id); getErr != nil {
			return getErr //nolint:wrapcheck // the repository named the failure
		}

		provider, getErr = repository.NewNotificationProvider(tx).Get(ctx, route.ProviderID)

		return getErr //nolint:wrapcheck // the repository named the failure
	})
	if err != nil {
		return nil, nil, fmt.Errorf("reading the notification route: %w", err)
	}

	return route, provider, nil
}

// protect builds the stored fields: each secret takes the key of the acting workspace.
func (m *Manager) protect(
	ctx context.Context,
	schema *settings.Schema,
	values map[string]any,
) (model.Fields, error) {
	key := secret.WorkspaceKey(pluginapi.ActingWorkspaceFromContext(ctx))
	fields := make(model.Fields, len(values))

	for name, value := range values {
		kind := schema.Kinds[name]
		if kind != model.FieldKindSecret {
			fields[name] = model.SettingEntry{Kind: kind, Value: value}

			continue
		}

		plaintext, ok := value.(string)
		if !ok {
			return nil, &FieldError{
				Pointer: fieldsPrefix + "/" + name,
				Detail:  "the value has the wrong type",
			}
		}

		ciphertext, err := m.cipher.Encrypt(ctx, key, plaintext)
		if err != nil {
			return nil, fmt.Errorf("protecting the field %q: %w", name, err)
		}

		fields[name] = model.SettingEntry{Kind: model.FieldKindSecret, Value: ciphertext}
	}

	return fields, nil
}

// masked answers the fields that a reader sees: the mask for each held secret, and the
// empty string for each declared secret that holds no value.
func (m *Manager) masked(provider *model.NotificationProvider) model.Fields {
	fields := make(model.Fields, len(provider.Fields))

	for name, entry := range provider.Fields {
		if entry.Kind == model.FieldKindSecret {
			entry.Value = settings.SecretMask
		}

		fields[name] = entry
	}

	entry, found := m.kinds.Get(provider.Kind)
	if !found {
		return fields
	}

	for name, kind := range entry.Provider.Kinds {
		if _, held := fields[name]; !held && kind == model.FieldKindSecret {
			fields[name] = model.SettingEntry{Kind: model.FieldKindSecret, Value: ""}
		}
	}

	return fields
}

// checkValues merges the input into the current values and validates the result. A
// null or an empty text removes a field, and the mask keeps a secret. It answers the
// plaintext of every field that the result holds.
func checkValues(
	schema *settings.Schema,
	input map[string]any,
	current map[string]any,
	prefix string,
) (map[string]any, error) {
	if err := settings.Validate(schema, input); err != nil {
		return nil, prefixed(err, prefix)
	}

	values := mergeValues(schema, input, current)

	if missing := missingRequired(schema, values, prefix); len(missing) > 0 {
		return nil, &settings.InvalidError{Fields: missing}
	}

	return values, nil
}

// mergeValues applies the input to a copy of the current values.
func mergeValues(
	schema *settings.Schema,
	input map[string]any,
	current map[string]any,
) map[string]any {
	values := maps.Clone(current)
	if values == nil {
		values = map[string]any{}
	}

	for name, value := range input {
		switch {
		case schema.Kinds[name] == model.FieldKindSecret && value == settings.SecretMask:
			continue
		case value == nil || value == "":
			delete(values, name)
		default:
			values[name] = value
		}
	}

	return values
}

// missingRequired names each required field that the values do not hold.
func missingRequired(
	schema *settings.Schema,
	values map[string]any,
	prefix string,
) []settings.FieldFailure {
	var missing []settings.FieldFailure

	for _, name := range slices.Sorted(maps.Keys(schema.Required)) {
		if _, held := values[name]; !held {
			missing = append(missing, settings.FieldFailure{
				Pointer: "#" + prefix + "/" + name, Detail: detailRequired,
			})
		}
	}

	return missing
}

// prefixed moves each pointer of a rejected save under the member that carried it.
func prefixed(err error, prefix string) error {
	invalid, ok := err.(*settings.InvalidError) //nolint:errorlint // Validate answers it unwrapped
	if !ok {
		return fmt.Errorf("validating the fields: %w", err)
	}

	moved := make([]settings.FieldFailure, 0, len(invalid.Fields))
	for _, failure := range invalid.Fields {
		moved = append(moved, settings.FieldFailure{
			Pointer: "#" + prefix + strings.TrimPrefix(failure.Pointer, "#"),
			Detail:  failure.Detail,
		})
	}

	return &settings.InvalidError{Fields: moved}
}

func secretChanged(schema *settings.Schema, before map[string]any, after map[string]any) bool {
	for name, kind := range schema.Kinds {
		if kind == model.FieldKindSecret && before[name] != after[name] {
			return true
		}
	}

	return false
}

func nameOf(given *string) (string, error) {
	if given == nil {
		return "", &FieldError{Pointer: pointerName, Detail: detailRequired}
	}

	name := strings.TrimSpace(*given)

	switch length := utf8.RuneCountInString(name); {
	case length == 0:
		return "", &FieldError{Pointer: pointerName, Detail: "the value is empty"}
	case length > MaxNameLength:
		return "", &FieldError{Pointer: pointerName, Detail: "the value is too long"}
	default:
		return name, nil
	}
}

// fixedRouteMember names the first member of the input that a route never changes, in
// the order of the body, or answers the empty string.
func fixedRouteMember(input RouteInput) string {
	switch {
	case input.PluginID != nil:
		return pointerPluginID
	case input.TopicID != nil:
		return pointerTopicID
	case input.ProviderID != nil:
		return pointerProviderID
	default:
		return ""
	}
}

func enabledOf(given *bool) bool {
	return given == nil || *given
}

// providerFieldError reads an absent provider of a route as a refused member, because
// the route names it in its body and not in its path.
func providerFieldError(err error) error {
	if errors.Is(err, errs.ErrNotificationProviderNotFound) {
		return &FieldError{
			Pointer: pointerProviderID,
			Detail:  "the workspace holds no such provider",
		}
	}

	return err
}
