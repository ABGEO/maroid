---
id: ADR-0001
title: PostgreSQL 18 and the native uuidv7
type: adr
status: accepted
created: 2026-09-12
updated: 2026-09-12
decided: 2026-09-12
changes: [DAT-008, DAT-009]
supersedes:
superseded_by:
---

# PostgreSQL 18 and the native uuidv7

## Context

`DAT-009` gives the reason for its own rule: "PostgreSQL 17 carries no `uuidv7()`
function, and `uuid-ossp` reaches version 4." The application therefore generates
every row identifier with `uuid.NewV7()`.

`DAT-008` enables `uuid-ossp` in the core migrations. No migration and no Go file
calls a function from that extension. `gen_random_uuid()` is built in, and the
plugins call `uuid.NewV7()` in Go.

The `IDENT` feature adds `public.users`. The owner creates each row with a SQL
statement, and no Go code inserts into that table. See open question 2 of
`features/ident/requirements.md`. The row needs an identifier that no Go code
produces.

PostgreSQL 18 carries `uuidv7()`. Both images that Maroid runs have a release:
`timescale/timescaledb:2.30.0-pg18-oss` for `docker-compose.yaml` and
`timescale/timescaledb-ha:pg18.6-ts2.30.0-all` for `chart/values.yaml`.
No production data exists yet, so the database is recreated, not migrated.

## Decision

Maroid runs PostgreSQL 18. The database generates every row identifier:
a table declares `id UUID PRIMARY KEY DEFAULT uuidv7()`, and Go reads the value
back with `RETURNING id`. The core migrations drop `uuid-ossp`.

## Rationale

One rule covers every table, with no case split for a table that Go never inserts
into. The default removes a class of defect: a Go caller that forgets the
identifier now gets a v7 value instead of a zero UUID.

`OWN-005` already gives the database the value of `user_id` on a scoped table.
The identifier follows the same direction, so one table has one source for both
columns that the caller does not name.

`uuid-ossp` costs an extension in every database and answers no call.

## Alternatives

| Alternative                                                                                                                 | Why we did not select it                                                                                                       |
| --------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------ |
| Stay on PostgreSQL 17 and write a `uuid_generate_v7()` function of 12 lines.                                                | Maroid then owns a function that PostgreSQL 18 provides. The version move deletes it again.                                    |
| Stay on PostgreSQL 17 and let the owner paste a UUID into each `INSERT`.                                                     | A version 4 value pasted by mistake is accepted with no error, and it loses the time ordering that `DAT-009` exists for.        |
| Move to PostgreSQL 18, keep `uuid.NewV7()` for a table that Go inserts into, and default only a table with no Go write path. | The rule then carries a case split, and the author of a new table decides which half applies.                                  |

## Consequences

`DAT-008` loses its reference to `uuid-ossp` and names the version, because
`uuidv7()` needs it. The new text:

> The database is PostgreSQL 18 or later, because `DAT-009` uses `uuidv7()`.
>
> PostgreSQL carries the TimescaleDB extension. The core migrations enable it.
>
> A table of measurements over time declares `WITH (tsdb.hypertable)`.

`DAT-009` loses the paragraph about PostgreSQL 17. The new text:

> Every table that a Maroid migration creates has the column `id` of the type
> `UUID`. The value is a UUID version 7, and the column declares
> `DEFAULT uuidv7()`. An insert does not name `id`. A caller that needs the value
> reads it with `RETURNING id`.
>
> A natural key or a content hash becomes a unique constraint, never the primary
> key. A hypertable puts the partitioning column in the key: `PRIMARY KEY (id, time)`.
>
> **Why:** Version 7 sorts by time, so the index keeps its locality and the row
> carries its creation order. One source for the value means no caller can omit it.

- Positive: `public.users` needs no Go write path. One rule covers every table.
- Negative: a caller that needs the identifier before the insert reads it back
  with `RETURNING id` instead of holding it in advance.
- Work that follows:
  - Change the image tag in `docker-compose.yaml` and in `chart/values.yaml`.
  - Delete `apps/hub/db/migrations/20251107142304_extension_uuid_ossp_create.*`.
  - Recreate the local database and the deployed database.
  - `features/ident/spec.md` declares `public.users` under this rule.
  - No plugin table carries a `UUID` identifier today. `jasmine` gives `plants.id`
    and `environments.id` the type `TEXT`, and `pensions`, `tbilisi-energy`, and
    `telasi` make a content hash or a natural key the primary key. They contradict
    `DAT-009` now, and this ADR does not change them. One feature for each plugin
    converts its tables.
