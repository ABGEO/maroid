---
id: DRIVVO
title: The widget and the pages of the car log account
type: spec
status: approved
created: 2026-11-25
updated: 2026-11-26
approved_by: Temuri
approved_on: 2026-11-25
constrained_by: [UI, RES, SEC]
requirements: features/drivvo/requirements-pages.md
---

# Specification: The widget and the pages of the car log account

`spec.md` holds the components, the data model, the declarations, and the computation of the
figures. This file holds the design decisions of the routes that the pages read, the pages, and
the widget.

## Design decisions

### `DRIVVO-DD-012`

**Realizes:** `DRIVVO-FR-027` to `DRIVVO-FR-032`
**Decision:** `GET /vehicles` answers every vehicle of the copy as one page with no `limit` and no
`cursor`, ordered active first, then by `name`, then by `id`. An item answers `id`, `name`, `make`,
`model`, `year`, `active`, `principal`, `default`, and `units` (`distance`, `volume`,
`consumption_shape`). `default` marks the one vehicle that a page with no vehicle opens: the
principal vehicle, else the first active vehicle, else the first vehicle. A navigation entry page
reads the list and calls `host.navigate` to `/<page>/<default id>`. With an empty list it
shows "Maroid holds no vehicle yet. The collection runs at the start of each hour." and a link to
`/account`. Every page with a vehicle shows a select of the list, with " (inactive)" after an
inactive name, and a choice navigates to the same page of the chosen vehicle. A page with a vehicle
calls `host.setTitle` with the name of the vehicle, so the trail reads "Overview" then "Hatchback". A route under
`/vehicles/{vehicleId}` that names no vehicle of the copy answers 404 `vehicle-not-found`, and the
page shows "This vehicle is not in Maroid." with a link to `/overview`.
**Rationale:** The server owns the rule of the default vehicle, so the widget and the pages agree
and a unit test covers it. An account holds few vehicles, so one page answers them all. The address
names the vehicle, so a reload keeps it (`PLROUTE-FR-006`).
**Alternatives:** The page picks the principal vehicle. The widget then repeats the rule in Go.

### `DRIVVO-DD-013`

**Realizes:** `DRIVVO-FR-033` to `DRIVVO-FR-036`
**Decision:** A route that takes a period reads `from` and `to`, each optional, as days
`YYYY-MM-DD`. The period runs from the start of `from` to the end of `to` in
`pluginapi.ActingTimeZone(ctx)`. An absent `from` or `to` leaves that end open. A page computes the
days of a choice in `host.user.timeZone`, else in the zone of the browser: this month is the first
day of the month to today; the last 12 months is the first day of the month 11 months back to
today; this year is 1 January to today; all time sends neither; a range sends the two days that the
member picks. The overview opens on the last 12 months, the history and the lists on all time. A
page keeps the choice while it stays mounted. Every amount travels as the money object with the
currency of the settings. A page labels a distance and a volume with the `units` of the vehicle.
A page writes each moment with the time helpers of `@maroid/plugin-sdk`, in the zone of the member.
**Rationale:** The member and the server read one calendar, so a month on the chart is the month the
member lives in (`RES-001`). The address holds no query (`PLROUTE`, out of scope), so the period
does not survive a reload.
**Alternatives:** Moments in the query. Each client then computes the months of the server again.

### `DRIVVO-DD-014`

**Realizes:** `DRIVVO-FR-039` to `DRIVVO-FR-047`
**Decision:** `GET /vehicles/{vehicleId}/overview` answers `from` and `to` as applied, `spend` with
`fuel`, `maintenance`, `expense`, and `total`; `income`; `months`, one item for each calendar month
of the period with `month` (`YYYY-MM`), `fuel`, `maintenance`, `expense`, and `income`; `types`, one
item for each maintenance type and expense type with spend, with `kind`, `name`, and `amount`, the
largest first; `distance`; `cost_per_distance` with `fuel` and `total`; and `consumption` with
`value` and `shape`. With all time, `months` runs from the month of the first entry to the current
month. A value that `DRIVVO-DD-009` or `DRIVVO-DD-010` cannot give is null. `Overview.svelte` shows
one card for each of the total spend, the income, the distance, the two values per distance, and the
consumption; a `ColumnLineChart` of `@maroid/ui` with a column series for each kind of spend and a
line for the income; and a table of the types.
**Rationale:** One read answers one page. The months answer a zero, so the chart keeps every slot.
**Alternatives:** One route for each card. The page then reads the rows of the vehicle six times.

### `DRIVVO-DD-015`

**Realizes:** `DRIVVO-FR-048` to `DRIVVO-FR-056`
**Decision:** `GET /vehicles/{vehicleId}/log-entries` takes `kind`, a comma-separated list of the
five kinds, default every kind; `from`; `to`; `limit` (1 to 100, default 20); `cursor`; and `sort`,
which takes `occurred_at` with the default `-occurred_at`. A tie sorts by `id` in the same
direction, and the cursor of `RES-006` keys on both and carries `kind`, `from`, and `to`. An item answers `id`, `kind`, `occurred_at`,
`odometer`, `amount` (absent for an odometer entry), `title`, `labels` (the names of the fuels or
the types, in the order of the lines), and for a refuel `volume`, `unit_price` of the first line,
`full_tank`, and `consumption` of the part that the refuel ends; for a maintenance `place`; for an
income `income_type`. `GET /vehicles/{vehicleId}/log-entry-totals` takes one required `kind`,
`from`, and `to`, and answers `kind`, `count`, `amount`, and for a refuel `volume`. `History.svelte`
shows a toggle for each kind, and a table with the columns of `DRIVVO-FR-048`. `EntryList.svelte`
takes the kind from its route, shows the totals above the table, and the columns of
`DRIVVO-FR-050` to `DRIVVO-FR-054`. Each table follows `next` and `prev`. A row opens
`/<page>/<vehicle>/<id>`, under the page that lists it.
**Rationale:** One route serves the history and the four lists. A page carries no total (`RES-005`),
so the totals of a list take a route of their own.
**Alternatives:** One route for each kind. Five routes repeat one query.

### `DRIVVO-DD-016`

**Realizes:** `DRIVVO-FR-057` to `DRIVVO-FR-059`
**Decision:** `GET /vehicles/{vehicleId}/log-entries/{logEntryId}` answers `id`, `kind`,
`occurred_at`, `odometer`, `note`, and each member of the table of `DRIVVO-FR-058` that the kind
holds: `amount`, `title`, `station`, `place`, `payment_method`, `reason`, and `income_type` as names;
`missed_refuel`; `battery_start_pct`, `battery_end_pct`, and `charge_minutes` when stored; `lines`
with `position`, `item`, `amount`, and for a fuel line `unit_price`, `volume`, `full_tank`, and
`free`; and `instalments` with `number`, `due_on`, and `amount`. An entry of another vehicle answers
404 `log-entry-not-found`. `Entry.svelte` calls `host.setTitle` with the kind and the day, and shows
a link back to the page that lists it.
**Rationale:** A member that a kind does not hold stays absent, as in the telecom plugin
(`MAGTIACC-DD-013`).
**Alternatives:** One object with every member, null when absent. The page then checks the kind and
the null.

### `DRIVVO-DD-017`

**Realizes:** `DRIVVO-FR-060` to `DRIVVO-FR-067`
**Decision:** `GET /vehicles/{vehicleId}/reminders` answers one page with no `limit`, in the order of
`DRIVVO-DD-011`. An item answers `id`, `kind`, `type`, `repeats`, `note`, `due`, and when known
`due_on`, `due_odometer`, `days_left`, and `distance_left`. A value past due is negative.
`Reminders.svelte` marks a due reminder, and writes "in 12 days", "in 1,200 km", or "250 km past".
The widget `car` reads the vehicle of `default`, and answers a `ListAnswer` with up to three items:

| Label           | Meta                                                   | Link              |
| --------------- | ------------------------------------------------------ | ----------------- |
| "Latest refuel" | `<day> · <volume> <unit> · <amount> <currency>`        | `/refuels/<id>`   |
| "Consumption"   | `<value, one decimal> <unit>`, over the last 12 months | `/overview/<id>`  |
| The type name   | "due", "in <n> days", or "in <n> <distance unit>"      | `/reminders/<id>` |

An absent value drops its item. `More` is `/overview/<id>`. No item answers
`EmptyAnswer("No refuel and no reminder yet.")`. No vehicle answers
`NotSetUpAnswer("Connect the Drivvo account.", "/account")`. The day is the day of the member.
**Rationale:** Each line has its own link, so a list widget fits three facts in size `s`.
**Alternatives:** A stat widget with the consumption as the value. The refuel and the reminder then
share one caption line.

## Retired identifiers

This file has no retired identifier.
