package repository_test

import (
	"encoding/json"
	"log/slog"
	"testing"

	"github.com/golang-migrate/migrate/v4"
	"github.com/jmoiron/sqlx"
	"github.com/mcuadros/go-defaults"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/apps/hub/internal/config"
	"github.com/abgeo/maroid/apps/hub/internal/outbox"
	"github.com/abgeo/maroid/apps/hub/internal/secret"
	"github.com/abgeo/maroid/apps/hub/internal/testbao"
	"github.com/abgeo/maroid/libs/testdb"
)

const (
	// versionSecretKeyBackfill queues the creation of the key of every record.
	versionSecretKeyBackfill = uint(20261126100000)
	maxBackfillCycles        = 20
)

// backfillBench holds a database migrated up to the backfill, with 3 user records,
// 5 workspaces, and a task that already creates the key of the first workspace.
type backfillBench struct {
	instance *testdb.Instance
	migrator *migrate.Migrate
	keys     []secret.Key
}

func backfillUnderTest(t *testing.T) *backfillBench {
	t.Helper()

	instance := testdb.Start(t)
	bench := &backfillBench{instance: instance, migrator: newMigrator(t, instance)}

	require.NoError(t, bench.migrator.Migrate(versionWorkspaceCascade))

	for _, name := range []string{"Ana", "Beka", "Gio"} {
		id := bench.insert(t, "users", "first_name", name)
		bench.keys = append(bench.keys, secret.UserKey(id))
	}

	names := []string{"A", "B", "C", "D", "E"}
	workspaces := make([]string, 0, len(names))

	for _, name := range names {
		id := bench.insert(t, "workspaces", "name", name)
		workspaces = append(workspaces, id)
		bench.keys = append(bench.keys, secret.WorkspaceKey(id))
	}

	waiting := secret.CreateTask(secret.Owner{Kind: secret.OwnerWorkspace, ID: workspaces[0]})

	payload, err := json.Marshal(waiting.Payload)
	require.NoError(t, err)

	_, err = instance.DB.Exec(
		`INSERT INTO public.outbox_tasks (type, payload, dedupe_key) VALUES ($1, $2, $3);`,
		waiting.Type, payload, waiting.DedupeKey,
	)
	require.NoError(t, err)

	return bench
}

func (b *backfillBench) insert(t *testing.T, table string, column string, value string) string {
	t.Helper()

	var id string

	require.NoError(t, b.instance.DB.Get(&id,
		`INSERT INTO public.`+table+` (`+column+`) VALUES ($1) RETURNING id;`, value))

	return id
}

// creations lists the key of each task that creates one.
func (b *backfillBench) creations(t *testing.T) []secret.Key {
	t.Helper()

	var payloads []json.RawMessage

	require.NoError(t, b.instance.DB.Select(&payloads,
		`SELECT payload FROM public.outbox_tasks WHERE type = $1;`, secret.TypeCreate))

	keys := make([]secret.Key, 0, len(payloads))

	for _, payload := range payloads {
		var owner secret.Owner

		require.NoError(t, json.Unmarshal(payload, &owner))

		key, err := owner.Key()
		require.NoError(t, err)

		keys = append(keys, key)
	}

	return keys
}

func runUntilDrained(t *testing.T, database *sqlx.DB, keys secret.Keys) {
	t.Helper()

	handlers := outbox.NewRegistry()
	require.NoError(t, handlers.Register(
		secret.NewCreateHandler(slog.New(slog.DiscardHandler), database, keys),
	))

	var cfg config.Outbox

	defaults.SetDefaults(&cfg)

	worker := outbox.NewWorker(slog.New(slog.DiscardHandler), database, handlers, cfg)

	for cycle := 0; ; cycle++ {
		var left int

		require.NoError(t, database.Get(&left, `SELECT count(*) FROM public.outbox_tasks;`))

		if left == 0 {
			return
		}

		require.Less(t, cycle, maxBackfillCycles, "the outbox drains")
		require.NoError(t, worker.Cycle(t.Context()))
	}
}

// OWNKEY-SC-008: The backfill queues one creation for each user record and each
// workspace, and its down migration leaves none.
func TestTheBackfillQueuesOneCreationForEachRecord(t *testing.T) {
	t.Parallel()

	bench := backfillUnderTest(t)

	require.NoError(t, bench.migrator.Migrate(versionSecretKeyBackfill))
	assert.ElementsMatch(t, bench.keys, bench.creations(t))

	require.NoError(t, bench.migrator.Steps(-1))
	assert.Empty(t, bench.creations(t))
}

// OWNKEY-SC-009: After the backfill, the worker creates every key and keeps the 2
// keys that an administrator created and rotated.
func TestTheBackfillKeepsTheKeysThatExist(t *testing.T) {
	t.Parallel()

	bench := backfillUnderTest(t)
	server := testbao.Start(t)
	kept := map[secret.Key]bool{bench.keys[0]: true, bench.keys[len(bench.keys)-1]: true}

	for key := range kept {
		path := testbao.TransitMount + "/keys/" + string(key)

		_, err := server.Root.Logical().WriteWithContext(t.Context(), path, nil)
		require.NoError(t, err)

		_, err = server.Root.Logical().WriteWithContext(t.Context(), path+"/rotate", nil)
		require.NoError(t, err)
	}

	require.NoError(t, bench.migrator.Migrate(versionSecretKeyBackfill))
	runUntilDrained(t, bench.instance.DB, server.Cipher(t))

	for _, key := range bench.keys {
		held, err := server.Root.Logical().
			ReadWithContext(t.Context(), testbao.TransitMount+"/keys/"+string(key))
		require.NoError(t, err)
		require.NotNil(t, held, key)

		version := json.Number("1")
		if kept[key] {
			version = "2"
		}

		assert.Equal(t, version, held.Data["latest_version"], key)
	}
}
