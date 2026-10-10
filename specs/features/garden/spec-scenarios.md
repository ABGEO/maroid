---
id: GARDEN
title: The scenarios of the plants, their environments, and their care
type: spec
status: approved
created: 2026-10-10
updated: 2026-10-10
approved_by: Temuri
approved_on: 2026-10-10
constrained_by: [TST, OWN, TRC]
requirements: features/garden/requirements.md
---

# Specification: The scenarios of the plants, their environments, and their care

`spec.md` holds the design. This file holds the scenarios. `SPC-001` divides them.

Every integration scenario runs against the container that `testdb.Start` gives,
with the migrations of the hub and of jasmine applied. Workspaces H and G enable
jasmine. A scenario acts in H unless it says otherwise. A route scenario calls the
handler with the acting workspace in the context, and the permission check of the
hub is not part of it. "Today" is the UTC date of the database at the run.

## Environments and plants

### `GARDEN-SC-001`

**Verifies:** `GARDEN-FR-001`
**Layer:** integration

**Given** H holds no environment.
**When** a client posts the name "Office", renames it to "Study", then posts a name of 101 characters.
**Then** the first two calls answer 201 and 200 with the new name. The third answers 422 with the pointer `#/name`.

### `GARDEN-SC-002`

**Verifies:** `GARDEN-FR-002`
**Layer:** integration

**Given** `E001` holds one inactive plant.
**When** a client deletes `E001`.
**Then** the answer is 409 with the type `/problems/dev.maroid.jasmine/environment-occupied`, and `E001` remains. After the delete of the plant, the delete of `E001` answers 204.

### `GARDEN-SC-003`

**Verifies:** `GARDEN-FR-003`
**Layer:** integration

**Given** `E001`.
**When** a client posts a plant with the common name "Gardenia", the scientific name "Gardenia jasminoides", `E001`, the acquired date of today, and care notes of 10000 characters.
**Then** the answer is 201 and carries every value, `status` `active`, and `environment.display_id` `E001`.

### `GARDEN-SC-004`

**Verifies:** `GARDEN-FR-003`
**Layer:** integration

**Given** `E001` in H, and `E001` in G.
**When** a client posts four plants: one acquired tomorrow, one acquired the day after tomorrow, one with a scientific name of 151 characters, and one with the environment of G.
**Then** the first answers 201. The others answer 422 with the pointers `#/acquired_on`, `#/scientific_name`, and `#/environment_id`.

### `GARDEN-SC-005`

**Verifies:** `GARDEN-FR-004`
**Layer:** integration

**Given** `P001` in `E001` with two maintenance entries, and `E002`.
**When** a client puts `P001` with the environment `E002` and a new common name.
**Then** the answer carries `display_id` `P001`, `E002`, and the new name. `GET /maintenance-entries?plant_id=` answers the two entries.

### `GARDEN-SC-006`

**Verifies:** `GARDEN-FR-005`
**Layer:** integration

**Given** a new plant `P001`.
**When** a client puts `status` `inactive`, then `active`, then `dormant`.
**Then** the plant answers `active` after the create, `inactive` and `active` after the first two puts. The third answers 422 with the pointer `#/status`.

### `GARDEN-SC-007`

**Verifies:** `GARDEN-FR-006`
**Layer:** integration

**Given** `P001` with two care intervals and three entries.
**When** a client deletes `P001`.
**Then** the answer is 204. The tables `care_intervals` and `maintenance_entries` hold no row for `P001`.

## Display identifiers

### `GARDEN-SC-008`

**Verifies:** `GARDEN-FR-007`
**Layer:** unit

**Given** `FormatDisplayID`.
**When** it formats `('P', 7)`, `('P', 15)`, `('E', 999)`, and `('P', 1000)`.
**Then** it answers `P007`, `P015`, `E999`, and `P1000`.

### `GARDEN-SC-009`

**Verifies:** `GARDEN-FR-007`
**Layer:** integration

**Given** H holds `E001` and `P001` to `P014`, and G holds nothing.
**When** a client creates a plant in H, an environment in H, and a plant in G.
**Then** they get `P015`, `E002`, and `P001`.

### `GARDEN-SC-010`

**Verifies:** `GARDEN-FR-008`, `GARDEN-INV-001`
**Layer:** integration

**Given** H holds `P001` to `P015`.
**When** a client deletes `P015` and creates a plant. Then 20 goroutines each create one plant at the same moment.
**Then** the first create gets `P016`. The 20 creates get 20 distinct numbers from `P017` to `P036`. An insert of a second row with the number 16 fails on the unique constraint.

### `GARDEN-SC-011`

**Verifies:** `GARDEN-FR-009`
**Layer:** integration

**Given** `P001`.
**When** a client puts `P001` with every value and `"display_id": "P099"`.
**Then** the answer is 400 with the type `/problems/http/member-unknown`, and `P001` keeps its values.

## Maintenance

### `GARDEN-SC-012`

**Verifies:** `GARDEN-FR-011`
**Layer:** integration

**Given** `P001`.
**When** a client posts one entry for each of the eight types, then one with the type `repot`.
**Then** the eight answer 201. The ninth answers 422 with the pointer `#/type`.

### `GARDEN-SC-013`

**Verifies:** `GARDEN-FR-012`
**Layer:** integration

**Given** `P003`.
**When** a client posts `{"plant_ids": ["<P003>"], "type": "watering"}`.
**Then** the answer is 201 with one item. Its `performed_at` lies between the times before and after the call, and its `note` is absent.

### `GARDEN-SC-014`

**Verifies:** `GARDEN-FR-012`
**Layer:** integration

**Given** `P003`.
**When** a client posts an entry at yesterday 19:00 UTC, one 1 minute after the current time, and one with a note of 1001 characters.
**Then** the first answers 201. The second answers 422 with the pointer `#/performed_at`. The third answers 422 with the pointer `#/note`.

### `GARDEN-SC-015`

**Verifies:** `GARDEN-FR-013`
**Layer:** integration

**Given** `E001` holds 13 plants.
**When** a client posts one watering for the 13 plants with the note "Bottom watering".
**Then** the answer is 201 with 13 items in the order of `plant_ids`, each with the same `performed_at` and note.

### `GARDEN-SC-016`

**Verifies:** `GARDEN-FR-013`
**Layer:** integration

**Given** `P001` to `P003`, and an identifier that names no plant of H.
**When** a client posts the three plants and that identifier at index 3. Then 101 identifiers. Then `P001` two times.
**Then** the first answers 422 with the pointer `#/plant_ids/3`, and no entry lands. The second and the third answer 422 with the pointer `#/plant_ids`.

### `GARDEN-SC-017`

**Verifies:** `GARDEN-FR-014`
**Layer:** manual

**Given** the page `/attention` shows watering for `P003` as due.
**When** the owner clicks "Done" on that line.
**Then** the line leaves the list after one click. The page `/maintenance` shows a watering of `P003` at the current time.

### `GARDEN-SC-018`

**Verifies:** `GARDEN-FR-015`, `GARDEN-FR-016`
**Layer:** integration

**Given** watering every 7 days on `P001` with `set_at` 3 days ago, and one watering entry of today.
**When** a client puts the entry with the type `misting`, a time of 9 days ago, and a note. Then a client deletes it.
**Then** after the put, the entry carries the three values, and the watering interval answers `days_since` 3. After the delete, `GET` of the entry answers 404.

### `GARDEN-SC-019`

**Verifies:** `GARDEN-FR-017`
**Layer:** integration

**Given** `P007` is `inactive`.
**When** a client posts a repotting of `P007` with the note "merged with P008".
**Then** the answer is 201.

## Care schedule

### `GARDEN-SC-020`

**Verifies:** `GARDEN-FR-018`, `GARDEN-INV-003`
**Layer:** integration

**Given** `P001`.
**When** a client puts `watering` 7 days, `fertilizing` 30 days, `watering` 10 days, `pruning` 0 days, `pruning` 366 days, and deletes `fertilizing`.
**Then** the puts of 7, 30, and 10 answer 200. The two pruning puts answer 422 with the pointer `#/days`. The plant holds one care interval: `watering`, 10 days.

### `GARDEN-SC-021`

**Verifies:** `GARDEN-FR-019`
**Layer:** integration

**Given** watering every 7 days, and the latest watering at 23:30 UTC, 7 calendar days before today.
**When** the state query runs at any time of today.
**Then** `days_since` is 7 and `counted_from` is the date of that watering.

### `GARDEN-SC-022`

**Verifies:** `GARDEN-FR-020`
**Layer:** integration

**Given** `P001` has no entry. Its watering interval of 7 days has `set_at` 8 days ago.
**When** a client puts `watering` 7 days again. Then a client puts `watering` 14 days.
**Then** after the first put, `days_since` is 8 and the state is `due`. After the second, `days_since` is 0 and the state is `on_schedule`.

### `GARDEN-SC-023`

**Verifies:** `GARDEN-FR-021`
**Layer:** integration

**Given** watering every 7 days on an active plant.
**When** the latest watering is 6, 7, 10, and 11 days ago.
**Then** the state is `on_schedule`, `due`, `due`, and `overdue`.

### `GARDEN-SC-024`

**Verifies:** `GARDEN-FR-021`
**Layer:** integration

**Given** misting every 1 day.
**When** the latest misting is 1 day ago, then 2 days ago. Then the plant becomes `inactive`.
**Then** the state is `due`, then `overdue`, then null. `GET /care-intervals?state=due,overdue` omits the inactive plant.

## Display

### `GARDEN-SC-025`

**Verifies:** `GARDEN-FR-022`
**Layer:** integration

**Given** `E001` holds two active plants and one inactive plant. `E002` holds one active plant.
**When** a client reads `/environments`, `/plants`, `/plants?status=inactive`, and `/plants?environment_id=<E002>`.
**Then** `E001` carries `active_plant_count` 2 and `E002` 1. The plant reads answer 3, 1, and 1 plants. A cursor of the second read is stale on the fourth.

### `GARDEN-SC-026`

**Verifies:** `GARDEN-FR-023`
**Layer:** integration

**Given** `P003` "Gardenia" in `E001` "Office", watering every 7 days, the latest 9 days ago.
**When** a client reads `/care-intervals?state=due,overdue`.
**Then** the item carries `plant.display_id` `P003`, `plant.common_name`, `plant.environment.display_id` `E001`, `type` `watering`, `days_since` 9, `days` 7, and `state` `due`.

## Reminder

### `GARDEN-SC-027`

**Verifies:** `GARDEN-FR-026`, `GARDEN-FR-032`
**Layer:** integration

**Given** a fake dispatcher. In H: `P003` "Gardenia" in `E001` "Office", watering overdue at 11 of 7 days; `P001` "Rose" in `E001`, watering due at 7 of 7 days; `P004` "Fern" in `E002` "Kitchen", misting due at 2 of 2 days. In G: one plant with a due care interval.
**When** `CareReminder.Run` runs for H, then for G.
**Then** the dispatcher receives two messages on the configured channel. The body of the first is:

```
E001 Office
P003 Gardenia: watering, 11 days, every 7 days, overdue
P001 Rose: watering, 7 days, every 7 days, due

E002 Kitchen
P004 Fern: misting, 2 days, every 2 days, due
```

### `GARDEN-SC-028`

**Verifies:** `GARDEN-FR-026`
**Layer:** integration

**Given** a workspace with every care interval `on_schedule`. Then a configuration with an empty channel.
**When** `CareReminder.Run` runs. Then `CronJobs()` runs.
**Then** the dispatcher receives nothing, and `Run` answers nil. `CronJobs()` answers no job.

### `GARDEN-SC-029`

**Verifies:** `GARDEN-FR-028`
**Layer:** integration

**Given** a due care interval, and a dispatcher that fails the first call.
**When** `Run` runs two times.
**Then** the first answers an error that wraps the error of the dispatcher. The second sends the same message.

## Isolation

### `GARDEN-SC-030`

**Verifies:** `GARDEN-INV-002`
**Layer:** integration

**Given** `E001` in G, and the acting workspace H.
**When** a statement inserts a plant that names the identifier of the `E001` of G.
**Then** the insert fails on the composite foreign key.

### `GARDEN-SC-031`

**Verifies:** `GARDEN-FR-010`, `GARDEN-FR-022`, `GARDEN-FR-024`
**Layer:** manual

**Given** the browser runs at UTC+4, and an entry at 22:30 UTC yesterday.
**When** the owner opens the four pages.
**Then** every plant and every environment shows its display identifier. The maintenance log shows the entry at 02:30 today.

## Measurements

### `GARDEN-SC-032`

**Verifies:** `GARDEN-NFR-001`
**Layer:** integration

**Given** H holds 500 active plants in 10 environments, 4000 care intervals, and 100000 entries spread over 365 days.
**When** a client reads `/care-intervals?state=due,overdue&limit=100` 20 times, and the test keeps the slowest call.
**Then** the slowest call answers in less than 1 second, from the call of the handler to the last byte.

### `GARDEN-SC-033`

**Verifies:** `GARDEN-NFR-002`
**Layer:** integration

**Given** 50 workspaces, each with the data of `GARDEN-SC-032`, and a fake dispatcher that waits 300 ms for each send.
**When** `Run` runs one time for each workspace, one after the other, as `runForEachWorkspace` does.
**Then** the last send ends less than 60 seconds after the first run starts.

## Retired identifiers

This file has no retired identifier.
