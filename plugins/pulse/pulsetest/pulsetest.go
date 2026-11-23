// Package pulsetest holds the fixtures that the tests of pulse share: the schema, two
// people, a fixed clock, and a fake messenger.
package pulsetest

import (
	"context"
	"io/fs"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/libs/testdb"
	"github.com/abgeo/maroid/plugins/pulse/db"
)

// The people of the scenarios. Nina lives in Asia/Tbilisi, and Beka never turns
// pulse on.
const (
	Nina = "01927f4e-3c2a-7b1d-9e8f-0a1b2c3d4e01"
	Beka = "01927f4e-3c2a-7b1d-9e8f-0a1b2c3d4e02"
)

// hubFixtures stands in for the core migrations of the hub, which a plugin test does
// not own: the trigger function, and the users that a scoped table references.
const hubFixtures = `
CREATE OR REPLACE FUNCTION set_updated_at() RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TABLE public.users (id UUID NOT NULL PRIMARY KEY);
INSERT INTO public.users (id) VALUES ('` + Nina + `'), ('` + Beka + `');`

// Tables names every table of the schema of pulse.
func Tables() []string {
	return []string{
		"settings", "questions", "choices", "checkins", "checkin_questions",
		"rounds", "prompts", "question_values",
	}
}

// Tbilisi is the zone of Nina.
func Tbilisi() *time.Location {
	return zone("Asia/Tbilisi")
}

// Berlin is the zone that Nina moves to in a scenario.
func Berlin() *time.Location {
	return zone("Europe/Berlin")
}

func zone(name string) *time.Location {
	location, err := time.LoadLocation(name)
	if err != nil {
		panic(err)
	}

	return location
}

// Schema starts a database with the schema of pulse, and answers the PluginDB that
// the hub gives the plugin.
func Schema(t *testing.T) pluginapi.PluginDB {
	t.Helper()

	instance := testdb.Start(t)

	_, err := instance.DB.Exec(hubFixtures)
	require.NoError(t, err)

	migrations, err := fs.Sub(db.Migrations, "migrations")
	require.NoError(t, err)

	identifier := pluginapi.ParsePluginID("dev.maroid.pulse")
	instance.Migrate(t, identifier.ToSafeName("_"), migrations)

	return testdb.PluginDB(instance, identifier)
}

// ActingAs answers a context whose acting user is the user. Nina acts in her zone,
// and any other person in UTC.
func ActingAs(user string) context.Context {
	location := time.UTC
	if user == Nina {
		location = Tbilisi()
	}

	return ActingIn(user, location)
}

// ActingIn answers a context whose acting user is the user, in the zone.
func ActingIn(user string, location *time.Location) context.Context {
	return pluginapi.ContextWithActingTimeZone(
		pluginapi.ContextWithActingUser(context.Background(), user), location)
}

// Count answers the single number that the query reads as the user.
func Count(t *testing.T, database pluginapi.PluginDB, user string, query string, args ...any) int {
	t.Helper()

	var total int

	require.NoError(t, database.WithTx(ActingAs(user), func(tx *sqlx.Tx) error {
		return tx.Get(&total, query, args...)
	}))

	return total
}

// Exec runs the statement as the user.
func Exec(t *testing.T, database pluginapi.PluginDB, user string, statement string, args ...any) {
	t.Helper()

	require.NoError(t, database.WithTx(ActingAs(user), func(tx *sqlx.Tx) error {
		_, err := tx.Exec(statement, args...)

		return err //nolint:wrapcheck // a fixture of a test.
	}))
}

// ID reads the identifier that the query answers, as the user.
func ID(t *testing.T, database pluginapi.PluginDB, user string, query string, args ...any) string {
	t.Helper()

	var id string

	require.NoError(t, database.WithTx(ActingAs(user), func(tx *sqlx.Tx) error {
		return tx.Get(&id, query, args...)
	}))

	return id
}
