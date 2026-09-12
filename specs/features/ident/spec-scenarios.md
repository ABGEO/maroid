---
id: IDENT
title: The scenarios of the user record and the ownership of a row
type: spec
status: approved
created: 2026-09-12
updated: 2026-09-12
approved_by: Temuri
approved_on: 2026-09-12
constrained_by: [TST, OWN, TRC]
requirements: features/ident/requirements.md
---

# Specification: The scenarios of the user record and the ownership of a row

`spec.md` holds the design. This file holds the scenarios, because the two together
pass the size that `LNG-013` gives. See `SPC-001`.

`IDENT-SC-001` through `IDENT-SC-005` and `IDENT-SC-010` build the table
`test_scope.notes` in the container, with the column and the policy that `OWN-005`
and `OWN-006` give. Two records exist in `public.users`, user A and user B.

## `IDENT-SC-001`

**Verifies:** `IDENT-FR-004`, `IDENT-NFR-001`
**Layer:** integration

**Given** user A holds three notes and user B holds three notes.
**When** a transaction with user A as the acting user reads every note.
**Then** the result holds three rows, and the count of the rows of user B is zero.

## `IDENT-SC-002`

**Verifies:** `IDENT-FR-003`
**Layer:** integration

**Given** a transaction with user A as the acting user.
**When** an insert names every column except `user_id`.
**Then** the stored row carries the identifier of user A.

## `IDENT-SC-003`

**Verifies:** `IDENT-FR-005`
**Layer:** integration

**Given** a note of user B.
**When** a transaction with user A as the acting user updates it, then deletes it.
**Then** each statement reports zero affected rows, and the note of user B stays
unchanged.

## `IDENT-SC-004`

**Verifies:** `IDENT-FR-006`
**Layer:** integration

**Given** a note of user B and an identifier that no row holds.
**When** a transaction with user A as the acting user reads each of the two.
**Then** both reads return `sql.ErrNoRows`, and the handler answers 404 for both.

## `IDENT-SC-005`

**Verifies:** `IDENT-INV-002`
**Layer:** integration

**Given** a context that carries no acting user.
**When** `WithTx` reads every note, then inserts one.
**Then** the read returns zero rows and the insert fails with the policy error.

## `IDENT-SC-006`

**Verifies:** `IDENT-FR-002`
**Layer:** unit

**Given** a valid token whose subject names no record, and a fake resolver that
returns `sql.ErrNoRows`.
**When** the request reaches `auth.Middleware`.
**Then** the answer is 401 and the next handler does not run.

## `IDENT-SC-007`

**Verifies:** `IDENT-FR-002`, `IDENT-FR-009`, `IDENT-FR-011`
**Layer:** integration

**Given** user B holds three notes, and the owner sets `status = 'blocked'`.
**When** user B sends a request with a token that is still valid, and then sends a
Telegram update.
**Then** the request gets 401, the update is dropped, and the three notes of user B
stay in the table.

## `IDENT-SC-008`

**Verifies:** `IDENT-FR-010`
**Layer:** integration

**Given** a running hub and a person with no record.
**When** one `INSERT` adds the record, and that person sends the next request.
**Then** the request succeeds with no restart of the hub.

## `IDENT-SC-009`

**Verifies:** `IDENT-FR-007`
**Layer:** unit

**Given** a job that declares `CronScopePerUser`, and a fake lister with two active
users.
**When** the scheduler fires the entry one time.
**Then** `Run` runs two times, and the context of each run carries a different one
of the two identifiers.

## `IDENT-SC-010`

**Verifies:** `IDENT-FR-008`
**Layer:** integration

**Given** the table `dev_maroid_jasmine.plants`, which carries no `user_id` and no
policy.
**When** user A reads it, and then user B reads it.
**Then** both read every row.

## `IDENT-SC-011`

**Verifies:** `IDENT-FR-001`, `IDENT-INV-003`
**Layer:** integration

**Given** a record with the Telegram identifier 123456789.
**When** a second insert names the same Telegram identifier.
**Then** the unique constraint refuses it.

## `IDENT-SC-012`

**Verifies:** `IDENT-NFR-002`
**Layer:** integration

**Given** 200 records in `public.users` and a hub with a warm connection pool.
**When** 1000 requests arrive.
**Then** the time from the arrival of the request to the first statement of the
handler at the database is 10 milliseconds or less at the 95th percentile.


## Retired identifiers

This file has no retired identifier.
