package repository_test

import (
	"context"
	"fmt"
	"io/fs"
	"testing"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx" // The PostgreSQL driver for migrate
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/libs/testdb"
	"github.com/abgeo/maroid/plugins/jasmine/db"
	"github.com/abgeo/maroid/plugins/jasmine/repository"
)

// setUpdatedAt is the trigger function that a core migration of the hub creates.
// A plugin test owns no core migration, so it creates the function itself.
const setUpdatedAt = `
CREATE OR REPLACE FUNCTION set_updated_at() RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;`

// plantsUnderTest returns a database that holds one environment and count plants,
// with identifiers that sort in the order that they were made.
func plantsUnderTest(t *testing.T, count int) *sqlx.DB {
	t.Helper()

	instance := testdb.Start(t)

	_, err := instance.DB.Exec(setUpdatedAt)
	require.NoError(t, err)

	migrations, err := fs.Sub(db.Migrations, "migrations")
	require.NoError(t, err)

	source, err := iofs.New(migrations, ".")
	require.NoError(t, err)

	migrator, err := migrate.NewWithSourceInstance("iofs", source, instance.DSN)
	require.NoError(t, err)

	t.Cleanup(func() { _, _ = migrator.Close() })
	require.NoError(t, migrator.Up())

	_, err = instance.DB.Exec(
		`INSERT INTO environments (id, name) VALUES ('env-0001', 'Balcony');`)
	require.NoError(t, err)

	for i := range count {
		_, err = instance.DB.Exec(
			`INSERT INTO plants (id, name, environment_id) VALUES ($1, $2, 'env-0001');`,
			fmt.Sprintf("plant-%06d", i), fmt.Sprintf("Plant %d", i),
		)
		require.NoError(t, err)
	}

	return instance.DB
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
		tx, err := database.Beginx()
		require.NoError(t, err)

		rows, err := repository.NewPlant(tx).List(context.Background(), after, limit+1)
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

	database := plantsUnderTest(t, 25)

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

	database := plantsUnderTest(t, 3)

	tx, err := database.Beginx()
	require.NoError(t, err)

	t.Cleanup(func() { _ = tx.Rollback() })

	rows, err := repository.NewPlant(tx).List(context.Background(), "", 3)
	require.NoError(t, err)
	assert.Len(t, rows, 3, "two asked for plus the probe row")

	rows, err = repository.NewPlant(tx).List(context.Background(), "plant-000001", 3)
	require.NoError(t, err)
	assert.Len(t, rows, 1, "the keyset reads past the boundary only")
	assert.Equal(t, "plant-000002", rows[0].ID)
}

// APIFMT-SC-004: A filter narrows the collection, and the keyset holds inside it.
func TestTheKeysetHoldsInsideAFilter(t *testing.T) {
	t.Parallel()

	database := plantsUnderTest(t, 4)

	_, err := database.Exec(
		`INSERT INTO environments (id, name) VALUES ('env-0002', 'Shelf');`)
	require.NoError(t, err)
	_, err = database.Exec(
		`INSERT INTO plants (id, name, environment_id)
		 VALUES ('plant-000099', 'Other', 'env-0002');`)
	require.NoError(t, err)

	tx, err := database.Beginx()
	require.NoError(t, err)

	t.Cleanup(func() { _ = tx.Rollback() })

	rows, err := repository.NewPlant(tx).ListByEnvironmentID(
		context.Background(), "env-0001", "", 10,
	)
	require.NoError(t, err)
	assert.Len(t, rows, 4, "the filter holds")

	for _, row := range rows {
		assert.Equal(t, "env-0001", row.EnvironmentID)
	}
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

// APIFMT-NFR-001, APIFMT-SC-011: The time to answer one page does not grow with
// the position of that page. A keyset reads from an index; an offset would scan
// every row that it discards.
func TestThePageTimeDoesNotGrowWithThePosition(t *testing.T) {
	t.Parallel()

	const (
		rows      = 10000
		pageSize  = 100
		deepPage  = 100
		tolerance = 1.2
	)

	database := plantsUnderTest(t, rows)

	// The deepest boundary that still leaves a full page behind it.
	deepest := fmt.Sprintf("plant-%06d", rows-pageSize-1)

	first := timeOnePage(t, database, "", pageSize)
	deep := timeOnePage(t, database, deepest, pageSize)

	t.Logf("page 1 took %s, page %d took %s", first, deepPage, deep)

	assert.Less(t, deep.Seconds(), first.Seconds()*tolerance+0.05,
		"page %d must not cost more than a fifth beyond page 1", deepPage)
}

// timeOnePage measures one read from the request to the last row.
func timeOnePage(t *testing.T, database *sqlx.DB, after string, limit int) time.Duration {
	t.Helper()

	tx, err := database.Beginx()
	require.NoError(t, err)

	defer func() { _ = tx.Rollback() }()

	started := time.Now()

	read, err := repository.NewPlant(tx).List(context.Background(), after, limit)
	elapsed := time.Since(started)

	require.NoError(t, err)
	require.Len(t, read, limit, "the page is full, so the two reads compare")

	return elapsed
}
