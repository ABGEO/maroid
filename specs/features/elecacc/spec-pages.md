---
id: ELECACC
title: The widget and the pages of the electricity account
type: spec
status: approved
created: 2026-10-14
updated: 2026-10-14
approved_by: Temuri
approved_on: 2026-10-14
constrained_by: [UI, TS, RES, API, SEC]
requirements: features/elecacc/requirements.md
---

# Specification: The widget and the pages of the electricity account

`spec.md` holds the components, the data model, and the declarations. This file holds the
design decisions of the widget, of the transactions page, of the chart, of the bills page, of
the download, and of the money object.

## Design decisions

### `ELECACC-DD-011`

**Realizes:** `ELECACC-FR-032`
**Decision:** `LatestReading` reads the charges of the latest date of a charge, and
`ReadingsOf` makes one reading. `StatData{Value: "<usage>", Unit: "kWh", Caption: "<charge> ₾, meter <value> kWh, <dd/mm/yyyy>", Link: "/transactions"}`.
The usage and the meter value take at most one decimal. The charge takes two.
**Rationale:** One reading fits one stat, as in the gas plugin.
**Alternatives:** The latest charge alone. A split reading then shows half its usage.

### `ELECACC-DD-012`

**Realizes:** `ELECACC-FR-037`, `ELECACC-FR-038`
**Decision:** The order is `transaction_date DESC, id DESC`, fixed, and the cursor of
`page.Read` keys on the identifier, as in the gas plugin. A row answers `operation` and
`transaction_date`, then each of `amount`, `balance`, `usage`, `meter_value`, and `tariff`
that it holds. A charge answers all five. A payment answers `amount` and `balance`. The first
reading of a new meter answers `meter_value`. `GET /transactions` takes `limit` (1 to 100,
default 20), `cursor`, and `from`, a date `YYYY-MM-DD`.
**Rationale:** The portal leaves a value empty when the kind has none, and the answer follows it.
**Alternatives:** A member `kind`. No rule of the portal names the kinds (`ELECACC-DD-008`).

### `ELECACC-DD-013`

**Realizes:** `ELECACC-FR-039`
**Decision:** `UsageChart.svelte` repeats the chart of the gas plugin: SVG markup, `d3-scale`
`4.0.2` and `d3-shape` `3.2.0`, the usage of each reading as a column in kWh on the left axis,
and the charge as a line in lari on the right axis. The page reads `from` 12 months before
today with `limit=100`, follows `next` to the last page, and groups the charges of each date
into one reading, as `ReadingsOf` does.
**Rationale:** The two pages read alike. The plugin cannot import the gas plugin (`PLG-008`).
**Alternatives:** A route of readings. The client groups at most 24 rows.

### `ELECACC-DD-014`

**Realizes:** `ELECACC-FR-040`, `ELECACC-FR-041`
**Decision:** `GET /bills` takes `limit` (1 to 100, default 20) and `cursor`. The order is
`month DESC, id DESC`, fixed. A row answers `id`, `month` as `YYYY-MM`, `bill_number`,
`issued_on` as a date of Tbilisi, and `document`, the absolute address of the download, which
the handler builds from the same external address as the links of the page (`RES-005`).
**Rationale:** A link that the server answers keeps the address in one place, and `Z-217`
asks for an absolute one.
**Alternatives:** The page builds the address. The page then knows the prefix of the hub.

### `ELECACC-DD-015`

**Realizes:** `ELECACC-FR-042`, `ELECACC-INV-005`
**Decision:** `GET /bills/{billId}/document` reads the bill through `Store`, so row level
security and the assigned customer limit it, and answers 404 for any other identifier.
`Documents.Open` downloads `service_bill_id` through `account.call`, and the route answers 200
with `Content-Type: application/pdf`, `Content-Disposition: attachment; filename="<bill_number>.pdf"`,
and `Cache-Control: private, no-store`. A failure answers the problem of section 4.5 of `spec.md`.
`Bills.svelte` renders `document` as `<a href download>`. The link names the origin of the hub,
and the session cookie of the hub is `SameSite=Lax`, so the browser sends it on that navigation
from the deck.
**Rationale:** The portal serves the document only inside a session. `ApiClient` of the SDK
reads JSON alone, and a link needs no change to it.
**Alternatives:** A method of `ApiClient` that answers a blob. It changes a shared library for one page.
A copy of each document in Maroid. The requirements exclude it.

### `ELECACC-DD-016`

**Realizes:** `ELECACC-FR-037`
**Decision:** `amount` and `balance` travel as the money object of `Z-173`, with `currency`
`GEL`, as in the gas plugin. `tariff` travels as a decimal number of lari for each kWh,
because it is a rate and no amount.
**Rationale:** `RES-001` binds Zalando, and the gas plugin set the precedent.
**Alternatives:** A money object for the tariff. `Z-173` names an amount, not a rate.

## Retired identifiers

This file has no retired identifier.
