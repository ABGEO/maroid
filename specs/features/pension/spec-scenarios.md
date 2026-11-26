---
id: PENSION
title: The scenarios of the pension account of a user
type: spec
status: approved
created: 2026-11-27
updated: 2026-11-27
approved_by: Temuri
approved_on: 2026-11-27
constrained_by: [TST, TRC]
requirements: features/pension/requirements.md
---

# Specification: The scenarios of the pension account of a user

A unit scenario runs the plugin against an `httptest` server that plays the pension service,
a fake `PluginSecrets`, a fake `PluginSettings`, and a fake `Store`. "The fake service" is that
server. An integration scenario runs the migrations and the repositories against the database
of `libs/testdb`. User A acts in workspace W, with the time zone Asia/Tbilisi, unless the
scenario says otherwise.

## `PENSION-SC-001`

**Verifies:** `PENSION-FR-001`, `PENSION-FR-002`
**Layer:** unit

**Given** the plugin.
**When** the test reflects the settings schema of `SettingsModel`.
**Then** the schema holds `username` (free text, user scope, required) and `password`
(secret, user scope, required), and no other field.

## `PENSION-SC-002`

**Verifies:** `PENSION-FR-003`
**Layer:** unit

**Given** stored credentials and no `session` secret.
**When** user A reads `GET /connections/self`.
**Then** the answer is 200 with `state: not_connected` and `credentials_stored: true`. The fake
service records no call.

## `PENSION-SC-003`

**Verifies:** `PENSION-FR-004`
**Layer:** unit

**Given** no settings for user A.
**When** user A reads `GET /connections/self`, then sends `PUT /connections/self`.
**Then** the read answers 200 with `credentials_stored: false`. The write answers 422 with the
type `/problems/dev.maroid.pensions/credentials-absent`. The fake service records no call.

## `PENSION-SC-004`

**Verifies:** `PENSION-FR-005`
**Layer:** unit

**Given** a `session` secret, and the fake service answers the participant with
`firstName: Nino`, `lastName: Beridze`, `cummSavings: 12480.55`.
**When** user A reads `GET /connections/self`.
**Then** the answer is 200 with `state: connected`, `holder.full_name: "Nino Beridze"`, and
`holder.asset_value` of `{amount: 12480.55, currency: GEL}`.

## `PENSION-SC-005`

**Verifies:** `PENSION-FR-006`
**Layer:** unit

**Given** a `session` secret, and the fake service answers the participant with status 503.
**When** user A reads `GET /connections/self`.
**Then** the answer is 502 with the type `/problems/dev.maroid.pensions/service-unavailable`.
The `session` secret stays.

## `PENSION-SC-006`

**Verifies:** `PENSION-FR-007`, `PENSION-FR-008`
**Layer:** unit

**Given** stored credentials, no `session` secret, and the fake service accepts the sign in.
**When** user A sends `PUT /connections/self`.
**Then** the answer is 200 with `state: connected`. The `session` secret of user A holds the
token. The fake service records the sign in and the participant read, and no other call.

## `PENSION-SC-007`

**Verifies:** `PENSION-FR-008`, `PENSION-INV-001`
**Layer:** unit

**Given** a `session` secret with the token `old`, and the fake service signs in with `new`.
**When** user A sends `PUT /connections/self`.
**Then** the `session` secret holds `new` alone.

## `PENSION-SC-008`

**Verifies:** `PENSION-FR-008`
**Layer:** unit

**Given** stored credentials, and the fake service rejects the sign in.
**When** user A sends `PUT /connections/self`.
**Then** the answer is 422 with the type `/problems/dev.maroid.pensions/credentials-rejected`.
No `session` secret exists.

## `PENSION-SC-009`

**Verifies:** `PENSION-FR-009`, `PENSION-FR-012`
**Layer:** unit

**Given** a `session` secret, and the store holds contributions of user A.
**When** user A sends `DELETE /connections/self`, then sends it again.
**Then** each answer is 204. No `session` secret exists. The settings and the contributions stay.

## `PENSION-SC-010`

**Verifies:** `PENSION-FR-010`
**Layer:** unit

**Given** a `session` secret, and the fake service answers the first participant read with 401
and accepts a new sign in.
**When** user A reads `GET /connections/self`.
**Then** the answer is 200 `connected`. The `session` secret holds the new token. The fake service
records two participant reads.

## `PENSION-SC-011`

**Verifies:** `PENSION-FR-010`
**Layer:** unit

**Given** a `session` secret, and the fake service answers every participant read with 401 and
accepts each sign in.
**When** user A reads `GET /connections/self`.
**Then** the answer is 502 `service-unavailable`. The fake service records one sign in.

## `PENSION-SC-012`

**Verifies:** `PENSION-FR-011`
**Layer:** unit

**Given** a `session` secret, and the fake service answers the participant read with 401 and
rejects the new sign in.
**When** user A reads `GET /connections/self`.
**Then** the answer is 200 `not_connected`. No `session` secret exists.

## `PENSION-SC-013`

**Verifies:** `PENSION-FR-013`
**Layer:** unit

**Given** the plugin.
**When** the test reads `CronJobs`.
**Then** it holds one job, `account_collector`, with the scope `CronScopePerUser`.

## `PENSION-SC-014`

**Verifies:** `PENSION-FR-014`
**Layer:** unit

**Given** a configuration with no `cron_schedule`.
**When** the plugin decodes it.
**Then** the schedule of `account_collector` is `0 10 1 * *`.

## `PENSION-SC-015`

**Verifies:** `PENSION-FR-015`
**Layer:** unit

**Given** stored credentials and no `session` secret.
**When** `account_collector` runs for user A.
**Then** the run answers nil. The fake service records no call. The store records no write.

## `PENSION-SC-016`

**Verifies:** `PENSION-FR-016`
**Layer:** unit

**Given** the clock reads 2026-11-01 07:00 in Asia/Tbilisi, and a `session` secret.
**When** `account_collector` runs for user A.
**Then** the contributions request sends `startDate=2026-10-01` and `endDate=2026-10-31`.

## `PENSION-SC-017`

**Verifies:** `PENSION-FR-016`
**Layer:** unit

**Given** the fake service answers the October contributions in three pages, of 10, 10, and 3
rows, with `pageCount: 3`.
**When** `Collect` runs for October 2026.
**Then** the snapshot holds 23 contributions. A month with no contribution gives an empty list,
and `Collect` answers no error.

## `PENSION-SC-018`

**Verifies:** `PENSION-FR-017`
**Layer:** unit

**Given** the chart answers two dates of October 2026, with `date` in the form
`2026-10-15T00:00:00`.
**When** `Collect` runs for October 2026.
**Then** the snapshot holds two asset values, on 2026-10-15 and the other date, with the value,
the units, and the unit price of each. An empty `result` gives no asset value and no error.

## `PENSION-SC-019`

**Verifies:** `PENSION-FR-018`
**Layer:** integration

**Given** the table holds 1964 unit prices of portfolio 7.
**When** `Save` writes 1985 unit prices of portfolio 7, the first 1964 the same as before.
**Then** the table holds 1985 rows of portfolio 7, and the statement rewrote 21.

## `PENSION-SC-020`

**Verifies:** `PENSION-FR-019`
**Layer:** integration

**Given** user A holds a summary and the shares of portfolios 7 and 9.
**When** `Save` writes a snapshot with a new summary and the share of portfolio 7 alone.
**Then** user A holds one summary with the new values, and one share, of portfolio 7. The
units of the summary equal the units of that share.

## `PENSION-SC-021`

**Verifies:** `PENSION-FR-020`
**Layer:** unit

**Given** a `session` secret, and the fake service answers page 2 of the contributions with 500.
**When** `Collect` runs for October 2026.
**Then** `Collect` answers an error that wraps `ErrRequestFailed` and the status 500. The store
records no write.

## `PENSION-SC-022`

**Verifies:** `PENSION-FR-020`
**Layer:** integration

**Given** user A holds three contributions of October 2026, and the last statement of `Save`
fails.
**When** `Save` runs with a snapshot of five contributions.
**Then** user A holds the same three contributions, with the same values.

## `PENSION-SC-023`

**Verifies:** `PENSION-FR-021`
**Layer:** integration

**Given** a snapshot holds one basis with the sources of the person, the employer, and the state.
**When** `Save` writes it twice, and the second time the amount of the employer differs.
**Then** user A holds three contributions. The contribution of the employer holds the second amount.

## `PENSION-SC-024`

**Verifies:** `PENSION-FR-022`
**Layer:** integration

**Given** user A holds the asset value 12000 on 2026-10-15.
**When** `Save` writes the asset value 12010 on 2026-10-15.
**Then** user A holds one asset value on 2026-10-15, of 12010.

## `PENSION-SC-025`

**Verifies:** `PENSION-FR-023`
**Layer:** unit

**Given** a `session` secret of user A, and the clock reads 2026-11-27.
**When** the administrator runs `backfill --user A --from 2026-08 --to 2026-10`.
**Then** `Collect` runs for August, September, and October 2026 with user A as the acting user,
and the command prints one line of counts for each month. A second run stores no new contribution.

## `PENSION-SC-026`

**Verifies:** `PENSION-FR-023`, `PENSION-FR-024`
**Layer:** unit

**Given** the clock reads 2026-11-27.
**When** the administrator runs `backfill` with `--from 2026-10 --to 2026-08`, then with
`--from 2026-10 --to 2026-11`.
**Then** the first fails with `ErrReversedPeriod`, and the second with `ErrPeriodNotEnded`. The
fake service records no call.

## `PENSION-SC-027`

**Verifies:** `PENSION-FR-025`
**Layer:** unit

**Given** no `session` secret of user A.
**When** the administrator runs `backfill --user A --from 2026-08 --to 2026-10`.
**Then** the command fails with an error that wraps `ErrNotConnected`. The fake service records no call.

## `PENSION-SC-028`

**Verifies:** `PENSION-FR-026`
**Layer:** unit

**Given** the plugin.
**When** the test reads `Permissions` and `Routes`.
**Then** `account.connect` has the lowest role `viewer`, and each of the three routes names it.

## `PENSION-SC-029`

**Verifies:** `PENSION-INV-002`
**Layer:** integration

**Given** user A holds a contribution, an asset value, a summary, and a share, in workspace W.
**When** user B, a member of W, reads each table of a user.
**Then** user B reads no row. User B reads the employer and the unit prices that the rows of
user A name.

## `PENSION-SC-030`

**Verifies:** `PENSION-INV-003`, `PENSION-INV-004`
**Layer:** unit

**Given** the fake service answers the sign in with a token and the participant with a
`personalId`, and a connect and a collection run with a logger that records every line.
**When** the test reads every answer body and every log line.
**Then** none holds the token, the password, or the personal number.

## `PENSION-SC-031`

**Verifies:** `PENSION-FR-008`, `PENSION-FR-010`, `PENSION-FR-011`
**Layer:** manual

**Given** the owner's pension account, and the hub on the developer machine.
**When** the owner connects with a wrong password, then with the correct one, then replaces
`session` with a token of the day before, then runs `backfill` for the previous month.
**Then** the wrong password answers `credentials-rejected`. The old token answers 401, and the
plugin signs in again. The record of the answers corrects `PENSION-DD-002` and `PENSION-DD-003`.

## Retired identifiers

This file has no retired identifier.
