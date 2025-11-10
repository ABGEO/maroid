// Package db provides access to the application's migrations.
package db

import "embed"

//go:embed migrations/*.sql
var migrations embed.FS

// GetMigrationsFS returns the embedded filesystem containing the database migration files.
func GetMigrationsFS() embed.FS {
	return migrations
}
