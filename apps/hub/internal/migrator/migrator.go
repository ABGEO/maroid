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
	"strings"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx" // The PostgreSQL driver for migrate
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jmoiron/sqlx"

	"github.com/abgeo/maroid/apps/hub/db"
	"github.com/abgeo/maroid/apps/hub/internal/config"
	"github.com/abgeo/maroid/apps/hub/internal/domain/errs"
	"github.com/abgeo/maroid/libs/pluginapi"
)

// Migration target constants define which components should have their database migrations executed.
const (
	TargetCore = "core"
	TargetAll  = "all"
)

// Migrator is responsible for running database migrations for core and plugin components.
type Migrator struct {
	config   *config.Config
	logger   *slog.Logger
	database *sqlx.DB
	plugins  []pluginapi.Plugin
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
	plugins []pluginapi.Plugin,
) *Migrator {
	return &Migrator{
		config: cfg,
		logger: logger.With(
			slog.String("component", "migrator"),
		),
		database: database,
		plugins:  plugins,
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
			slog.String("migration-component", component),
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
	coreFS, err := m.getCoreFilesystem()
	if err != nil {
		return nil, err
	}

	if target == TargetCore {
		return &migrationPlan{
			filesystems: map[string]fs.FS{TargetCore: coreFS},
			order:       []string{TargetCore},
		}, nil
	}

	pluginFS, err := m.collectPluginFilesystems()
	if err != nil {
		return nil, err
	}

	pluginIDs := slices.Collect(maps.Keys(pluginFS))

	switch target {
	case TargetAll:
		pluginFS[TargetCore] = coreFS

		return &migrationPlan{
			filesystems: pluginFS,
			order:       append([]string{TargetCore}, pluginIDs...),
		}, nil

	default:
		if slices.Contains(pluginIDs, target) {
			return &migrationPlan{
				filesystems: pluginFS,
				order:       []string{target},
			}, nil
		}

		return nil, fmt.Errorf("%w: invalid plugin ID %s", errs.ErrUnknownMigrationTarget, target)
	}
}

func (m *Migrator) getCoreFilesystem() (fs.FS, error) {
	coreFS, err := fs.Sub(db.GetMigrationsFS(), "migrations")
	if err != nil {
		return nil, fmt.Errorf("failed to access core migrations: %w", err)
	}

	return coreFS, nil
}

func (m *Migrator) collectPluginFilesystems() (map[string]fs.FS, error) {
	filesystems := make(map[string]fs.FS)

	for _, plugin := range m.plugins {
		pluginID := plugin.Meta().ID.String()

		migrationPlugin, ok := plugin.(pluginapi.MigrationPlugin)
		if !ok {
			continue
		}

		filesystem, err := migrationPlugin.Migrations()
		if err != nil {
			return nil, fmt.Errorf("failed to read migrations for plugin %s: %w", pluginID, err)
		}

		filesystems[pluginID] = filesystem
	}

	return filesystems, nil
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
	replacer := strings.NewReplacer(".", "_", "-", "_")

	return replacer.Replace(component)
}
