package provider_test

import (
	"strings"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"github.com/abgeo/maroid/apps/hub/internal/database"
	"github.com/abgeo/maroid/apps/hub/internal/dex"
	"github.com/abgeo/maroid/apps/hub/internal/dex/dextest"
	"github.com/abgeo/maroid/apps/hub/internal/domain/errs"
	"github.com/abgeo/maroid/apps/hub/internal/model"
	"github.com/abgeo/maroid/apps/hub/internal/provider"
	"github.com/abgeo/maroid/apps/hub/internal/repository"
	"github.com/abgeo/maroid/libs/testdb"
)

const (
	ninaEmail    = "nina@home.example"
	anaEmail     = "ana@home.example"
	goodPassword = "correct horse battery"
	newPassword  = "staple battery horse"
	localPreset  = `{"maroidPreset":"local"}`
)

type localFixture struct {
	instance *testdb.Instance
	memory   *dextest.Memory
	accounts *provider.Accounts
}

func localUnderTest(t *testing.T, connectors ...dex.Connector) *localFixture {
	t.Helper()

	instance := migrated(t)
	memory := dextest.New(connectors...)
	manager := provider.NewManager(memory, instance.DB, provider.Settings{Issuer: issuer})

	return &localFixture{
		instance: instance,
		memory:   memory,
		accounts: provider.NewAccounts(memory, instance.DB, manager),
	}
}

func withLocal() dex.Connector {
	return dextest.Connector("local", "local", "Email", localPreset)
}

func (f *localFixture) user(t *testing.T, name string) string {
	t.Helper()

	var id string

	require.NoError(t, f.instance.DB.Get(&id,
		`INSERT INTO public.users (first_name) VALUES ($1) RETURNING id;`, name))

	return id
}

func (f *localFixture) identitiesOf(t *testing.T, userID string) []model.Identity {
	t.Helper()

	identities, err := database.FetchTx(t.Context(), f.instance.DB,
		func(tx *sqlx.Tx) ([]model.Identity, error) {
			return repository.NewIdentity(tx).ListByUser(t.Context(), userID)
		},
	)
	require.NoError(t, err)

	return identities
}

func (f *localFixture) password(t *testing.T, email string) *dex.Password {
	t.Helper()

	passwords, err := f.memory.ListPasswords(t.Context())
	require.NoError(t, err)

	for i := range passwords {
		if passwords[i].Email == email {
			return &passwords[i]
		}
	}

	return nil
}

// IDPROV-SC-017, IDPROV-SC-030: A local account lands with the identifier of the record
// in Dex and in the identity. An email address that another account holds is refused,
// and the record gains nothing.
func TestAnAdministratorGivesALocalAccount(t *testing.T) {
	t.Parallel()

	fixture := localUnderTest(t, withLocal())
	ana, nina, beka := fixture.user(t, "Ana"), fixture.user(t, "Nina"), fixture.user(t, "Beka")

	require.NoError(t, fixture.accounts.Give(t.Context(), ana, anaEmail, []byte(goodPassword)))
	require.NoError(t, fixture.accounts.Give(t.Context(), nina, ninaEmail, []byte(goodPassword)))

	stored := fixture.password(t, ninaEmail)
	require.NotNil(t, stored)
	assert.Equal(t, nina, stored.UserID)

	identities := fixture.identitiesOf(t, nina)
	require.Len(t, identities, 1)
	assert.Equal(t, "local", identities[0].Provider)
	assert.Equal(t, nina, identities[0].ProviderUserID)

	err := fixture.accounts.Give(t.Context(), beka, anaEmail, []byte(goodPassword))
	require.ErrorIs(t, err, errs.ErrLocalAccountExists)
	assert.Empty(t, fixture.identitiesOf(t, beka))
}

// IDPROV-SC-017: A record that holds a local account gets no second one.
func TestARecordHoldsOneLocalAccount(t *testing.T) {
	t.Parallel()

	fixture := localUnderTest(t, withLocal())
	nina := fixture.user(t, "Nina")

	require.NoError(t, fixture.accounts.Give(t.Context(), nina, ninaEmail, []byte(goodPassword)))

	err := fixture.accounts.Give(t.Context(), nina, "other@home.example", []byte(goodPassword))
	require.ErrorIs(t, err, errs.ErrLocalAccountExists)
	assert.Nil(t, fixture.password(t, "other@home.example"))
}

// IDPROV-SC-017: With no local provider, a local account is refused.
func TestALocalAccountNeedsTheLocalProvider(t *testing.T) {
	t.Parallel()

	fixture := localUnderTest(t)
	nina := fixture.user(t, "Nina")

	err := fixture.accounts.Give(t.Context(), nina, ninaEmail, []byte(goodPassword))
	require.ErrorIs(t, err, errs.ErrLocalProviderAbsent)
	assert.Empty(t, fixture.identitiesOf(t, nina))
}

// IDPROV-SC-017: Dex compares an email address without regard to case, so the hub
// stores it in lower case. A malformed address names /email.
func TestTheEmailAddressIsCheckedAndLowered(t *testing.T) {
	t.Parallel()

	fixture := localUnderTest(t, withLocal())
	nina := fixture.user(t, "Nina")

	err := fixture.accounts.Give(t.Context(), nina, "not an address", []byte(goodPassword))
	requireField(t, err, "/email")

	require.NoError(t, fixture.accounts.Give(t.Context(), nina, "Nina@Home.Example",
		[]byte(goodPassword)))
	assert.NotNil(t, fixture.password(t, ninaEmail))
}

// IDPROV-SC-018: A new password replaces the old one.
func TestAnAdministratorResetsAPassword(t *testing.T) {
	t.Parallel()

	fixture := localUnderTest(t, withLocal())
	nina := fixture.user(t, "Nina")

	require.NoError(t, fixture.accounts.Give(t.Context(), nina, ninaEmail, []byte(goodPassword)))
	require.NoError(t, fixture.accounts.Reset(t.Context(), nina, []byte(newPassword)))

	hash := fixture.memory.Hash(ninaEmail)
	require.NoError(t, bcrypt.CompareHashAndPassword(hash, []byte(newPassword)))
	require.Error(t, bcrypt.CompareHashAndPassword(hash, []byte(goodPassword)))

	cost, err := bcrypt.Cost(hash)
	require.NoError(t, err)
	assert.Equal(t, 12, cost, "the cost that Dex recommends")

	other := fixture.user(t, "Beka")
	require.ErrorIs(t, fixture.accounts.Reset(t.Context(), other, []byte(newPassword)),
		errs.ErrLocalAccountNotFound)
}

// IDPROV-SC-019: A removal deletes the identity and the password. The last identity of
// a record stays, with its password.
func TestAnAdministratorRemovesALocalAccount(t *testing.T) {
	t.Parallel()

	fixture := localUnderTest(t, withLocal())
	nina, beka := fixture.user(t, "Nina"), fixture.user(t, "Beka")

	require.NoError(t, fixture.accounts.Give(t.Context(), nina, ninaEmail, []byte(goodPassword)))
	require.NoError(t, fixture.accounts.Give(t.Context(), beka, anaEmail, []byte(goodPassword)))

	require.NoError(t, database.WithTx(t.Context(), fixture.instance.DB, func(tx *sqlx.Tx) error {
		return repository.NewIdentity(tx).
			Attach(t.Context(), nina, "telegram", "104", model.Profile{})
	}))

	require.NoError(t, fixture.accounts.Remove(t.Context(), nina))
	assert.Nil(t, fixture.password(t, ninaEmail))
	assert.Len(t, fixture.identitiesOf(t, nina), 1)

	require.ErrorIs(t, fixture.accounts.Remove(t.Context(), beka), errs.ErrLastIdentity)
	assert.NotNil(t, fixture.password(t, anaEmail))
	assert.Len(t, fixture.identitiesOf(t, beka), 1)
}

// IDPROV-SC-022: A password of 12 characters and one of 72 bytes land. One of 11
// characters and one of 73 bytes name /password.
func TestThePasswordLengthIsChecked(t *testing.T) {
	t.Parallel()

	fixture := localUnderTest(t, withLocal())

	cases := []struct {
		password string
		lands    bool
	}{
		{strings.Repeat("a", 11), false},
		{strings.Repeat("a", 12), true},
		{strings.Repeat("ა", 24), true},
		{strings.Repeat("a", 73), false},
	}

	for i, one := range cases {
		user := fixture.user(t, "User")
		err := fixture.accounts.Give(t.Context(), user, strings.Repeat("x", i+1)+"@home.example",
			[]byte(one.password))

		if one.lands {
			require.NoError(t, err, "case %d", i)
		} else {
			requireField(t, err, "/password")
		}
	}
}

// IDPROV-SC-029: A Dex that does not answer leaves the record with no identity.
func TestAnUnavailableDexLeavesNoIdentity(t *testing.T) {
	t.Parallel()

	fixture := localUnderTest(t, withLocal())
	nina := fixture.user(t, "Nina")

	fixture.memory.FailOnce("CreatePassword", errs.ErrIDPUnavailable)

	err := fixture.accounts.Give(t.Context(), nina, ninaEmail, []byte(goodPassword))
	require.ErrorIs(t, err, errs.ErrIDPUnavailable)
	assert.Empty(t, fixture.identitiesOf(t, nina))
}

// IDPROV-DD-003: A password that a timed out call wrote for the same record lets the
// retry land.
func TestARetryOfAGiveLands(t *testing.T) {
	t.Parallel()

	fixture := localUnderTest(t, withLocal())
	nina := fixture.user(t, "Nina")

	require.NoError(t, fixture.memory.CreatePassword(t.Context(), dex.Password{
		Email: ninaEmail, Hash: []byte("hash"), UserID: nina,
	}))

	require.NoError(t, fixture.accounts.Give(t.Context(), nina, ninaEmail, []byte(goodPassword)))
	assert.Len(t, fixture.identitiesOf(t, nina), 1)
}

// IDPROV-SC-025: EnsureProvider adds the local provider once.
func TestEnsureProviderAddsTheLocalProvider(t *testing.T) {
	t.Parallel()

	fixture := localUnderTest(t)

	require.NoError(t, fixture.accounts.EnsureProvider(t.Context()))
	require.NoError(t, fixture.accounts.EnsureProvider(t.Context()))

	connectors, err := fixture.memory.ListConnectors(t.Context())
	require.NoError(t, err)
	require.Len(t, connectors, 1)
	assert.Equal(t, "Email", connectors[0].Name)
}

// IDPROV-SC-025: The command line asks whether a record holds a local account before it
// gives one or resets one.
func TestHasAccountReadsTheIdentity(t *testing.T) {
	t.Parallel()

	fixture := localUnderTest(t, withLocal())
	nina, beka := fixture.user(t, "Nina"), fixture.user(t, "Beka")

	require.NoError(t, fixture.accounts.Give(t.Context(), nina, ninaEmail, []byte(goodPassword)))

	held, err := fixture.accounts.HasAccount(t.Context(), nina)
	require.NoError(t, err)
	assert.True(t, held)

	held, err = fixture.accounts.HasAccount(t.Context(), beka)
	require.NoError(t, err)
	assert.False(t, held)
}
