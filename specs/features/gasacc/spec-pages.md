---
id: GASACC
title: The widget and the transactions page of the gas account
type: spec
status: approved
created: 2026-10-14
updated: 2026-10-14
approved_by: Temuri
approved_on: 2026-10-14
constrained_by: [UI, TS, RES, API]
requirements: features/gasacc/requirements.md
---

# Specification: The widget and the transactions page of the gas account

`spec.md` holds the components, the data model, and the declarations. This file holds the
design decisions of the widget, of the transactions page, of the chart, and of the money object.

## Design decisions

### `GASACC-DD-011`

**Realizes:** `GASACC-FR-029`
**Decision:** `StatData{Value: "<consumption>", Unit: "m³", Caption: "<amount> ₾, meter <value> m³, <dd/mm/yyyy>", Link: "/transactions"}`.
The consumption and the meter value take no decimal. The amount takes two.
**Rationale:** One reading fits one stat. The household reads the usage first.
**Alternatives:** A list of one line. A list shows several things.

### `GASACC-DD-012`

**Realizes:** `GASACC-FR-034`, `GASACC-FR-035`
**Decision:** The order is `transaction_date DESC, id DESC`, fixed. The boundary of a cursor is
the identifier of a row, and the condition compares `(transaction_date, id)` with the one of
that row. `page.Read` keys the cursor on the identifier. A reading answers `usage`. A
transaction with a meter value above zero, a reading or a control reading, answers `meter_value`.
A payment answers neither.
**Rationale:** A reading and a payment share a date, so the date alone cannot order the page.
**Alternatives:** A filter of the kind. The chart reads 12 months, at most 30 rows.

### `GASACC-DD-013`

**Realizes:** `GASACC-FR-036`
**Decision:** `base_url` is the origin of the gas service. The client calls the API under
`base_url` and `/api`. The route answers `bill_url` and `meter_photo_url`: `base_url`, a slash,
and the stored path, which starts with `api/`. A null path answers no member. The page renders
each as a link with `target="_blank"` and `rel="noopener noreferrer"`.
**Rationale:** The owner chose a link to the gas service. The gas service serves the file with
no session. Maroid stores no file. A stored path is relative to the origin, so a link, a
download, and a call of the API share one base, and no code takes a URL apart.
**Alternatives:** A route of Maroid that downloads the file. It holds the file in memory and
needs a token for each click.

### `GASACC-DD-014`

**Realizes:** `GASACC-FR-037`
**Decision:** `UsageChart.svelte` draws the chart as SVG markup. `d3-scale` gives the scales
and `d3-shape` the path of the line, and D3 touches no element. The x axis is a band for each
reading date. The consumption of each reading is a column on the left axis, in cubic metres,
from zero. The amount is a line on the right axis, in lari. The columns take `fill-primary`,
the line `stroke-accent`, the axes `fill-base-content`, and the grid `stroke-base-300`. The
width follows the container. A pointer over a date shows a daisyUI tooltip with the usage and
the amount. The SVG carries `role="img"` and a label, and the table is the accessible form of
the same transactions. The page reads `from` 12 months before today with `limit=100`, follows
`next` to the last page, and keeps the readings. The chart lives in the plugin.
**Rationale:** The chart is plain markup, so it takes the classes of the daisyUI theme, and a
change of the theme recolors it with no code. The secondary color shares the hue of the primary in both themes,
so a line of it fades over the columns in the dark theme. The accent is green in both. Both D3 modules are small and stable releases.
**Alternatives:** A chart in `@maroid/ui`. The owner keeps it in the plugin. A library of charts
on a canvas. A canvas reads no CSS variable.

### `GASACC-DD-015`

**Realizes:** `GASACC-FR-034`
**Decision:** `amount` and `balance` of a transaction travel as the money object of `Z-173`:
`{"amount": 11.33, "currency": "GEL"}`, `amount` a decimal number and `currency` the ISO 4217 code.
The gas service bills in lari alone, so `MoneyResponse` sets `GEL`.
**Rationale:** `RES-001` binds Zalando, and `RES-002` holds no deviation from `Z-173`. The owner
chose the object over a deviation.
**Alternatives:** A bare number in lari, with a deviation in `RES-002`. Every client then knows the
currency from the documentation alone.

## Retired identifiers

This file has no retired identifier.
