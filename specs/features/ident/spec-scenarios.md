---
id: IDENT
title: The scenarios of the user record and the ownership of a row
type: spec
status: approved
created: 2026-09-12
updated: 2026-10-01
approved_by: Temuri
approved_on: 2026-09-12
constrained_by: [TST, OWN, TRC]
requirements: features/ident/requirements.md
---

# Specification: The scenarios of the user record and the ownership of a row

`spec.md` holds the design. This file holds the scenarios, because the two together
pass the size that `LNG-013` gives. See `SPC-001`.

`IDENT-SC-001` through `IDENT-SC-005` and `IDENT-SC-010` build the table
`test_scope.notes` in the container, with the `workspace_id` column and the policy
that `OWN-005` and `OWN-006` give. Two records exist in `public.users`, user A and
user B. Two records exist in `public.workspaces`, workspace A with user A as its
manager, and workspace B with user B as its manager.

## `IDENT-SC-001`

**Verifies:** `IDENT-FR-004`, `IDENT-NFR-001`
**Layer:** integration

**Given** workspace A holds three notes and workspace B holds three notes.
**When** a transaction with workspace A as the acting workspace reads every note.
**Then** the result holds three rows, and the count of the rows of workspace B is zero.

## `IDENT-SC-002`

**Verifies:** `IDENT-FR-003`
**Layer:** integration

**Given** a transaction with workspace A as the acting workspace.
**When** an insert names every column except `workspace_id`.
**Then** the stored row carries the identifier of workspace A.

## `IDENT-SC-003`

**Verifies:** `IDENT-FR-005`
**Layer:** integration

**Given** a note of workspace B.
**When** a transaction with workspace A as the acting workspace updates it, then
deletes it.
**Then** each statement reports zero affected rows, and the note of workspace B
stays unchanged.

## `IDENT-SC-004`

**Verifies:** `IDENT-FR-006`
**Layer:** integration

**Given** a note of workspace B and an identifier that no row holds.
**When** a transaction with workspace A as the acting workspace reads each of the two.
**Then** both reads return `sql.ErrNoRows`, and the handler answers 404 for both.

## `IDENT-SC-005`

**Verifies:** `IDENT-INV-002`
**Layer:** integration

**Given** a context that carries no acting workspace.
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

**Given** workspace B holds three notes, and an administrator sets
`status = 'blocked'` on user B.
**When** user B sends a request with a token that is still valid, and then sends a
Telegram update.
**Then** the request gets 401, the update is dropped, and the three notes of
workspace B stay in the table.

## `IDENT-SC-008`

**Verifies:** `IDENT-FR-010`
**Layer:** integration

**Given** a running hub and a person with no record.
**When** an administrator adds the record and its identity, and that person sends
the next request.
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

**Given** the table `dev_maroid_jasmine.plants`, which carries no `workspace_id` and
no policy.
**When** a member of workspace A reads it, and then a member of workspace B reads it.
**Then** both read every row.

## `IDENT-SC-012`

**Verifies:** `IDENT-NFR-002`
**Layer:** integration

**Given** 200 records in `public.users`, each a member of one workspace, and a hub
with a warm connection pool.
**When** 1000 requests arrive, each under the path of the workspace of its user.
**Then** the time from the arrival of the request to the first statement of the
handler at the database is 10 milliseconds or less at the 95th percentile.

## `IDENT-SC-013`

**Verifies:** `IDENT-FR-006`
**Layer:** integration

**Given** user A, who is a member of workspace A and not of workspace B.
**When** user A sends a request under the path of workspace B.
**Then** the answer is 404, `not-found`, and the next handler does not run.

## `IDENT-SC-014`

**Verifies:** `IDENT-FR-007`
**Layer:** unit

**Given** a job that declares `CronScopePerWorkspace`, and a fake lister with two
workspaces that enable the plugin of the job.
**When** the scheduler fires the entry one time.
**Then** `Run` runs two times, the context of each run carries a different one of
the two workspace identifiers, and no context carries an acting user.

## Retired identifiers

| ID             | Retired    | Reason                                                                                    |
| -------------- | ---------- | ------------------------------------------------------------------------------------------- |
| `IDENT-SC-011` | 2026-09-15 | It proves the unique constraint on `telegram_id`, which the migration of `EXTID` drops. `EXTID-SC-007` succeeds it. |
