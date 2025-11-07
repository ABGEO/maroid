// Package database provides functionality for initializing and managing
// database connections used by the application.
package database

import (
	"fmt"

	_ "github.com/jackc/pgx/stdlib" // The PostgreSQL driver for sqlx
	"github.com/jmoiron/sqlx"

	"github.com/abgeo/maroid/apps/hub/internal/config"
)

// New creates and returns a new PostgreSQL connection using the provided configuration.
// It returns an error if the connection cannot be established.
func New(cfg *config.Config) (*sqlx.DB, error) {
	db, err := sqlx.Connect("pgx", cfg.Database.DSN())
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	return db, nil
}
