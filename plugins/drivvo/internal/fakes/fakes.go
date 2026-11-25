package fakes

import (
	"context"
	"sync"
	"time"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/plugins/drivvo/model"
)

// The workspaces and the credentials of the tests.
const (
	WorkspaceW = "0199c5e2-8f3a-7c21-9b4e-2f6a1d0c5e78"
	WorkspaceX = "0199c5e2-8f3a-7c21-9b4e-2f6a1d0c5e79"
	Email      = "driver@example.com"
	Password   = " secret "

	secretSession = "session"
)

// Acting answers a context that acts in the workspace.
func Acting(ctx context.Context, workspaceID string) context.Context {
	return pluginapi.ContextWithActingWorkspace(ctx, workspaceID)
}

// Secrets is a PluginSecrets in memory, keyed by the acting workspace and the name.
type Secrets struct {
	mu      sync.Mutex
	entries map[string]string
}

// NewSecrets answers secrets that hold no secret.
func NewSecrets() *Secrets {
	return &Secrets{entries: map[string]string{}}
}

// WithSession stores the session of the token, ending at the moment, for the workspace.
func (f *Secrets) WithSession(workspaceID, token string, expiresAt time.Time) *Secrets {
	stored, _ := model.Session{Token: token, ExpiresAt: expiresAt}.MarshalSecret()

	f.mu.Lock()
	defer f.mu.Unlock()

	f.entries[workspaceID+"/"+secretSession] = stored

	return f
}

func secretKey(ctx context.Context, name string) string {
	return pluginapi.ActingWorkspaceFromContext(ctx) + "/" + name
}

// Get answers the secret, or pluginapi.ErrSecretAbsent.
func (f *Secrets) Get(ctx context.Context, _ pluginapi.SecretScope, name string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	value, found := f.entries[secretKey(ctx, name)]
	if !found {
		return "", pluginapi.ErrSecretAbsent
	}

	return value, nil
}

// Put stores the secret.
func (f *Secrets) Put(ctx context.Context, scope pluginapi.SecretScope, name, value string) error {
	return f.PutUntil(ctx, scope, name, value, time.Time{})
}

// PutUntil stores the secret, and ignores the expiry.
func (f *Secrets) PutUntil(
	ctx context.Context,
	_ pluginapi.SecretScope,
	name string,
	value string,
	_ time.Time,
) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.entries[secretKey(ctx, name)] = value

	return nil
}

// Delete removes the secret.
func (f *Secrets) Delete(ctx context.Context, _ pluginapi.SecretScope, name string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	delete(f.entries, secretKey(ctx, name))

	return nil
}

// Session answers the stored session of the workspace, and false when none is stored.
func (f *Secrets) Session(workspaceID string) (model.Session, bool) {
	f.mu.Lock()
	stored, found := f.entries[workspaceID+"/"+secretSession]
	f.mu.Unlock()

	if !found {
		return model.Session{}, false
	}

	session, err := model.ParseSession(stored)

	return session, err == nil
}

// Settings answers the stored credentials, or pluginapi.ErrSettingsAbsent with none.
type Settings struct {
	Values map[string]any
}

// Get answers the values.
func (f *Settings) Get(context.Context) (map[string]any, error) {
	if f.Values == nil {
		return nil, pluginapi.ErrSettingsAbsent
	}

	return f.Values, nil
}

// Credentials answers the settings of the tests.
func Credentials() *Settings {
	return &Settings{Values: map[string]any{"email": Email, "password": Password}}
}

// Store keeps each snapshot that Replace received, by the acting workspace.
type Store struct {
	mu        sync.Mutex
	snapshots map[string][]model.Snapshot
}

// NewStore answers a store that received no snapshot.
func NewStore() *Store {
	return &Store{snapshots: map[string][]model.Snapshot{}}
}

// Replace keeps the snapshot.
func (f *Store) Replace(ctx context.Context, snapshot model.Snapshot) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	workspaceID := pluginapi.ActingWorkspaceFromContext(ctx)
	f.snapshots[workspaceID] = append(f.snapshots[workspaceID], snapshot)

	return nil
}

// Snapshots answers the snapshots of the workspace, in order.
func (f *Store) Snapshots(workspaceID string) []model.Snapshot {
	f.mu.Lock()
	defer f.mu.Unlock()

	return append([]model.Snapshot(nil), f.snapshots[workspaceID]...)
}
