---
id: DRIVVO
title: The widget and the pages of the car log account
type: requirements
status: approved
created: 2026-11-25
updated: 2026-11-26
approved_by: Temuri
approved_on: 2026-11-25
constrained_by: [UI, SEC]
---

# Requirements: The widget and the pages of the car log account

`requirements.md` holds the problem, the definitions, the connection, the collection, the
permissions, and the invariants. This file holds what a member reads. "A page" in this file
is the overview, the history, a list, the detail of a log entry, or the reminders page.

## Functional requirements

### The vehicle of a page

### `DRIVVO-FR-027`

A page must show one vehicle of the car log copy, and let a member choose any other vehicle of the car log copy.

### `DRIVVO-FR-028`

The choice of the vehicle must list the active vehicles first, then each inactive vehicle with a mark.

### `DRIVVO-FR-029`

A page opened with no vehicle in its address must show the principal vehicle.

**Examples:**

- Normal case: the account marks the hatchback as its principal vehicle. The overview opens on the hatchback.
- Limit case: the account marks no vehicle as principal. The page shows the first active vehicle by name.

### `DRIVVO-FR-030`

The address of a page must name the vehicle that the page shows.

**Why:** A reload, and a link that a member sends, open the same vehicle (`PLROUTE-FR-006`).

### `DRIVVO-FR-031`

A page must report the vehicle as not found, with a link to the page of the principal vehicle, when its address names a vehicle that the car log copy does not hold.

**Examples:**

- Unwanted case: a member deletes a vehicle in the mobile application. A saved link to its overview reports the vehicle as not found.

### `DRIVVO-FR-032`

A page must report that Maroid holds no vehicle, with a link to the connection page, when the car log copy holds no vehicle.

### `DRIVVO-FR-033`

A page must read the car log copy alone, with no call to the car log service.

### `DRIVVO-FR-034`

A page must show an amount in the currency of the car log settings, a distance in the distance unit of the vehicle, and a volume in the volume unit of the car log settings.

**Examples:**

- Normal case: the account sets GEL and litres, and the vehicle sets kilometres. A refuel shows "40.00 L, 120.00 GEL, 152,300 km".

### `DRIVVO-FR-035`

A page must show the moment of a log entry in the time zone of the member.

### `DRIVVO-FR-036`

A member must choose the period of the overview, the history, and each list: this month, the last 12 months, this year, all time, or the range between two days.
The default of the overview is the last 12 months. The default of the history and of each list is all time.

### The amounts

### `DRIVVO-FR-037`

The amount of a refuel, a maintenance, or an expense must be the sum of the amounts of its lines. A fuel line with the free mark counts zero.

### `DRIVVO-FR-038`

The spend of one maintenance type or one expense type must take a share of each instalment in proportion to the amount of its line.

**Examples:**

- Normal case: a maintenance of 300.00 GEL holds an oil change of 100.00 and a filter of 200.00, in two instalments of 150.00 due in January and in February. January counts 50.00 for the oil change and 100.00 for the filter.

### The overview

### `DRIVVO-FR-039`

The overview must show the spend of the period for fuel, for maintenance, and for expenses, and the sum of the three.

### `DRIVVO-FR-040`

The overview must show the spend of each calendar month of the period, for fuel, for maintenance, and for expenses.

**Examples:**

- Normal case: an insurance expense of 600.00 GEL on 2026-01-10 holds three instalments of 200.00, due in January, February, and March. Each of the three months counts 200.00.
- Limit case: a month with no spend shows zero.

### `DRIVVO-FR-041`

The overview must show the spend of the period for each maintenance type and for each expense type.

### `DRIVVO-FR-042`

The overview must show the income of the period, and the income of each calendar month of the period, apart from the spend.

### `DRIVVO-FR-043`

The overview must show the distance of the period: the highest odometer value minus the lowest odometer value of the log entries of the vehicle in the period.

**Examples:**

- Limit case: the period holds one log entry. The overview shows no distance.

### `DRIVVO-FR-044`

The overview must show the fuel spend per distance, and the spend per distance, of the period.

**Examples:**

- Normal case: a spend of 1,500.00 GEL, of which 900.00 is fuel, over 3,000 km. The overview shows 0.30 GEL/km for fuel and 0.50 GEL/km in all.
- Limit case: the period has no distance. The overview shows no value per distance.

### `DRIVVO-FR-045`

The overview must show the average consumption of the period, in the consumption format of the car log settings.

**Examples:**

- Normal case: a full tank at 10,000 km, 30 L with no full tank mark at 10,450 km, and 20 L to a full tank at 10,800 km. The consumption is 6.25 L/100 km, or 16.0 km/L.
- Limit case: the period holds one refuel with a full tank mark. The overview shows no consumption.

### `DRIVVO-FR-046`

The average consumption must count each part between two refuels with a full tank mark whose second refuel is in the period.

### `DRIVVO-FR-047`

The consumption must skip a part between two full tank marks that holds a refuel with the missed refuel mark.

**Examples:**

- Normal case: the second refuel of the example of `DRIVVO-FR-045` holds the missed refuel mark. The overview shows no consumption for that part.

### The history and the lists

### `DRIVVO-FR-048`

The history must list every log entry of the vehicle in the period, the latest moment first.
Each row shows the moment, the kind, the odometer value, the amount, and the types or the fuels that the log entry names.

**Examples:**

- Limit case: an odometer entry shows no amount.

### `DRIVVO-FR-049`

A member must filter the history by one or more kinds of log entry.

### `DRIVVO-FR-050`

The refuel list must list the refuels of the vehicle in the period, the latest moment first.
Each row shows the moment, the odometer value, the fuels, the volume, and the amount.

### `DRIVVO-FR-051`

A row of the refuel list must show the consumption of the part that the refuel ends, when `DRIVVO-FR-047` does not skip it.

### `DRIVVO-FR-052`

The maintenance list must list the maintenances of the vehicle in the period, the latest moment first.
Each row shows the moment, the odometer value, the title, the maintenance types, and the amount.

### `DRIVVO-FR-053`

The expense list must list the expenses of the vehicle in the period, the latest moment first.
Each row shows the moment, the odometer value, the title, the expense types, and the amount.

### `DRIVVO-FR-054`

The income list must list the incomes of the vehicle in the period, the latest moment first.
Each row shows the moment, the odometer value, the income type, and the amount.

### `DRIVVO-FR-055`

Each list must show the total amount of the period. The refuel list also shows the total volume.

### `DRIVVO-FR-056`

The history and each list must show at most 20 log entries at a time, with a way to the next and the previous log entries.

### The detail of a log entry

### `DRIVVO-FR-057`

A log entry in the history or in a list must open its detail under its row, on the same page.
The detail links to the page of the log entry.

**Examples:**

- Normal case: a click on a refuel shows its fuels, its station, and its note under the row. A second click closes it.
- Limit case: an odometer entry with no note shows that it holds no other value.

### `DRIVVO-FR-058`

The detail of a log entry must show every value that the car log copy holds of it:

| Kind           | Values                                                                                                                                                           |
| -------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Refuel         | Moment, odometer, each fuel line, amount, station, payment method, reason, missed refuel mark, start and end charge of the battery, minutes of the charge, note. |
| Maintenance    | Moment, odometer, title, each line, amount, place, payment method, each instalment, note.                                                                        |
| Expense        | Moment, odometer, title, each line, amount, place, payment method, reason, each instalment, note.                                                                |
| Income         | Moment, odometer, income type, amount, each instalment, note.                                                                                                    |
| Odometer entry | Moment, odometer, note.                                                                                                                                          |

### `DRIVVO-FR-059`

The address of the detail must name the log entry. The detail reports the log entry as not found when the car log copy does not hold it.

### The reminders page

### `DRIVVO-FR-060`

The reminders page must list each reminder of the vehicle with its maintenance type or expense type, its due point, and what is left: the days to the due day, and the distance to the due odometer value.

### `DRIVVO-FR-061`

The due point of a reminder that applies one time must be its own day and its own odometer value.

### `DRIVVO-FR-062`

The due point of a reminder that repeats must count from the latest log entry of the vehicle that names its type: the day of that log entry plus the interval of the reminder, and the odometer value of that log entry plus the distance of the reminder.

**Why:** The car log service shows the same due point. Two different due points mislead the member.

**Examples:**

- Normal case: an oil change each 10,000 km or each 12 months. The latest oil change is on 2026-03-01 at 150,000 km. The due point is 2027-03-01 or 160,000 km.
- Limit case: no log entry names the type. The due point is the day and the odometer value of the reminder.

### `DRIVVO-FR-063`

A reminder must show as due when the day in the time zone of the member reaches its due day, or when the latest odometer reaches its due odometer value.

**Examples:**

- Normal case: the due point is 160,000 km, and the latest odometer is 160,250 km. The reminder is due, 250 km past.
- Limit case: the due day is in 40 days, and the due odometer value is passed. The reminder is due.

### `DRIVVO-FR-064`

The reminders page must list the due reminders first, then the others by the days left, then the reminders with no due day by the distance left.

### The widget

### `DRIVVO-FR-065`

The plugin must declare a widget for the principal vehicle that shows the day, the volume, and the amount of the latest refuel, the average consumption of the last 12 months, and the first reminder in the order of `DRIVVO-FR-064` with what is left.

**Examples:**

- Normal case: "Refuel 2026-11-20: 40.00 L, 120.00 GEL. 6.8 L/100 km. Oil change in 1,200 km."
- Limit case: the vehicle holds no reminder. The widget shows the refuel and the consumption alone.

### `DRIVVO-FR-066`

The widget must answer "not set up", with a link to the connection page, when the car log copy holds no vehicle.

### `DRIVVO-FR-067`

The widget must link to the overview of the principal vehicle.

### `DRIVVO-FR-068`

The widget must show the latest odometer and the spend of the current month of the principal vehicle.

**Examples:**

- Normal case: "Odometer 128,450 km. This month 312.40 GEL."
- Limit case: the current month holds no refuel, no maintenance, and no expense. The widget leaves the spend out.

## Retired identifiers

This file has no retired identifier.
