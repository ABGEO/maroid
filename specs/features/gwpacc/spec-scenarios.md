---
id: GWPACC
title: The scenarios of the water account of a workspace
type: spec
status: approved
created: 2026-10-12
updated: 2026-10-12
approved_by: Temuri
approved_on: 2026-10-12
constrained_by: [TST, TRC]
requirements: features/gwpacc/requirements.md
---

# Specification: The scenarios of the water account of a workspace

A unit scenario runs the plugin against an `httptest` server that plays the water service,
a fake `SecretProvider`, a fake `SettingsProvider`, and fake repositories. "The fake
service" is that server. An integration scenario runs the migrations and the repositories
against PostgreSQL. An editor E and a viewer V act in workspace W unless the scenario says
otherwise. The account holds customers `C1` and `C2`.

## `GWPACC-SC-001`

**Verifies:** `GWPACC-FR-001`, `GWPACC-FR-002`
**Layer:** unit

**Given** the plugin.
**When** the test reflects the settings schema of `SettingsModel`.
**Then** the schema holds `username` (free text, workspace scope, required) and `password`
(secret, workspace scope, required), and no other field.

## `GWPACC-SC-002`

**Verifies:** `GWPACC-FR-003`, `GWPACC-FR-004`
**Layer:** unit

**Given** no `token` secret, and no settings.
**When** V reads `GET /connections/self`.
**Then** the answer is 200 with `state: not_connected` and `credentials_stored: false`. The fake service records no call.

## `GWPACC-SC-003`

**Verifies:** `GWPACC-FR-005`, `GWPACC-FR-006`
**Layer:** unit

**Given** a `token` secret, `C2` assigned, and the fake service lists `C1` with a communal
number and `C2` with none, each with its address.
**When** V reads `GET /connections/self`.
**Then** the answer is 200 with `state: connected` and two customers, each with its
number and address. `C1` carries its communal number, and `C2` carries none. Only `C2`
carries `assigned: true`.

## `GWPACC-SC-004`

**Verifies:** `GWPACC-FR-007`
**Layer:** unit

**Given** a `token` secret, and the fake service answers the list with 503.
**When** V reads `GET /connections/self`.
**Then** the answer is 502 `/problems/dev.maroid.gwp/service-unavailable`. The `token` secret stays.

## `GWPACC-SC-005`

**Verifies:** `GWPACC-FR-008`, `GWPACC-FR-009`
**Layer:** unit

**Given** stored settings, and the fake service accepts them with the token `T1`.
**When** E sends `PUT /connections/self`.
**Then** the answer is 200 with `state: connected` and the customers. The `token` secret holds `T1`.

## `GWPACC-SC-006`

**Verifies:** `GWPACC-FR-009`
**Layer:** unit

**Given** (a) a `token` secret `T1` and `C1` assigned, and the fake service gives `T2`;
(b) the fake service answers the sign in with `success: false`; (c) no settings.
**When** E sends `PUT /connections/self`.
**Then** (a) 200, the secret holds `T2`, and `C1` stays assigned; (b) 422
`credentials-rejected`, and no secret exists; (c) 422 `credentials-absent`.

## `GWPACC-SC-007`

**Verifies:** `GWPACC-FR-010`, `GWPACC-FR-011`
**Layer:** unit

**Given** a `token` secret and `C1` assigned.
**When** E sends `DELETE /connections/self`.
**Then** the answer is 204. No `token` secret and no assignment exist. A second `DELETE` answers 204 too.

## `GWPACC-SC-008`

**Verifies:** `GWPACC-FR-012`
**Layer:** unit

**Given** a `token` secret, `C1` assigned, and the fake service answers the list with 401.
**When** V reads `GET /connections/self`.
**Then** the answer is 200 with `state: not_connected`. The `token` secret is gone. `C1` stays assigned.

## `GWPACC-SC-009`

**Verifies:** `GWPACC-FR-013`, `GWPACC-FR-015`, `GWPACC-INV-001`
**Layer:** unit

**Given** a `token` secret and `C1` assigned.
**When** E sends `PUT /customer-assignments/self` with `C2`.
**Then** the answer is 200, and `C2` is the one assignment of W. The fake service records one call of `/RecordDisplay`.

## `GWPACC-SC-010`

**Verifies:** `GWPACC-FR-014`
**Layer:** unit

**Given** a `token` secret.
**When** E sends `PUT /customer-assignments/self` with (a) `C9`, which the account does not hold;
(b) an empty `customer_number`; (c) a body of 5 KiB.
**Then** (a) 422 `customer-unknown`; (b) 422 `/problems/http/validation-failed` with the
pointer `#/customer_number`; (c) 413 `/problems/http/content-too-large`. No assignment
exists in any case.

## `GWPACC-SC-011`

**Verifies:** `GWPACC-FR-015`
**Layer:** unit

**Given** a `token` secret, and the fake service answers `/RecordDisplay` with 503.
**When** E sends `PUT /customer-assignments/self` with `C1`.
**Then** the answer is 200, and `C1` is assigned. The log holds one warn line.

## `GWPACC-SC-012`

**Verifies:** `GWPACC-FR-016`
**Layer:** unit

**Given** a `token` secret, `C1` assigned, and the fake service reports one meter of `C1` and one of `C2`.
**When** the job runs in W.
**Then** the reading repository receives one insert, for the meter of `C1`.

## `GWPACC-SC-013`

**Verifies:** `GWPACC-FR-017`
**Layer:** unit

**Given** (a) no `token` secret; (b) a `token` secret and no assignment.
**When** the job runs in W.
**Then** the job answers nil, logs a skip, and the fake service records no call.

## `GWPACC-SC-014`

**Verifies:** `GWPACC-FR-018`
**Layer:** unit

**Given** a `token` secret `T1`, `C1` assigned, and stored settings. The fake service
answers `/RecordDisplay` with 401 for `T1`, gives `T2` at the sign in, and answers for `T2`.
**When** the job runs in W.
**Then** the secret holds `T2`, and the readings of `C1` are inserted.

## `GWPACC-SC-015`

**Verifies:** `GWPACC-FR-018`, `GWPACC-FR-012`, `GWPACC-FR-019`
**Layer:** manual

**Given** a workspace connected to a real water account, the token of an earlier sign in
that the water service no longer accepts, and a wrong password in a second workspace.
**When** the owner reads the page, assigns a customer, and runs the job in both workspaces.
**Then** each answer lands in the condition of section 4.5, the date of each reading
parses, and the customer number of a reading equals the one of the list. Build step 7 records the outcome.

## `GWPACC-SC-016`

**Verifies:** `GWPACC-FR-019`
**Layer:** unit

**Given** a `token` secret, `C1` assigned, and the fake service reports readings of `C2` only.
**When** the job runs in W.
**Then** the job answers an error that matches `errs.ErrCustomerAbsent`. The reading repository receives no insert.

## `GWPACC-SC-017`

**Verifies:** `GWPACC-FR-020`
**Layer:** integration

**Given** workspace W.
**When** the repository inserts the same reading twice, then a reading of the same meter on a
later date, then the first date again with a corrected value.
**Then** `readings` holds two rows for W, and the first date holds the corrected value.

## `GWPACC-SC-018`

**Verifies:** `GWPACC-FR-021`
**Layer:** unit

**Given** a `token` secret, `C1` assigned, and two readings of `C1`.
**When** E assigns `C2`, then sends `DELETE /connections/self`.
**Then** the reading repository receives no delete.

## `GWPACC-SC-019`

**Verifies:** `GWPACC-FR-022`, `GWPACC-FR-025`
**Layer:** integration

**Given** `C1` assigned, meter `A` with 1204.0 on 15/08/2026 and 1216.5 on 15/09/2026,
meter `B` with 80.0 on 15/08/2026 and 78.5 on 15/09/2026, meter `C` with 50.0 on
15/09/2026 alone, and a reading of `C2`.
**When** V loads the widget `usage`.
**Then** the answer is a list of three lines: "Meter A: 12.5 m³" with "1216.5 m³ on
15/09/2026", "Meter B: -1.5 m³" with "78.5 m³ on 15/09/2026", and "Meter C" with
"50.0 m³ on 15/09/2026", and `More` is `/readings`. No call reaches the water service.

## `GWPACC-SC-020`

**Verifies:** `GWPACC-FR-023`, `GWPACC-FR-024`
**Layer:** integration

**Given** (a) no assignment; (b) `C1` assigned and no reading of `C1`.
**When** V loads the widget `usage`.
**Then** (a) `not_set_up` with the link `/account`; (b) `empty`.

## `GWPACC-SC-021`

**Verifies:** `GWPACC-FR-026`, `GWPACC-FR-027`
**Layer:** unit

**Given** the plugin.
**When** the test reads `Permissions`, `Routes`, and the meta of the widget.
**Then** `account.read` has the lowest role `viewer` and guards `GET /connections/self` and
the widget. `account.connect` has `editor` and guards the other three routes.

## `GWPACC-SC-022`

**Verifies:** `GWPACC-INV-002`
**Layer:** integration

**Given** workspaces W and X, each with an assignment and readings.
**When** a transaction that acts in X reads both tables.
**Then** it sees only the rows of X.

## `GWPACC-SC-023`

**Verifies:** `GWPACC-INV-003`
**Layer:** unit

**Given** a `token` secret `T1`, and the settings with the password `P1`.
**When** E calls each route, and the fake service fails each call in turn.
**Then** no answer body and no log line holds `T1` or `P1`.

## `GWPACC-SC-024`

**Verifies:** `GWPACC-FR-013`
**Layer:** unit

**Given** no `token` secret.
**When** E sends `PUT /customer-assignments/self` with `C1`.
**Then** the answer is 409 `/problems/dev.maroid.gwp/not-connected`. No assignment exists,
and the fake service records no call.

## `GWPACC-SC-025`

**Verifies:** `GWPACC-FR-028`, `GWPACC-FR-029`, `GWPACC-FR-033`
**Layer:** integration

**Given** `C1` assigned, meters `A` and `B`, and 24 monthly readings of each, 48 in all.
**When** V reads `GET /readings`, then follows `next` twice.
**Then** the pages hold 20, 20, and 8 readings, the latest date first, and the two meters
of one date in the order of their identifiers. The first page carries `next` and no
`prev`, and the last carries `prev` and no `next`. Each item but the earliest of each
meter carries its usage.

## `GWPACC-SC-026`

**Verifies:** `GWPACC-FR-028`
**Layer:** integration

**Given** readings of `C1` and of `C2`, and `C2` assigned.
**When** V reads `GET /readings`.
**Then** every item is a reading of `C2`.

## `GWPACC-SC-027`

**Verifies:** `GWPACC-FR-030`
**Layer:** integration

**Given** `C1` assigned, meter `A` with 100.0 on 15/09/2025, 110.0 on 15/10/2025, and 200.0 on 15/09/2026.
**When** V reads `GET /readings?from=2025-10-12`, then `GET /readings?from=12-10-2025`.
**Then** the first answer holds 200.0 with a usage of 90.0, and 110.0 with a usage of 10.0,
which counts from the reading before `from`. The second answers 400
`/problems/http/request-invalid`, and its detail names `from`.

## `GWPACC-SC-028`

**Verifies:** `GWPACC-FR-031`
**Layer:** integration

**Given** no assigned customer.
**When** V reads `GET /readings`.
**Then** the answer is 409 `/problems/dev.maroid.gwp/not-assigned`.

## `GWPACC-SC-029`

**Verifies:** `GWPACC-FR-032`
**Layer:** integration

**Given** `C1` assigned, and no reading of `C1`.
**When** V reads `GET /readings`.
**Then** the answer is 200 with an empty `items`, and neither `next` nor `prev`.

## `GWPACC-SC-030`

**Verifies:** `GWPACC-FR-030`, `GWPACC-FR-031`, `GWPACC-FR-032`
**Layer:** manual

**Given** the deck, and a workspace with 24 months of readings of two meters.
**When** the owner opens the readings page, switches the theme, pages through the list,
then removes the assignment and opens the page again.
**Then** the chart shows a column and a line for each meter over the last 12 months, and
takes the colors of each theme. The list pages by 20. With no assignment, the page asks
for the setup with a link to the account page.

## `GWPACC-SC-031`

**Verifies:** `GWPACC-FR-026`
**Layer:** unit

**Given** the plugin.
**When** the test reads the routes of the readings handler.
**Then** `GET /readings` names `account.read`.

## Retired identifiers

This file has no retired identifier.
