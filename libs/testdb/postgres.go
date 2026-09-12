// Package testdb starts a PostgreSQL instance that one integration test owns.
package testdb

import (
	"context"
	"fmt"
	"io/fs"
	"testing"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx" // The PostgreSQL driver for migrate
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "github.com/jackc/pgx/stdlib" // The PostgreSQL driver for sqlx
	"github.com/jmoiron/sqlx"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	// Image is the PostgreSQL that an integration test runs against.
	Image = "timescale/timescaledb:2.30.0-pg18-oss"

	// Role is the login role that Start connects as.
	Role = "maroid"

	// Database is the database that Role owns.
	Database = "maroid"

	superUser      = "postgres"
	password       = "password"
	startTimeout   = 3 * time.Minute
	readyOccurs    = 2
	connectRetries = 30
	retryInterval  = time.Second
)

// Instance is a running PostgreSQL that one test owns.
type Instance struct {
	// DB connects as Role, which owns Database.
	DB *sqlx.DB
	// DSN is the connection string that DB uses.
	DSN string
}

// Start launches PostgreSQL in a container and returns a handle that connects as
// Role. The role owns the database, so a policy that declares
// FORCE ROW LEVEL SECURITY applies to every statement that the test runs.
//
// The container stops when the test ends. The test is skipped under -short.
func Start(t *testing.T) *Instance {
	t.Helper()

	if testing.Short() {
		t.Skip("skipping the integration test, it needs Docker")
	}

	ctx, cancel := context.WithTimeout(context.Background(), startTimeout)
	defer cancel()

	container, err := postgres.Run(
		ctx,
		Image,
		postgres.WithDatabase(superUser),
		postgres.WithUsername(superUser),
		postgres.WithPassword(password),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(readyOccurs).
				WithStartupTimeout(startTimeout),
		),
	)
	if err != nil {
		t.Fatalf("starting the PostgreSQL container: %v", err)
	}

	t.Cleanup(func() {
		if err := testcontainers.TerminateContainer(container); err != nil {
			t.Logf("terminating the PostgreSQL container: %v", err)
		}
	})

	endpoint, err := container.Endpoint(ctx, "")
	if err != nil {
		t.Fatalf("reading the container endpoint: %v", err)
	}

	createRole(t, endpoint)

	return &Instance{
		DB:  connect(t, endpoint, Role, Database),
		DSN: DSN(endpoint, Role, Database),
	}
}

// Migrate applies every migration in filesystem to the schema.
// It creates the schema when the schema does not exist.
func (i *Instance) Migrate(t *testing.T, schema string, filesystem fs.FS) {
	t.Helper()

	if _, err := i.DB.Exec(fmt.Sprintf(`CREATE SCHEMA IF NOT EXISTS %q`, schema)); err != nil {
		t.Fatalf("creating the schema %q: %v", schema, err)
	}

	source, err := iofs.New(filesystem, ".")
	if err != nil {
		t.Fatalf("reading the migrations: %v", err)
	}

	instance, err := migrate.NewWithSourceInstance("iofs", source, i.migrationDSN(schema))
	if err != nil {
		t.Fatalf("initializing the migrator: %v", err)
	}

	t.Cleanup(func() {
		sourceErr, databaseErr := instance.Close()
		if sourceErr != nil {
			t.Logf("closing the migration source: %v", sourceErr)
		}

		if databaseErr != nil {
			t.Logf("closing the migration database: %v", databaseErr)
		}
	})

	if err = instance.Up(); err != nil {
		t.Fatalf("running the migrations: %v", err)
	}
}

// DSN builds the connection string of the endpoint, for the given role.
func DSN(endpoint string, role string, database string) string {
	return fmt.Sprintf("pgx://%s:%s@%s/%s?sslmode=disable", role, password, endpoint, database)
}

func (i *Instance) migrationDSN(schema string) string {
	return fmt.Sprintf(
		`%s&x-migrations-table-quoted=true&x-migrations-table="%s"."schema_migrations"`+
			`&options=-csearch_path%%3D%s,public`,
		i.DSN,
		schema,
		schema,
	)
}

func createRole(t *testing.T, endpoint string) {
	t.Helper()

	bootstrap := []string{
		`CREATE ROLE ` + Role + ` LOGIN PASSWORD '` + password + `' ` +
			`NOSUPERUSER NOCREATEROLE NOBYPASSRLS`,
		`CREATE DATABASE ` + Database + ` OWNER ` + Role,
	}

	admin := connect(t, endpoint, superUser, superUser)
	for _, statement := range bootstrap {
		if _, err := admin.Exec(statement); err != nil {
			t.Fatalf("bootstrapping the role %q: %v", Role, err)
		}
	}

	owner := connect(t, endpoint, superUser, Database)
	if _, err := owner.Exec(`CREATE EXTENSION IF NOT EXISTS timescaledb`); err != nil {
		t.Fatalf("creating the TimescaleDB extension: %v", err)
	}
}

func connect(t *testing.T, endpoint string, role string, database string) *sqlx.DB {
	t.Helper()

	var (
		db  *sqlx.DB
		err error
	)

	for range connectRetries {
		db, err = sqlx.Connect("pgx", DSN(endpoint, role, database))
		if err == nil {
			break
		}

		time.Sleep(retryInterval)
	}

	if err != nil {
		t.Fatalf("connecting to %q as %q: %v", database, role, err)
	}

	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Logf("closing the connection to %q: %v", database, err)
		}
	})

	return db
}
