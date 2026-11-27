---
id: PARKHIST
title: The pages of the parking history
type: spec
status: approved
created: 2026-11-27
updated: 2026-11-27
approved_by: Temuri
approved_on: 2026-11-27
constrained_by: [UI, TS, RES, API, SEC, OWN, PLG, TST]
requirements: features/parkhist/requirements-pages.md
---

# Specification: The pages of the parking history

`spec.md` holds the collection, the data model, and the build plan. This file holds the read
routes, the widget link, the three pages, their design decisions, and their scenarios. Every
route reads the database alone (`PARKHIST-FR-044`). Row level security limits each read to the
acting user (`PARKHIST-INV-001`). Only the overview reads `/connections/self` of `PARKACC`, for the
figures of `PARKHIST-FR-022`.

## 1. Components

Every path is under `plugins/parking/`.

| Path                                                | Action | Holds                                                                                                                                                            |
| --------------------------------------------------- | ------ | ---------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `model/read.go`                                     | create | `SessionFilter` (`From`, `To`, `Plate`, `Free *bool`), `TransactionFilter` (`From`, `To`, `Kind`), `SessionTotal`, `TransactionTotal`, `ChargeMonth`, `LotUsage` |
| `repository/session.go`                             | change | `Page`, `Total`, `Plates`, `LotUsages`                                                                                                                           |
| `repository/transaction.go`                         | change | `Page`, `Total`, `ChargeMonths`                                                                                                                                  |
| `repository/keyset.go`                              | create | The keyset condition and the order of a list, as the pensions plugin writes it                                                                                   |
| `service/history_read.go`                           | create | The read methods of `HistoryStore`, each in one `WithTx`                                                                                                         |
| `dto/{session,transaction,month,lot}.go`            | create | The answers of section 2. `MoneyResponse` of `dto/connection.go` stays the money object                                                                          |
| `handler/{session,transaction,query}.go`            | create | `SessionHandler`, `TransactionHandler`, the parsers of `from`, `to`, `plate`, `free`, and `kind`                                                                 |
| `widget/balance.go`                                 | change | `Link` of a connected answer is `/overview` (`PARKHIST-DD-010`)                                                                                                  |
| `permission/permission.go`                          | change | `HistoryRead = "history.read"`                                                                                                                                   |
| `main.go`                                           | change | The new routes, `history.read`, and the UI routes of section 2                                                                                                   |
| `ui/src/entry.ts`                                   | change | The page routes of section 2                                                                                                                                     |
| `ui/src/pages/{overview,sessions,transactions}/`    | create | `Overview.svelte`, `Sessions.svelte`, `Transactions.svelte`                                                                                                      |
| `ui/src/components/`                                | create | `PeriodPicker.svelte`, `NoData.svelte`, `NoAccess.svelte`, `ChargeChart.svelte`, `BalanceCard.svelte`, `LotList.svelte`, `FilterBar.svelte`                      |
| `ui/src/lib/{address,chart,format,pages,period}.ts` | create | The copies of the pensions plugin (`UI-006` forbids an import), with the filters and the kinds of this feature                                                   |
| `ui/src/api/{session,transaction}.ts`               | create | One function for each route of section 2                                                                                                                         |

Each handler depends on an interface that it declares, and `HistoryStore` meets it.

## 2. Declarations

**HTTP routes.** The prefix of `PARKACC` applies. `api.yaml` holds the bodies. Each route names `history.read`.

| Method | Path                  | Access        | Takes                                                                   | Realizes                                                   |
| ------ | --------------------- | ------------- | ----------------------------------------------------------------------- | ---------------------------------------------------------- |
| `GET`  | `/sessions`           | Authenticated | `from`, `to`, `plate`, `free`, `limit` (1 to 100, default 20), `cursor` | `PARKHIST-FR-029` to `PARKHIST-FR-034`, `PARKHIST-NFR-001` |
| `GET`  | `/sessions/total`     | Authenticated | `from`, `to`, `plate`, `free`                                           | `PARKHIST-FR-027`, `PARKHIST-FR-035`, `PARKHIST-NFR-001`   |
| `GET`  | `/plates`             | Authenticated | Nothing                                                                 | `PARKHIST-FR-033`                                          |
| `GET`  | `/lot-usages`         | Authenticated | `from`                                                                  | `PARKHIST-FR-028`                                          |
| `GET`  | `/transactions`       | Authenticated | `from`, `to`, `kind`, `limit` (1 to 100, default 20), `cursor`          | `PARKHIST-FR-036` to `PARKHIST-FR-041`, `PARKHIST-NFR-001` |
| `GET`  | `/transactions/total` | Authenticated | `from`, `to`, `kind`                                                    | `PARKHIST-FR-042`, `PARKHIST-NFR-001`                      |
| `GET`  | `/charge-months`      | Authenticated | `from`                                                                  | `PARKHIST-FR-025`, `PARKHIST-FR-026`                       |

`GET /connections/self` of `PARKACC` gives the balance and the free parkings left today
(`PARKHIST-FR-022`). It does not change. `/sessions` and `/transactions` answer pages (`RES-005`). `/plates`, `/lot-usages`, and
`/charge-months` are bounded collections, and take no `limit` and no `cursor`. `from` and `to` are
dates `YYYY-MM-DD`, each one included, in the time zone of the acting user. An absent `from` reads
from the first row. `free` is `true` or `false`. `kind` is `top_up`, `parking_payment`,
`permit_purchase`, or `other`.

**Permission.**

| Name           | Description                                  | Lowest   | Realizes          |
| -------------- | -------------------------------------------- | -------- | ----------------- |
| `history.read` | Read your own past sessions and transactions | `viewer` | `PARKHIST-FR-045` |

**Widget.** The widget `balance` of `PARKACC` keeps its key, kind, size, `sessions.read`, and its
read of the connection. A connected answer links to `/overview` (`PARKHIST-FR-021`).

**UI routes.** The first route is the landing page of the plugin.

| Page route      | Navigation entry | Page                  | Realizes                                                  |
| --------------- | ---------------- | --------------------- | --------------------------------------------------------- |
| `/overview`     | "Overview"       | `Overview.svelte`     | `PARKHIST-FR-022` to `PARKHIST-FR-028`, `PARKHIST-FR-043` |
| `/sessions`     | "Sessions"       | `Sessions.svelte`     | `PARKHIST-FR-029` to `PARKHIST-FR-035`, `PARKHIST-FR-043` |
| `/transactions` | "Transactions"   | `Transactions.svelte` | `PARKHIST-FR-036` to `PARKHIST-FR-043`                    |
| `/account`      | "Account"        | `Account.svelte`      | `PARKACC`, no change                                      |

**Indexes.** The migrations of `spec.md` add
`sessions_user_started_at_idx (user_id, started_at DESC, id DESC)` and
`transactions_user_occurred_at_idx (user_id, occurred_at DESC, id DESC)` (`PARKHIST-DD-012`).

**Errors.** A `from` or a `to` that is not a date, a `from` after a `to`, a `free` or a `kind`
outside its values, or a parameter that a route does not declare answers 400
`/problems/http/request-invalid`, with a detail that names the parameter. A `plate` of no stored
session answers an empty page and a total of zero.

## 3. Design decisions

### `PARKHIST-DD-010`

**Realizes:** `PARKHIST-FR-021`
**Decision:** `widget/balance.go` sets `Link: "/overview"` on the stat answer. The answer
`NotSetUpAnswer` keeps `/account`.
**Rationale:** A connected member wants the history. A member with no connection must connect
first. The widget reads the connection as `PARKACC` gives, and this feature does not change it.
**Alternatives:** A second widget. The home page then shows the balance twice.

### `PARKHIST-DD-011`

**Realizes:** `PARKHIST-FR-029` to `PARKHIST-FR-031`, `PARKHIST-FR-036`, `PARKHIST-FR-037`, `PARKHIST-FR-039`
**Decision:** The order of `/sessions` is `started_at DESC, id DESC`, and of
`/transactions` `occurred_at DESC, id DESC`, fixed. The cursor of `page.Read` keys on both, and
carries the filters, so a change of a filter retires it. A session answers `id`,
`lot_number`, `address`, `plate`, `started_at`, `ended_at`, `duration_seconds`, `amount`, and
`free`. A transaction answers `id`, `occurred_at`, `kind`, `description`, `amount`,
`balance_after`, and `session` (`lot_number`, `address`) when the join finds one.
**Rationale:** Two rows can share a time, so the time alone does not order a page. The page
formats each time in the time zone of the deck, and a duration as hours and minutes.
**Alternatives:** A sort that the client names. No requirement asks for one.

### `PARKHIST-DD-012`

**Realizes:** `PARKHIST-FR-027`, `PARKHIST-FR-035`, `PARKHIST-FR-042`, `PARKHIST-NFR-001`
**Decision:** `/sessions/total` answers `count`, `duration_seconds` (the sum of
`ended_at - started_at`), and `amount`, with the filter of the list, in one statement.
`/transactions/total` answers `count`, `top_ups` (the sum of `top_up`), and `charges` (the sum
of `parking_payment` and `permit_purchase`). The overview sends `/sessions/total` with
the `from` of the period. The two indexes of section 2 serve each list and each total.
**Rationale:** A user holds about 150 rows of each list a year. A read with any filter scans one
range of one index of the acting user.
**Alternatives:** The sums as members of the page. `Page` of `libs/rest/page` holds the items and
the links alone, as `PENSION-DD-015` found.

### `PARKHIST-DD-013`

**Realizes:** `PARKHIST-FR-025`, `PARKHIST-FR-026`
**Decision:** `/charge-months` answers, for each month from `from` that holds a transaction,
`month` as `YYYY-MM`, `charges` (the sum of the charges), and `balance` (`balance_after` of the
latest transaction of the month). The month is `date_trunc('month', occurred_at AT TIME ZONE
:zone)`, with the name of `pluginapi.ActingTimeZone`. `ChargeChart.svelte` draws
`ColumnLineChart` of `@maroid/ui` with one slot for each month of the period, keyed
`YYYY-MM-01`: `charges` as columns on the left axis, `balance` as the line on the right axis.
`period.ts` gives `from`: the first day of the month 11 months back, 35 months back, or none.
The default is 12 months.
**Rationale:** The database sums each month in one statement. One slot for each month keeps a
month with no transaction on the axis, through the `dates` property of the chart.
**Alternatives:** The client sums the pages of `/transactions`. All months then cost 4 requests of 100.

### `PARKHIST-DD-014`

**Realizes:** `PARKHIST-FR-028`
**Decision:** `/lot-usages` groups the sessions from `from` by `lot_number`, and answers
at most 5: `lot_number`, the `address` of the latest session of the lot, `count`, and
`amount`. The order is `count DESC, max(started_at) DESC`.
**Rationale:** The capture writes one lot with two spellings of its address. The latest spelling
is the one the parking service gives now.
**Alternatives:** A group by `lot_number` and `address`. One lot then fills two places of the 5.

### `PARKHIST-DD-015`

**Realizes:** `PARKHIST-FR-032`, `PARKHIST-FR-041`
**Decision:** The handler turns `from` into 00:00 of that day in `pluginapi.ActingTimeZone`, and
`to` into 00:00 of the next day, and the repository reads `>= from AND < to`. The sessions page
and the transactions page keep their filters in the query of the address, as `PENSION-DD-020` gives.
**Rationale:** The columns hold points in time (`DAT-010`). A day of a person begins at their own
midnight. A reload keeps the filter, and a link names one plate.
**Alternatives:** The days of Tbilisi for every user. It breaks the filter of a user in another zone.

### `PARKHIST-DD-016`

**Realizes:** `PARKHIST-FR-022` to `PARKHIST-FR-024`, `PARKHIST-FR-043`, `PARKHIST-FR-044`
**Decision:** `BalanceCard.svelte` reads `GET /connections/self` and shows `account.balance` and
`account.free_parkings_left`. A 502 shows "The parking service is unavailable." in the card.
`state: not_connected` shows "Not connected" with a link to `/account`. A role without
`account.connect` shows no card. A page reads its list with no filter first, and an empty answer
shows `NoData.svelte`: "No data yet. Connect the parking account on the Account page. The
collection runs each hour." with a link to `/account`. The overview shows `NoData.svelte` in place
of its stored parts when both lists are empty, and keeps the card. A role without `history.read`
sends no read of the stored data, and the page shows `NoAccess.svelte`.
**Rationale:** The connection route already reads both figures, and removes a rejected token
(`PARKACC-FR-019`). The card fails alone, so the stored parts stay. The first read of a list tells
a page whether Maroid holds data, so no route of state exists.
**Alternatives:** A route that reports the connection from the plugin secret. The owner decided on
2026-11-27 that the history pages show what Maroid holds, and nothing about the connection.

## 4. Scenarios

A unit scenario of a handler runs it against a fake. An integration scenario runs the
repositories against the database of `libs/testdb`. User A acts in workspace W, in the time zone
Asia/Tbilisi.

### `PARKHIST-SC-020`

**Verifies:** `PARKHIST-FR-021`
**Layer:** unit

**Given** a connection with a balance of 9.08, and no connection for user B.
**When** the widget loads for A, then for B.
**Then** A answers `ready` with the link `/overview`. B answers `not_set_up` with the link `/account`.

### `PARKHIST-SC-021`

**Verifies:** `PARKHIST-FR-022`, `PARKHIST-FR-025`, `PARKHIST-FR-026`
**Layer:** manual

**Given** a connected account with a balance of 9.08 GEL and 4 free parkings left, and transactions in January and March 2026, on the hub of the developer machine in November 2026.
**When** the owner opens the overview with the period of 12 months, then of every month.
**Then** the card shows 9.08 GEL and 4. The axis holds 12 slots from 2025-12-01 to 2026-11-01. February holds no column and no point. The second period starts at January 2026.

### `PARKHIST-SC-022`

**Verifies:** `PARKHIST-FR-025`
**Layer:** integration

**Given** user A holds a top up of 10.00 on 2026-01-05, a parking payment of 1.07 on 2026-01-20 with `balance_after` 8.93, a parking payment of 0.55 at `2026-01-31T21:30:00Z`, and user B holds a charge in January.
**When** user A reads `/charge-months?from=2026-01-01`.
**Then** the answer holds 2026-01 with `charges` 1.07 and `balance` 8.93, and 2026-02 with `charges` 0.55, because 21:30 UTC on 31 January is 1 February in Tbilisi. No charge of user B counts.

### `PARKHIST-SC-023`

**Verifies:** `PARKHIST-FR-028`
**Layer:** integration

**Given** user A holds sessions on 7 lots: A1637 four times, A894 and A203 twice each, with the latest of A203 after the latest of A894, and four lots once each.
**When** user A reads `/lot-usages`.
**Then** the answer holds 5 lots: A1637, A203, A894, then two of the four, each with its count and the sum of its amounts.

### `PARKHIST-SC-024`

**Verifies:** `PARKHIST-FR-029`, `PARKHIST-FR-030`, `PARKHIST-FR-031`
**Layer:** integration

**Given** user A holds 45 sessions, two of them with the same `started_at`, one with an amount of 0.
**When** user A reads `/sessions`, then follows `next` twice, then `prev` once.
**Then** the pages hold 20, 20, and 5 rows, the latest start first, each row once. Each row holds every member of `PARKHIST-DD-011`. The row with an amount of 0 holds `free: true`. `prev` answers the second page again.

### `PARKHIST-SC-025`

**Verifies:** `PARKHIST-FR-027`, `PARKHIST-FR-032`, `PARKHIST-FR-033`, `PARKHIST-FR-034`, `PARKHIST-FR-035`
**Layer:** integration

**Given** user A holds 30 sessions of the plate AB071EO in November 2026, 8 of them free, and 15 of another plate.
**When** user A reads `/sessions` and `/sessions/total` with `plate=AB071EO`, `from=2026-11-01`, `to=2026-11-30`, and `free=false`.
**Then** both answers hold the 22 paid sessions of that plate in November alone. The total counts 22, not the first 20, with the sum of their durations and their amounts.

### `PARKHIST-SC-026`

**Verifies:** `PARKHIST-FR-032`
**Layer:** unit

**Given** the route of the sessions.
**When** a request names `from=2026-11-30&to=2026-11-01`, then `from=2026-13-01`, then `free=yes`.
**Then** each answers 400 `/problems/http/request-invalid`, with a detail that names the parameter.

### `PARKHIST-SC-027`

**Verifies:** `PARKHIST-FR-033`
**Layer:** integration

**Given** user A holds sessions of two plates, and user B of a third.
**When** user A reads `/plates`, then `/sessions?plate=<the plate of B>`.
**Then** the first answer holds the two plates of user A. The second answers an empty page.

### `PARKHIST-SC-028`

**Verifies:** `PARKHIST-FR-016`, `PARKHIST-FR-036`, `PARKHIST-FR-037`, `PARKHIST-FR-038`, `PARKHIST-FR-039`
**Layer:** integration

**Given** user A holds 25 transactions: a top up, a parking payment that names a stored session on A1637, a parking payment that names an absent one, and a transaction of `other`.
**When** user A reads `/transactions`, then follows `next`.
**Then** the pages hold 20 and 5 rows, the latest moment first. The first payment holds `session` with A1637. The second holds no `session`. Each row holds its `kind`, and the amount with no sign.

### `PARKHIST-SC-029`

**Verifies:** `PARKHIST-FR-040`, `PARKHIST-FR-041`, `PARKHIST-FR-042`
**Layer:** integration

**Given** user A holds in October 2026 three top ups of 10.00, twelve parking payments of 0.50, and one permit purchase of 25.00.
**When** user A reads `/transactions/total?from=2026-10-01&to=2026-10-31`, then `/transactions?kind=permit_purchase`.
**Then** the total holds `count` 16, `top_ups` 30.00, and `charges` 31.00. The list holds the one permit purchase.

### `PARKHIST-SC-030`

**Verifies:** `PARKHIST-FR-043`
**Layer:** manual

**Given** user B, connected 10 minutes ago, and user C, never connected, both with no row, and user D with transactions and no session.
**When** each one opens the three pages.
**Then** B and C see "No data yet" with a link to the Account page on each page. D sees the transactions page and the chart of the overview, and "No data yet" on the sessions page. The network panel holds no request of `/connections/self` from the sessions page or the transactions page.

### `PARKHIST-SC-031`

**Verifies:** `PARKHIST-FR-044`
**Layer:** integration

**Given** a fake parking service that fails every call.
**When** each route of section 2 answers.
**Then** each answer succeeds, and the fake records no call.

### `PARKHIST-SC-032`

**Verifies:** `PARKHIST-FR-045`
**Layer:** unit

**Given** the plugin.
**When** the test reads `Permissions` and `Routes`.
**Then** `history.read` has the lowest role `viewer`, and each route of section 2 names it.

### `PARKHIST-SC-033`

**Verifies:** `PARKHIST-NFR-001`
**Layer:** integration

**Given** user A holds 5,000 sessions of three plates and 5,000 transactions over 10 years, and user B holds the same.
**When** the test reads each list and each total 20 times with each filter: none, the dates, `plate` or `kind`, `free`, and all of them.
**Then** the 95th percentile of each route stays below 300 milliseconds, from the request to the last byte, on the database of the developer machine.

### `PARKHIST-SC-034`

**Verifies:** `PARKHIST-FR-023`, `PARKHIST-FR-024`
**Layer:** manual

**Given** user A holds stored data, on the hub of the developer machine.
**When** the owner opens the overview with the parking service blocked at the network, then after a disconnect on the Account page.
**Then** the first view shows "The parking service is unavailable." in the card and every stored part. The second shows "Not connected" with a link to the Account page, and every stored part.

## Retired identifiers

This file has no retired identifier.
