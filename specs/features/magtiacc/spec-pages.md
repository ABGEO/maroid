---
id: MAGTIACC
title: The widget and the pages of the telecom account
type: spec
status: approved
created: 2026-11-24
updated: 2026-11-24
approved_by: Temuri
approved_on: 2026-11-24
constrained_by: [UI, TS, RES, API, SEC]
requirements: features/magtiacc/requirements.md
---

# Specification: The widget and the pages of the telecom account

`spec.md` holds the components, the data model, and the declarations. This file holds the
design decisions of the connection page, the subscriber page, the widget, the traffic page, and
the money page.

## Design decisions

### `MAGTIACC-DD-012`

**Realizes:** `MAGTIACC-FR-003` to `MAGTIACC-FR-019`
**Decision:** `Account.svelte` repeats the page of the electricity plugin. A subscriber row
shows the number, the label, and the kind ("Mobile", "Home line", or "Other"). A row of kind
`other` shows no assign button.
**Rationale:** The two pages read alike. The route refuses kind `other` too (`MAGTIACC-FR-018`).
**Alternatives:** Hide a subscriber of kind `other`. The holder then wonders where the domain went.

### `MAGTIACC-DD-013`

**Realizes:** `MAGTIACC-FR-032` to `MAGTIACC-FR-036`
**Decision:** `GET /subscriber-states/self` answers `subscriber_number`, `kind`, `status`, and
`balance`. A mobile subscriber adds `plan` (`name`, `price`, `renews_on` as a date of Tbilisi,
`days_left`) when it holds one, and `allowances` (`data_mb`, `call_seconds`, `sms`). A home line
adds `plan` (`price`, `days_left`) and `speed`, the text of the portal, for example `70 Mbps`.
`Subscriber.svelte` shows one card for the balance and one for the plan, and for a mobile
subscriber one value for each allowance, because the portal answers no total to fill a meter. The
call time shows as `mm:ss`. A 409 shows the setup prompt with a link to `/account`. A 502 shows
"Magti is unavailable", with a button that reads again.
**Rationale:** A member that a kind does not hold stays absent, as in the transactions of the
electricity plugin (`ELECACC-DD-012`).
**Alternatives:** One object with every member, null when absent. The page then checks the kind
and the null.

### `MAGTIACC-DD-014`

**Realizes:** `MAGTIACC-FR-034`, `MAGTIACC-FR-037` to `MAGTIACC-FR-039`
**Decision:** `Load` calls `States.Read`. `ErrNotConnected` and `ErrNotAssigned` answer
`NotSetUpAnswer("Connect the Magti account and assign a subscriber.", "/account")`. Any other
error is the error of the widget. A state answers
`StatData{Value: "<balance, two decimals>", Unit: "₾", Caption: <caption>, Link: "/subscriber"}`.

| Subscriber           | Caption                                                     |
| -------------------- | ----------------------------------------------------------- |
| Mobile, with a plan  | `<data_mb, no decimal> MB left, renews in <days_left> days` |
| Mobile, with no plan | `No active plan`                                            |
| Home line            | `<price> ₾ plan, renews in <days_left> days`                |

`days_left` of 1 writes "1 day", and 0 writes "renews today".
**Rationale:** The balance answers the question "can I pay for the next period", so it is the value.
**Alternatives:** A list widget with each allowance. It needs size `m` for four lines.

### `MAGTIACC-DD-015`

**Realizes:** `MAGTIACC-FR-040` to `MAGTIACC-FR-044`
**Decision:** `GET /traffic-days` takes `from` and `to`, both required dates `YYYY-MM-DD`, at most
366 days apart, `limit` (1 to 100, default 20), `cursor`, and `sort`, which takes `day` with the
default `day` (`RES-005`). One subscriber holds one row of a day, so no tie occurs. A row answers `day`, and each of `data_mb`, `call_in_seconds`, `call_out_seconds`,
`sms_sent`, and `sms_received` that it holds. `Traffic.svelte` offers 30, 90, and 365 days,
reads `from` and `to` with `limit=100`, and follows `next` to the last page. It draws three
`ColumnLineChart` of `@maroid/ui` with columns alone: the data in MB, the call minutes out and in
as two series, and the SMS sent and received as two series. Every day of the period keeps its
slot on the axis, and a day with no row draws no column. A chart holds no line, so it draws no
right axis.
The totals sum the rows in the page.
**Rationale:** 365 rows are four pages. The client sums at most 366 rows, so a route of totals
adds no value. The axis names the month, as in the pulse plugin.
**Alternatives:** A route of totals. It repeats a sum that the page already holds the rows for.

### `MAGTIACC-DD-016`

**Realizes:** `MAGTIACC-FR-045`, `MAGTIACC-FR-046`
**Decision:** `GET /money-entries` takes `limit` (1 to 100, default 20), `cursor`, and `sort`,
which takes `occurred_at` with the default `-occurred_at` (`RES-005`). A tie sorts by `id` in
the same direction, and the cursor of `RES-006` keys on both. A row answers `id`, `occurred_at` as RFC 3339 in UTC, `direction`, `kind`,
and `amount` as the money object. `Money.svelte` shows the time in the zone of the browser
(`ADR-0024`), a top up in green with a plus sign, and a charge with a minus sign.
**Rationale:** A moment stays UTC on the wire.
**Alternatives:** The time of Tbilisi on the wire. It breaks the rule of the moment.

### `MAGTIACC-DD-017`

**Realizes:** `MAGTIACC-FR-047`
**Decision:** `GET /monthly-money-totals` takes `from`, a month `YYYY-MM`, required, at most 11 months
before the current month of Tbilisi, and answers the page of every month from `from` to the
current month, with `month`, `top_ups`, and `charges` as money objects. `sort` takes `month`,
with the default `-month`. The handler orders the filled months in memory. A month with no money entry answers zero for both.
`Money.svelte` reads `from` 11 months before the current month, and shows the 12 months above
the list. `MonthTotals` groups by the month of `occurred_at` in Tbilisi in SQL, and the handler
fills the empty months. The collection holds at most 12 items, so it answers one page with no
`limit` and no `cursor`, and `next` stays absent (`RES-005`).
**Rationale:** A charge of a plan, an SMS, and a call each make a money entry, so one year holds
hundreds. A sum in the database reads one index.
**Alternatives:** Sum in the client. It reads every entry of a year for 24 numbers.

## Retired identifiers

This file has no retired identifier.
