package provider

import (
	"cmp"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"slices"
	"strings"

	"github.com/abgeo/maroid/apps/hub/internal/dex"
	"github.com/abgeo/maroid/apps/hub/internal/domain/errs"
	"github.com/abgeo/maroid/apps/hub/internal/model"
	"github.com/abgeo/maroid/libs/rest/precondition"
)

// Provider is one provider of the instance, read from a connector of Dex. It holds
// no client secret.
type Provider struct {
	ID              string
	Name            string
	Preset          Preset
	Static          bool
	Issuer          string
	ClientID        string
	ClientSecretSet bool
	UserIDKey       string
	Scopes          []string
	Options         map[string]json.RawMessage
	RedirectURI     string
	// Version moves with each change of the name or the config in Dex.
	Version int64
	// IdentityCount is the count of the identities that a removal deletes.
	IdentityCount int
	// AdministratorsWithoutSignIn holds each active administrator whose identities all
	// belong to this provider, so a removal leaves them with no sign in.
	AdministratorsWithoutSignIn []model.User
}

// Input is a new provider. A nil member is absent from the request.
type Input struct {
	Preset       Preset
	ID           *string
	Name         *string
	Issuer       *string
	ClientID     *string
	ClientSecret *string
	UserIDKey    *string
	Scopes       *[]string
	Options      map[string]json.RawMessage
}

// Change is a merge patch of a provider. A nil member keeps its value, and a null
// option removes that option.
type Change struct {
	Name         *string
	ClientID     *string
	ClientSecret *string
	Scopes       *[]string
	Options      map[string]json.RawMessage
}

// Service reads, adds, and changes the providers of the instance.
type Service interface {
	List(ctx context.Context) ([]Provider, error)
	Get(ctx context.Context, id string) (*Provider, error)
	Create(ctx context.Context, input Input) (*Provider, error)
	// Change refuses a provider whose version moved when ifMatch names one.
	Change(ctx context.Context, id string, change Change, ifMatch *int64) (*Provider, error)
	// Remove deletes a stored provider and every identity of it, together or not at all.
	Remove(ctx context.Context, id string) error
}

// Identities reads and deletes the identities that belong to a provider.
type Identities interface {
	CountByProvider(ctx context.Context) (map[string]int, error)
	AdministratorsBySoleProvider(ctx context.Context) (map[string][]model.User, error)
	DeleteByProvider(
		ctx context.Context,
		provider string,
		beforeCommit func(context.Context) error,
	) error
}

// Settings holds what the presets read from the configuration of the hub.
type Settings struct {
	// Issuer is the address of Dex. The redirect address of every OIDC provider
	// derives from it.
	Issuer string
	// TelegramBotID is the client identifier that a Telegram provider takes when the
	// request names none.
	TelegramBotID string
	Discoverer    Discoverer
}

// Manager is the Service that Dex backs. It caches nothing, so a change in Dex
// shows at the next read.
type Manager struct {
	client        dex.Client
	identities    Identities
	redirectURI   string
	telegramBotID string
	discoverer    Discoverer
}

var _ Service = (*Manager)(nil)

// NewManager creates a Manager.
func NewManager(client dex.Client, identities Identities, settings Settings) *Manager {
	return &Manager{
		client:        client,
		identities:    identities,
		redirectURI:   strings.TrimSuffix(settings.Issuer, "/") + "/callback",
		telegramBotID: settings.TelegramBotID,
		discoverer:    settings.Discoverer,
	}
}

// List answers every provider, ordered by the identifier.
func (m *Manager) List(ctx context.Context) ([]Provider, error) {
	connectors, err := m.client.ListConnectors(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing the providers: %w", err)
	}

	providers := make([]Provider, 0, len(connectors))
	for _, connector := range connectors {
		providers = append(providers, m.read(connector))
	}

	slices.SortFunc(providers, func(a, b Provider) int { return cmp.Compare(a.ID, b.ID) })

	if err = m.report(ctx, providers); err != nil {
		return nil, err
	}

	return providers, nil
}

// Get answers one provider.
func (m *Manager) Get(ctx context.Context, id string) (*Provider, error) {
	connector, err := m.find(ctx, id)
	if err != nil {
		return nil, err
	}

	providers := []Provider{m.read(connector)}
	if err = m.report(ctx, providers); err != nil {
		return nil, err
	}

	return &providers[0], nil
}

// Create adds a provider of one preset. Dex accepts any config, so every check runs
// here before the write.
func (m *Manager) Create(ctx context.Context, input Input) (*Provider, error) {
	connector, err := m.build(input)
	if err != nil {
		return nil, err
	}

	if _, err = m.find(ctx, connector.ID); err == nil {
		return nil, fmt.Errorf("adding the provider %s: %w", connector.ID, errs.ErrProviderExists)
	} else if !errors.Is(err, errs.ErrProviderNotFound) {
		return nil, err
	}

	if input.Preset != PresetLocal {
		if err = m.discoverer.Discover(ctx, issuerOf(input)); err != nil {
			return nil, fieldError("/issuer", "the issuer publishes no OIDC discovery document")
		}
	}

	if err = m.client.CreateConnector(ctx, connector); err != nil {
		return nil, fmt.Errorf("adding the provider %s: %w", connector.ID, err)
	}

	providers := []Provider{m.read(connector)}
	if err = m.report(ctx, providers); err != nil {
		return nil, err
	}

	return &providers[0], nil
}

// Change applies a merge patch to a provider that the hub stored.
func (m *Manager) Change(
	ctx context.Context,
	id string,
	change Change,
	ifMatch *int64,
) (*Provider, error) {
	connector, err := m.find(ctx, id)
	if err != nil {
		return nil, err
	}

	current := m.read(connector)
	if current.Static {
		return nil, fmt.Errorf("changing the provider %s: %w", id, errs.ErrProviderStatic)
	}

	if ifMatch != nil && *ifMatch != current.Version {
		return nil, fmt.Errorf("changing the provider %s: %w", id, precondition.ErrModified)
	}

	var config map[string]json.RawMessage
	if err = json.Unmarshal(connector.Config, &config); err != nil {
		return nil, fmt.Errorf("reading the config of %s: %w", id, err)
	}

	if connector.Name, err = applyChange(
		current.Preset,
		config,
		change,
		connector.Name,
	); err != nil {
		return nil, err
	}

	if connector.Config, err = json.Marshal(config); err != nil {
		return nil, fmt.Errorf("writing the config of %s: %w", id, err)
	}

	if err = m.client.UpdateConnector(ctx, connector); err != nil {
		return nil, fmt.Errorf("changing the provider %s: %w", id, err)
	}

	providers := []Provider{m.read(connector)}
	if err = m.report(ctx, providers); err != nil {
		return nil, err
	}

	return &providers[0], nil
}

// Remove deletes a stored provider and every identity of it. A local provider takes
// every local account with it, because each local connector reads one password store.
func (m *Manager) Remove(ctx context.Context, id string) error {
	connector, err := m.find(ctx, id)
	if errors.Is(err, errs.ErrProviderNotFound) {
		return m.removeLeftovers(ctx, id, err)
	}

	if err != nil {
		return err
	}

	removed := m.read(connector)
	if removed.Static {
		return fmt.Errorf("removing the provider %s: %w", id, errs.ErrProviderStatic)
	}

	err = m.identities.DeleteByProvider(ctx, id, func(ctx context.Context) error {
		if removed.Preset == PresetLocal {
			if err := m.deletePasswords(ctx); err != nil {
				return err
			}
		}

		return m.client.DeleteConnector(ctx, id)
	})
	if err != nil {
		return fmt.Errorf("removing the provider %s: %w", id, err)
	}

	return nil
}

// removeLeftovers deletes the identities of a provider that Dex no longer holds. A
// commit that failed after Dex removed the connector leaves them, and a retry ends
// here.
func (m *Manager) removeLeftovers(ctx context.Context, id string, absent error) error {
	counts, err := m.identities.CountByProvider(ctx)
	if err != nil {
		return fmt.Errorf("counting the identities of %s: %w", id, err)
	}

	if counts[id] == 0 {
		return absent
	}

	err = m.identities.DeleteByProvider(ctx, id, func(context.Context) error { return nil })
	if err != nil {
		return fmt.Errorf("removing the identities of %s: %w", id, err)
	}

	return nil
}

// deletePasswords removes every local account before the connector goes, so a retry
// after a failure still finds the connector to remove.
func (m *Manager) deletePasswords(ctx context.Context) error {
	passwords, err := m.client.ListPasswords(ctx)
	if err != nil {
		return fmt.Errorf("listing the local accounts: %w", err)
	}

	for _, password := range passwords {
		if err = m.client.DeletePassword(ctx, password.Email); err != nil {
			return fmt.Errorf("deleting a local account: %w", err)
		}
	}

	return nil
}

// report writes the count of identities and the administrators who would hold no sign
// in into each provider.
func (m *Manager) report(ctx context.Context, providers []Provider) error {
	counts, err := m.identities.CountByProvider(ctx)
	if err != nil {
		return fmt.Errorf("counting the identities of each provider: %w", err)
	}

	sole, err := m.identities.AdministratorsBySoleProvider(ctx)
	if err != nil {
		return fmt.Errorf("reading the administrators of each provider: %w", err)
	}

	for i := range providers {
		providers[i].IdentityCount = counts[providers[i].ID]
		providers[i].AdministratorsWithoutSignIn = sole[providers[i].ID]
	}

	return nil
}

func (m *Manager) find(ctx context.Context, id string) (dex.Connector, error) {
	connectors, err := m.client.ListConnectors(ctx)
	if err != nil {
		return dex.Connector{}, fmt.Errorf("listing the providers: %w", err)
	}

	for _, connector := range connectors {
		if connector.ID == id {
			return connector, nil
		}
	}

	return dex.Connector{}, fmt.Errorf("reading the provider %s: %w", id, errs.ErrProviderNotFound)
}

func (m *Manager) read(connector dex.Connector) Provider {
	read := Provider{
		ID:      connector.ID,
		Name:    connector.Name,
		Static:  true,
		Version: version(connector),
	}

	var config map[string]json.RawMessage
	if json.Unmarshal(connector.Config, &config) != nil {
		return read
	}

	var preset Preset
	if json.Unmarshal(config[MarkerKey], &preset) != nil || !preset.known() {
		return read
	}

	read.Static = false
	read.Preset = preset

	if preset == PresetLocal {
		return read
	}

	read.Issuer = text(config[keyIssuer])
	read.ClientID = text(config[keyClientID])
	read.ClientSecretSet = text(config[keyClientSecret]) != ""
	read.UserIDKey = text(config[keyUserIDKey])
	read.RedirectURI = m.redirectURI

	if preset == PresetOIDC {
		_ = json.Unmarshal(config[keyScopes], &read.Scopes)
	}

	if options := optionsOf(config); len(options) > 0 {
		read.Options = options
	}

	return read
}

// version answers a positive integer from the SHA-256 of the name and the config, so
// that the entity tag of precondition carries it.
func version(connector dex.Connector) int64 {
	digest := sha256.New()
	digest.Write([]byte(connector.Name))
	digest.Write([]byte{0})
	digest.Write(connector.Config)

	return int64(binary.BigEndian.Uint64(digest.Sum(nil)[:8]) & math.MaxInt64)
}

// optionsOf answers the members of a config that no field of a preset covers.
func optionsOf(config map[string]json.RawMessage) map[string]json.RawMessage {
	options := make(map[string]json.RawMessage, len(config))

	for key, value := range config {
		if !slices.Contains(FixedKeys(), key) {
			options[key] = value
		}
	}

	return options
}

// text answers the string that a member holds, or nothing when it holds another value.
func text(member json.RawMessage) string {
	var value string

	_ = json.Unmarshal(member, &value)

	return value
}
