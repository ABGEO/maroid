---
id: GWPACC
title: The readings page of the water account
type: spec
status: approved
created: 2026-10-12
updated: 2026-10-13
approved_by: Temuri
approved_on: 2026-10-12
constrained_by: [PLG, OWN, RES, API, ERR, UI, TS, GO, BLD, TST]
requirements: features/gwpacc/requirements.md
---

# Specification: The readings page of the water account

## 1. Summary

The plugin gains the route `GET /readings`, a keyset page of the readings of the assigned
customer, the latest date first, with the filter `from`. The remote gains the page
`/readings`: a daisyUI table with the shared `Pagination`, and a chart of the last 12 months
that Svelte draws as SVG over the scales of D3, with the classes of the daisyUI theme. The
widget links to the page. The page and the route read only the database.

## 2. Guideline compliance

| Rule      | Guideline | How this specification obeys it |
| --------- | --------- | ------------------------------- |
| `RES-005`, `RES-006` | REST | The route answers the `Page` of `libs/rest/page`, with `limit`, `cursor`, `self`, `first`, `next`, and `prev`. |
| `OWN-005`, `OWN-008` | Ownership | The repository writes no filter on `workspace_id`. The policy scopes every read. |
| `ERR-002` | Errors | `not-assigned` has the form `/problems/dev.maroid.gwp/<slug>`. |
| `SEC-013` | Security | The route names `account.read`. |
| `UI-005`, `UI-006`, `UI-007` | Web UI | The remote adds `/readings`, imports the shared `Pagination` and `createPager` from `@maroid/ui`, and reaches the route through `host.api`. |
| `TS-002` | Frontend | The page and the chart take their props through `$props`. |

## 3. Design

### 3.1 Components

| Path | Action | Holds |
| ---- | ------ | ----- |
| `plugins/gwp/model/reading.go` | change | `Reading.ID` |
| `plugins/gwp/repository/reading.go` | change | `List(ctx, seek page.Seek, customerNumber string, from *time.Time) ([]model.Reading, error)` |
| `plugins/gwp/service/store.go` | change | `ReadingsPage(ctx, asked page.Request, from *time.Time) (page.Window[model.Reading], error)`. It reads the assignment and the page in one transaction, and answers `errs.ErrNotAssigned` |
| `plugins/gwp/dto/reading.go` | create | `MeterReadingResponse`, `NewMeterReadingResponseList`. `ReadingResponse` already names the answer of the water service |
| `plugins/gwp/handler/reading.go` | create | `ReadingHandler`, `NewReadingHandler`, `ReadingLister` |
| `plugins/gwp/handler/problem.go` | change | `typeNotAssigned` |
| `plugins/gwp/widget/usage.go` | change | `ListData.More` is `/readings` |
| `plugins/gwp/main.go` | change | `Routes` adds the readings route. `UIManifest` adds `/readings` |
| `plugins/gwp/ui/package.json` | change | `d3-scale` at exactly `4.0.2` and `d3-shape` at exactly `3.2.0`. `@types/d3-scale` `4.0.9` and `@types/d3-shape` `3.2.0` as development dependencies |
| `plugins/gwp/ui/src/api/{reading,types,index}.ts` | create, change | `Reading`, `ReadingPage`, `createReadingApi` with `page(cursorLink?)` and `since(from)` |
| `plugins/gwp/ui/src/lib/chart.ts` | create | `chartSeries(readings)`: the dates, the meters, and the usage and the value of each meter on each date. `meterTone(index)`: the daisyUI classes of a meter |
| `plugins/gwp/ui/src/pages/readings/{Readings,UsageChart}.svelte` | create | The page and the chart |
| `plugins/gwp/ui/src/entry.ts` | change | `/readings` |

### 3.2 Data model

No table changes. `List` reads the index `readings_customer_meter_date_idx` through its
leading columns `workspace_id` and `customer_number`.

### 3.3 Declarations

**HTTP route.** The prefix is `/workspaces/{workspaceId}/plugins/dev.maroid.gwp/api`.

| Method | Path | Access | Permission | Realizes |
| ------ | ---- | ------ | ---------- | -------- |
| `GET` | `/readings` | Authenticated | `account.read` | `GWPACC-FR-028` to `GWPACC-FR-033` |

The route takes `limit` (1 to 100, default 20), `cursor`, and the filter `from`, a date
`YYYY-MM-DD`. It takes no `sort`. Each item is:

| Member | Type | From |
| ------ | ---- | ---- |
| `id` | string, UUID | `readings.id` |
| `serial_number` | string | `serial_number` |
| `reading_date` | string, `date` | `reading_date` |
| `value` | number | `value` |
| `usage` | number, absent for the earliest reading of a meter | `value` minus the `value` of the reading before it, of the same meter |

**UI routes.**

| Page route | Navigation entry | Page | Realizes |
| ---------- | ---------------- | ---- | -------- |
| `/readings` | "Readings" | `Readings.svelte` | `GWPACC-FR-028` to `GWPACC-FR-032` |

**Widget.** `usage` answers `More: "/readings"` (`GWPACC-FR-034`).

### 3.4 Errors

| Condition | Answer |
| --------- | ------ |
| No assigned customer | 409 `/problems/dev.maroid.gwp/not-assigned` |
| `from` is not a date `YYYY-MM-DD` | 400 `/problems/http/request-invalid`, naming `from` |
| `limit`, `cursor`, or `sort` that `RES-006` refuses | 400, as `libs/rest/page` answers |
| No reading of the assigned customer | 200 with an empty `items` |

## 4. Design decisions

### `GWPACC-DD-011`

**Realizes:** `GWPACC-FR-028`, `GWPACC-FR-029`
**Decision:** The order is `reading_date DESC, id DESC`, fixed. The boundary of a cursor is
the identifier of a row, and the condition compares `(reading_date, id)` with the one of
that row, as `maintenance_entries` of jasmine does.
The query computes the usage with `LAG(value) OVER (PARTITION BY serial_number ORDER BY
reading_date)` over every reading of the customer, then applies `from` and the keyset, so
the first reading after `from` still counts from the one before it.
**Rationale:** Two meters share a date, so the date alone cannot order the page. The
identifier breaks the tie, and `page.Read` keys a cursor on it.
**Alternatives:** A `sort` parameter. No requirement asks for a second order.

### `GWPACC-DD-012`

**Realizes:** `GWPACC-FR-030`, `GWPACC-FR-033`
**Decision:** The chart reads the same route with `from` set to the day 12 months before
today and `limit=100`, and follows `next` until the last page.
**Rationale:** One route serves the list and the chart. The deck knows the date of the
person, so the page sets the bound. One meter fills 12 rows of one page.
**Alternatives:** A route of series. It repeats the read of the list for one view.

### `GWPACC-DD-013`

**Realizes:** `GWPACC-FR-030`
**Decision:** `UsageChart.svelte` draws the chart as SVG markup. `d3-scale` gives the scales
and `d3-shape` the path of each line, and D3 touches no element. The x axis is a band for
each reading date, with an inner band for each meter. Each meter shows its usage as a
column on the left axis, which starts at zero and reaches below it for a negative usage,
and its value as a line on the right axis. A reading with no usage draws no column. The
width follows the container. A pointer over a date shows a daisyUI tooltip with the
usage and the value of each meter on that date. The SVG carries `role="img"` and a
label, and the table below it is the accessible form of the same readings. The chart
stays in the plugin until a second plugin needs one.
**Rationale:** The owner chose D3 over a library of charts. The two modules are small, and
the releases are stable: `d3-scale` 4.0.2 dates from 2021 and `d3-shape` 3.2.0 from 2022.
The chart is plain markup, so it takes the components and the classes of daisyUI.
**Alternatives:** ApexCharts. It is not free for a company above 2 million US dollars of
revenue, its core is 146 KB gzipped, and its colors need a conversion. LayerChart. Four
releases in one week. Chart.js. It draws on a canvas, which reads no CSS variable.

### `GWPACC-DD-014`

**Realizes:** `GWPACC-FR-030`
**Decision:** Each meter takes the classes of one daisyUI color, in the order `primary`,
`secondary`, `accent`, `info`: `fill-<color>` for its columns and `stroke-<color>` for its
line. The axes take `fill-base-content`, and the grid `stroke-base-300`. The source names
each class in full, so Tailwind finds it.
**Rationale:** A class resolves the variable of the theme in the browser, so a change of
the theme, by the deck or by the scheme of the system, recolors the chart with no code.
**Alternatives:** Colors that a script reads from the theme. It needs a listener for each
way the theme changes.

### `GWPACC-DD-015`

**Realizes:** `GWPACC-FR-031`, `GWPACC-FR-032`
**Decision:** The route answers 409 `not-assigned` when the workspace holds no assigned
customer, and an empty page when it holds no reading. The page shows the alert with the
link to `/account` on the 409, and "No reading yet." on an empty first page.
**Rationale:** The client tells the two states apart by the status, with no second call.
**Alternatives:** An empty page in both cases. The page then cannot ask for the setup.

## 5. Scenarios

`spec-scenarios.md` holds `GWPACC-SC-025` to `GWPACC-SC-031`.

## Retired identifiers

This file has no retired identifier.
