---
id: PENSION
title: The widget and the pages of the pension account
type: requirements
status: approved
created: 2026-11-27
updated: 2026-11-27
approved_by: Temuri
approved_on: 2026-11-27
constrained_by: [UI, SEC, OWN]
---

# Requirements: The widget and the pages of the pension account

`requirements.md` holds the problem, the connection, the collection, and the invariants.
This file holds what a member reads: the widget, the overview page, the contributions page,
and the portfolios page. `PENSION-INV-002` limits each of them to the data of the acting user.

## Functional requirements

### The widget

### `PENSION-FR-027`

The plugin must declare a widget that shows the asset value of the account summary, the gain, and the date of the collection that stored the summary.

**Examples:**

- Normal case: the summary holds 12,480.55 GEL and a gain of 1,904.10 GEL, collected on 2026-11-01. The widget shows "12,480.55 ₾" with "Gain 1,904.10 ₾, 01/11/2026".

### `PENSION-FR-028`

The widget must answer "not set up", with a link to the connection page, when Maroid holds no connection and no account summary for the acting user.

### `PENSION-FR-029`

The widget must answer "empty" when Maroid holds a connection and no account summary for the acting user.

**Why:** A member who connected today waits for the next collection (`PENSION-FR-008`). The
widget names the wait, not a setup.

### `PENSION-FR-030`

The widget must show the stored account summary when the acting user ended the connection.

**Why:** Maroid keeps the collected data after the end (`PENSION-FR-012`).

### `PENSION-FR-031`

The widget must link to the overview page.

### The overview page

### `PENSION-FR-032`

The overview page must show the account summary of the acting user: the asset value, the gain, the withheld amount, the sum of the contributions of each source, and the sum of all sources.

### `PENSION-FR-033`

The overview page must show the date of the collection that stored the account summary.

### `PENSION-FR-034`

The overview page must show a chart of the asset values within the chosen period, as a line, and of the sum of the contributions of each month, as columns.

**Examples:**

- Normal case: the period is the last 12 months. The chart holds 12 columns and the asset values of those months.
- Limit case: a month holds no contribution. The month holds no column, and keeps its place on the axis.
- Limit case: the period holds one asset value. The line is one point.

### `PENSION-FR-035`

A member must choose the period of the chart: the last 12 months, the last 3 years, or every month that Maroid holds. The last 12 months is the default.

### The contributions page

### `PENSION-FR-036`

The contributions page must list the contributions of the acting user, the latest date first.

### `PENSION-FR-037`

Each row of the contributions page must show the date, the employer, the source, the type, the month of the salary, the gross salary, and the amount.

**Examples:**

- Limit case: a contribution names no employer. The row shows no employer, and shows the other values.

### `PENSION-FR-038`

The contributions page must show at most 20 contributions at a time, with a way to the next and the previous contributions.

### `PENSION-FR-039`

A member must filter the contributions page by one employer that a contribution of the acting user names.

**Examples:**

- Normal case: the member chooses one of four employers. The page lists the contributions of that employer alone.
- Unwanted case: the address names an employer of no contribution of the acting user. The page lists no contribution.

### `PENSION-FR-040`

A member must filter the contributions page by a first date and a last date, each one included.

**Examples:**

- Normal case: from 2026-01-01 to 2026-03-31. The page lists the contributions dated in the first quarter.
- Unwanted case: the first date is after the last date. The page reports the period as invalid, and lists nothing.

### `PENSION-FR-041`

The contributions page must show the sum of the amounts of every contribution that the filter matches, not only of the shown 20.

### The portfolios page

### `PENSION-FR-042`

The portfolios page must list the portfolio shares of the acting user. Each row shows the portfolio, the asset manager, the ticker, the risk tier, the chosen percentage, the current percentage, the amount, the units, and the unit price.

### `PENSION-FR-043`

The portfolios page must show a chart of the unit price of each portfolio of the shares within the chosen period, with the period choices of `PENSION-FR-035`.

**Examples:**

- Normal case: the period is the last 3 years. The chart holds one line for each portfolio.
- Limit case: Maroid holds no unit price of a portfolio in the period. The chart holds no line for it, and the legend names it.

### All pages

### `PENSION-FR-044`

The overview page and the portfolios page must ask for the setup, with a link to the connection page, when Maroid holds no connection and no account summary for the acting user.

### `PENSION-FR-045`

Each page must report "no data yet" when Maroid holds a connection and none of the data that the page shows.

**Examples:**

- Normal case: the member connected today. The overview page reports that the first collection runs on the first day of the next month.

### `PENSION-FR-046`

The widget and each page must read the data that Maroid holds, with no call to the pension service.

**Why:** A view of a page costs no call, and shows the data when the pension service is unavailable.

### Permissions

### `PENSION-FR-047`

A member must hold a permission whose lowest role is viewer to read the widget, the overview page, the contributions page, and the portfolios page.

**Why:** The data belongs to the member, as the connection does (`PENSION-FR-026`).

## Non-functional requirements

### `PENSION-NFR-001`

The route of the contributions page must answer in less than 300 milliseconds for a user who holds 2,000 contributions, with any filter. Measure at the hub, from the request to the last byte of the answer, on the database of the developer machine.

**Why:** A list with a sum over every match must not grow slower with the history of a person.

## Retired identifiers

This file has no retired identifier.
