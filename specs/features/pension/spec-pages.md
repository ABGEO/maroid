---
id: PENSION
title: The widget and the pages of the pension account
type: spec
status: approved
created: 2026-11-27
updated: 2026-11-27
approved_by: Temuri
approved_on: 2026-11-27
constrained_by: [UI, TS, RES, API, SEC, OWN, PLG, TST]
requirements: features/pension/requirements-pages.md
---

# Specification: The widget and the pages of the pension account

`spec.md` holds the connection, the collection, the data model, and the build plan. This
file holds what a member reads: the read routes, the widget, the three pages, their design
decisions, and their scenarios. Every route and every page reads the database alone
(`PENSION-FR-046`). Row level security limits each read to the acting user (`PENSION-INV-002`).

## 1. Components

Every path is under `plugins/pensions/`.

| Path                                                         | Action | Holds                                                                                                                                                                                                   |
| ------------------------------------------------------------ | ------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `model/read.go`                                              | create | `SummaryView` with `Connected` and `Summary *AccountSummary`. `ContributionFilter` with `EmployerCode`, `From`, `To`. `MonthSum`                                                                        |
| `repository/contribution.go`                                 | change | `Contribution.Page`, `Contribution.Total`, `Contribution.MonthSums`                                                                                                                                     |
| `repository/{employer,asset_value,unit_price}.go`            | change | `Employer.OfContributions`, `AssetValue.Since`, `PortfolioUnitPrice.WeeklySince`                                                                                                                        |
| `repository/{summary,portfolio_share}.go`                    | change | `AccountSummary.Get`, `PortfolioShare.List`                                                                                                                                                             |
| `repository/keyset.go`                                       | create | The keyset condition and the order of the contributions page                                                                                                                                            |
| `service/store_read.go`                                      | create | The read methods of `Store`, each in one `WithTx`                                                                                                                                                       |
| `service/reader.go`                                          | create | `Reader`: `Summary(ctx)`, which joins the stored summary with the presence of the `session` secret                                                                                                      |
| `dto/{summary,contribution,asset_value,portfolio}.go`        | create | The answers of section 2, and `MoneyResponse` of `spec.md`                                                                                                                                              |
| `handler/{summary,contribution,asset_value,portfolio}.go`    | create | `SummaryHandler`, `ContributionHandler`, `AssetValueHandler`, `PortfolioHandler`                                                                                                                        |
| `widget/{doc,asset_value}.go`                                | create | `NewAssetValue`                                                                                                                                                                                         |
| `permission/permission.go`                                   | change | `AccountRead = "account.read"`                                                                                                                                                                          |
| `main.go`                                                    | change | `Widgets`, the new routes, `account.read`, and the UI routes of section 2                                                                                                                               |
| `ui/src/pages/{overview,contributions,portfolios}/`          | create | `Overview.svelte`, `Contributions.svelte`, `Portfolios.svelte`                                                                                                                                          |
| `ui/src/components/`                                         | create | `PeriodPicker.svelte`, `SetupNotice.svelte`, `NoAccess.svelte`, `ValueChart.svelte`, `UnitPriceChart.svelte`, `SummaryCards.svelte`                                                                     |
| `ui/src/lib/{address,chart,format,pages,period,state}.ts`    | create | The filter in the address (`PENSION-DD-020`). The series of the overview chart. Money, percent, and date formats. Every page of a collection. The three periods. The state of a page (`PENSION-DD-013`) |
| `ui/src/api/{summary,contribution,asset_value,portfolio}.ts` | create | One function for each route of section 2                                                                                                                                                                |

Each handler depends on an interface that it declares, and `Store` or `Reader` meets it.

## 2. Declarations

**HTTP routes.** The prefix of `spec.md` applies. `api.yaml` holds the bodies.

| Method | Path                      | Access        | Permission     | Takes                                                              | Realizes                                                               |
| ------ | ------------------------- | ------------- | -------------- | ------------------------------------------------------------------ | ---------------------------------------------------------------------- |
| `GET`  | `/account-summaries/self` | Authenticated | `account.read` | Nothing                                                            | `PENSION-FR-032`, `PENSION-FR-033`, `PENSION-FR-044`, `PENSION-FR-045` |
| `GET`  | `/asset-values`           | Authenticated | `account.read` | `from`, `limit` (1 to 100, default 20), `cursor`                   | `PENSION-FR-034`                                                       |
| `GET`  | `/contribution-months`    | Authenticated | `account.read` | `from`                                                             | `PENSION-FR-034`                                                       |
| `GET`  | `/contributions`          | Authenticated | `account.read` | `employer`, `from`, `to`, `limit` (1 to 100, default 20), `cursor` | `PENSION-FR-036` to `PENSION-FR-040`, `PENSION-NFR-001`                |
| `GET`  | `/contributions/total`    | Authenticated | `account.read` | `employer`, `from`, `to`                                           | `PENSION-FR-041`, `PENSION-NFR-001`                                    |
| `GET`  | `/employers`              | Authenticated | `account.read` | Nothing                                                            | `PENSION-FR-039`                                                       |
| `GET`  | `/portfolio-shares`       | Authenticated | `account.read` | Nothing                                                            | `PENSION-FR-042`                                                       |
| `GET`  | `/portfolio-unit-prices`  | Authenticated | `account.read` | `from`, `limit` (1 to 100, default 20), `cursor`                   | `PENSION-FR-043`                                                       |

`/contributions`, `/asset-values`, and `/portfolio-unit-prices` answer pages, the latest date
first, because each one grows past a few hundred rows over the years (`RES-005`). Every other route
is a bounded collection or a self resource: it takes no `limit` and no `cursor`. `from` and `to` are dates `YYYY-MM-DD`, each one included. An absent
`from` reads from the first row.

**Permission.**

| Name           | Description                                                        | Lowest   | Realizes         |
| -------------- | ------------------------------------------------------------------ | -------- | ---------------- |
| `account.read` | Read your own pension summary, contributions, and portfolio shares | `viewer` | `PENSION-FR-047` |

**Widget.**

| Key           | Title   | Kind   | Size | Permission     | Realizes                             |
| ------------- | ------- | ------ | ---- | -------------- | ------------------------------------ |
| `asset-value` | Pension | `stat` | `s`  | `account.read` | `PENSION-FR-027` to `PENSION-FR-031` |

**UI routes.** The first route is the landing page of the plugin.

| Page route       | Navigation entry | Page                   | Realizes                                                                 |
| ---------------- | ---------------- | ---------------------- | ------------------------------------------------------------------------ |
| `/overview`      | "Overview"       | `Overview.svelte`      | `PENSION-FR-032` to `PENSION-FR-035`, `PENSION-FR-044`, `PENSION-FR-045` |
| `/contributions` | "Contributions"  | `Contributions.svelte` | `PENSION-FR-036` to `PENSION-FR-041`, `PENSION-FR-045`                   |
| `/portfolios`    | "Portfolios"     | `Portfolios.svelte`    | `PENSION-FR-042`, `PENSION-FR-043`, `PENSION-FR-044`, `PENSION-FR-045`   |
| `/account`       | "Account"        | `Account.svelte`       | `spec.md`, section 4.3                                                   |

**Indexes.** The migration of `contributions` in `spec.md` adds
`contributions_user_paid_on_idx (user_id, paid_on DESC, id DESC)` and
`contributions_user_employer_paid_on_idx (user_id, employer_code, paid_on DESC, id DESC)`
(`PENSION-DD-015`).

**Errors.** A `from` or a `to` that is not a date, a `from` after a `to`, or a parameter that a
collection route does not declare answers 400 `/problems/http/request-invalid`, with a detail that names
the parameter. An `employer` that no contribution names answers an empty page and a total of
zero (`PENSION-FR-039`).

## 3. Design decisions

### `PENSION-DD-012`

**Realizes:** `PENSION-FR-027` to `PENSION-FR-031`
**Decision:** `NewAssetValue` reads `Reader.Summary`. A summary answers
`StatData{Value: "<asset value>", Unit: "₾", Caption: "Gain <gain> ₾, <dd/mm/yyyy>", Link: "/overview"}`,
with two decimals and the date of `collected_at` in `ActingTimeZone`. No summary and no session
answers `NotSetUpAnswer("Connect the pension account.", "/account")`. No summary and a session
answers `EmptyAnswer("The first collection runs on the first day of the next month.")`.
**Rationale:** The summary is one row. A disconnected member keeps the summary, and the widget
shows it, because the summary decides before the session does.
**Alternatives:** A list widget of the latest contributions. The owner chose the stat.

### `PENSION-DD-013`

**Realizes:** `PENSION-FR-044`, `PENSION-FR-045`, `PENSION-FR-046`
**Decision:** `GET /account-summaries/self` answers 200 with `connected` and, when a summary
exists, `summary`. `connected` reports that the `session` secret exists, with no call to the
pension service. `state.ts` maps the answer: a summary is `ready`, no summary with `connected` is
`empty`, and neither is `not_set_up`. The overview page and the portfolios page read it first. A
role without `account.read` sends no read: each of the three pages shows `NoAccess.svelte`.
**Rationale:** One read decides the state of every page with no call outside Maroid. The connection
route of `spec.md` calls the pension service, so a page cannot use it.
**Alternatives:** 404 for an absent summary. A page then cannot tell `empty` from `not_set_up`.

### `PENSION-DD-014`

**Realizes:** `PENSION-FR-036`, `PENSION-FR-038`
**Decision:** The order of `/contributions` is `paid_on DESC, id DESC`, fixed, and the cursor of
`page.Read` keys on both. The cursor carries `employer`, `from`, and `to`, so a change of a filter
retires it. A row answers `id`, `paid_on`, `employer` (`code` and `name`, absent when none),
`source`, `type`, `salary_month` as `YYYY-MM` (absent when the year or the month is null),
`gross_salary`, and `amount`, each amount a money object.
**Rationale:** Two contributions of one basis share a date, so the date alone does not order a page.
`source` and `type` travel as the pension service gives them; the page shows them as they are.
**Alternatives:** A sort that the client names. No requirement asks for one.

### `PENSION-DD-015`

**Realizes:** `PENSION-NFR-001`, `PENSION-FR-041`
**Decision:** `/contributions/total` answers `amount`, a money object, and `count`, with the same
filter as the page, in one `SELECT sum(amount), count(*)`. The two indexes of section 2 serve the
page and the total, with and without `employer`.
**Rationale:** A user holds about 500 contributions after 7 years. Each read then scans one range
of one index. The page and the total are two requests, so a page of the list needs no sum.
**Alternatives:** The sum as a member of the page. `Page` of `libs/rest/page` holds the items and
the links alone, and the page links would repeat the sum on every page.

### `PENSION-DD-016`

**Realizes:** `PENSION-FR-034`, `PENSION-FR-035`
**Decision:** `ValueChart.svelte` draws `ColumnLineChart` of `@maroid/ui` with one slot for each
month of the period, keyed `YYYY-MM-01`. A column is the sum of `/contribution-months` of that
month, on the left axis. The line is the last asset value of `/asset-values` within that month,
on the right axis. The page reads every page of `/asset-values` with `limit=100`. `period.ts` gives `from`: the first day of the month 11 months back, 35 months
back, or no `from` for every month. The default is 12 months.
**Rationale:** The pension service reports two or three asset values a month, on dates that do not
align with the columns. One slot for each month keeps a month with no contribution on the axis,
through the `dates` property of the chart.
**Alternatives:** One slot for each date of an asset value. The columns then fall between the points.

### `PENSION-DD-017`

**Realizes:** `PENSION-FR-034`
**Decision:** `/contribution-months` answers `month` as `YYYY-MM` and `amount`, the sum of the
contributions whose `paid_on` falls in that month, from `from` to the last contribution.
**Rationale:** The database sums 500 rows in one statement. The client reads one row for each month.
**Alternatives:** The client sums the pages of `/contributions`. All months then cost 5 requests of 100.

### `PENSION-DD-018`

**Realizes:** `PENSION-FR-043`
**Decision:** `/portfolio-unit-prices` answers, for each portfolio of the shares of the acting
user, the last unit price of each ISO week from `from`: `portfolio_id`, `priced_on`, and
`unit_price`. The statement is `SELECT DISTINCT ON (portfolio_id, date_trunc('week', priced_on))`,
restricted to `portfolio_id IN (SELECT portfolio_id FROM portfolio_shares)`. `UnitPriceChart.svelte`
draws one line for each portfolio, with no column, and names each portfolio in the legend by its
ticker. A portfolio with no unit price in the period keeps its legend entry. The page reads every
page with `limit=100`, and the keyset of `PENSION-DD-014` orders the weekly rows by `priced_on` and
`id`.
**Rationale:** Every month costs about 300 points for each portfolio in place of 2000. The policy of
`portfolio_shares` limits the portfolios to those of the acting user, so the shared table reaches
a user only through a row of their own.
**Alternatives:** Every day. The chart then holds 2000 slots that no pointer can tell apart.

### `PENSION-DD-019`

**Realizes:** `PENSION-FR-039`, `PENSION-FR-042`
**Decision:** `/employers` answers each employer that a contribution of the acting user names,
by name: `code` and `name`. `/portfolio-shares` answers each share with the English names of the
portfolio, the asset manager, and the risk tier, the ticker, `target_share` and `current_share`
as percents, `amount` as a money object, `units`, and `unit_price`.
**Rationale:** The join through `contributions` applies the policy, so a user reads no employer of
another user. The deck is in English, and the pension service gives an English name for each.
**Alternatives:** The Georgian names. The stored rows keep them for a later choice of language.

### `PENSION-DD-020`

**Realizes:** `PENSION-FR-039`, `PENSION-FR-040`
**Decision:** The contributions page keeps `employer`, `from`, and `to` in the query of its address.
It reads them from `window.location` when it mounts, and writes each change with
`history.replaceState`, which keeps `history.state` of the deck. A change of a filter reads the
first page again.
**Rationale:** A reload keeps the filter, and a link names one employer. `PluginHost` gives a path
alone, and a change to it changes the contract of every plugin.
**Alternatives:** The filter in the state of the page alone. A reload then loses it. A query member
of `PluginHost`. It needs an ADR, and no other plugin asks for it.

## 4. Scenarios

A unit scenario of a handler runs it against a fake reader. An integration scenario runs the
repositories against the database of `libs/testdb`. User A acts in workspace W.

### `PENSION-SC-032`

**Verifies:** `PENSION-FR-027`, `PENSION-FR-031`
**Layer:** unit

**Given** a summary of 12480.55 with a gain of 1904.10, collected at 2026-11-01 06:00 UTC, and
the time zone Asia/Tbilisi.
**When** the widget loads.
**Then** it answers `ready` with the value `12,480.55`, the unit `₾`, the caption
`Gain 1,904.10 ₾, 01/11/2026`, and the link `/overview`.

### `PENSION-SC-033`

**Verifies:** `PENSION-FR-028`, `PENSION-FR-029`, `PENSION-FR-030`
**Layer:** unit

**Given** three users: B with no summary and no session, C with no summary and a session, D
with a summary and no session.
**When** the widget loads for each one.
**Then** B answers `not_set_up` with the link `/account`, C answers `empty`, and D answers `ready`.

### `PENSION-SC-034`

**Verifies:** `PENSION-FR-032`, `PENSION-FR-033`, `PENSION-FR-044`, `PENSION-FR-045`
**Layer:** unit

**Given** the same three users as `PENSION-SC-033`.
**When** each one reads `GET /account-summaries/self`.
**Then** B gets `connected: false` and no `summary`, C gets `connected: true` and no `summary`,
and D gets `connected: false` and a `summary` with every member and `collected_at`.

### `PENSION-SC-035`

**Verifies:** `PENSION-FR-034`, `PENSION-FR-035`
**Layer:** manual

**Given** the asset values of 2026-01-10, 2026-01-25, and 2026-03-12, and the contribution months
2026-01 and 2026-03, on the hub of the developer machine in November 2026.
**When** the owner opens the overview with the period of 12 months.
**Then** the axis holds 12 slots from 2025-12-01 to 2026-11-01. January holds a column and the
asset value of 2026-01-25. February holds no column and no point.

### `PENSION-SC-036`

**Verifies:** `PENSION-FR-034`
**Layer:** integration

**Given** user A holds contributions in January and March 2026, and user B holds one in February.
**When** user A reads `/contribution-months?from=2026-01-01`.
**Then** the answer holds 2026-01 and 2026-03 with the sum of each, and no 2026-02.

### `PENSION-SC-037`

**Verifies:** `PENSION-FR-036`, `PENSION-FR-037`, `PENSION-FR-038`
**Layer:** integration

**Given** user A holds 45 contributions, three of them on 2026-10-15, one with no employer.
**When** user A reads `/contributions`, then follows `next` twice.
**Then** the pages hold 20, 20, and 5 rows, the latest date first, each row once. The row with no
employer holds no `employer` member.

### `PENSION-SC-038`

**Verifies:** `PENSION-FR-039`, `PENSION-FR-040`, `PENSION-FR-041`
**Layer:** integration

**Given** user A holds 30 contributions of employer `204567890` and 15 of another employer.
**When** user A reads `/contributions` and `/contributions/total` with `employer=204567890`,
`from=2026-01-01`, and `to=2026-03-31`, then with an employer of no contribution.
**Then** both answers hold the contributions of that employer in the first quarter alone. The total
counts every match, not the first 20. The unknown employer answers an empty page and a total of 0.

### `PENSION-SC-039`

**Verifies:** `PENSION-FR-040`
**Layer:** unit

**Given** the contributions route.
**When** a request names `from=2026-04-01&to=2026-03-01`, then `from=2026-13-01`.
**Then** each answers 400 `/problems/http/request-invalid`, with a detail that names the parameter.

### `PENSION-SC-040`

**Verifies:** `PENSION-FR-039`
**Layer:** integration

**Given** user A holds contributions of two employers, and user B of a third.
**When** user A reads `/employers`.
**Then** the answer holds the two employers of user A, and not the third.

### `PENSION-SC-041`

**Verifies:** `PENSION-FR-042`
**Layer:** unit

**Given** a share of portfolio 7 with `target_share` 100 and `current_share` 98.7.
**When** user A reads `/portfolio-shares`.
**Then** the row holds the English names, the ticker, both percents, the amount as a money object,
the units, and the unit price.

### `PENSION-SC-042`

**Verifies:** `PENSION-FR-043`
**Layer:** integration

**Given** user A holds the share of portfolio 7, the table holds daily unit prices of portfolios 7
and 9 for 2026, and user A holds no share of 9.
**When** user A reads `/portfolio-unit-prices?from=2026-01-01&limit=20`, then follows `next` to
the last page.
**Then** the pages hold one row for each ISO week of 2026 for portfolio 7, each the last price of
its week, each row once, and no row of portfolio 9.

### `PENSION-SC-043`

**Verifies:** `PENSION-FR-046`
**Layer:** integration

**Given** a fake pension service that fails every call.
**When** the widget loads and each route of section 2 answers.
**Then** each answer succeeds, and the fake service records no call.

### `PENSION-SC-044`

**Verifies:** `PENSION-FR-047`
**Layer:** unit

**Given** the plugin.
**When** the test reads `Permissions`, `Routes`, and `Widgets`.
**Then** `account.read` has the lowest role `viewer`, and each route of section 2 and the widget name it.

### `PENSION-SC-045`

**Verifies:** `PENSION-NFR-001`
**Layer:** integration

**Given** user A holds 2,000 contributions of four employers over 10 years, and user B holds 2,000.
**When** the test reads `/contributions` and `/contributions/total` 20 times with each filter: none,
`employer`, `from` and `to`, and all three.
**Then** the 95th percentile of each route stays below 300 milliseconds, from the request to the
last byte, on the database of the developer machine.

## Retired identifiers

This file has no retired identifier.
