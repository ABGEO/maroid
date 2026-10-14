---
id: ELECACC
title: The widget and the pages of the electricity account
type: requirements
status: approved
created: 2026-10-14
updated: 2026-10-14
approved_by: Temuri
approved_on: 2026-10-14
constrained_by: [UI, SEC]
---

# Requirements: The widget and the pages of the electricity account

`requirements.md` holds the problem, the connection, the collection, the bill, the
permissions, and the invariants. This file holds what a member reads, because one file
passed the size that `LNG-013` gives.

## Functional requirements

### The widget

### `ELECACC-FR-032`

The plugin must declare a widget that shows the latest reading of the assigned customer: its date, its usage, its meter value, and its charge.

### `ELECACC-FR-033`

The widget must answer "not set up", with a link to the connection page, when the workspace holds no assigned customer.

### `ELECACC-FR-034`

The widget must answer "empty" when Maroid holds no reading of the assigned customer.

### `ELECACC-FR-035`

The widget must read the transactions that Maroid holds, with no call to the electricity service.

### `ELECACC-FR-036`

The widget must link to the transactions page.

### The transactions page

### `ELECACC-FR-037`

The transactions page must list the transactions of the assigned customer, the latest date first.
Each row shows the date, the name of the operation, the amount, and the balance.
A charge also shows its usage, its meter value, and its tariff.

### `ELECACC-FR-038`

The transactions page must show at most 20 transactions at a time, with a way to the next and the previous transactions.

### `ELECACC-FR-039`

The transactions page must show a chart of the readings dated within the last 12 months:
the usage of each reading as a column, and its charge as a line.

**Examples:**

- Normal case: 12 readings and 12 payments. The chart shows 12 columns and one line of 12 points.
- Limit case: the two charges of 2026-04-28. The chart shows one column of 166.5 kWh.
- Limit case: a reading dated 13 months ago. The list shows it. The chart does not.

### The bills page

### `ELECACC-FR-040`

The bills page must list the bills of the assigned customer, the latest month first.
Each row shows the month, the bill number, and the date of issue.

### `ELECACC-FR-041`

The bills page must show at most 20 bills at a time, with a way to the next and the previous bills.

### `ELECACC-FR-042`

A member must download the document of a bill from the bills page.

**Why:** The electricity service serves a bill only inside a session, so a link to it fails.

**Examples:**

- Normal case: the member clicks the bill of September 2026. The browser receives the document.
- Unwanted case: the electricity service fails. The member receives an answer that reports it unavailable.

### Both pages

### `ELECACC-FR-043`

The transactions page and the bills page must ask for the setup, with a link to the connection page, when the workspace holds no assigned customer.

### `ELECACC-FR-044`

The transactions page must report "no transaction", and the bills page "no bill", when Maroid holds none of the assigned customer.

### `ELECACC-FR-045`

The list, the chart, and the list of bills must read the data that Maroid holds, with no call to the electricity service.

## Retired identifiers

This file has no retired identifier.
