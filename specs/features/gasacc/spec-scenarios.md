---
id: GASACC
title: The scenarios of the gas account of a workspace
type: spec
status: approved
created: 2026-10-13
updated: 2026-10-14
approved_by: Temuri
approved_on: 2026-10-13
constrained_by: [TST, TRC]
requirements: features/gasacc/requirements.md
---

# Specification: The scenarios of the gas account of a workspace

A unit scenario runs the plugin against an `httptest` server that plays the gas service, a
fake `SecretProvider`, a fake `SettingsProvider`, a fake `Notifier`, and fake stores. "The fake
service" is that server. An integration scenario runs the migrations and the repositories
against PostgreSQL. An editor E and a viewer V act in workspace W unless the scenario says
otherwise. The account holds customers `C1` (`757235-992`) and `C2` (`340309-649`).

## `GASACC-SC-001`

**Verifies:** `GASACC-FR-001`, `GASACC-FR-002`, `GASACC-FR-003`
**Layer:** unit

**Given** the plugin.
**When** the test reflects the settings schema of `SettingsModel`.
**Then** the schema holds `username` (free text, workspace scope, required) and `password`
(secret, workspace scope, required), and no other field.

## `GASACC-SC-002`

**Verifies:** `GASACC-FR-004`, `GASACC-FR-005`
**Layer:** unit

**Given** no `token` secret, and no settings.
**When** V reads `GET /connections/self`.
**Then** the answer is 200 with `state: not_connected` and `credentials_stored: false`. The fake service records no call.

## `GASACC-SC-003`

**Verifies:** `GASACC-FR-006`, `GASACC-FR-007`, `GASACC-DD-001`
**Layer:** unit

**Given** a `token` secret, `C2` assigned, and the fake service lists `C1` and `C2` with a name, an address, and a personal number.
**When** V reads `GET /connections/self`.
**Then** the answer is 200 with `state: connected` and two customers. `C1` carries
`customer_number: "757235-992"`, its name, and its address. Only `C2` carries `assigned: true`.

## `GASACC-SC-004`

**Verifies:** `GASACC-FR-008`
**Layer:** unit

**Given** a `token` secret, and the fake service answers the list with 503.
**When** V reads `GET /connections/self`.
**Then** the answer is 502 `service-unavailable`. The `token` secret stays.

## `GASACC-SC-005`

**Verifies:** `GASACC-FR-009`, `GASACC-FR-010`
**Layer:** unit

**Given** the settings, a `token` secret, and `C1` assigned.
**When** E sends `PUT /connections/self`, and the fake service accepts the sign in with token T2.
**Then** the answer is 200 with `state: connected`. The `token` secret holds T2. `C1` stays assigned.

## `GASACC-SC-006`

**Verifies:** `GASACC-FR-010`
**Layer:** unit

**Given** the settings, and no `token` secret.
**When** E sends `PUT /connections/self` twice: the fake service answers a `code` other than 0, then `recaptchaRequested: true`.
**Then** the first answer is 422 `credentials-rejected`, the second 422 `sign-in-blocked`. No `token` secret exists after either.

## `GASACC-SC-007`

**Verifies:** `GASACC-FR-011`, `GASACC-FR-012`
**Layer:** unit

**Given** a `token` secret, `C1` assigned, and stored transactions.
**When** E sends `DELETE /connections/self`.
**Then** the answer is 204. No `token` secret and no assignment remain. The transactions stay.

## `GASACC-SC-008`

**Verifies:** `GASACC-FR-013`
**Layer:** unit

**Given** a `token` secret, and the fake service rejects the token on the list.
**When** V reads `GET /connections/self`.
**Then** the answer is 200 with `state: not_connected`. The `token` secret is gone. The assignment stays.

## `GASACC-SC-009`

**Verifies:** `GASACC-FR-014`, `GASACC-INV-001`
**Layer:** integration

**Given** `C1` assigned to W.
**When** E assigns `C2`.
**Then** `customer_assignments` holds one row for W, with `340309-649`.

## `GASACC-SC-010`

**Verifies:** `GASACC-FR-015`
**Layer:** unit

**Given** a `token` secret, and the fake service lists `C1` and `C2`.
**When** E sends `PUT /customer-assignments/self` with `999999-111`.
**Then** the answer is 422 `customer-unknown`. No assignment is stored.

## `GASACC-SC-012`

**Verifies:** `GASACC-FR-017`, `GASACC-DD-006`
**Layer:** unit

**Given** `C1` assigned, a `token` secret, and the run at 2026-11-01 00:30 in Tbilisi, which is 2026-10-31 20:30 UTC.
**When** the job runs.
**Then** the fake service records the transactions request from `2026-10-01` to `2026-10-31`.

## `GASACC-SC-013`

**Verifies:** `GASACC-FR-018`
**Layer:** unit

**Given** two workspaces: one with no `token` secret, one with a token and no assignment.
**When** the job runs in each.
**Then** each run answers nil, and the fake service records no call.

## `GASACC-SC-014`

**Verifies:** `GASACC-FR-019`
**Layer:** unit

**Given** `C1` assigned, an expired `token` secret, and the settings.
**When** the job runs, and the fake service rejects the old token, then accepts a sign in with T2.
**Then** the run stores the transactions. The `token` secret holds T2.

## `GASACC-SC-015`

**Verifies:** `GASACC-FR-019`
**Layer:** unit

**Given** `C1` assigned, an expired `token` secret, and a wrong password.
**When** the job runs.
**Then** the run answers an error that wraps `ErrCredentialsRejected`. Nothing is stored.

## `GASACC-SC-016`

**Verifies:** `GASACC-FR-020`
**Layer:** unit

**Given** `C1` assigned, and the fake service lists `C2` only.
**When** the job runs.
**Then** the run answers an error that wraps `ErrCustomerAbsent`. The fake service records no transactions request. Nothing is stored.

## `GASACC-SC-017`

**Verifies:** `GASACC-FR-021`, `GASACC-DD-008`
**Layer:** integration

**Given** the reading of `C1` on 2026-09-28, amount 11.33, and the payment of `C1` on the same date.
**When** the store inserts both twice, then the reading with amount 11.50.
**Then** `transactions` holds two rows. The reading holds 11.50. `Insert` answered true for the
first insert of each row, and false for every other insert.

## `GASACC-SC-018`

**Verifies:** `GASACC-FR-022`
**Layer:** integration

**Given** transactions of `C1`.
**When** the workspace assigns `C2`, then deletes the assignment.
**Then** every transaction of `C1` stays.

## `GASACC-SC-019`

**Verifies:** `GASACC-FR-023`, `GASACC-FR-024`, `GASACC-DD-009`
**Layer:** unit

**Given** W with `C1` assigned and a `token` secret, and the fake service answers one reading and one payment for each month.
**When** the test runs `backfill --workspace W --from 2025-10-15 --to 2026-01-10`.
**Then** the fake service records four transactions requests: `2025-10-15` to `2025-10-31`,
November, December, and `2026-01-01` to `2026-01-10`. Each request ran in the acting workspace
W with no acting user. The store holds eight transactions. The fake notifier records no call.

## `GASACC-SC-020`

**Verifies:** `GASACC-FR-023`
**Layer:** unit

**Given** the command.
**When** the test runs `backfill` with `--from 2026-02-01 --to 2026-01-01`, then with `--from 2026-13-01`, then with `--workspace abc`.
**Then** each run returns an error from `PreRunE`. The fake service records no call.

## `GASACC-SC-021`

**Verifies:** `GASACC-FR-025`
**Layer:** unit

**Given** W with no `token` secret, then W with a token and no assignment.
**When** the test runs `backfill` in each.
**Then** the first run returns an error that wraps `ErrNotConnected`, the second one that wraps `ErrNotAssigned`.

## `GASACC-SC-022`

**Verifies:** `GASACC-FR-026`, `GASACC-FR-028`
**Layer:** unit

**Given** the plugin, and a new reading of 2026-09-28: meter value 1532, usage 21, amount 11.33.
**When** the test reads `NotificationTopics`, then the job sends the bill.
**Then** the topics hold `monthly_bill` with the name "Monthly bill". The body of the message
holds `2026.09.28`, `1532 მ³`, `21 მ³`, and `11.33 ₾`.

## `GASACC-SC-023`

**Verifies:** `GASACC-FR-027`, `GASACC-DD-007`
**Layer:** unit

**Given** `C1` assigned, and the fake service answers two new readings, 2026-09-28 and 2026-08-27, each with a bill and a meter photo, a payment, and a control reading of 2026-09-30 with a meter value and no charge.
**When** the job runs.
**Then** the fake notifier records one message on `monthly_bill`, with the figures of 2026-09-28, a PDF, and a JPEG.
A second run with the same answer records no message.

## `GASACC-SC-024`

**Verifies:** `GASACC-FR-027`, `GASACC-DD-010`
**Layer:** unit

**Given** a new reading, and the fake service answers the meter photo with 500.
**When** the job runs.
**Then** the fake notifier records one message with the PDF only. The run answers nil.

## `GASACC-SC-025`

**Verifies:** `GASACC-FR-029`, `GASACC-FR-032`, `GASACC-FR-033`, `GASACC-DD-011`
**Layer:** unit

**Given** a store whose latest reading is 2026-09-28: consumption 21, meter value 1532, amount 11.33.
**When** the widget loads.
**Then** the answer is a stat with value `21`, unit `m³`, caption `11.33 ₾, meter 1532 m³, 28/09/2026`, and link `/transactions`. The fake service records no call.

## `GASACC-SC-026`

**Verifies:** `GASACC-FR-030`, `GASACC-FR-031`
**Layer:** unit

**Given** a store with no assignment, then a store with an assignment and a payment only.
**When** the widget loads in each.
**Then** the first answer is "not set up" with the link `/account`. The second is "empty".

## `GASACC-SC-027`

**Verifies:** `GASACC-FR-034`, `GASACC-FR-035`, `GASACC-FR-040`, `GASACC-DD-012`
**Layer:** integration

**Given** `C1` assigned, 45 transactions of `C1`, and 3 of `C2`.
**When** V reads `GET /transactions`, then follows `next`.
**Then** the first page holds the 20 latest of `C1`, a reading and a payment of one date
ordered by `id` descending. A reading carries `usage` and `meter_value`. A payment carries
neither. The second page holds the next 20. No transaction of `C2` appears.

## `GASACC-SC-028`

**Verifies:** `GASACC-FR-036`, `GASACC-DD-013`
**Layer:** unit

**Given** a reading with `bill_path` `api/Customer/GetCustomerBillPdf/x.pdf` and no meter photo, and a payment.
**When** V reads `GET /transactions`.
**Then** the reading carries `bill_url: "https://my.te.ge/api/Customer/GetCustomerBillPdf/x.pdf"` and no `meter_photo_url`. The payment carries neither.

## `GASACC-SC-029`

**Verifies:** `GASACC-FR-037`, `GASACC-DD-014`
**Layer:** manual

**Given** the deck, and 14 months of transactions of `C1`.
**When** a member opens `/transactions` in the light theme and in the dark theme.
**Then** the chart shows 12 columns and one line of 12 points. The reading of 13 months ago is in the list and not in the chart.

## `GASACC-SC-030`

**Verifies:** `GASACC-FR-038`, `GASACC-FR-039`
**Layer:** unit

**Given** no assignment, then an assignment with no transaction.
**When** V reads `GET /transactions`.
**Then** the first answer is 409 `not-assigned`. The second is 200 with an empty `items`.

## `GASACC-SC-031`

**Verifies:** `GASACC-FR-041`, `GASACC-FR-042`
**Layer:** unit

**Given** the plugin.
**When** the test reads `Permissions`, `Routes`, and `Widgets`.
**Then** `account.read` has the lowest role `viewer`, and names both `GET` routes and the widget.
`account.connect` has `editor`, and names the three writes.

## `GASACC-SC-032`

**Verifies:** `GASACC-INV-002`
**Layer:** integration

**Given** an assignment and transactions in workspace W, and none in workspace X.
**When** a read runs in X.
**Then** it sees no assignment and no transaction.

## `GASACC-SC-033`

**Verifies:** `GASACC-INV-003`, `GASACC-INV-004`
**Layer:** unit

**Given** the fake service answers the sign in with token T, and the customers with `personalN` `62909012413`.
**When** E connects, and V reads `GET /connections/self`.
**Then** no body and no log line holds T, the password, or `62909012413`.

## `GASACC-SC-034`

**Verifies:** `GASACC-FR-043`, `GASACC-DD-007`
**Layer:** unit

**Given** a `token` secret, and the fake service lists `C1` and `C2` and answers transactions.
**When** E assigns `C1`, then assigns `C2`.
**Then** each answer is 200 and marks the customer that E assigned. The fake service records the
list of the customers, and no request of the transactions. The store holds no transaction. The
fake notifier records nothing.

## Retired identifiers

| ID              | Retired    | Reason                                                                 |
| --------------- | ---------- | ---------------------------------------------------------------------- |
| `GASACC-SC-011` | 2026-10-14 | `GASACC-FR-016` is retired. `GASACC-SC-034` verifies `GASACC-FR-043`. |
