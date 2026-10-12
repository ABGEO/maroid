---
id: GWPACC
title: The readings page of the water account
type: spec
status: approved
created: 2026-10-12
updated: 2026-10-12
approved_by: Temuri
approved_on: 2026-10-12
constrained_by: [PLG, OWN, RES, API, ERR, UI, TS, GO, BLD, TST]
requirements: features/gwpacc/requirements.md
---

# Specification: The readings page of the water account

## 1. Summary

The plugin gains the route `GET /readings`, a keyset page of the readings of the assigned
customer, the latest date first, with the filter `from`. The remote gains the page
`/readings`: a daisyUI table with the shared `Pagination`, and a mixed ApexCharts chart of
the last 12 months that takes its colors from the daisyUI theme. The widget links to the
page. The page and the route read only the database.

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
| `plugins/gwp/repository/store.go` | change | `ReadingsPage(ctx, asked page.Request, from *time.Time) (page.Window[model.Reading], error)`. It reads the assignment and the page in one transaction, and answers `errs.ErrNotAssigned` |
| `plugins/gwp/dto/reading.go` | create | `ReadingResponse`, `NewReadingResponseList` |
| `plugins/gwp/handler/reading.go` | create | `ReadingHandler`, `NewReadingHandler`, `ReadingLister` |
| `plugins/gwp/handler/problem.go` | change | `typeNotAssigned` |
| `plugins/gwp/widget/usage.go` | change | `ListData.More` is `/readings` |
| `plugins/gwp/main.go` | change | `Routes` adds the readings route. `UIManifest` adds `/readings` |
| `plugins/gwp/ui/package.json` | change | `apexcharts` at exactly `7.6.1` |
| `plugins/gwp/ui/src/api/{reading,types,index}.ts` | create, change | `Reading`, `ReadingPage`, `createReadingApi` with `page(cursorLink?)` and `since(from)` |
| `plugins/gwp/ui/src/lib/chart.ts` | create | `usageChartOptions(readings, palette)`, `themePalette()` |
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
**Decision:** The chart is an ApexCharts `line` chart with mixed series: for each meter, a
`column` series of the usage on the left axis and a `line` series of the value on the
right axis. The x axis is the reading date. ApexCharts is pinned at `7.6.1`, and the page
imports it in a dynamic `import()`.
**Rationale:** daisyUI Charts builds on ApexCharts, so the chart reads as the gallery
does. Its example code is a paid product, so the plugin writes its own options. The
dynamic import keeps the chart out of the bundle of the account page.
**Alternatives:** A chart in CSS alone. It cannot draw two axes. Chart.js. It is not what daisyUI Charts uses.

### `GWPACC-DD-014`

**Realizes:** `GWPACC-FR-030`
**Decision:** `themePalette` reads `--color-primary`, `--color-secondary`, `--color-accent`,
`--color-info`, `--color-base-content`, and `--color-base-300` with `getComputedStyle`,
converts each to `rgb()` through a canvas, and the chart takes them as its colors, its
text, and its grid. A `MutationObserver` on the `data-theme` attribute of the root
rebuilds the chart when the theme changes.
**Rationale:** ApexCharts writes a color into an SVG attribute, where `var()` does not
resolve, and its color code reads no `oklch()`, which daisyUI 5 gives. A theme switch of
the deck changes the variables, so the chart follows it.
**Alternatives:** Fixed colors. The chart then breaks the dark theme.

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
