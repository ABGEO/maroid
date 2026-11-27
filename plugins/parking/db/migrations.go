package db

import "embed"

// Migrations holds every SQL migration of the plugin.
//
//go:embed migrations/*.sql
var Migrations embed.FS
