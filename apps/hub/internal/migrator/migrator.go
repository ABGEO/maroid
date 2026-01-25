// Package migrator provides functionality to run database migrations
// for both the core application and plugin components.
package migrator

import (
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"maps"
	"slices"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx" // The PostgreSQL driver for migrate
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jmoiron/sqlx"

	"github.com/abgeo/maroid/apps/hub/internal/config"
	"github.com/abgeo/maroid/apps/hub/internal/domain/errs"
	"github.com/abgeo/maroid/apps/hub/internal/registry"
	"github.com/abgeo/maroid/libs/pluginapi"
)

// Migration target constants define which components should have their database migrations executed.
const (
	TargetCore = "core"
	TargetAll  = "all"
)

// Migrator is responsible for running database migrations for core and plugin components.
type Migrator struct {
	config            *config.Config
	logger            *slog.Logger
	database          *sqlx.DB
	migrationRegistry *registry.MigrationRegistry
}

type migrationPlan struct {
	filesystems map[string]fs.FS
	order       []string
}

// New creates a new Migrator instance.
func New(
	cfg *config.Config,
	logger *slog.Logger,
	database *sqlx.DB,
	migrationRegistry *registry.MigrationRegistry,
) *Migrator {
	return &Migrator{
		config: cfg,
		logger: logger.With(
			slog.String("component", "migrator"),
		),
		database:          database,
		migrationRegistry: migrationRegistry,
	}
}

// Up runs the migrations up to the specified target component or "all".
func (m *Migrator) Up(target string) error {
	plan, err := m.buildMigrationPlan(target)
	if err != nil {
		return err
	}

	for _, component := range plan.order {
		m.logger.Info(
			"running migrations up",
			slog.String("migration_component", component),
		)

		if component != TargetCore {
			schema := buildSchemaName(component)
			if err = m.ensureSchema(schema); err != nil {
				return err
			}
		}

		if err = m.migrateComponent(component, plan.filesystems[component]); err != nil {
			return err
		}
	}

	return nil
}

func (m *Migrator) ensureSchema(schema string) error {
	query := fmt.Sprintf(`CREATE SCHEMA IF NOT EXISTS "%s"`, schema)

	_, err := m.database.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to ensure schema %q: %w", schema, err)
	}

	return nil
}

func (m *Migrator) migrateComponent(component string, filesystem fs.FS) error {
	schema := "public"
	if component != TargetCore {
		schema = buildSchemaName(component)
	}

	instance, err := m.newMigrateInstance(filesystem, schema)
	if err != nil {
		return err
	}

	err = instance.Up()
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("failed to run up migrations: %w", err)
	}

	return nil
}

func (m *Migrator) buildMigrationPlan(target string) (*migrationPlan, error) {
	migrations := m.migrationRegistry.All()
	plan := &migrationPlan{
		filesystems: migrations,
		order:       slices.Collect(maps.Keys(migrations)),
	}

	switch target {
	case TargetCore:
		plan.order = []string{TargetCore}

		return plan, nil

	case TargetAll:
		return plan, nil

	default:
		if _, exists := migrations[target]; !exists {
			return nil, fmt.Errorf(
				"%w: invalid plugin ID %s",
				errs.ErrUnknownMigrationTarget,
				target,
			)
		}

		plan.order = []string{target}

		return plan, nil
	}
}

func (m *Migrator) newMigrateInstance(
	filesystem fs.FS,
	schema string,
) (*migrate.Migrate, error) {
	source, err := iofs.New(filesystem, ".")
	if err != nil {
		return nil, fmt.Errorf("failed to initialize migrations IOFS: %w", err)
	}

	dsn := fmt.Sprintf(
		`%s?x-migrations-table-quoted=true&x-migrations-table="%s"."schema_migrations"&options=-csearch_path%%3D%s,public`,
		m.config.Database.DSN(),
		schema,
		schema,
	)

	instance, err := migrate.NewWithSourceInstance("iofs", source, dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize migrator: %w", err)
	}

	return instance, nil
}

func buildSchemaName(component string) string {
	return pluginapi.ParsePluginID(component).ToSafeName("_")
}
