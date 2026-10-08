// Package dextest holds a Dex in memory for the tests of the hub.
package dextest

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"sync"

	"github.com/abgeo/maroid/apps/hub/internal/dex"
	"github.com/abgeo/maroid/apps/hub/internal/domain/errs"
)

// Memory is a dex.Client that holds the connectors and the passwords in memory, and
// answers the flags of Dex as the real client does.
type Memory struct {
	mu         sync.Mutex
	connectors []dex.Connector
	passwords  []dex.Password
	failure    error
}

var _ dex.Client = (*Memory)(nil)

// New creates a Memory that holds the connectors.
func New(connectors ...dex.Connector) *Memory {
	return &Memory{connectors: connectors}
}

// Connector builds a connector with its config as JSON text.
func Connector(id string, kind string, name string, config string) dex.Connector {
	return dex.Connector{ID: id, Type: kind, Name: name, Config: json.RawMessage(config)}
}

// Fail makes every later call answer the error, or answer normally when it is nil.
func (m *Memory) Fail(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.failure = err
}

// ListConnectors answers every connector.
func (m *Memory) ListConnectors(_ context.Context) ([]dex.Connector, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failure != nil {
		return nil, m.failure
	}

	return slices.Clone(m.connectors), nil
}

// CreateConnector stores a connector, or answers that its identifier exists.
func (m *Memory) CreateConnector(_ context.Context, connector dex.Connector) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failure != nil {
		return m.failure
	}

	if m.connectorAt(connector.ID) >= 0 {
		return fmt.Errorf("creating the connector %s: %w", connector.ID, errs.ErrProviderExists)
	}

	m.connectors = append(m.connectors, connector)

	return nil
}

// UpdateConnector replaces a connector.
func (m *Memory) UpdateConnector(_ context.Context, connector dex.Connector) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failure != nil {
		return m.failure
	}

	at := m.connectorAt(connector.ID)
	if at < 0 {
		return fmt.Errorf("updating the connector %s: %w", connector.ID, errs.ErrProviderNotFound)
	}

	m.connectors[at] = connector

	return nil
}

// DeleteConnector removes a connector.
func (m *Memory) DeleteConnector(_ context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failure != nil {
		return m.failure
	}

	at := m.connectorAt(id)
	if at < 0 {
		return fmt.Errorf("deleting the connector %s: %w", id, errs.ErrProviderNotFound)
	}

	m.connectors = slices.Delete(m.connectors, at, at+1)

	return nil
}

// ListPasswords answers every password, with no hash, as Dex does.
func (m *Memory) ListPasswords(_ context.Context) ([]dex.Password, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failure != nil {
		return nil, m.failure
	}

	passwords := make([]dex.Password, 0, len(m.passwords))
	for _, one := range m.passwords {
		one.Hash = nil
		passwords = append(passwords, one)
	}

	return passwords, nil
}

// CreatePassword stores a password, or answers that its email address exists.
func (m *Memory) CreatePassword(_ context.Context, password dex.Password) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failure != nil {
		return m.failure
	}

	if m.passwordAt(password.Email) >= 0 {
		return fmt.Errorf("creating a password: %w", errs.ErrLocalAccountExists)
	}

	m.passwords = append(m.passwords, password)

	return nil
}

// UpdatePassword replaces the hash of a password.
func (m *Memory) UpdatePassword(_ context.Context, email string, hash []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failure != nil {
		return m.failure
	}

	at := m.passwordAt(email)
	if at < 0 {
		return fmt.Errorf("updating a password: %w", errs.ErrLocalAccountNotFound)
	}

	m.passwords[at].Hash = hash

	return nil
}

// DeletePassword removes a password.
func (m *Memory) DeletePassword(_ context.Context, email string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failure != nil {
		return m.failure
	}

	at := m.passwordAt(email)
	if at < 0 {
		return fmt.Errorf("deleting a password: %w", errs.ErrLocalAccountNotFound)
	}

	m.passwords = slices.Delete(m.passwords, at, at+1)

	return nil
}

// Hash answers the hash that a password holds, which the real Dex never answers.
func (m *Memory) Hash(email string) []byte {
	m.mu.Lock()
	defer m.mu.Unlock()

	if at := m.passwordAt(email); at >= 0 {
		return m.passwords[at].Hash
	}

	return nil
}

func (m *Memory) connectorAt(id string) int {
	return slices.IndexFunc(m.connectors, func(one dex.Connector) bool { return one.ID == id })
}

func (m *Memory) passwordAt(email string) int {
	return slices.IndexFunc(m.passwords, func(one dex.Password) bool { return one.Email == email })
}
