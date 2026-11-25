---
id: DRIVVO
title: The scenarios of the pages and the widget of the car log account
type: spec
status: approved
created: 2026-11-25
updated: 2026-11-26
approved_by: Temuri
approved_on: 2026-11-25
constrained_by: [TST, TRC]
requirements: features/drivvo/requirements-pages.md
---

# Specification: The scenarios of the pages and the widget of the car log account

`spec-scenarios.md` holds the setting of every scenario: the fake service, the members, the
vehicles, the car log settings, and now.

## `DRIVVO-SC-029`

**Verifies:** `DRIVVO-FR-027` to `DRIVVO-FR-029`, `DRIVVO-DD-012`
**Layer:** unit

**Given** the vehicles "Zeta" (active, not principal), "Hatchback" (active, principal), and "Old
sedan" (inactive).
**When** V reads `GET /vehicles`, then the test unmarks the principal vehicle and reads again.
**Then** the order is "Hatchback", "Zeta", "Old sedan", and `default` marks "Hatchback". The second
answer marks "Hatchback" as `default`, the first active by name.

## `DRIVVO-SC-030`

**Verifies:** `DRIVVO-FR-029`, `DRIVVO-FR-030`
**Layer:** manual

**Given** the deck with the plugin.
**When** V opens the navigation entry "Overview", reloads the page, and picks "Old sedan".
**Then** the address reads `/overview/<Hatchback>`, the reload shows "Hatchback", and the
choice moves the address to `/overview/<Old sedan>`.

## `DRIVVO-SC-031`

**Verifies:** `DRIVVO-FR-031`
**Layer:** integration

**Given** an identifier that names no vehicle of W.
**When** V reads its overview, its entries, and its reminders.
**Then** each answer is 404 `vehicle-not-found`.

## `DRIVVO-SC-032`

**Verifies:** `DRIVVO-FR-032`, `DRIVVO-FR-066`
**Layer:** unit

**Given** a copy with no vehicle.
**When** V reads `GET /vehicles` and the widget.
**Then** the list is empty, and the widget answers not set up with the link `/account`.

## `DRIVVO-SC-033`

**Verifies:** `DRIVVO-FR-033`
**Layer:** unit

**Given** a copy of one vehicle, and a fake service that fails every call.
**When** V reads every route under `/vehicles`.
**Then** each answer is 200, and the fake service records no call.

## `DRIVVO-SC-034`

**Verifies:** `DRIVVO-FR-034`
**Layer:** integration

**Given** the settings of the account.
**When** V reads `GET /vehicles` and an entry.
**Then** "Hatchback" answers `units` `km`, `l`, and `volume_per_100`, and the amount answers
`{"amount": 90.0, "currency": "GEL"}`.

## `DRIVVO-SC-035`

**Verifies:** `DRIVVO-FR-035`, `DRIVVO-FR-036`
**Layer:** manual

**Given** V with the time zone `Europe/Berlin`, and a refuel at 2026-11-24T14:30:00Z.
**When** V opens the history, then the overview.
**Then** the history shows the refuel at 15:30 on 2026-11-24 and the period all time. The overview
opens on 2025-12-01 to 2026-11-25.

## `DRIVVO-SC-036`

**Verifies:** `DRIVVO-FR-036`, `DRIVVO-DD-013`
**Layer:** unit

**Given** a refuel at 2026-10-31T21:30:00Z, which is 1 November in Tbilisi.
**When** V reads the overview with `from=2026-11-01`, then with `from=2026-11-30&to=2026-11-01`.
**Then** the first answer counts the refuel in November. The second is 400 `request-invalid`.

## `DRIVVO-SC-037`

**Verifies:** `DRIVVO-FR-038`, `DRIVVO-FR-041`
**Layer:** unit

**Given** a maintenance of 300.00 with an oil change of 100.00 and a filter of 200.00, in two
instalments of 150.00 due on 2026-01-15 and 2026-02-15.
**When** `figures.Spend` runs for January 2026.
**Then** January counts 50.00 for the oil change and 100.00 for the filter.

## `DRIVVO-SC-038`

**Verifies:** `DRIVVO-FR-039`, `DRIVVO-FR-040`, `DRIVVO-FR-042`
**Layer:** unit

**Given** in 2026: an insurance expense of 600.00 on 10 January in three instalments of 200.00 due in
January, February, and March; a refuel of 120.00 in March; an income of 50.00 in March; nothing in
April.
**When** V reads the overview from 2026-01-01 to 2026-04-30.
**Then** `spend` answers fuel 120.00, expense 600.00, and total 720.00. `income` is 50.00. Each of
January and February counts expense 200.00. March counts expense 200.00, fuel 120.00, and income
50.00. April counts zero.

## `DRIVVO-SC-039`

**Verifies:** `DRIVVO-FR-043`, `DRIVVO-FR-044`
**Layer:** unit

**Given** entries with odometers 10,000 and 13,000 in the period, a spend of 1,500.00 of which 900.00
is fuel, and in a second case one entry.
**When** V reads the overview in each case.
**Then** the first answers the distance 3000, fuel 0.30 per km, and total 0.50 per km. The second
answers null for the distance and both values per distance.

## `DRIVVO-SC-040`

**Verifies:** `DRIVVO-FR-045`, `DRIVVO-FR-046`, `DRIVVO-DD-010`
**Layer:** unit

**Given** a full refuel at 10,000 km, 30 L not full at 10,450 km, and 20 L full at 10,800 km.
**When** `figures.Consumption` runs with format 4, then with format 1, then with one full refuel alone.
**Then** it answers 6.25 with `volume_per_100`, then 16.0 with `distance_per_volume`, then null.

## `DRIVVO-SC-041`

**Verifies:** `DRIVVO-FR-047`, `DRIVVO-FR-051`
**Layer:** unit

**Given** the refuels of `DRIVVO-SC-040`, with `missed_refuel` on the second.
**When** V reads the refuel list.
**Then** the refuel at 10,800 km answers no consumption, and the overview answers null.

## `DRIVVO-SC-042`

**Verifies:** `DRIVVO-FR-048`, `DRIVVO-FR-049`, `DRIVVO-DD-015`
**Layer:** integration

**Given** one entry of each kind on five days.
**When** V reads the history, then with `kind=refuel,odometer`.
**Then** the first answer lists five items, the latest first, and the odometer entry holds no amount.
The second lists the refuel and the odometer entry.

## `DRIVVO-SC-043`

**Verifies:** `DRIVVO-FR-050`, `DRIVVO-FR-052` to `DRIVVO-FR-055`
**Layer:** integration

**Given** two refuels of 40 L for 120.00 and 30 L for 90.00, a maintenance at the place "Garage", and
an income of the type "Ride".
**When** V reads the four lists and the totals of each kind.
**Then** the refuel items answer volume, unit price, and full tank. The maintenance answers `place`
"Garage", and the income `income_type` "Ride". The refuel totals answer count 2, 210.00, and 70 L.

## `DRIVVO-SC-044`

**Verifies:** `DRIVVO-FR-056`
**Layer:** integration

**Given** 45 refuels.
**When** V reads the history with no `limit`, then follows `next` two times.
**Then** the pages hold 20, 20, and 5 items. The last page holds no `next`, and the first no `prev`.

## `DRIVVO-SC-045`

**Verifies:** `DRIVVO-FR-057`, `DRIVVO-FR-058`, `DRIVVO-DD-016`
**Layer:** integration

**Given** the refuel of `DRIVVO-SC-019` at the station "Gulf" paid by "Card", and an odometer entry.
**When** V reads the detail of each.
**Then** the refuel answers both lines, the station, the payment method, and the three values of the
charge. The odometer entry answers `odometer` and `note` alone.

## `DRIVVO-SC-046`

**Verifies:** `DRIVVO-FR-059`
**Layer:** integration

**Given** an entry of "Old sedan".
**When** V reads it under the path of "Hatchback", then under an identifier that the copy lacks.
**Then** each answer is 404 `log-entry-not-found`.

## `DRIVVO-SC-047`

**Verifies:** `DRIVVO-FR-060`, `DRIVVO-FR-061`, `DRIVVO-DD-017`
**Layer:** unit

**Given** a reminder that applies one time, for the expense type "Insurance" on 2026-12-07.
**When** V reads the reminders.
**Then** the item answers `type` "Insurance", `due_on` 2026-12-07, `days_left` 12, and `due` false.

## `DRIVVO-SC-048`

**Verifies:** `DRIVVO-FR-062`, `DRIVVO-DD-011`
**Layer:** unit

**Given** a reminder of the maintenance type "Oil change" each 10,000 km or each 12 months, the
latest oil change on 2026-03-01 at 150,000 km, and in a second case no oil change.
**When** `figures.DueReminders` runs.
**Then** the first answers 2027-03-01 and 160,000 km. The second answers the day and the odometer of
the reminder.

## `DRIVVO-SC-049`

**Verifies:** `DRIVVO-FR-063`
**Layer:** unit

**Given** the latest odometer 160,250 km, a reminder due at 160,000 km, and a second reminder due in
40 days or at 159,000 km.
**When** `figures.DueReminders` runs.
**Then** both are due. The first answers `distance_left` -250.

## `DRIVVO-SC-050`

**Verifies:** `DRIVVO-FR-064`
**Layer:** unit

**Given** one due reminder, one due in 5 days, one due in 30 days, and one with an odometer alone,
1,000 km left.
**When** V reads the reminders.
**Then** the order is the due one, 5 days, 30 days, then the one with an odometer alone.

## `DRIVVO-SC-051`

**Verifies:** `DRIVVO-FR-065`, `DRIVVO-FR-067`
**Layer:** unit

**Given** "Hatchback" with a refuel of 40 L for 120.00 on 2026-11-20, a consumption of 6.8, and an
oil change 1,200 km away.
**When** the widget loads.
**Then** it answers three items: "Latest refuel" with "2026-11-20 · 40.00 L · 120.00 GEL",
"Consumption" with "6.8 L/100 km", and "Oil change" with "in 1,200 km". `More` is the overview of
"Hatchback".

## `DRIVVO-SC-052`

**Verifies:** `DRIVVO-FR-065`
**Layer:** unit

**Given** "Hatchback" with refuels and no reminder, and in a second case with no entry and no reminder.
**When** the widget loads in each case.
**Then** the first answers two items, the refuel and the consumption. The second answers empty.

## Retired identifiers

This file has no retired identifier.
