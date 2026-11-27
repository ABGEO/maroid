---
id: PARKHIST
title: The scenarios of the parking history
type: spec
status: approved
created: 2026-11-27
updated: 2026-11-27
approved_by: Temuri
approved_on: 2026-11-27
constrained_by: [TST]
requirements: features/parkhist/requirements.md
---

# Scenarios: The parking history of a user

A unit scenario runs the collector against a fake parking service and a fake store. An
integration scenario runs the repositories against the database of `libs/testdb`. A page of the
fake holds 20 rows and the `result.page` of the capture. User A acts in the time zone Asia/Tbilisi.

## `PARKHIST-SC-001`

**Verifies:** `PARKHIST-FR-001`, `PARKHIST-FR-002`
**Layer:** unit

**Given** the plugin with an empty config.
**When** the test reads `CronJobs`.
**Then** one job holds the identifier `history_collector`, the schedule `15 * * * *`, and the scope `CronScopePerUser`.

## `PARKHIST-SC-002`

**Verifies:** `PARKHIST-FR-003`
**Layer:** unit

**Given** user A with no `session` secret.
**When** the job runs for user A.
**Then** it answers nil, logs a skip at info, and the fake parking service records no call.

## `PARKHIST-SC-003`

**Verifies:** `PARKHIST-FR-004`
**Layer:** unit

**Given** a pending connection of user A and a fake parking service.
**When** user A verifies the code, as `PARKACC-SC-009` does.
**Then** the fake records no call of `/parking/history` and no call of `/parking/person/payment-history`, and the store records no write.

## `PARKHIST-SC-004`

**Verifies:** `PARKHIST-FR-005`
**Layer:** unit

**Given** the store holds every row of page 0 of both lists.
**When** `Collect` runs with `ReachRecent`.
**Then** the fake records page 0 of each list alone, and the store records no write.

## `PARKHIST-SC-005`

**Verifies:** `PARKHIST-FR-005`
**Layer:** unit

**Given** page 0 of the sessions holds one new row and 19 held rows, page 1 holds 20 held rows, and page 0 of the transactions holds one new row dated before a held row of the same page.
**When** `Collect` runs with `ReachRecent`.
**Then** the fake records pages 0 and 1 of the sessions, and pages 0 and 1 of the transactions. The store saves one session and one transaction.

## `PARKHIST-SC-006`

**Verifies:** `PARKHIST-FR-006`, `PARKHIST-FR-007`
**Layer:** unit

**Given** the store holds no row, and the fake reports 24 pages of sessions and 3 pages of transactions.
**When** `Collect` runs with `ReachRecent` and `collector_pages` 5.
**Then** the fake records pages 0 to 4 of the sessions and pages 0 to 2 of the transactions. The store saves 100 sessions and every transaction. The log holds one warning that names `sessions` and "run backfill", and none for the transactions.

## `PARKHIST-SC-007`

**Verifies:** `PARKHIST-FR-008`
**Layer:** unit

**Given** page 0 holds one row with `status` `ACTIVE` and no `endDate`, then 19 held rows.
**When** `Collect` runs with `ReachRecent`.
**Then** the fake records page 0 alone, and the store saves no session.

## `PARKHIST-SC-008`

**Verifies:** `PARKHIST-FR-009`
**Layer:** integration

**Given** user A holds 10 sessions, the store holds no transaction, and page 2 of the transactions answers 500.
**When** `Collect` runs with `ReachFull`.
**Then** it answers an error that matches `ErrRequestFailed`. User A holds the same 10 sessions and no transaction.

## `PARKHIST-SC-009`

**Verifies:** `PARKHIST-FR-009`
**Layer:** unit

**Given** a collector that fails for user A and succeeds for user B.
**When** the job runs for A, then for B.
**Then** the run of A answers the error, and the run of B answers nil.

## `PARKHIST-SC-010`

**Verifies:** `PARKHIST-FR-010`, `PARKHIST-FR-017`
**Layer:** integration

**Given** user A holds 10 sessions and a `session` secret, and page 0 of the sessions answers 401.
**When** the job runs for user A, then runs again.
**Then** the first run deletes the secret, stores nothing, and answers nil after a log at info. The second run makes no call. User A holds the same 10 sessions.

## `PARKHIST-SC-011`

**Verifies:** `PARKHIST-FR-011`
**Layer:** integration

**Given** user A holds the session 45304769 with an amount of 0.55.
**When** the store saves it again with 0.55, then with 0.60.
**Then** user A holds one row of 45304769. Its amount is 0.60. The first save changes no `collected_at`.

## `PARKHIST-SC-012`

**Verifies:** `PARKHIST-FR-012`, `PARKHIST-FR-013`
**Layer:** unit

**Given** a row of the capture: `id` 45304769, `FINISHED`, `type` `BOTH`, `govNumber` AB071EO, lot A1637 with its address, `startDate` `2026-11-21T13:26:08.347292`, `endDate` `2026-11-21T14:14:21.809195`, `amount` 0.55.
**When** `HistorySession.ToModel` runs.
**Then** the model holds every value, `started_at` 09:26:08.347292 UTC, `ended_at` 10:14:21.809195 UTC, and `parking_type` `BOTH`.

## `PARKHIST-SC-013`

**Verifies:** `PARKHIST-FR-014`, `PARKHIST-FR-015`
**Layer:** unit

**Given** four rows of the ledger: `FILL_BALANCE` with the description `" Fill Balance "`, `ZONE_PARKING_PAYMENT` with `parkingId` 45304769, `CARD_PAYMENT` with `cardId` set, and `REFUND`. Each `operationDate` is `2026-11-21T10:14:21.827+0000`.
**When** `PaymentEntry.ToModel` runs for each.
**Then** the kinds are `top_up`, `parking_payment`, `permit_purchase`, and `other`, each `service_kind` keeps the value, the first description is `Fill Balance`, the second links 45304769, and each moment is 10:14:21.827 UTC.

## `PARKHIST-SC-014`

**Verifies:** `PARKHIST-FR-018`, `PARKHIST-FR-019`
**Layer:** unit

**Given** user A holds 30 sessions of the 468 that the fake reports on 24 pages, and the fake reports 393 transactions on 20 pages.
**When** `backfill --user <A>` runs.
**Then** the fake records every page of both lists, the store saves 468 sessions and 393 transactions, and the command prints `438 new sessions, 393 new transactions`.

## `PARKHIST-SC-015`

**Verifies:** `PARKHIST-FR-020`
**Layer:** unit

**Given** user A with no `session` secret.
**When** `backfill --user <A>` runs, then `backfill --user abc`.
**Then** the first fails with `ErrNotConnected` and no call. The second fails in `PreRunE` with `ErrInvalidUser`.

## `PARKHIST-SC-016`

**Verifies:** `PARKHIST-INV-001`
**Layer:** integration

**Given** user A and user B each hold sessions and transactions, and both are members of workspace W.
**When** user A reads each table, and writes a row with the `external_id` of a row of B.
**Then** user A reads no row of B. The write adds a row of A, and the row of B stays.

## `PARKHIST-SC-017`

**Verifies:** `PARKHIST-INV-002`
**Layer:** unit

**Given** rows of the capture that hold `walletNumber`, `personId`, and `ownerCode`.
**When** they pass through the DTOs, the store, and each route of `spec-pages.md`.
**Then** no answer and no log line holds the wallet number or the personal identifier.

## `PARKHIST-SC-018`

**Verifies:** `PARKHIST-FR-018`
**Layer:** unit

**Given** a fake that reports a full page of new rows on every page, with no end.
**When** `Collect` runs with `ReachFull`.
**Then** it answers `ErrPageLimit` after page 499 of the sessions, and the store records no write.

## `PARKHIST-SC-019`

**Verifies:** `PARKHIST-FR-005`, `PARKHIST-FR-013`, `PARKHIST-FR-018`
**Layer:** manual

**Given** the account of the owner, connected on the hub of the developer machine.
**When** the owner runs `backfill`, parks once, and waits for the next scheduled run.
**Then** the sessions page shows the times that the mobile application shows, and the new session appears after the run. The log of the run names two calls of each list.

## Retired identifiers

This file has no retired identifier.
