// Package database provides functionality for initializing and managing
// database connections used by the application.
package database

import (
	"fmt"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib" // The PostgreSQL driver for sqlx
	"github.com/jmoiron/sqlx"

	"github.com/abgeo/maroid/apps/hub/internal/config"
)

// The limits close a connection before an idle timeout outside the hub, such as a
// NAT or a proxy, can drop it.
const (
	connMaxIdleTime = 5 * time.Minute
	connMaxLifetime = 30 * time.Minute
)

// New creates and returns a new PostgreSQL connection using the provided configuration.
// It returns an error if the connection cannot be established.
func New(cfg *config.Config) (*sqlx.DB, error) {
	db, err := sqlx.Connect("pgx", cfg.Database.DSN())
	if err != nil {
		return nil, fmt.Errorf("connecting to database: %w", err)
	}

	db.SetConnMaxIdleTime(connMaxIdleTime)
	db.SetConnMaxLifetime(connMaxLifetime)

	return db, nil
}
