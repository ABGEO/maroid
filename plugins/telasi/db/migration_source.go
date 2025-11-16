// Package db provides access to the embedded SQL migration files.
package db

import "embed"

// Migrations contains all SQL migration files embedded in the plugin.
//
//go:embed migrations/*.sql
var Migrations embed.FS
