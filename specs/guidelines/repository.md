---
id: REP
title: The repository layer
type: guideline
status: active
created: 2026-09-11
updated: 2026-09-11
scope: [plugins/*/repository/, plugins/*/model/]
related: [DAT, PKG, GO, OWN]
---

# The repository layer

`DAT` gives the schema and the migrations. This guideline gives the Go code that
reads and writes them. `OWN-008` keeps the user filter out of that code.

## REP-001

Data access lives in `repository/`. One file holds one entity.

## REP-002

Declare `<Entity>Repository` as the interface. Name the implementation `<Entity>`.
Assert the implementation at compile time. See `GO-007`.

## REP-003

A repository takes `*sqlx.Tx`. It does not take `*sqlx.DB` and it does not take
`pluginapi.PluginDB`.

The caller creates the repository inside `PluginDB.WithTx`.

**Why:** `DAT-004` sets the search path on the transaction. A repository that holds
the pool would read the wrong schema.

## REP-004

A query uses named parameters and a `Context` variant:
`NamedExecContext`, `GetContext`, `SelectContext`.

Never build a query by string concatenation with a value.

## REP-005

An entity lives in `model/`. Every field carries a `db` tag.
An entity carries no `json` tag of a third party. See `EXT-006`.

## REP-006

A write that a retry can repeat is idempotent in the database.
Use a natural key or a content hash with `ON CONFLICT ... DO NOTHING`.

**Why:** A cron job that reruns after a failure must not duplicate a bill.

## REP-007

Wrap an error with the operation and the entity: `"inserting BillingItem: %w"`.
See `GO-004`.

## Retired identifiers

This file has no retired identifier.
