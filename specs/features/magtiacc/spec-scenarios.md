---
id: MAGTIACC
title: The scenarios of the telecom account of a workspace
type: spec
status: approved
created: 2026-11-24
updated: 2026-11-24
approved_by: Temuri
approved_on: 2026-11-24
constrained_by: [TST, TRC]
requirements: features/magtiacc/requirements.md
---

# Specification: The scenarios of the telecom account of a workspace

A unit scenario runs the plugin against an `httptest` server that plays the portal, a fake
`SecretProvider`, a fake `SettingsProvider`, and fake stores. "The fake portal" is that server.
It answers the sign in with `{"message":"success"}` and the cookie `PHPSESSID=K` unless the
scenario says otherwise. An integration scenario runs the migrations and the repositories
against PostgreSQL. An editor E and a viewer V act in workspace W unless the scenario says
otherwise. The account holds `599000001` (mobile, label empty), `322000001` (home line, label
"House"), and `d_example.ge` (other). "Now" is 2026-11-25 09:00 in Tbilisi.

## `MAGTIACC-SC-001`

**Verifies:** `MAGTIACC-FR-001`, `MAGTIACC-FR-002`
**Layer:** unit

**Given** the plugin.
**When** the test reflects the settings schema of `SettingsModel`.
**Then** the schema holds `username` (free text, workspace scope, required) and `password`
(secret, workspace scope, required), and no other field.

## `MAGTIACC-SC-002`

**Verifies:** `MAGTIACC-FR-003`, `MAGTIACC-FR-004`
**Layer:** unit

**Given** no `session` secret, and no settings.
**When** V reads `GET /connections/self`.
**Then** the answer is 200 with `state: not_connected` and `credentials_stored: false`. The fake portal records no call.

## `MAGTIACC-SC-003`

**Verifies:** `MAGTIACC-FR-005`, `MAGTIACC-FR-006`, `MAGTIACC-DD-004`
**Layer:** unit

**Given** a `session` secret, and the assignment of `322000001`.
**When** V reads `GET /connections/self`.
**Then** the answer lists three subscribers: `599000001` `mobile`, `322000001` `home_line` with
the label "House" and `assigned: true`, and `d_example.ge` `other`. The fake portal records one
selection of each of the first two numbers, and none of `d_example.ge`.

## `MAGTIACC-SC-004`

**Verifies:** `MAGTIACC-FR-007`
**Layer:** unit

**Given** a `session` secret, and a fake portal that answers 503 to `GET /ka/mymagti/`.
**When** V reads `GET /connections/self`.
**Then** the answer is 502 `/problems/dev.maroid.magti/service-unavailable`. The `session` secret stays.

## `MAGTIACC-SC-005`

**Verifies:** `MAGTIACC-FR-008`, `MAGTIACC-FR-009`, `MAGTIACC-DD-001`, `MAGTIACC-DD-002`
**Layer:** unit

**Given** the settings of W.
**When** E sends `PUT /connections/self`.
**Then** the fake portal receives `subscriber`, `password`, and the `csrf-token-login` of the form.
The answer is 200 with `state: connected` and three subscribers. The secret holds `PHPSESSID=K`.

- Limit case: W holds a session and the assignment of `599000001`. The new session replaces the old one, and the assignment stays.
- Unwanted case: the fake portal answers `invalid-credentials`. The answer is 422 `credentials-rejected`, and W holds no `session`.

## `MAGTIACC-SC-006`

**Verifies:** `MAGTIACC-FR-010`, `MAGTIACC-FR-011`
**Layer:** unit

**Given** the settings of W.
**When** E sends `PUT /connections/self`, and the fake portal answers `two-factor-code-required`.
**Then** the answer is 422 `code-required`, and W holds no `session`.

- Unwanted case: the fake portal answers `locked`. The answer is 422 `account-locked`, and W holds no `session`.
- Unwanted case: the fake portal answers `invalid-token`. The answer is 502 `service-unavailable`.

## `MAGTIACC-SC-007`

**Verifies:** `MAGTIACC-FR-012`, `MAGTIACC-FR-013`
**Layer:** unit

**Given** a `session` secret, the assignment of `599000001`, and stored traffic.
**When** E sends `DELETE /connections/self`.
**Then** the answer is 204. W holds no `session` and no assignment. The settings and the traffic stay.

- Limit case: W holds no connection. The answer is 204.

## `MAGTIACC-SC-008`

**Verifies:** `MAGTIACC-FR-014`, `MAGTIACC-DD-003`
**Layer:** unit

**Given** a `session` secret, the assignment of `599000001`, and a fake portal that answers the
body `false` to the first `GET /request/accounts.php`.
**When** V reads `GET /subscriber-states/self`.
**Then** the plugin signs in once, writes the new `session`, repeats the read, and answers 200.

- Unwanted case: the fake portal answers a redirect to the repeat too. The answer is 502 `service-unavailable`.

## `MAGTIACC-SC-009`

**Verifies:** `MAGTIACC-FR-015`, `MAGTIACC-FR-023`
**Layer:** unit

**Given** a `session` secret, the assignment of `599000001`, a fake portal that rejects the
session, and answers `invalid-credentials` to the sign in.
**When** V reads `GET /connections/self`.
**Then** the answer is 200 `not_connected`. The `session` secret is gone. The assignment stays.

- Normal case: the job runs in W instead. `Run` answers an error that wraps `ErrCredentialsRejected`.
- Limit case: the sign in answers `locked`. `Run` answers an error that wraps `ErrAccountLocked`.

## `MAGTIACC-SC-010`

**Verifies:** `MAGTIACC-FR-016`, `MAGTIACC-INV-001`
**Layer:** integration

**Given** a `session` secret.
**When** E sends `PUT /subscriber-assignments/self` with `599000001`, then with `322000001`.
**Then** each answer is 200, marks the new number, and carries an `ETag`. `subscriber_assignments`
holds one row of W: `322000001`, kind `home_line`.

- Unwanted case: E sends the second `PUT` with the `ETag` of the first answer, after V assigned
  `599000001` again in between. The answer is 412 `precondition-failed`, and the assignment of V stays.

## `MAGTIACC-SC-011`

**Verifies:** `MAGTIACC-FR-017`, `MAGTIACC-FR-018`
**Layer:** unit

**Given** a `session` secret, and the assignment of `599000001`.
**When** E sends `PUT /subscriber-assignments/self` with `599999999`.
**Then** the answer is 422 `subscriber-unknown`, and the assignment stays.

- Unwanted case: the number is `d_example.ge`. The answer is 422 `subscriber-unsupported`, and the assignment stays.

## `MAGTIACC-SC-012`

**Verifies:** `MAGTIACC-FR-019`, `MAGTIACC-DD-008`
**Layer:** unit

**Given** a `session` secret.
**When** E assigns `599000001`.
**Then** the fake portal records no request to `/request/statistics.php`, and the stores record no write of traffic or money.

## `MAGTIACC-SC-013`

**Verifies:** `MAGTIACC-FR-020`, `MAGTIACC-FR-021`
**Layer:** unit

**Given** the assignment of `599000001`, and the default configuration.
**When** the job runs at now.
**Then** `Meta().Schedule` is `0 9 * * *`. The fake portal receives five step 3 calls with
`date[from]` and `date[to]` `24/11/2026`: `MB`, `MIN`, and `SMS` with `expense`, and `GEL` with
`expense` and with `income`.

## `MAGTIACC-SC-014`

**Verifies:** `MAGTIACC-FR-022`
**Layer:** unit

**Given** W holds no `session`.
**When** the job runs.
**Then** `Run` answers nil, and the fake portal records no call.

- Limit case: W holds a session and no assignment. `Run` answers nil, with no call.
- Limit case: W assigns `322000001`. `Run` answers nil, with no call.

## `MAGTIACC-SC-015`

**Verifies:** `MAGTIACC-FR-024`
**Layer:** unit

**Given** the assignment of `599000001`, and a fake portal whose list no longer holds it.
**When** the job runs.
**Then** `Run` answers an error that wraps `ErrSubscriberAbsent`. The stores record no write.

## `MAGTIACC-SC-016`

**Verifies:** `MAGTIACC-FR-025`, `MAGTIACC-FR-028`, `MAGTIACC-DD-009`
**Layer:** integration

**Given** the `MB` history `[{day: 1795464000000, chartData: 812.4}, {day: 0, chartData: 3},
{sum: true, chartData: 812.4}]`, and the `MIN` history with `chartData: 12.5`, `chartDataIn: 3`
on the same day.
**When** the job runs, then runs again with `chartData: 815.0` for `MB`.
**Then** `traffic_days` holds one row of 2026-11-24: `data_mb` 815.0, `call_out_seconds` 750,
`call_in_seconds` 180, and null for the SMS columns.

- Limit case: the moment `1795462200000` is 2026-11-23 in Tbilisi.

## `MAGTIACC-SC-017`

**Verifies:** `MAGTIACC-FR-026`, `MAGTIACC-DD-010`
**Layer:** integration

**Given** the `GEL` `income` history with `{transactionDate: "13/11/2026 01:50:48", entryAmount:
"17.30", actionType: "Top up"}`, and the `expense` history with two rows at
`13/11/2026 01:50:49` of `2.00`, one `Plan` and one `SMS`.
**When** the collection of 2026-11-13 runs two times.
**Then** `money_entries` holds three rows: one `in` of 17.30, and two `out` of 2.00. The first run
answers three new entries, the second none. `occurred_at` is `2026-11-12T21:50:48Z`.

## `MAGTIACC-SC-018`

**Verifies:** `MAGTIACC-FR-027`
**Layer:** integration

**Given** W holds traffic and money entries of `599000001`.
**When** E assigns `322000001`, then ends the connection.
**Then** the traffic and the money entries of `599000001` stay.

## `MAGTIACC-SC-019`

**Verifies:** `MAGTIACC-FR-029`, `MAGTIACC-FR-031`, `MAGTIACC-DD-011`
**Layer:** unit

**Given** the assignment of `599000001`, and a fake portal that answers no row before 2026-01-01.
**When** the administrator runs `backfill --workspace W --from 2025-11-15 --to 2026-01-10`.
**Then** the command calls `Collect` three times: 2025-11-15 to 2025-11-30, 2025-12-01 to
2025-12-31, and 2026-01-01 to 2026-01-10. It prints 0, 0, and the counts of January, and succeeds.

## `MAGTIACC-SC-020`

**Verifies:** `MAGTIACC-FR-029`, `MAGTIACC-FR-030`
**Layer:** unit

**Given** the assignment of `599000001`.
**When** the administrator runs `backfill` with `--from 2026-02-01 --to 2026-01-01`.
**Then** `PreRunE` fails, and the fake portal records no call.

- Unwanted case: W assigns `322000001`. The command fails with `ErrHomeLine`.
- Unwanted case: W holds no `session`. The command fails with `ErrNotConnected`.

## `MAGTIACC-SC-021`

**Verifies:** `MAGTIACC-FR-032` to `MAGTIACC-FR-034`, `MAGTIACC-DD-005`, `MAGTIACC-DD-013`
**Layer:** unit

**Given** the assignment of `599000001`. `accounts.php` answers `status` "Active", the balances
`BALANCE` `0.03`, `INTERNET` `2589.29`, `MINUTES` `56:16`, `SMS` `100.0`, and `untilDate`
`13/12/2026 01:50:47`. `cocktail-package.php` answers `price` `17.30`.
**When** V reads `GET /subscriber-states/self` two times.
**Then** each answer holds `balance` 0.03 GEL, `plan.renews_on` 2026-12-13, `plan.days_left` 18,
and `allowances` 2589.29 MB, 3376 seconds, and 100 SMS. The fake portal records two reads of `accounts.php`.

- Normal case: W assigns `322000001`. `accounts-isp.php` answers `package_sum` 40 and the balances
  `27.70` and `70 Mbps`, and `account-data.php` answers `daysRemaining` 20. The answer holds no `allowances`.
- Limit case: `activeCocktail` holds no `cocktailId`. The answer holds no `plan`, and the fake portal records no call of `cocktail-package.php`.

## `MAGTIACC-SC-022`

**Verifies:** `MAGTIACC-FR-035`, `MAGTIACC-FR-036`
**Layer:** unit

**Given** a `session` secret, and no assignment.
**When** V reads `GET /subscriber-states/self`.
**Then** the answer is 409 `not-assigned`.

- Unwanted case: no `session`. The answer is 409 `not-connected`.
- Unwanted case: `accounts.php` answers 503. The answer is 502 `service-unavailable`.

## `MAGTIACC-SC-023`

**Verifies:** `MAGTIACC-FR-037` to `MAGTIACC-FR-039`, `MAGTIACC-DD-014`
**Layer:** unit

**Given** the state of `MAGTIACC-SC-021`.
**When** the widget loads.
**Then** it answers `ready` with `Value` "0.03", `Unit` "₾", `Caption` "2589 MB left, renews in 18 days", and `Link` "/subscriber".

- Normal case: the home line. `Caption` is "40.00 ₾ plan, renews in 20 days".
- Limit case: no assignment. It answers `not_set_up` with the link "/account".
- Unwanted case: the fake portal answers 503. `Load` answers an error.

## `MAGTIACC-SC-024`

**Verifies:** `MAGTIACC-FR-040` to `MAGTIACC-FR-044`, `MAGTIACC-DD-015`
**Layer:** integration

**Given** 30 traffic rows of `599000001` from 2026-10-26 to 2026-11-24, without 2026-11-02.
**When** V reads `GET /traffic-days?from=2026-10-26&to=2026-11-24&limit=20`, then follows `next`.
**Then** the pages hold 20 and 9 rows, the earliest day first. No row holds 2026-11-02.

- Limit case: a row with null SMS columns answers no `sms_sent` and no `sms_received`.
- Limit case: `sort=-day`. The pages hold the latest day first.
- Unwanted case: `from=2025-01-01&to=2026-11-24`. The answer is 400 `request-invalid`.
- Unwanted case: `sort=data_mb`. The answer is 400 `request-invalid`, and `detail` names `data_mb`.

## `MAGTIACC-SC-025`

**Verifies:** `MAGTIACC-FR-045` to `MAGTIACC-FR-047`, `MAGTIACC-DD-016`, `MAGTIACC-DD-017`
**Layer:** integration

**Given** the money entries of `MAGTIACC-SC-017`, and a top up of 2.00 on 2026-11-15 and of 4.90 on 2026-11-05.
**When** V reads `GET /money-entries`, then `GET /monthly-money-totals?from=2025-12`.
**Then** the entries come the latest first. The months answer 12 items. 2026-11 holds
`top_ups` 24.20 GEL and `charges` 4.00 GEL. 2026-10 holds zero for both.

- Limit case: `GET /monthly-money-totals?from=2025-12&sort=month`. The earliest month comes first.
- Unwanted case: `from=2025-01`. The answer is 400 `request-invalid`.

## `MAGTIACC-SC-026`

**Verifies:** `MAGTIACC-FR-048` to `MAGTIACC-FR-051`
**Layer:** unit

**Given** no assignment.
**When** V reads `GET /traffic-days`, `GET /money-entries`, and `GET /monthly-money-totals`.
**Then** each answer is 409 `not-assigned`.

- Normal case: W assigns `322000001`. Each answer is 409 `home-line`.
- Limit case: W assigns `599000001` and holds no row. Each answer is 200 with an empty page, and `monthly-money-totals` with zero totals.
- Normal case: every read records no call of the fake portal.

## `MAGTIACC-SC-027`

**Verifies:** `MAGTIACC-FR-052`, `MAGTIACC-FR-053`
**Layer:** unit

**Given** the permissions of the plugin.
**When** the test reads each route and the widget.
**Then** every `GET` and the widget name `account.read`, lowest `viewer`. Every `PUT` and
`DELETE` names `account.connect`, lowest `editor`.

## `MAGTIACC-SC-028`

**Verifies:** `MAGTIACC-NFR-001`, `MAGTIACC-DD-006`
**Layer:** unit

**Given** `state_timeout` 4s, and a fake portal that holds `accounts.php` for 10 seconds.
**When** V reads `GET /subscriber-states/self`.
**Then** the answer is 502 `service-unavailable` no later than 4.5 seconds after the request.

- Unwanted case: the configuration sets `state_timeout` to 6s. `New` fails.

## `MAGTIACC-SC-029`

**Verifies:** `MAGTIACC-INV-002`
**Layer:** integration

**Given** the assignment, traffic, and money entries of W, and a workspace W2 with none.
**When** the repositories read as W2.
**Then** they answer no row of W.

## `MAGTIACC-SC-030`

**Verifies:** `MAGTIACC-INV-003`, `MAGTIACC-INV-004`
**Layer:** unit

**Given** a fake portal whose answers hold `signContractToken`, a name, and a personal number.
**When** V reads each route, and the widget loads.
**Then** no answer body holds the password, `K`, the CSRF token, `signContractToken`, the name, or the personal number.

## `MAGTIACC-SC-031`

**Verifies:** `MAGTIACC-FR-034`, `MAGTIACC-DD-007`
**Layer:** unit

**Given** the assignment of `599000001`, and a fake portal that holds each selection for 100 milliseconds.
**When** V reads `GET /connections/self` and `GET /subscriber-states/self` at the same time.
**Then** the fake portal records the selection of `599000001` directly before the read of `accounts.php`, with no other selection between them.

## Retired identifiers

This file has no retired identifier.
