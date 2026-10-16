---
id: GARDEN
title: The care of the next three days
type: requirements
status: approved
created: 2026-10-16
updated: 2026-10-16
approved_by: Temuri
approved_on: 2026-10-16
constrained_by: [OWN, RES]
---

# Requirements: The care of the next three days

`requirements.md` holds the plants, the care schedule, the attention list, the reminder,
and the bot command. This file holds the upcoming list and the links of the view of the
attention list, because one file passed the size that `LNG-013` gives.

## Definitions

| Term          | Meaning                                                                                                                         |
| ------------- | ------------------------------------------------------------------------------------------------------------------------------- |
| due date      | The date on which a care interval becomes due: the date that `GARDEN-FR-019` or `GARDEN-FR-020` counts from, plus the interval. |
| upcoming list | Each care interval of an active plant whose due date is 1 to 3 days after the current UTC date.                                 |

## Functional requirements

### `GARDEN-FR-039`

The view of the attention list must show the upcoming list after the attention list.

**Why:** A person plans the care of the next days, for example before a trip.

**Examples:**

- Normal case: watering every 7 days, the latest 6 days ago. The due date is tomorrow. The line is in the upcoming list.
- Limit case: the latest 4 days ago. The due date is in 3 days. The line is in the upcoming list. At 3 days ago, it is not.
- Unwanted case: the latest 7 days ago. The care state is due. The line is in the attention list only.
- Unwanted case: the plant is inactive. No list holds the line.

### `GARDEN-FR-040`

Maroid must sort the upcoming list by due date, earliest first, then by the display identifier of the environment, then by the display identifier of the plant, then by the order of `GARDEN-FR-011`.

**Examples:**

- Normal case: `P009` is due tomorrow and `P002` in 2 days. `P009` comes first.
- Limit case: `P002` and `P009` of one environment are due tomorrow. `P002` comes first.

### `GARDEN-FR-041`

Each line of the upcoming list must show the plant, the maintenance type, the care interval, and the due date as "tomorrow", "in 2 days", or "in 3 days".

### `GARDEN-FR-042`

A line of the upcoming list must offer no action that records a maintenance entry.

**Why:** An editor who acts early records the entry from the plant page.

### `GARDEN-FR-043`

When the upcoming list is empty, the view must show "No care in the next 3 days."

### `GARDEN-FR-044`

Each line of the attention list and of the upcoming list must link to the plant page of its plant.

**Why:** A person who reads a line checks the plant, its notes, and its history before the care.

## Non-functional requirements

### `GARDEN-NFR-004`

The upcoming list must answer in less than 1 second, for a workspace with 500 plants and 100000 maintenance entries. Measure at the hub, from the request to the last byte of the answer.

## Constraints from the guidelines

| Rule      | Guideline | Effect on this feature                                    |
| --------- | --------- | --------------------------------------------------------- |
| `OWN-004` | Ownership | The upcoming list holds the care of the acting workspace. |
| `RES-005` | REST      | The upcoming list answers a page.                         |
