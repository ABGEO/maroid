---
id: ELECACC
title: The scenarios of the electricity account of a workspace
type: spec
status: approved
created: 2026-10-14
updated: 2026-10-14
approved_by: Temuri
approved_on: 2026-10-14
constrained_by: [TST, TRC]
requirements: features/elecacc/requirements.md
---

# Specification: The scenarios of the electricity account of a workspace

A unit scenario runs the plugin against an `httptest` server that plays the portal, a fake
`SecretProvider`, a fake `SettingsProvider`, a fake `Notifier`, and fake stores. "The fake
portal" is that server. It answers the sign in with cookie K and CSRF token T unless the
scenario says otherwise. An integration scenario runs the migrations and the repositories
against PostgreSQL. An editor E and a viewer V act in workspace W unless the scenario says
otherwise. The account holds customers `C1` (`8370764`) and `C2` (`8370999`).

## `ELECACC-SC-001`

**Verifies:** `ELECACC-FR-001`, `ELECACC-FR-002`
**Layer:** unit

**Given** the plugin.
**When** the test reflects the settings schema of `SettingsModel`.
**Then** the schema holds `username` (free text, workspace scope, required) and `password`
(secret, workspace scope, required), and no other field.

## `ELECACC-SC-002`

**Verifies:** `ELECACC-FR-003`, `ELECACC-FR-004`
**Layer:** unit

**Given** no `session` secret, and no settings.
**When** V reads `GET /connections/self`.
**Then** the answer is 200 with `state: not_connected` and `credentials_stored: false`. The fake portal records no call.

## `ELECACC-SC-003`

**Verifies:** `ELECACC-FR-005`, `ELECACC-FR-006`, `ELECACC-DD-004`
**Layer:** unit

**Given** a `session` secret, `C2` assigned, and the fake portal lists `C1` and `C2`, each detail with an address, an `owner`, and an `idNumber`.
**When** V reads `GET /connections/self`.
**Then** the answer is 200 with `state: connected` and two customers. `C1` carries
`customer_number: "8370764"`, its label, and its address. Only `C2` carries `assigned: true`.
No member holds the `owner` or the `idNumber`.

## `ELECACC-SC-004`

**Verifies:** `ELECACC-FR-007`
**Layer:** unit

**Given** a `session` secret, and the fake portal answers the list with 503.
**When** V reads `GET /connections/self`.
**Then** the answer is 502 `service-unavailable`. The `session` secret stays.

## `ELECACC-SC-005`

**Verifies:** `ELECACC-FR-008`, `ELECACC-FR-009`, `ELECACC-DD-002`
**Layer:** unit

**Given** the settings, an old `session` secret, and `C1` assigned.
**When** E sends `PUT /connections/self`.
**Then** the fake portal records `GET /`, `POST /login` with the CSRF token of `GET /`, and
`GET /private` with the cookie of the redirect. The answer is 200 with `state: connected`. The
`session` secret holds K and T. `C1` stays assigned. No request followed a redirect.

## `ELECACC-SC-006`

**Verifies:** `ELECACC-FR-009`, `ELECACC-DD-002`
**Layer:** unit

**Given** the settings, and no `session` secret.
**When** E sends `PUT /connections/self` three times: the fake portal answers the sign in with
a 302 to `/?tabId=password&error=error.bad_credentials`, then with a 302 to `/`, then with a
first page that holds no `_csrf`.
**Then** the first answer is 422 `credentials-rejected`, the second and the third 502
`service-unavailable`. No `session` secret exists after any. A connect whose new session the
portal rejects, and whose second sign in it rejects, answers 422 `credentials-rejected` too.

## `ELECACC-SC-007`

**Verifies:** `ELECACC-FR-010`, `ELECACC-FR-011`
**Layer:** unit

**Given** a `session` secret, `C1` assigned, and stored transactions and bills.
**When** E sends `DELETE /connections/self`.
**Then** the answer is 204. No `session` secret and no assignment remain. The transactions and the bills stay.

## `ELECACC-SC-008`

**Verifies:** `ELECACC-FR-012`, `ELECACC-DD-003`
**Layer:** unit

**Given** a `session` secret with cookie K0, and the fake portal answers 401 to K0 and 200 to K.
**When** V reads `GET /connections/self`, then the job runs.
**Then** each signs in once. The answer is 200 with `state: connected`. The `session` secret
holds K. A fake portal that answers 401 to K too makes the read answer 502, after one sign in.

## `ELECACC-SC-009`

**Verifies:** `ELECACC-FR-013`, `ELECACC-FR-020`
**Layer:** unit

**Given** a `session` secret that the fake portal rejects, `C1` assigned, and the fake portal rejects the credentials.
**When** V reads `GET /connections/self`, and in a second run the job runs.
**Then** the read answers 200 with `state: not_connected`. The job answers an error that
wraps `ErrCredentialsRejected`. No `session` secret remains. `C1` stays assigned.

## `ELECACC-SC-010`

**Verifies:** `ELECACC-FR-014`, `ELECACC-INV-001`
**Layer:** integration

**Given** `C1` assigned to W.
**When** E assigns `C2`.
**Then** `customer_assignments` holds one row for W, with `8370999`.

## `ELECACC-SC-011`

**Verifies:** `ELECACC-FR-015`
**Layer:** unit

**Given** a `session` secret, and the fake portal lists `C1` and `C2`.
**When** E sends `PUT /customer-assignments/self` with `1111111`.
**Then** the answer is 422 `customer-unknown`. No assignment is stored.

## `ELECACC-SC-013`

**Verifies:** `ELECACC-FR-017`, `ELECACC-FR-018`, `ELECACC-DD-005`
**Layer:** unit

**Given** the default configuration, `C1` assigned, a `session` secret, and the run at
2026-11-01 00:30 in Tbilisi, which is 2026-10-31 20:30 UTC.
**When** the job runs.
**Then** the schedule of the job is `0 10 1 * *`. The fake portal records the requests with the month `2026-10`.

## `ELECACC-SC-014`

**Verifies:** `ELECACC-FR-019`
**Layer:** unit

**Given** two workspaces: one with no `session` secret, one with a session and no assignment.
**When** the job runs in each.
**Then** each run answers nil, and logs a skip. The fake portal records no call.

## `ELECACC-SC-015`

**Verifies:** `ELECACC-FR-021`
**Layer:** unit

**Given** `C1` assigned, and the fake portal lists `C2` alone.
**When** the job runs.
**Then** the run answers an error that wraps `ErrCustomerAbsent`. The store holds nothing new. The fake notifier records nothing.

## `ELECACC-SC-016`

**Verifies:** `ELECACC-FR-022`, `ELECACC-DD-007`
**Layer:** integration

**Given** the two charges of 2026-04-28 at the meter values 1163 and 1317, and the payment of 2026-04-01.
**When** the repository inserts the three rows, then the same three, then the charge of 1317 with the amount 36.90.
**Then** the first insert answers true three times, the second false three times, and the
third false. The table holds three rows. The charge of 1317 holds 36.90.

## `ELECACC-SC-017`

**Verifies:** `ELECACC-FR-023`
**Layer:** integration

**Given** the bill `8370764-2026-9`.
**When** the repository inserts it twice.
**Then** `bills` holds one row, with `month` 2026-09-01.

## `ELECACC-SC-018`

**Verifies:** `ELECACC-FR-024`
**Layer:** integration

**Given** `C1` assigned, with stored transactions and bills.
**When** E assigns `C2`, then sends `DELETE /connections/self`.
**Then** the store still holds the transactions and the bills of `C1`.

## `ELECACC-SC-019`

**Verifies:** `ELECACC-FR-025`, `ELECACC-FR-026`, `ELECACC-DD-009`
**Layer:** unit

**Given** `C1` assigned and a `session` secret.
**When** the administrator runs `backfill --workspace W --from 2025-10 --to 2026-09`.
**Then** the fake portal records 12 months, from `2025-10` to `2026-9`. The command prints the count of each month. The fake notifier records nothing.

## `ELECACC-SC-020`

**Verifies:** `ELECACC-FR-025`, `ELECACC-FR-027`
**Layer:** unit

**Given** a workspace with no `session` secret, then one with no assignment.
**When** the administrator runs `backfill` in each, and then runs it with `--from 2026-09 --to 2026-08`.
**Then** the first two fail with `ErrNotConnected` and `ErrNotAssigned`. The third fails in `PreRunE`, and the fake portal records no call.

## `ELECACC-SC-021`

**Verifies:** `ELECACC-FR-028`, `ELECACC-FR-029`, `ELECACC-FR-030`, `ELECACC-DD-008`, `ELECACC-DD-010`
**Layer:** unit

**Given** `C1` assigned, the stored bill `8370764-2026-4`, and the fake portal answers the
charges of 2026-04-28 (1163, 13.2 kWh, 2.52) and (1317, 153.3 kWh, 36.87) for the first time.
**When** the job runs.
**Then** `NotificationTopics` declares `monthly_bill`. The fake notifier records one message on
it, titled `თელმიკო | ქვითარი`, that shows 2026-04-28, 1317 kWh, 166.5 kWh, and 39.39 ₾, with the
attachment `8370764-2026-4.pdf` of type `application/pdf`.

## `ELECACC-SC-022`

**Verifies:** `ELECACC-FR-031`, `ELECACC-DD-010`
**Layer:** unit

**Given** the reading of 2026-09-28, and the fake portal answers the document with 500, then a month that holds no stored bill.
**When** the bill of the reading goes out.
**Then** the send answers nil each time. The fake notifier records one message with no attachment each time. The log holds a warning.

## `ELECACC-SC-023`

**Verifies:** `ELECACC-FR-032`, `ELECACC-FR-035`, `ELECACC-FR-036`, `ELECACC-DD-011`
**Layer:** unit

**Given** `C1` assigned, and the stored charges of 2026-04-28 of `ELECACC-SC-021`.
**When** the widget loads.
**Then** it answers `ready` with the value `166.5`, the unit `kWh`, the caption `39.39 ₾, meter 1317 kWh, 28/04/2026`, and the link `/transactions`. The fake portal records no call.

## `ELECACC-SC-024`

**Verifies:** `ELECACC-FR-033`, `ELECACC-FR-034`
**Layer:** unit

**Given** no assignment, then an assignment with a payment and no charge.
**When** the widget loads.
**Then** the first answer is `not set up` with the link `/account`. The second is `empty`.

## `ELECACC-SC-025`

**Verifies:** `ELECACC-FR-037`, `ELECACC-FR-038`, `ELECACC-FR-045`, `ELECACC-DD-012`, `ELECACC-DD-016`
**Layer:** integration

**Given** 45 stored transactions of `C1`: charges, payments, and the first reading of a new meter.
**When** V reads `GET /transactions`, then follows `next`.
**Then** the first page holds the 20 latest, the latest date first, and a `next` link. A charge
carries `amount` `{"amount": 39.11, "currency": "GEL"}`, `balance`, `usage`, `meter_value`, and
`tariff`. A payment carries `amount` and `balance` alone. The first reading carries
`meter_value` alone. The fake portal records no call.

## `ELECACC-SC-026`

**Verifies:** `ELECACC-FR-039`, `ELECACC-DD-013`
**Layer:** manual

**Given** the deck, and 14 months of transactions of `C1`, with the split charges of 2026-04-28.
**When** a member opens `/transactions` in the light theme and in the dark theme.
**Then** the chart shows 12 columns and one line of 12 points. April shows one column of 166.5 kWh. The reading of 13 months ago is in the list and not in the chart.

## `ELECACC-SC-027`

**Verifies:** `ELECACC-FR-040`, `ELECACC-FR-041`, `ELECACC-FR-045`, `ELECACC-DD-014`
**Layer:** integration

**Given** 24 stored bills of `C1`.
**When** V reads `GET /bills`.
**Then** the page holds 20 bills, `2026-09` first, each with `month`, `bill_number`, `issued_on`,
and an absolute `document` that ends with `/bills/<id>/document`, and a `next` link. The fake portal records no call.

## `ELECACC-SC-028`

**Verifies:** `ELECACC-FR-042`, `ELECACC-DD-015`
**Layer:** unit

**Given** the stored bill `8370764-2026-9`, and the fake portal answers its document with a PDF of 70 KiB.
**When** V reads `GET /bills/<id>/document`, then reads it again while the fake portal answers 500.
**Then** the first answer is 200 with `Content-Type: application/pdf`,
`Content-Disposition: attachment; filename="8370764-2026-9.pdf"`, and the bytes of the PDF.
The second is 502 `service-unavailable`.

## `ELECACC-SC-029`

**Verifies:** `ELECACC-FR-042`, `ELECACC-INV-005`
**Layer:** integration

**Given** a bill B of `C1` in workspace W, a bill X in workspace Y, and `C2` assigned to W after B was stored.
**When** V reads the document of X in W, then the document of B in W.
**Then** both answers are 404. The fake portal records no call.

## `ELECACC-SC-030`

**Verifies:** `ELECACC-FR-043`, `ELECACC-FR-044`
**Layer:** unit

**Given** no assignment, then an assignment with no transaction and no bill.
**When** V reads `GET /transactions` and `GET /bills`.
**Then** the first answers are 409 `not-assigned`. The second are 200 with an empty `items`.

## `ELECACC-SC-031`

**Verifies:** `ELECACC-FR-046`, `ELECACC-FR-047`
**Layer:** unit

**Given** the plugin.
**When** the test reads `Permissions`, `Routes`, and `Widgets`.
**Then** `account.read` has the lowest role `viewer`, and names the four `GET` routes and the
widget. `account.connect` has `editor`, and names the three writes.

## `ELECACC-SC-032`

**Verifies:** `ELECACC-INV-002`
**Layer:** integration

**Given** an assignment, transactions, and bills in workspace W, and none in workspace X.
**When** a read runs in X.
**Then** it sees no assignment, no transaction, and no bill.

## `ELECACC-SC-033`

**Verifies:** `ELECACC-INV-003`, `ELECACC-INV-004`
**Layer:** unit

**Given** the settings with the username `01001010101`, and the fake portal answers the
detail of `C1` with the `owner` `Holder Name` and the `idNumber` `62909012413`.
**When** E connects, V reads `GET /connections/self`, and the job runs.
**Then** no body and no log line holds K, T, the password, `01001010101`, `Holder Name`, or `62909012413`.

## `ELECACC-SC-034`

**Verifies:** `ELECACC-FR-048`, `ELECACC-DD-006`
**Layer:** unit

**Given** a `session` secret, and the fake portal lists `C1` and `C2`.
**When** E assigns `C1`, then assigns `C2`.
**Then** each answer is 200 and marks the customer that E assigned. The fake portal records the
list of the customers and their details, and no request of the transactions or the bills. The
store holds no transaction and no bill. The fake notifier records nothing.

## Retired identifiers

| ID               | Retired    | Reason                                                                   |
| ---------------- | ---------- | ------------------------------------------------------------------------ |
| `ELECACC-SC-012` | 2026-10-14 | `ELECACC-FR-016` is retired. `ELECACC-SC-034` verifies `ELECACC-FR-048`. |
