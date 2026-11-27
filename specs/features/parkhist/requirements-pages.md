---
id: PARKHIST
title: The pages of the parking history
type: requirements
status: approved
created: 2026-11-27
updated: 2026-11-27
approved_by: Temuri
approved_on: 2026-11-27
constrained_by: [UI, SEC, OWN]
---

# Requirements: The pages of the parking history

`requirements.md` holds the problem, the collection, the stored data, and the invariants.
This file holds what a member reads: the overview page, the sessions page, the
transactions page, and the change to the balance widget. `PARKHIST-INV-001` limits each of them to
the data of the acting user. Only the connection page and the figures of `PARKHIST-FR-022` call the
parking service.

## Functional requirements

### The widget

### `PARKHIST-FR-021`

The balance widget must link to the overview page when Maroid holds a connection for the acting user.

**Examples:**

- Normal case: a connected member presses the balance on the home page. The deck opens the overview page.
- Limit case: Maroid holds no connection. The widget answers "not set up" with a link to the connection page, as before.

### The overview page

### `PARKHIST-FR-022`

The overview page must show the balance and the count of the free parkings left today, as the connection page reads them at the time of the view.

**Why:** Both change with each session. The connection page already reads them from the parking service.

**Examples:**

- Normal case: the parking service reports 9.08 GEL and 4. The overview shows 9.08 GEL and 4.

### `PARKHIST-FR-023`

The overview page must report the parking service as unavailable in place of the figures of `PARKHIST-FR-022`, and show the stored data, when the parking service fails.

**Why:** An outage at the parking service is no reason to hide the history that Maroid holds.

### `PARKHIST-FR-024`

The overview page must report "not connected", with a link to the connection page, in place of the figures of `PARKHIST-FR-022`, when Maroid holds no connection for the acting user.

**Examples:**

- Normal case: the connection ended after 30 days. The overview shows "not connected" and the stored data.

### `PARKHIST-FR-025`

The overview page must show a chart of the sum of the charges of each month within the chosen period, as columns, and of the balance at the end of each month, as a line.

**Examples:**

- Normal case: the period is the last 12 months. The chart holds 12 columns and the balance of each month.
- Limit case: a month holds no transaction. The month holds no column and no point, and keeps its place on the axis.

### `PARKHIST-FR-026`

A member must choose the period of the overview page: the last 12 months, the last 3 years, or every month that Maroid holds. The last 12 months is the default.

### `PARKHIST-FR-027`

The overview page must show, for the chosen period: the count of the sessions, the sum of their durations, and the sum of their amounts.

### `PARKHIST-FR-028`

The overview page must list the 5 parking lots with the most sessions in the chosen period, each with the count of its sessions and the sum of their amounts.

**Examples:**

- Limit case: two parking lots hold the same count. The one with the later last session comes first.
- Limit case: the period holds sessions on 3 parking lots. The list holds 3.

### The sessions page

### `PARKHIST-FR-029`

The sessions page must list the sessions of the acting user, the latest start first.

### `PARKHIST-FR-030`

Each row of the sessions page must show the date, the start, the end, the duration, the number and the address of the parking lot, the plate, the amount, and a mark for a free parking.

### `PARKHIST-FR-031`

The sessions page must show at most 20 sessions at a time, with a way to the next and the previous sessions.

### `PARKHIST-FR-032`

A member must filter the sessions page by a first date and a last date of the start, each one included, in the time zone of the acting user.

**Examples:**

- Normal case: from 2026-11-01 to 2026-11-30. The page lists the sessions that started in November.
- Unwanted case: the first date is after the last date. The page reports the period as invalid, and lists nothing.

### `PARKHIST-FR-033`

A member must filter the sessions page by one plate that a stored session of the acting user names.

**Examples:**

- Unwanted case: the address names a plate of no stored session. The page lists no session.

### `PARKHIST-FR-034`

A member must filter the sessions page to the free parkings alone, or to the sessions that are not free parkings alone.

### `PARKHIST-FR-035`

The sessions page must show, for every session that the filter matches, not only the shown 20: the count, the sum of the durations, and the sum of the amounts.

### The transactions page

### `PARKHIST-FR-036`

The transactions page must list the transactions of the acting user, the latest moment first.

### `PARKHIST-FR-037`

Each row of the transactions page must show the moment, the kind, the amount, the balance after the transaction, and the parking lot of the linked session.

**Examples:**

- Limit case: a top up links no session. The row shows no parking lot.

### `PARKHIST-FR-038`

The transactions page must show a top up as an addition to the balance, and a charge as a subtraction. It must show the amount of another kind with no direction.

### `PARKHIST-FR-039`

The transactions page must show at most 20 transactions at a time, with a way to the next and the previous transactions.

### `PARKHIST-FR-040`

A member must filter the transactions page by one kind of `PARKHIST-FR-015`.

### `PARKHIST-FR-041`

A member must filter the transactions page by a first date and a last date of the moment, each one included, in the time zone of the acting user.

### `PARKHIST-FR-042`

The transactions page must show, for every transaction that the filter matches: the sum of the top ups and the sum of the charges.

### All pages

### `PARKHIST-FR-043`

Each page must report "no data yet", with a link to the connection page, when Maroid holds none of the data that the page shows for the acting user.

**Why:** A page does not ask the parking service whether a connection works. The same message serves a member who never connected and a member who waits for the first collection.

**Examples:**

- Normal case: the member connected 10 minutes ago. The sessions page reports "no data yet", and names the connection page and the next collection.
- Limit case: Maroid holds transactions and no session. The transactions page lists the transactions. The sessions page reports "no data yet".

### `PARKHIST-FR-044`

Each page must read the data that Maroid holds, with no call to the parking service, except the figures of `PARKHIST-FR-022` on the overview page.

**Why:** A view of a page costs no call, and shows the data when the parking service is unavailable or the connection ended.

### Permissions

### `PARKHIST-FR-045`

A member must hold a permission whose lowest role is viewer to read the stored data of the overview page, the sessions page, and the transactions page.

**Why:** The data belongs to the member, as the connection does (`PARKACC-FR-023`). A separate permission lets an owner withhold the history and grant the balance.

## Non-functional requirements

### `PARKHIST-NFR-001`

The route of the sessions page and the route of the transactions page must each answer in less than 300 milliseconds for a user who holds 5,000 rows of it, with any filter. Measure at the hub, from the request to the last byte of the answer, on the database of the developer machine.

**Why:** A list with a sum over every match must not grow slower with the history of a person.

## Retired identifiers

This file has no retired identifier.
