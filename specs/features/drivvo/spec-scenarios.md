---
id: DRIVVO
title: The scenarios of the car log account of a workspace
type: spec
status: approved
created: 2026-11-25
updated: 2026-11-25
approved_by: Temuri
approved_on: 2026-11-25
constrained_by: [TST, TRC]
requirements: features/drivvo/requirements.md
---

# Specification: The scenarios of the car log account of a workspace

A unit scenario runs the plugin against an `httptest` server that plays the car log service, a
fake `PluginSecrets`, a fake `PluginSettings`, and fake stores. "The fake service" is that server.
It answers the sign in with the token `T` and `expira_em_segundos: 86400` unless the scenario says
otherwise. An integration scenario runs the migrations and the repositories against PostgreSQL. An
editor E and a viewer V act in workspace W, with the time zone `Asia/Tbilisi`, unless the scenario
says otherwise. The account holds the vehicle "Hatchback" (`id_veiculo` 11, principal, active, km)
and "Old sedan" (12, inactive, km). The settings are litres, format 4 (L/100 km), and `GEL`. "Now" is
2026-11-25 10:00 in Tbilisi.

## The connection

## `DRIVVO-SC-001`

**Verifies:** `DRIVVO-FR-001`, `DRIVVO-FR-002`
**Layer:** unit

**Given** the plugin.
**When** the test reflects the settings schema of `SettingsModel`.
**Then** the schema holds `email` (format email, workspace scope, required) and `password` (secret,
workspace scope, required), and no other field.

## `DRIVVO-SC-002`

**Verifies:** `DRIVVO-FR-003`, `DRIVVO-FR-004`
**Layer:** unit

**Given** no `session` secret, and no settings.
**When** V reads `GET /connections/self`.
**Then** the answer is 200 with `state: not_connected` and `credentials_stored: false`. The fake
service records no call.

## `DRIVVO-SC-003`

**Verifies:** `DRIVVO-FR-005`, `DRIVVO-DD-004`
**Layer:** unit

**Given** a `session` secret.
**When** V reads `GET /connections/self`.
**Then** the answer lists "Hatchback" active and "Old sedan" inactive, with make and model. The
fake service records one `GET /veiculo/web` with `X-Token: T`.

## `DRIVVO-SC-004`

**Verifies:** `DRIVVO-FR-006`
**Layer:** unit

**Given** a `session` secret, and a fake service that answers 503 to `GET /veiculo/web`.
**When** V reads `GET /connections/self`.
**Then** the answer is 502 `service-unavailable`, and the `session` secret stays.

## `DRIVVO-SC-005`

**Verifies:** `DRIVVO-FR-007`, `DRIVVO-FR-008`, `DRIVVO-INV-001`, `DRIVVO-DD-001`
**Layer:** unit

**Given** the settings `driver@example.com` and `secret` with outer spaces.
**When** E calls `PUT /connections/self` two times.
**Then** each answer is 200 with two vehicles. The fake service records each sign in with
`senha` = `5ebe2294ecd0e0f08eab7690d2a6ee69`, the MD5 of `secret`, and `App-Platform: Web`. W holds one
`session` secret that ends at now plus 86400 seconds. The fake service records no read of an entry.

## `DRIVVO-SC-006`

**Verifies:** `DRIVVO-FR-008`
**Layer:** unit

**Given** a fake service that answers the sign in with 401, and in a second run with 200 and
`{"result": false, "exception": {"code": 1908}}`.
**When** E calls `PUT /connections/self` in each run.
**Then** each answer is 422 `credentials-rejected`, and W holds no `session` secret.

## `DRIVVO-SC-007`

**Verifies:** `DRIVVO-FR-009`
**Layer:** unit

**Given** a fake service that answers the sign in with a token and `alterar_senha: true`.
**When** E calls `PUT /connections/self`.
**Then** the answer is 422 `new-password-required`, and W holds no `session` secret.

## `DRIVVO-SC-008`

**Verifies:** `DRIVVO-FR-011`, `DRIVVO-FR-014`
**Layer:** integration

**Given** a `session` secret, and a copy of two vehicles and 40 entries.
**When** E calls `DELETE /connections/self`, then V reads `GET /vehicles/{Hatchback}/log-entries`.
**Then** the first answer is 204, and W holds no `session` secret. The second answer lists the same
entries as before.

## `DRIVVO-SC-009`

**Verifies:** `DRIVVO-FR-012`, `DRIVVO-DD-003`
**Layer:** unit

**Given** a `session` secret with the token `OLD`, and a fake service that answers 401 to `OLD`.
**When** the job runs for W.
**Then** the fake service records one sign in and the reads with `T`. W holds `session` with `T`.
A fake service that answers 401 to `T` too makes the run fail with `ErrRequestFailed` after one sign in.

## `DRIVVO-SC-010`

**Verifies:** `DRIVVO-FR-012`
**Layer:** unit

**Given** a `session` secret whose `expires_at` is one minute before now.
**When** V reads `GET /connections/self`.
**Then** the fake service records the sign in before `GET /veiculo/web`, and no 401.

## `DRIVVO-SC-011`

**Verifies:** `DRIVVO-FR-013`
**Layer:** unit

**Given** a `session` secret, a fake service that answers 401 to every data call, and a sign in that
answers 1908.
**When** V reads `GET /connections/self`.
**Then** the answer is 200 `not_connected`, and W holds no `session` secret.

## The collection

## `DRIVVO-SC-012`

**Verifies:** `DRIVVO-FR-015`, `DRIVVO-FR-017`, `DRIVVO-DD-005`
**Layer:** unit

**Given** W with a `session` secret, and workspace X with none.
**When** the job runs for W, then for X.
**Then** the store of W receives one `Replace` with the snapshot of the account. The run of X logs a
skip at info, answers nil, and the fake service records no call for X.

## `DRIVVO-SC-013`

**Verifies:** `DRIVVO-FR-016`
**Layer:** unit

**Given** a configuration with no `cron_schedule`.
**When** the plugin builds its jobs.
**Then** `car_log_collector` declares `0 * * * *` and `CronScopePerWorkspace`.

## `DRIVVO-SC-014`

**Verifies:** `DRIVVO-FR-018`
**Layer:** unit

**Given** a fake service whose "Old sedan" holds 3 refuels and 1 reminder.
**When** the job runs for W.
**Then** the snapshot holds both vehicles, the 3 refuels of "Old sedan", and its reminder. The fake
service records the six reads of each vehicle.

## `DRIVVO-SC-015`

**Verifies:** `DRIVVO-FR-019`, `DRIVVO-DD-006`
**Layer:** integration

**Given** a copy with the refuels 101 and 102, and the maintenance 201 of 120.00.
**When** `Replace` stores a snapshot with the refuels 101 and 103, and 201 of 150.00.
**Then** the copy holds 101 with its `id` of before, 103, and 201 with 150.00 and its `id` of before.
It holds no 102. A second snapshot from another account with no vehicle 11 removes "Hatchback" and
each of its entries.

## `DRIVVO-SC-016`

**Verifies:** `DRIVVO-FR-020`
**Layer:** unit

**Given** a fake service that answers 500 to `GET /veiculo/12/despesa/completo`.
**When** the job runs for W.
**Then** the run fails with `ErrRequestFailed`, and the store receives no call of `Replace`.

## `DRIVVO-SC-017`

**Verifies:** `DRIVVO-FR-021`
**Layer:** unit

**Given** the failure of `DRIVVO-SC-016`.
**When** the scheduler logs the error of the run.
**Then** the log line names `workspace_id` of W and the path `/veiculo/12/despesa/completo`, and holds
no token, no email address, and no password.

## `DRIVVO-SC-018`

**Verifies:** `DRIVVO-FR-022`, `DRIVVO-DD-007`
**Layer:** unit

**Given** a refuel with `data` `2026-11-24 18:30:00`, and an instalment with `data_vencimento`
`2026-12-01 00:00:00`.
**When** the reader decodes them.
**Then** the refuel occurs at 2026-11-24T14:30:00Z, and the instalment is due on 2026-12-01.

## `DRIVVO-SC-019`

**Verifies:** `DRIVVO-FR-023`, `DRIVVO-FR-037`, `DRIVVO-DD-008`
**Layer:** unit

**Given** a refuel with fuel 1 (30 L, 90.00, full), fuel 2 (5 L, 15.00, `sem_custo`), no fuel 3,
`bateria_inicial_pct` 20, `bateria_final_pct` 80, and `duracao_minutos` 45.
**When** the reader decodes it.
**Then** the entry holds two lines at positions 1 and 2, the amount 90.00, and the three values of the
charge.

## `DRIVVO-SC-020`

**Verifies:** `DRIVVO-FR-024`
**Layer:** unit

**Given** an expense with `parcelas` of numbers 1, 2, and 3, of 200.00 each.
**When** the reader decodes it.
**Then** the entry holds three instalments with their days and amounts.

## `DRIVVO-SC-021`

**Verifies:** `DRIVVO-FR-025`, `DRIVVO-FR-026`
**Layer:** unit

**Given** the plugin.
**When** the test reads `Permissions`, `Routes`, and `Widgets`.
**Then** every `GET` route and the widget name `account.read` (viewer). The `PUT` and the `DELETE`
of `/connections/self` name `account.connect` (editor).

## `DRIVVO-SC-022`

**Verifies:** `DRIVVO-NFR-001`
**Layer:** integration

**Given** a fake service with 3 vehicles and 2,000 entries, each answer delayed 200 ms.
**When** the job runs for W.
**Then** the run ends in less than 60 seconds, from its start to the commit of `Replace`.

## `DRIVVO-SC-023`

**Verifies:** `DRIVVO-NFR-002`
**Layer:** integration

**Given** a copy of one vehicle with 2,000 entries, a third of them with instalments.
**When** V reads the overview with all time, the history, the reminders, and the widget, ten times
each.
**Then** each answer arrives in less than 1 second, from the receipt of the request to the answer.

## `DRIVVO-SC-024`

**Verifies:** `DRIVVO-INV-002`
**Layer:** integration

**Given** W and X, each with a copy of one vehicle with `source_id` 11.
**When** a transaction of X reads `vehicles` and `log_entries`, and `Replace` runs for X.
**Then** X reads its own rows alone, and the rows of W stay.

## `DRIVVO-SC-025`

**Verifies:** `DRIVVO-INV-003`
**Layer:** integration

**Given** a copy with the refuel 101.
**When** a repository inserts a second refuel 101 for W.
**Then** the insert fails on `UNIQUE (workspace_id, kind, source_id)`.

## `DRIVVO-SC-026`

**Verifies:** `DRIVVO-INV-004`
**Layer:** integration

**Given** a copy with "Old sedan", 3 of its entries with lines, and its reminder.
**When** a transaction deletes "Old sedan".
**Then** no entry, line, instalment, or reminder of "Old sedan" remains.

## `DRIVVO-SC-027`

**Verifies:** `DRIVVO-INV-005`, `DRIVVO-INV-006`, `DRIVVO-DD-002`
**Layer:** unit

**Given** a sign in answer that holds `nome`, `email`, `telefone`, `cnh`, and the token.
**When** E connects, and V reads every route.
**Then** no answer holds the token, the password, its MD5, or any of the four values.

## `DRIVVO-SC-028`

**Verifies:** `DRIVVO-FR-008`, `DRIVVO-FR-012`, `DRIVVO-FR-022`, `DRIVVO-NFR-001`
**Layer:** manual

**Given** the account of the owner at the car log service.
**When** the owner connects, waits for an expired token, and runs the job.
**Then** the sign in, the sign in again, and the collection succeed. A refuel shows at the time that
the mobile application shows. The run ends in less than 60 seconds.

## Retired identifiers

This file has no retired identifier.
