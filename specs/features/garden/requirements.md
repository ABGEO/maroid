---
id: GARDEN
title: The plants of a workspace, their environments, and their care
type: requirements
status: approved
created: 2026-10-10
updated: 2026-10-10
approved_by: Temuri
approved_on: 2026-10-10
constrained_by: [OWN, DAT, SEC, NTF, JOB, CFG, RES]
---

# Requirements: The plants of a workspace, their environments, and their care

## 1. Problem

A person who keeps plants records each care action, and wants to know which plant
needs care before it suffers. Nothing tells them when a plant is due.

The plugin holds environments and plants today, with no workspace. Every person sees
every plant. It holds no care history and no schedule.

## 2. Users

- An editor: adds the plants, sets their schedules, and records each care action.
- A viewer: reads the plants, their history, and what is due.

## 3. Out of scope

- Photos of a plant.
- Charts and statistics of the care history.
- A bot command that records a care action.
- An import of records from another system.
- A notification preference for each workspace or each person.
- A link between two plants: a parent, a cutting, or a merge.

## 4. Definitions

| Term                | Meaning                                                                                              |
| ------------------- | ---------------------------------------------------------------------------------------------------- |
| environment         | A place in a workspace where plants live, for example "Office".                                      |
| plant               | One plant that a workspace keeps. It lives in one environment.                                       |
| display identifier  | The short name that a person reads for a plant or an environment: `P001`, `E001`.                    |
| maintenance type    | One kind of care action. `GARDEN-FR-011` gives the list.                                             |
| maintenance entry   | The record of one care action on one plant: a maintenance type, a time, and a note.                  |
| care interval       | The number of days between two care actions of one maintenance type, for one plant.                  |
| care state          | The state of one care interval of one plant: on schedule, due, or overdue.                           |
| reminder            | The one notification for one workspace that lists every care interval that is due or overdue.       |

## 5. Functional requirements

### Environments and plants

### `GARDEN-FR-001`

An editor must create, rename, and delete an environment. An environment holds a name of 1 to 100 characters.

### `GARDEN-FR-002`

Maroid must refuse to delete an environment that holds a plant, active or inactive.

**Why:** A plant always lives in one environment, and its history stays readable.

### `GARDEN-FR-003`

An editor must create a plant with these values:

| Value           | Rule                                       |
| --------------- | ------------------------------------------ |
| Common name     | Required. 1 to 100 characters.             |
| Scientific name | Optional. 1 to 150 characters.             |
| Environment     | Required. An environment of the workspace. |
| Acquired date   | Optional. A date that is not in the future. |
| Care notes      | Optional. 10000 characters maximum.        |

### `GARDEN-FR-004`

An editor must change each value of `GARDEN-FR-003`. A change of the environment keeps the display identifier and the history.

### `GARDEN-FR-005`

A plant must be active or inactive. Maroid must make a new plant active. An editor must change the state in both directions.

**Why:** A plant that dies or leaves keeps its history, and gets no reminder.

### `GARDEN-FR-006`

An editor must delete a plant. The delete removes its care intervals and its maintenance entries.

**Why:** A plant entered by mistake leaves no trace.

### Display identifiers

### `GARDEN-FR-007`

Maroid must give each new environment the display identifier `E` followed by the next number of the workspace, and each new plant `P` followed by the next number.

**Examples:**

- Normal case: the workspace holds `P001` to `P014`. The next plant gets `P015`.
- Limit case: the number has fewer than three digits. Maroid pads it with zeros: `P007`.
- Limit case: after `P999`, the next plant gets `P1000`.
- Normal case: a second workspace creates its first plant. It gets `P001`.

### `GARDEN-FR-008`

Maroid must never give a display identifier a second time in one workspace, also after a delete.

**Examples:**

- Unwanted case: the editor deletes `P015`, the highest. The next plant gets `P016`, not `P015`.

**Why:** A note or a photo outside Maroid that names `P015` names one plant forever.

### `GARDEN-FR-009`

Maroid must refuse a change to a display identifier.

### `GARDEN-FR-010`

Every page, list, and reminder that shows a plant or an environment must show its display identifier.

### Maintenance

### `GARDEN-FR-011`

The maintenance types must be exactly these: watering, fertilizing, pruning, repotting, pest treatment, cleaning, misting, other.

### `GARDEN-FR-012`

An editor must record a maintenance entry for one plant, with a maintenance type, a time, and a note of 0 to 1000 characters. The time is the current time unless the editor gives another.

**Examples:**

- Normal case: the editor records watering on `P003`, with no note, at the current time.
- Limit case: the editor records a watering of yesterday at 19:00.
- Unwanted case: the time is in the future. Maroid refuses the entry.

### `GARDEN-FR-013`

An editor must record one maintenance entry for each plant of a selection of 1 to 100 plants, with one maintenance type, one time, and one note. Maroid records all entries or none.

**Examples:**

- Normal case: the editor selects every plant of "Office" and records watering.
- Unwanted case: one selected plant was deleted a moment before. Maroid records no entry and names the plant.

### `GARDEN-FR-014`

An editor must record a maintenance entry from a line of the attention list, in one action. The entry takes the plant and the maintenance type of the line, and the current time.

### `GARDEN-FR-015`

An editor must change the maintenance type, the time, and the note of a maintenance entry.

### `GARDEN-FR-016`

An editor must delete a maintenance entry.

### `GARDEN-FR-017`

Maroid must accept a maintenance entry for an inactive plant.

**Why:** The owner records the last action on a plant after a merge or a loss.

### Care schedule

### `GARDEN-FR-018`

An editor must set a care interval of 1 to 365 days for each maintenance type of a plant, and remove it.

**Examples:**

- Normal case: `P001` gets watering every 7 days and fertilizing every 30 days.
- Unwanted case: an interval of 0 days. Maroid refuses it.

### `GARDEN-FR-019`

Maroid must count the days of a care interval as calendar days in UTC, from the date of the latest maintenance entry of that type.

**Why:** A watering at 19:00 and a check at 09:00 seven days later count as seven days.

### `GARDEN-FR-020`

When a plant has no maintenance entry of a type, Maroid must count from the date that the editor set the care interval.

**Why:** A new plant with a full schedule does not fill the next reminder.

### `GARDEN-FR-021`

Maroid must give each care interval of an active plant one care state:

| Care state  | Days since the latest entry |
| ----------- | --------------------------- |
| On schedule | Less than the interval.     |
| Due         | The interval to 1.5 times the interval. |
| Overdue     | More than 1.5 times the interval. |

**Examples:**

- Normal case: watering every 7 days, the latest 7 days ago. The state is due.
- Limit case: watering every 7 days, the latest 10 days ago. The state is due. At 11 days it is overdue.
- Limit case: every 1 day, the latest 1 day ago. Due. At 2 days, overdue.

### Display

### `GARDEN-FR-022`

A viewer must read these views:

| View              | Content                                                                                                  |
| ----------------- | -------------------------------------------------------------------------------------------------------- |
| Environment list  | Each environment with its count of active plants.                                                        |
| Plant list        | Each plant with its environment and state. A filter by environment and by state. Active plants by default. |
| Plant page        | The values of the plant, each care interval with its care state, and its maintenance entries, newest first. |
| Attention list    | Each care interval that is due or overdue, grouped by environment, overdue first.                       |
| Maintenance log   | The maintenance entries of the workspace, newest first. A filter by plant and by maintenance type.       |

### `GARDEN-FR-023`

Each line of the attention list must show the plant, the maintenance type, the days since the latest entry, the care interval, and the care state.

### `GARDEN-FR-024`

A view must show each time and each date in the time zone of the browser.

### Reminder

### `GARDEN-FR-025`

The configuration of the installation must hold the schedule of the reminder, and the notification channel.

**Why:** A preference for each workspace comes with a later feature.

### `GARDEN-FR-026`

At each run of the schedule, Maroid must send one reminder for each workspace that enables the plugin and holds a due or overdue care interval.

**Examples:**

- Normal case: two workspaces have due plants. The channel receives two reminders.
- Limit case: no care interval is due. Maroid sends nothing for that workspace.

### `GARDEN-FR-032`

A reminder must hold each line of `GARDEN-FR-023`, grouped by environment.

### `GARDEN-FR-028`

When the channel fails, Maroid must record the failure and send the reminder at the next run only.

### Change

### `GARDEN-FR-031`

The upgrade must remove every environment and plant that exists before this feature.

**Why:** They hold test data only, with no workspace.

## 6. Non-functional requirements

### `GARDEN-NFR-001`

The attention list must answer in less than 1 second, for a workspace with 500 plants and 100000 maintenance entries. Measure at the hub, from the request to the last byte of the answer.

### `GARDEN-NFR-002`

The reminder must reach the channel in less than 60 seconds after the scheduled time, for 50 workspaces of 500 plants each. Measure from the scheduled time to the call to the channel.

## 7. Invariants

### `GARDEN-INV-001`

One display identifier names at most one environment or plant of a workspace, at all times and across all deletes.

### `GARDEN-INV-002`

A plant, its environment, its care intervals, and its maintenance entries are in one workspace.

### `GARDEN-INV-003`

A plant holds at most one care interval for each maintenance type.

## 8. Constraints from the guidelines

| Rule      | Guideline     | Effect on this feature |
| --------- | ------------- | ---------------------- |
| `OWN-004` | Ownership     | Every record of this feature is scoped to a workspace. |
| `OWN-009` | Ownership     | The reminder is a job for each workspace. |
| `DAT-009` | Data          | The internal identifier is a UUID version 7. The display identifier is a second value. |
| `SEC-013` | Security      | The plugin declares a read and a write permission for maintenance. Read is viewer, write is editor. |
| `NTF-002` | Notifications | The reminder names a channel, not a transport. |
| `CFG-007` | Configuration | The channel in the installation configuration is a temporary exception. See question 3. |
| `JOB-006` | Jobs          | A slow reminder run does not start a second run. |
| `RES-005` | REST          | Each list answers a page. |

## 9. Open questions

| #   | Question | Owner | Answer |
| --- | -------- | ----- | ------ |
| 1   | `GARDEN-FR-019` needs a time zone. Is one time zone in the plugin configuration correct, with UTC as the default? | Owner | No. Maroid stores and counts in UTC. A view shows the time zone of the browser. |
| 2   | `CFG-007` puts a value of one workspace in the database. One channel for every workspace is a value of the installation. Is that the reading you accept until the notification rework? | Owner | Yes, until the notification rework. |

## Retired identifiers

| ID              | Retired    | Reason                                                              |
| --------------- | ---------- | ------------------------------------------------------------------- |
| GARDEN-FR-027   | 2026-10-10 | The reminder names no workspace. `GARDEN-FR-032` keeps its content. |
| GARDEN-FR-029   | 2026-10-10 | The measurements left the scope of this feature.                    |
| GARDEN-FR-030   | 2026-10-10 | The measurements left the scope of this feature.                    |
