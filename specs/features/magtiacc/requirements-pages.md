---
id: MAGTIACC
title: The widget and the pages of the telecom account
type: requirements
status: approved
created: 2026-11-24
updated: 2026-11-24
approved_by: Temuri
approved_on: 2026-11-24
constrained_by: [UI, SEC]
---

# Requirements: The widget and the pages of the telecom account

`requirements.md` holds the problem, the definitions, the connection, the collection, the
permissions, and the invariants. This file holds what a member reads.

## Functional requirements

### The subscriber page

### `MAGTIACC-FR-032`

The subscriber page must show the state of an assigned mobile subscriber: the status, the name and the price of the plan, the date of the next renewal, the balance, and the allowances left of data, call minutes, and SMS.

**Examples:**

- Normal case: status active, plan price 17.30 lari, renewal on 2026-12-13, balance 0.03 lari, 2589.29 MB, 56:16 minutes, and 100 SMS left.
- Limit case: the subscriber holds no active plan. The page shows the status and the balance, and says "no active plan".

### `MAGTIACC-FR-033`

The subscriber page must show the state of an assigned home line: the status, the price of the plan, the days left to the next renewal, the balance, and the speed of the internet.

**Examples:**

- Normal case: status active, plan price 40.00 lari, 20 days left, balance 27.70 lari, 70 Mbps.

### `MAGTIACC-FR-034`

The subscriber page and the widget must read the subscriber state from the telecom service at each read.

**Why:** The balance changes at each top up, and a copy of one day before misleads.

### `MAGTIACC-FR-035`

The subscriber page must ask for the setup, with a link to the connection page, when the workspace holds no connection or no assigned subscriber.

### `MAGTIACC-FR-036`

The subscriber page must report the telecom service as unavailable when the telecom service fails.

### The widget

### `MAGTIACC-FR-037`

The plugin must declare a widget that shows the balance of the assigned subscriber and the days left to its next renewal. For a mobile subscriber, it also shows the data left.

**Examples:**

- Normal case: a mobile subscriber. The widget shows 0.03 ₾, with "2589 MB left, renews in 19 days".
- Normal case: a home line. The widget shows 27.70 ₾, with "40.00 ₾ plan, renews in 20 days".
- Unwanted case: the telecom service fails. The widget shows the error of `WIDGET-FR-020`.

### `MAGTIACC-FR-038`

The widget must answer "not set up", with a link to the connection page, when the workspace holds no connection or no assigned subscriber.

### `MAGTIACC-FR-039`

The widget must link to the subscriber page.

### The traffic page

### `MAGTIACC-FR-040`

The traffic page must show a chart of the data of each day of the period, in megabytes.

### `MAGTIACC-FR-041`

The traffic page must show a chart of the call minutes in and the call minutes out of each day of the period.

### `MAGTIACC-FR-042`

The traffic page must show a chart of the SMS sent and the SMS received of each day of the period.

### `MAGTIACC-FR-043`

A member must choose the period of the traffic page: the last 30 days, the last 90 days, or the last 365 days. The default is the last 30 days.

**Examples:**

- Normal case: the member opens the page on 2026-11-25. The charts show 2026-10-26 to 2026-11-24.
- Limit case: Maroid holds no traffic of 2026-11-02. The charts show no value on that day, and no zero.

### `MAGTIACC-FR-044`

The traffic page must show the totals of the period: the data, the call minutes in, the call minutes out, the SMS sent, and the SMS received.

### The money page

### `MAGTIACC-FR-045`

The money page must list the money entries of the assigned subscriber, the latest moment first.
Each row shows the date and the time, the kind, the direction, and the amount.

**Examples:**

- Normal case: a top up of 17.30 lari on 2026-11-13 at 01:50, and a charge of 17.30 lari for the plan one second later.

### `MAGTIACC-FR-046`

The money page must show at most 20 money entries at a time, with a way to the next and the previous money entries.

### `MAGTIACC-FR-047`

The money page must show the total of the top ups and the total of the charges of each of the last 12 calendar months.

**Examples:**

- Normal case: November 2026 holds three top ups of 2.00, 17.30, and 4.90 lari. The total of the top ups is 24.20 lari.
- Limit case: a month with no money entry shows zero for both totals.

### Both pages

### `MAGTIACC-FR-048`

The traffic page and the money page must ask for the setup, with a link to the connection page, when the workspace holds no assigned subscriber.

### `MAGTIACC-FR-049`

The traffic page and the money page must report that the telecom service keeps no traffic and no money entries of a home line, when the assigned subscriber is a home line.

### `MAGTIACC-FR-050`

The traffic page must report "no traffic", and the money page "no money entry", when Maroid holds none of the assigned subscriber.

### `MAGTIACC-FR-051`

The charts, the totals, and the list must read the data that Maroid holds, with no call to the telecom service.

## Retired identifiers

This file has no retired identifier.
