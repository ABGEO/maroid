package dex_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/dexidp/dex/api/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/apps/hub/internal/dex"
	"github.com/abgeo/maroid/apps/hub/internal/domain/errs"
)

const (
	ninaEmail = "nina@home.example"
	typeOIDC  = "oidc"
)

// fakeDex answers each call from its fields. A nil answer means the call was not
// expected.
type fakeDex struct {
	api.UnimplementedDexServer

	connectors []*api.Connector
	hold       time.Duration
	created    *api.Connector
	updated    *api.UpdateConnectorReq
	password   *api.Password
	conflict   bool
}

func (f *fakeDex) ListConnectors(
	ctx context.Context,
	_ *api.ListConnectorReq,
) (*api.ListConnectorResp, error) {
	select {
	case <-time.After(f.hold):
	case <-ctx.Done():
		return nil, fmt.Errorf("holding the call: %w", ctx.Err())
	}

	return &api.ListConnectorResp{Connectors: f.connectors}, nil
}

func (f *fakeDex) CreateConnector(
	_ context.Context,
	request *api.CreateConnectorReq,
) (*api.CreateConnectorResp, error) {
	f.created = request.GetConnector()

	return &api.CreateConnectorResp{AlreadyExists: f.conflict}, nil
}

func (f *fakeDex) UpdateConnector(
	_ context.Context,
	request *api.UpdateConnectorReq,
) (*api.UpdateConnectorResp, error) {
	f.updated = request

	return &api.UpdateConnectorResp{NotFound: f.conflict}, nil
}

func (f *fakeDex) DeleteConnector(
	_ context.Context,
	_ *api.DeleteConnectorReq,
) (*api.DeleteConnectorResp, error) {
	return &api.DeleteConnectorResp{NotFound: f.conflict}, nil
}

func (f *fakeDex) CreatePassword(
	_ context.Context,
	request *api.CreatePasswordReq,
) (*api.CreatePasswordResp, error) {
	f.password = request.GetPassword()

	return &api.CreatePasswordResp{AlreadyExists: f.conflict}, nil
}

func (f *fakeDex) UpdatePassword(
	_ context.Context,
	_ *api.UpdatePasswordReq,
) (*api.UpdatePasswordResp, error) {
	return &api.UpdatePasswordResp{NotFound: f.conflict}, nil
}

func (f *fakeDex) DeletePassword(
	_ context.Context,
	_ *api.DeletePasswordReq,
) (*api.DeletePasswordResp, error) {
	return &api.DeletePasswordResp{NotFound: f.conflict}, nil
}

func (f *fakeDex) ListPasswords(
	_ context.Context,
	_ *api.ListPasswordReq,
) (*api.ListPasswordResp, error) {
	return &api.ListPasswordResp{Passwords: []*api.Password{f.password}}, nil
}

func clientOf(t *testing.T, fake *fakeDex) *dex.GRPC {
	t.Helper()

	client, err := dex.New(startDex(t, fake))
	require.NoError(t, err)

	t.Cleanup(func() { _ = client.Close() })

	return client
}

// IDPROV-SC-001: The client reads every connector that Dex holds, over mutual TLS,
// with its raw config.
func TestTheClientListsEveryConnector(t *testing.T) {
	t.Parallel()

	client := clientOf(t, &fakeDex{connectors: []*api.Connector{
		{Id: "mock", Type: "mockCallback", Name: "Mock", Config: []byte(`{}`)},
		{
			Id:     "telegram",
			Type:   typeOIDC,
			Name:   "Telegram",
			Config: []byte(`{"maroidPreset":"telegram"}`),
		},
	}})

	connectors, err := client.ListConnectors(t.Context())
	require.NoError(t, err)
	require.Len(t, connectors, 2)

	assert.Equal(t, "mock", connectors[0].ID)
	assert.Equal(t, "mockCallback", connectors[0].Type)
	assert.Equal(t, "Telegram", connectors[1].Name)
	assert.JSONEq(t, `{"maroidPreset":"telegram"}`, string(connectors[1].Config))
}

// IDPROV-SC-029: A call that Dex holds for 6 seconds ends at the deadline of
// 5 seconds, as the unavailability of the IdP.
func TestACallPastTheDeadlineEndsAsUnavailable(t *testing.T) {
	t.Parallel()

	client := clientOf(t, &fakeDex{hold: 6 * time.Second})

	started := time.Now()
	_, err := client.ListConnectors(t.Context())
	elapsed := time.Since(started)

	require.ErrorIs(t, err, errs.ErrIDPUnavailable)
	assert.GreaterOrEqual(t, elapsed, 5*time.Second)
	assert.Less(t, elapsed, 5500*time.Millisecond)
}

// IDPROV-SC-029: A Dex that refuses the connection is unavailable too.
func TestARefusedConnectionEndsAsUnavailable(t *testing.T) {
	t.Parallel()

	cfg := startDex(t, &fakeDex{})

	listener, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp", "127.0.0.1:0")
	require.NoError(t, err)

	cfg.Address = listener.Addr().String()
	require.NoError(t, listener.Close())

	client, err := dex.New(cfg)
	require.NoError(t, err)

	t.Cleanup(func() { _ = client.Close() })

	_, err = client.ListConnectors(t.Context())
	require.ErrorIs(t, err, errs.ErrIDPUnavailable)
}

// IDPROV-SC-003: Dex reports an identifier that exists in a successful answer. The
// client turns the flag into the error of the hub.
func TestAConnectorThatExistsIsAnError(t *testing.T) {
	t.Parallel()

	fake := &fakeDex{conflict: true}
	client := clientOf(t, fake)

	err := client.CreateConnector(t.Context(), dex.Connector{
		ID: "abgeo-cloud", Type: typeOIDC, Name: "ABGEO.cloud", Config: json.RawMessage(`{}`),
	})
	require.ErrorIs(t, err, errs.ErrProviderExists)

	err = client.UpdateConnector(t.Context(), dex.Connector{
		ID: "gone", Type: typeOIDC, Name: "Gone", Config: json.RawMessage(`{}`),
	})
	require.ErrorIs(t, err, errs.ErrProviderNotFound)

	require.ErrorIs(t, client.DeleteConnector(t.Context(), "gone"), errs.ErrProviderNotFound)
}

// IDPROV-SC-009: A change sends the name and the whole config, so the stored config
// is exactly what the hub built.
func TestAChangeSendsTheNameAndTheWholeConfig(t *testing.T) {
	t.Parallel()

	fake := &fakeDex{}
	client := clientOf(t, fake)

	require.NoError(t, client.UpdateConnector(t.Context(), dex.Connector{
		ID: "abgeo-cloud", Type: typeOIDC, Name: "Cloud", Config: json.RawMessage(`{"a":1}`),
	}))

	require.NotNil(t, fake.updated)
	assert.Equal(t, "abgeo-cloud", fake.updated.GetId())
	assert.Equal(t, "Cloud", fake.updated.GetNewName())
	assert.Equal(t, typeOIDC, fake.updated.GetNewType())
	assert.JSONEq(t, `{"a":1}`, string(fake.updated.GetNewConfig()))
}

// IDPROV-SC-017: Dex reports an email address that another password holds in a
// successful answer. The client turns the flag into the error of the hub.
func TestAPasswordThatExistsIsAnError(t *testing.T) {
	t.Parallel()

	fake := &fakeDex{conflict: true}
	client := clientOf(t, fake)

	err := client.CreatePassword(t.Context(), dex.Password{
		Email: ninaEmail, Hash: []byte("hash"), Name: ninaEmail, UserID: "u1",
	})
	require.ErrorIs(t, err, errs.ErrLocalAccountExists)
	assert.Equal(t, "u1", fake.password.GetUserId())
	assert.Equal(t, ninaEmail, fake.password.GetUsername())

	err = client.UpdatePassword(t.Context(), ninaEmail, []byte("hash"))
	require.ErrorIs(t, err, errs.ErrLocalAccountNotFound)

	err = client.DeletePassword(t.Context(), ninaEmail)
	require.ErrorIs(t, err, errs.ErrLocalAccountNotFound)
}

// IDPROV-SC-030: A password in Dex carries the identifier of the user record.
func TestTheClientReadsThePasswords(t *testing.T) {
	t.Parallel()

	fake := &fakeDex{password: &api.Password{Email: ninaEmail, UserId: "u1"}}
	client := clientOf(t, fake)

	passwords, err := client.ListPasswords(t.Context())
	require.NoError(t, err)
	require.Len(t, passwords, 1)
	assert.Equal(t, "u1", passwords[0].UserID)
}
