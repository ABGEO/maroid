package pluginapi_test

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/libs/testdb"
)

const fixture = `
CREATE TABLE public.users (
    id          UUID   NOT NULL PRIMARY KEY DEFAULT uuidv7(),
    telegram_id BIGINT NOT NULL UNIQUE
);

CREATE SCHEMA test_scope;

CREATE TABLE test_scope.notes (
    id      UUID NOT NULL PRIMARY KEY DEFAULT uuidv7(),
    user_id UUID NOT NULL
        DEFAULT NULLIF(current_setting('app.user_id', true), '')::uuid
        REFERENCES public.users (id),
    body    TEXT NOT NULL
);

ALTER TABLE test_scope.notes ENABLE ROW LEVEL SECURITY;
ALTER TABLE test_scope.notes FORCE ROW LEVEL SECURITY;

CREATE POLICY notes_user_isolation ON test_scope.notes
    USING (user_id = NULLIF(current_setting('app.user_id', true), '')::uuid)
    WITH CHECK (user_id = NULLIF(current_setting('app.user_id', true), '')::uuid);

CREATE SCHEMA dev_maroid_jasmine;

CREATE TABLE dev_maroid_jasmine.plants (
    id   TEXT NOT NULL PRIMARY KEY,
    name TEXT NOT NULL
);
`

// scope is a database that holds one scoped table and two user records.
type scope struct {
	instance *testdb.Instance
	notes    *pluginapi.PluginDB
	userA    string
	userB    string
}

func newScope(t *testing.T) *scope {
	t.Helper()

	instance := testdb.Start(t)

	_, err := instance.DB.ExecContext(t.Context(), fixture)
	require.NoError(t, err)

	return &scope{
		instance: instance,
		notes:    pluginapi.NewPluginDB(instance.DB, pluginapi.ParsePluginID("test.scope")),
		userA:    addUser(t, instance, 111),
		userB:    addUser(t, instance, 222),
	}
}

func addUser(t *testing.T, instance *testdb.Instance, telegramID int64) string {
	t.Helper()

	var id string

	err := instance.DB.GetContext(
		t.Context(),
		&id,
		`INSERT INTO public.users (telegram_id) VALUES ($1) RETURNING id;`,
		telegramID,
	)
	require.NoError(t, err)

	return id
}

// as returns a context that names the acting user.
func as(t *testing.T, user string) context.Context {
	t.Helper()

	return pluginapi.ContextWithActingUser(t.Context(), user)
}

// write stores one note for each body, as the given user.
func (s *scope) write(t *testing.T, user string, bodies ...string) {
	t.Helper()

	ctx := as(t, user)

	require.NoError(t, s.notes.WithTx(ctx, func(tx *sqlx.Tx) error {
		for _, body := range bodies {
			_, err := tx.ExecContext(ctx, `INSERT INTO notes (body) VALUES ($1);`, body)
			if err != nil {
				return fmt.Errorf("inserting the note %q: %w", body, err)
			}
		}

		return nil
	}))
}

// IDENT-SC-002: An insert that names no user_id carries the acting user.
func TestAnInsertCarriesTheActingUser(t *testing.T) {
	t.Parallel()

	scoped := newScope(t)
	ctx := as(t, scoped.userA)

	scoped.write(t, scoped.userA, "carried")

	var owner string

	require.NoError(t, scoped.notes.WithTx(ctx, func(tx *sqlx.Tx) error {
		return tx.GetContext(ctx, &owner, `SELECT user_id FROM notes WHERE body = 'carried';`)
	}))
	require.Equal(t, scoped.userA, owner)
}

// IDENT-SC-001: A read as user A returns the notes of user A and none of user B.
// IDENT-NFR-001: The count of the records of user B in the result is zero.
func TestAReadReturnsNoRecordOfAnotherUser(t *testing.T) {
	t.Parallel()

	scoped := newScope(t)
	scoped.write(t, scoped.userA, "a1", "a2", "a3")
	scoped.write(t, scoped.userB, "b1", "b2", "b3")

	var mine, ofOther int

	ctx := as(t, scoped.userA)

	require.NoError(t, scoped.notes.WithTx(ctx, func(tx *sqlx.Tx) error {
		if err := tx.GetContext(ctx, &mine, `SELECT count(*) FROM notes;`); err != nil {
			return fmt.Errorf("counting the notes: %w", err)
		}

		err := tx.GetContext(
			ctx, &ofOther,
			`SELECT count(*) FROM notes WHERE user_id = $1;`, scoped.userB,
		)
		if err != nil {
			return fmt.Errorf("counting the notes of the other user: %w", err)
		}

		return nil
	}))

	require.Equal(t, 3, mine)
	require.Equal(t, 0, ofOther)
}

// IDENT-SC-003: An update and a delete of the record of another user change no row.
func TestAWriteDoesNotReachTheRecordOfAnotherUser(t *testing.T) {
	t.Parallel()

	scoped := newScope(t)
	scoped.write(t, scoped.userB, "untouched")

	ctx := as(t, scoped.userA)

	var updated, deleted int64

	require.NoError(t, scoped.notes.WithTx(ctx, func(tx *sqlx.Tx) error {
		var err error

		updated, err = affected(
			ctx, tx,
			`UPDATE notes SET body = 'stolen' WHERE body = 'untouched';`,
		)
		if err != nil {
			return err
		}

		deleted, err = affected(ctx, tx, `DELETE FROM notes WHERE body = 'untouched';`)

		return err
	}))

	require.Equal(t, int64(0), updated)
	require.Equal(t, int64(0), deleted)

	ofOwner := as(t, scoped.userB)

	var body string

	require.NoError(t, scoped.notes.WithTx(ofOwner, func(tx *sqlx.Tx) error {
		return tx.GetContext(ofOwner, &body, `SELECT body FROM notes;`)
	}))
	require.Equal(t, "untouched", body)
}

// IDENT-SC-004: A read of the record of another user answers as a missing record.
// IDENT-FR-006: Neither answer tells the requester that the record exists.
func TestTheRecordOfAnotherUserReadsAsAMissingRecord(t *testing.T) {
	t.Parallel()

	scoped := newScope(t)
	scoped.write(t, scoped.userB, "hidden")

	ctx := as(t, scoped.userA)

	read := func(body string) error {
		var found string

		return scoped.notes.WithTx(ctx, func(tx *sqlx.Tx) error {
			return tx.GetContext(ctx, &found, `SELECT body FROM notes WHERE body = $1;`, body)
		})
	}

	require.ErrorIs(t, read("hidden"), sql.ErrNoRows)
	require.ErrorIs(t, read("no such note"), sql.ErrNoRows)
}

// IDENT-SC-005: A transaction with no acting user reads no scoped record and its
// insert fails. IDENT-INV-002 holds even when the caller forgets.
func TestNoActingUserReachesNoScopedRecord(t *testing.T) {
	t.Parallel()

	scoped := newScope(t)
	scoped.write(t, scoped.userA, "invisible")

	ctx := t.Context()

	var count int

	require.NoError(t, scoped.notes.WithTx(ctx, func(tx *sqlx.Tx) error {
		return tx.GetContext(ctx, &count, `SELECT count(*) FROM notes;`)
	}))
	require.Equal(t, 0, count)

	err := scoped.notes.WithTx(ctx, func(tx *sqlx.Tx) error {
		_, execErr := tx.ExecContext(ctx, `INSERT INTO notes (body) VALUES ('refused');`)
		if execErr != nil {
			return fmt.Errorf("inserting with no acting user: %w", execErr)
		}

		return nil
	})
	require.ErrorContains(t, err, "row-level security policy")
}

// IDENT-SC-010: A table from before this feature answers every user with every row.
func TestATableFromBeforeThisFeatureStaysShared(t *testing.T) {
	t.Parallel()

	scoped := newScope(t)
	shared := pluginapi.NewPluginDB(
		scoped.instance.DB,
		pluginapi.ParsePluginID("dev.maroid.jasmine"),
	)

	_, err := scoped.instance.DB.ExecContext(
		t.Context(),
		`INSERT INTO dev_maroid_jasmine.plants (id, name) VALUES ('p1', 'Jasmine');`,
	)
	require.NoError(t, err)

	for _, user := range []string{scoped.userA, scoped.userB} {
		ctx := as(t, user)

		var count int

		require.NoError(t, shared.WithTx(ctx, func(tx *sqlx.Tx) error {
			return tx.GetContext(ctx, &count, `SELECT count(*) FROM plants;`)
		}))
		require.Positive(t, count, "every user reads a shared table")
	}
}

func affected(ctx context.Context, tx *sqlx.Tx, query string) (int64, error) {
	result, err := tx.ExecContext(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("running %q: %w", query, err)
	}

	count, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("reading the affected rows of %q: %w", query, err)
	}

	return count, nil
}
