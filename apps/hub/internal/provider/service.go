package provider

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/abgeo/maroid/apps/hub/internal/dex"
	"github.com/abgeo/maroid/apps/hub/internal/domain/errs"
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
}

// Service reads the providers of the instance.
type Service interface {
	List(ctx context.Context) ([]Provider, error)
	Get(ctx context.Context, id string) (*Provider, error)
}

// Manager is the Service that Dex backs. It caches nothing, so a change in Dex
// shows at the next read.
type Manager struct {
	client      dex.Client
	redirectURI string
}

var _ Service = (*Manager)(nil)

// NewManager creates a Manager. The issuer is the address of Dex, and the redirect
// address of every OIDC provider derives from it.
func NewManager(client dex.Client, issuer string) *Manager {
	return &Manager{
		client:      client,
		redirectURI: strings.TrimSuffix(issuer, "/") + "/callback",
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

	return providers, nil
}

// Get answers one provider.
func (m *Manager) Get(ctx context.Context, id string) (*Provider, error) {
	providers, err := m.List(ctx)
	if err != nil {
		return nil, err
	}

	for i := range providers {
		if providers[i].ID == id {
			return &providers[i], nil
		}
	}

	return nil, fmt.Errorf("reading the provider %s: %w", id, errs.ErrProviderNotFound)
}

func (m *Manager) read(connector dex.Connector) Provider {
	read := Provider{ID: connector.ID, Name: connector.Name, Static: true}

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

	for _, fixed := range []string{
		MarkerKey, keyIssuer, keyClientID, keyClientSecret, keyRedirectURI, keyUserIDKey, keyScopes,
	} {
		delete(config, fixed)
	}

	if len(config) > 0 {
		read.Options = config
	}

	return read
}

// text answers the string that a member holds, or nothing when it holds another value.
func text(member json.RawMessage) string {
	var value string

	_ = json.Unmarshal(member, &value)

	return value
}
