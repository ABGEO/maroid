package repository_test

import (
	"context"
	"fmt"
	"io/fs"
	"testing"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5" // The PostgreSQL driver for migrate
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/libs/rest/page"
	"github.com/abgeo/maroid/libs/testdb"
	"github.com/abgeo/maroid/plugins/jasmine/db"
	"github.com/abgeo/maroid/plugins/jasmine/model"
	"github.com/abgeo/maroid/plugins/jasmine/repository"
)

const workspace = "01927f4e-3c2a-7b1d-9e8f-0a1b2c3d4e5f"

// hubFixtures stands in for the core migrations of the hub, which a plugin test
// does not own: the trigger function, and the workspace that a scoped table
// references.
const hubFixtures = `
CREATE OR REPLACE FUNCTION set_updated_at() RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TABLE public.workspaces (id UUID NOT NULL PRIMARY KEY);
INSERT INTO public.workspaces (id) VALUES ('` + workspace + `');`

// plantsUnderTest returns a database that holds one environment and count plants,
// and the identifiers of the plants in the order that they were made, which is
// the order that a UUID version 7 sorts in.
func plantsUnderTest(t *testing.T, count int) (*sqlx.DB, []string) {
	t.Helper()

	instance := testdb.Start(t)

	_, err := instance.DB.Exec(hubFixtures)
	require.NoError(t, err)

	migrations, err := fs.Sub(db.Migrations, "migrations")
	require.NoError(t, err)

	source, err := iofs.New(migrations, ".")
	require.NoError(t, err)

	migrator, err := migrate.NewWithSourceInstance("iofs", source, instance.MigrationDSN)
	require.NoError(t, err)

	t.Cleanup(func() { _, _ = migrator.Close() })
	require.NoError(t, migrator.Up())

	tx := begin(t, instance.DB)

	var environmentID string

	require.NoError(t, tx.Get(&environmentID,
		`INSERT INTO environments (number, name) VALUES (1, 'Balcony') RETURNING id;`))

	ids := make([]string, 0, count)

	for i := range count {
		var id string

		require.NoError(t, tx.Get(
			&id,
			`INSERT INTO plants (number, common_name, environment_id) VALUES ($1, $2, $3) RETURNING id;`,
			i+1,
			fmt.Sprintf("Plant %d", i),
			environmentID,
		))

		ids = append(ids, id)
	}

	require.NoError(t, tx.Commit())

	return instance.DB, ids
}

// begin starts a transaction that acts in the workspace, as PluginDB.WithTx does.
func begin(t *testing.T, database *sqlx.DB) *sqlx.Tx {
	t.Helper()

	tx, err := database.Beginx()
	require.NoError(t, err)

	t.Cleanup(func() { _ = tx.Rollback() })

	_, err = tx.Exec(`SELECT set_config('app.workspace_id', $1, true)`, workspace)
	require.NoError(t, err)

	return tx
}

// active keeps the plants that the fixture made, which are all active.
func active() repository.PlantFilter {
	return repository.PlantFilter{Status: model.PlantStatusActive}
}

// readEveryPage follows the keyset from the first page to the last, and returns
// the identifiers in the order that a client reads them.
func readEveryPage(t *testing.T, database *sqlx.DB, limit int) []string {
	t.Helper()

	var (
		seen  []string
		after string
	)

	for range 100 {
		tx := begin(t, database)

		rows, err := repository.NewPlant(tx).List(
			context.Background(),
			page.Seek{Direction: page.DirectionForward, Boundary: after, Limit: limit + 1},
			active(),
		)
		require.NoError(t, err)
		require.NoError(t, tx.Commit())

		if len(rows) == 0 {
			break
		}

		more := len(rows) > limit
		if more {
			rows = rows[:limit]
		}

		for _, row := range rows {
			seen = append(seen, row.ID)
		}

		if !more {
			break
		}

		after = rows[len(rows)-1].ID
	}

	return seen
}

// APIFMT-SC-004, APIFMT-INV-001: A client that reads every page of a collection
// which no write changes reads each item exactly once.
func TestTheKeysetReadsEveryRowExactlyOnce(t *testing.T) {
	t.Parallel()

	database, _ := plantsUnderTest(t, 25)

	seen := readEveryPage(t, database, 10)

	require.Len(t, seen, 25)
	assert.Len(t, unique(seen), 25, "no row arrives twice")

	for i := 1; i < len(seen); i++ {
		assert.Less(t, seen[i-1], seen[i], "the order holds between pages")
	}
}

// APIFMT-SC-004: A page reads one row more than the limit, so the caller tells a
// collection that continues from one that ends.
func TestTheListReadsOneRowMoreThanTheLimit(t *testing.T) {
	t.Parallel()

	database, ids := plantsUnderTest(t, 3)
	tx := begin(t, database)

	rows, err := repository.NewPlant(tx).
		List(context.Background(), page.Seek{Direction: page.DirectionForward, Limit: 3}, active())
	require.NoError(t, err)
	assert.Len(t, rows, 3, "two asked for plus the probe row")

	rows, err = repository.NewPlant(tx).List(
		context.Background(),
		page.Seek{Direction: page.DirectionForward, Boundary: ids[1], Limit: 3},
		active(),
	)
	require.NoError(t, err)
	assert.Len(t, rows, 1, "the keyset reads past the boundary only")
	assert.Equal(t, ids[2], rows[0].ID)
}

// APIFMT-SC-004: A backward read takes the rows nearest before the boundary, and
// answers them in the order of the collection.
func TestABackwardReadAnswersTheRowsBeforeTheBoundaryInOrder(t *testing.T) {
	t.Parallel()

	database, ids := plantsUnderTest(t, 6)
	tx := begin(t, database)

	rows, err := repository.NewPlant(tx).List(
		context.Background(),
		page.Seek{Direction: page.DirectionBackward, Boundary: ids[4], Limit: 2},
		active(),
	)
	require.NoError(t, err)
	require.Len(t, rows, 2)
	assert.Equal(t, ids[2], rows[0].ID)
	assert.Equal(t, ids[3], rows[1].ID)
}

func unique(values []string) []string {
	held := map[string]struct{}{}
	out := make([]string, 0, len(values))

	for _, value := range values {
		if _, seen := held[value]; !seen {
			held[value] = struct{}{}
			out = append(out, value)
		}
	}

	return out
}

// APIFMT-NFR-001, APIFMT-SC-011: The cost of one page does not grow with the
// position of that page. A keyset reads from an index; an offset would read
// every row that it discards.
//
// The test counts the rows that each read touches rather than the time it
// takes. A read of a hundred rows takes a fraction of a millisecond, where the
// noise of a shared machine drowns the difference, and the rows are the cause
// of the time.
func TestThePageCostDoesNotGrowWithThePosition(t *testing.T) {
	t.Parallel()

	const (
		rows      = 10000
		pageSize  = 100
		deepPage  = 100
		tolerance = 1.2
	)

	database, ids := plantsUnderTest(t, rows)

	// The boundary of page 100, the last full page of the collection.
	deepest := ids[rows-pageSize-1]

	first := rowsReadByOnePage(t, database, "", pageSize)
	deep := rowsReadByOnePage(t, database, deepest, pageSize)

	t.Logf("page 1 read %d rows, page %d read %d rows", first, deepPage, deep)

	assert.LessOrEqual(t, float64(deep), float64(first)*tolerance,
		"page %d must not read more than a fifth beyond page 1", deepPage)
}

// rowsReadByOnePage answers the count of plant rows that one read touches, from
// the counters that PostgreSQL keeps for the current transaction.
func rowsReadByOnePage(t *testing.T, database *sqlx.DB, after string, limit int) int64 {
	t.Helper()

	tx := begin(t, database)

	touched := func() int64 {
		var count int64

		require.NoError(t, tx.Get(&count, `
			SELECT coalesce(seq_tup_read, 0) + coalesce(idx_tup_fetch, 0)
			FROM pg_stat_xact_user_tables
			WHERE relname = 'plants';`))

		return count
	}

	before := touched()

	read, err := repository.NewPlant(tx).List(
		context.Background(),
		page.Seek{Direction: page.DirectionForward, Boundary: after, Limit: limit},
		active(),
	)
	require.NoError(t, err)
	require.Len(t, read, limit, "the page is full, so the two reads compare")

	return touched() - before
}
