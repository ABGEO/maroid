---
id: DAT
title: Data
type: guideline
status: active
created: 2026-09-11
updated: 2026-09-12
scope: [apps/hub/db/, apps/hub/internal/migrator/, plugins/*/db/, plugins/*/repository/]
related: [ARC, PLG, OWN]
---

# Data

This guideline holds the rules for the persistence of the data.
`ARC-004` selects PostgreSQL. `OWN` gives the user that a row belongs to.

## DAT-001

Each plugin owns one PostgreSQL schema.
The schema name is the plugin identifier. The character `_` replaces the dot and the hyphen.
The plugin `dev.maroid.jasmine` owns the schema `dev_maroid_jasmine`.

**Why:** A plugin cannot corrupt the data of another plugin by accident.
The schema name comes from the identifier, so the hub needs no configuration for it.

## DAT-002

The hub owns the schema `public`.

## DAT-003

A plugin does not read a table and does not write a table in a different schema.
A plugin gets data from another component through `pluginapi.Host`.

## DAT-004

A plugin reaches the database through `pluginapi.PluginDB`.
`PluginDB` sets the `search_path` to the schema of the plugin and to `public`.
A plugin does not use the `*sqlx.DB` handle directly.

**Why:** The search path enforces `DAT-001` and `DAT-003` at runtime.

## DAT-005

Migrations use golang-migrate.
A migration has two files: `<timestamp>_<name>.up.sql` and `<timestamp>_<name>.down.sql`.
The timestamp has the form `YYYYMMDDHHMMSS`.

## DAT-006

The migrations of a plugin live in `plugins/<name>/db/migrations/`.
The plugin embeds them and returns them from `Migrations()`.
The migrations of the hub live in `apps/hub/db/migrations/`.

**Why:** The migration travels inside the `.so` file. The operator copies one file.

## DAT-007

Each schema holds its own `schema_migrations` table.
The migration of one plugin does not block the migration of another plugin.

## DAT-008

The database is PostgreSQL 18 or later, because `DAT-009` uses `uuidv7()`.

PostgreSQL carries the TimescaleDB extension. The core migrations enable it.

A table of measurements over time declares `WITH (tsdb.hypertable)`.

## DAT-009

Every table that a Maroid migration creates has the column `id` of the type `UUID`.
The value is a UUID version 7, and the column declares `DEFAULT uuidv7()`.
An insert does not name `id`. A caller that needs the value reads it with
`RETURNING id`.

A natural key or a content hash becomes a unique constraint, never the primary key.
A hypertable puts the partitioning column in the key: `PRIMARY KEY (id, time)`.

**Why:** Version 7 sorts by time, so the index keeps its locality and the row
carries its creation order. One source for the value means no caller can omit it.

## Retired identifiers

This file has no retired identifier.
