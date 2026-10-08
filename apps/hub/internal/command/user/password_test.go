package user_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"strings"
	"testing"

	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"github.com/abgeo/maroid/apps/hub/db"
	"github.com/abgeo/maroid/apps/hub/internal/command/user"
	"github.com/abgeo/maroid/apps/hub/internal/dex"
	"github.com/abgeo/maroid/apps/hub/internal/dex/dextest"
	"github.com/abgeo/maroid/apps/hub/internal/domain/errs"
	"github.com/abgeo/maroid/apps/hub/internal/provider"
	users "github.com/abgeo/maroid/apps/hub/internal/user"
	"github.com/abgeo/maroid/libs/testdb"
)

const (
	zuraEmail     = "zura@home.example"
	firstPassword = "correct horse battery"
	nextPassword  = "staple battery horse"
	unknownRecord = "01998aa0-0000-7000-8000-000000000000"
)

var errReadFailed = errors.New("the read failed")

// fixedInput answers one password, or the error of a read that failed.
type fixedInput struct {
	password string
	err      error
}

func (f fixedInput) Password() ([]byte, error) {
	return []byte(f.password), f.err
}

type commandWorld struct {
	memory   *dextest.Memory
	records  *users.Manager
	accounts *provider.Accounts
	zura     string
}

func newCommandWorld(t *testing.T) *commandWorld {
	t.Helper()

	instance := testdb.Start(t)

	migrations, err := fs.Sub(db.GetMigrationsFS(), "migrations")
	require.NoError(t, err)

	instance.Migrate(t, "public", migrations)

	var zura string

	require.NoError(t, instance.DB.Get(
		&zura,
		`INSERT INTO public.users (first_name, is_administrator) VALUES ('Zura', true) RETURNING id;`,
	))

	memory := dextest.New()
	manager := provider.NewManager(memory, instance.DB, provider.Settings{
		Issuer: "https://auth.maroid.localhost",
	})

	return &commandWorld{
		memory:   memory,
		records:  users.NewManager(instance.DB, nil, nil, 0),
		accounts: provider.NewAccounts(memory, instance.DB, manager),
		zura:     zura,
	}
}

func (w *commandWorld) run(userID string, email string, input fixedInput) error {
	err := user.SetPassword(context.Background(), w.records, w.accounts, input, userID, email)
	if err != nil {
		return fmt.Errorf("running the command: %w", err)
	}

	return nil
}

func (w *commandWorld) connectors(t *testing.T) []dex.Connector {
	t.Helper()

	connectors, err := w.memory.ListConnectors(t.Context())
	require.NoError(t, err)

	return connectors
}

// IDPROV-SC-024: On a new instance the command adds the local provider and gives the
// first administrator a local account.
func TestTheCommandGivesTheFirstLocalAccount(t *testing.T) {
	t.Parallel()

	world := newCommandWorld(t)

	require.NoError(t, world.run(world.zura, zuraEmail, fixedInput{password: firstPassword}))

	connectors := world.connectors(t)
	require.Len(t, connectors, 1)
	assert.Equal(t, "local", connectors[0].ID)
	assert.Equal(t, "Maroid", connectors[0].Name)
	require.NoError(
		t,
		bcrypt.CompareHashAndPassword(world.memory.Hash(zuraEmail), []byte(firstPassword)),
	)
}

// IDPROV-SC-025: A record that holds a local account takes a new password, and the
// local provider returns when Dex lost it. A run that names --email fails and changes
// nothing.
func TestTheCommandResetsALocalAccount(t *testing.T) {
	t.Parallel()

	world := newCommandWorld(t)
	require.NoError(t, world.run(world.zura, zuraEmail, fixedInput{password: firstPassword}))
	require.NoError(t, world.memory.DeleteConnector(t.Context(), "local"))

	require.NoError(t, world.run(world.zura, "", fixedInput{password: nextPassword}))

	require.Len(t, world.connectors(t), 1, "the local provider returns")
	hash := world.memory.Hash(zuraEmail)
	require.NoError(t, bcrypt.CompareHashAndPassword(hash, []byte(nextPassword)))

	err := world.run(world.zura, "other@home.example", fixedInput{password: firstPassword})
	require.ErrorIs(t, err, errs.ErrLocalAccountEmail)
	assert.Equal(t, hash, world.memory.Hash(zuraEmail), "the refused run changes nothing")
}

// IDPROV-SC-025: A record with no local account needs --email, and an unknown record
// is refused. Neither run writes to Dex.
func TestTheCommandChecksTheRecordAndTheAddress(t *testing.T) {
	t.Parallel()

	world := newCommandWorld(t)

	require.ErrorIs(t, world.run(world.zura, "", fixedInput{password: firstPassword}),
		errs.ErrLocalAccountEmailMissing)
	require.ErrorIs(t, world.run(unknownRecord, zuraEmail, fixedInput{password: firstPassword}),
		errs.ErrUserNotFound)
	assert.Empty(t, world.connectors(t))
}

// IDPROV-SC-026: A read of the password that fails ends the run before any write.
func TestAFailedReadChangesNothing(t *testing.T) {
	t.Parallel()

	world := newCommandWorld(t)

	err := world.run(world.zura, zuraEmail, fixedInput{err: errReadFailed})
	require.ErrorIs(t, err, errReadFailed)
	assert.Empty(t, world.connectors(t))
}

// IDPROV-SC-026: The prompt asks twice, and two different values are refused.
func TestThePromptAsksTwice(t *testing.T) {
	t.Parallel()

	var prompts []string

	answers := []string{firstPassword, nextPassword}
	read := func(prompt string) ([]byte, error) {
		prompts = append(prompts, prompt)
		answer := answers[0]
		answers = answers[1:]

		return []byte(answer), nil
	}

	_, err := user.PromptTwice(read)
	require.ErrorIs(t, err, errs.ErrPasswordMismatch)
	assert.Len(t, prompts, 2)

	same := func(string) ([]byte, error) { return []byte(firstPassword), nil }

	password, err := user.PromptTwice(same)
	require.NoError(t, err)
	assert.Equal(t, firstPassword, string(password))
}

// IDPROV-SC-026: With no terminal the command reads the first line of the standard
// input, without its line break.
func TestTheStandardInputGivesTheFirstLine(t *testing.T) {
	t.Parallel()

	cases := []struct{ input, expected string }{
		{firstPassword + "\n", firstPassword},
		{firstPassword + "\r\nsecond\n", firstPassword},
		{firstPassword, firstPassword},
		{" a password with spaces at its ends \n", " a password with spaces at its ends "},
	}

	for _, one := range cases {
		password, err := user.FirstLine(strings.NewReader(one.input))
		require.NoError(t, err)
		assert.Equal(t, one.expected, string(password))
	}

	_, err := user.FirstLine(strings.NewReader(""))
	require.ErrorIs(t, err, errs.ErrPasswordMissing)
}

// IDPROV-SC-026: The command declares no flag that takes a password.
func TestTheCommandTakesNoPasswordFlag(t *testing.T) {
	t.Parallel()

	command := user.NewPasswordCommand(nil).Command()

	var names []string

	command.Flags().VisitAll(func(flag *pflag.Flag) { names = append(names, flag.Name) })
	assert.ElementsMatch(t, []string{"user", "email"}, names)
}

// IDPROV-SC-027: The invitation prints the identifier of the record on the first line
// and the address on the second.
func TestTheInvitationPrintsTheRecordFirst(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer

	require.NoError(
		t,
		user.WriteInvitation(&out, unknownRecord, "https://maroid.localhost/invite?token=t"),
	)
	assert.Equal(t, unknownRecord+"\nhttps://maroid.localhost/invite?token=t\n", out.String())
}
