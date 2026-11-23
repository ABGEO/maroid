---
id: PULSE
title: The check-ins of a person, the widget of today
type: requirements
status: approved
created: 2026-11-24
updated: 2026-11-24
approved_by: Temuri
approved_on: 2026-11-24
constrained_by: [OWN, UI]
---

# Requirements: The widget of today

[`requirements.md`](requirements.md) holds the problem and the definitions. The
`WIDGET` feature holds the kinds, the sizes, and the states of a widget.

## 4. Definitions

| Term     | Meaning                                                                                       |
| -------- | --------------------------------------------------------------------------------------------- |
| today    | The local day of the person at the moment of the read, in the time zone of the schedule.      |
| upcoming | An occurrence of today, later than the moment of the read, that the schedule will still fire. |

## 5. Functional requirements

### `PULSE-FR-038`

Pulse must declare a `list` widget of size `m`, with the title "Today", under the
permission that every page of pulse names.

**Why:** The person sees on the home page which check-ins of the day they answered,
and which ones still wait, with no click.

### `PULSE-FR-039`

The widget must show one item for each round of the person whose occurrence falls on
today, whatever its state.

**Examples:**

- Normal case: at 14:00, Morning of 09:00 is done and Midday of 13:00 was declined.
  The widget shows both.
- Limit case: Evening fired at 22:00 yesterday and expired today at 09:00. Its
  occurrence falls on yesterday, so the widget does not show it.
- Limit case: Nina deletes the check-in Midday after its round of 13:00. The item
  keeps the name Midday.
- Limit case: at 10:30 in `Asia/Tbilisi`, Morning of 09:00 is done. Nina changes the
  time zone to `America/Chicago`, where it is 00:30. Morning fell at 23:00 of the day
  before there, so the widget does not show it.

### `PULSE-FR-040`

The widget must show one item for each upcoming occurrence.

**Examples:**

- Normal case: at 14:00, Afternoon of 17:00 and Evening of 22:00 show as upcoming.
- Limit case: on a Saturday, Weekly of Sunday 18:00 does not show.
- Limit case: Nina pauses pulse until 18:00 today. Afternoon of 17:00 does not show.
  Evening of 22:00 shows.
- Unwanted case: Nina turns off the check-in Evening. Evening of 22:00 does not show.

### `PULSE-FR-041`

The widget must not show an occurrence of today that is earlier than the moment of
the read and fired no round.

**Why:** Pulse holds no record of it. A check-in added after its time, a pause, and a
hub that was down leave the same trace, so a label "missed" would be wrong for two of
the three.

**Examples:**

- Normal case: at 10:00 Nina adds a check-in at 09:00. It does not show today.
- Limit case: Midday of 13:00 is due and has not fired yet at 13:01. Midday does not
  show until its round exists.

### `PULSE-FR-042`

The widget must order its items by the moment of the occurrence, then by the name of
the check-in.

**Example:** Midday of 13:00 and Water of 13:00 show after Morning of 09:00, Midday
first.

### `PULSE-FR-043`

Each item must show the name of the check-in as its label, and the local time of the
occurrence with its state as its meta line.

The state reads "Upcoming", "Waiting to start", "In progress", "Done", "Declined", or
"Expired", as on the history page.

**Example:** "Morning", with "09:00 · Done".

### `PULSE-FR-044`

An item of a round in progress must show the count of the answered questions and the
count of the questions of the round in place of its state.

**Why:** A round in progress waits for the person. The count tells how much is left.

**Examples:**

- Normal case: Evening holds 7 questions, and Nina answered 3. The meta line reads
  "22:00 · 3 of 7 answered".
- Limit case: Nina skipped one question and answered two. The meta line reads
  "2 of 7 answered".

### `PULSE-FR-045`

Each item must link to the history page, narrowed to its check-in.

**Examples:**

- Normal case: the item Morning opens the history page with the filter Morning.
- Limit case: the item of an upcoming occurrence links the same way, though no round
  of it exists yet.
- Limit case: the check-in of a round was deleted. The item opens the history page
  with no filter.

### `PULSE-FR-046`

The widget must link to the history page to see all its items.

### `PULSE-FR-047`

The widget must answer "not set up" while pulse is off, with a message that asks the
person to turn pulse on, and a link to the preferences page.

**Example:** Beka never turned pulse on. The widget shows "Turn pulse on to receive
check-ins." and a link to the preferences page.

### `PULSE-FR-048`

The widget must answer "empty" with the end of the pause, in the time zone of the
schedule, while pulse is paused and no item applies.

**Examples:**

- Normal case: Nina paused pulse at 08:00 until 2026-12-01. The widget shows
  "Check-ins paused until 1 Dec 2026, 00:00.", the text of `/pause`.
- Limit case: Morning of 09:00 is done, and then Nina pauses pulse. The widget shows
  Morning, and no message.

### `PULSE-FR-049`

The widget must answer "empty" with the message "No check-in today." while pulse is
on, not paused, and no item applies.

**Example:** Nina turned off every check-in. The widget shows "No check-in today."

## 8. Constraints from the guidelines

| Rule      | Guideline | Effect on this feature                                                           |
| --------- | --------- | -------------------------------------------------------------------------------- |
| `OWN-003` | Ownership | The widget loads as the acting user, so each member reads their own day.         |
| `OWN-008` | Ownership | The policy of each table narrows the read to the acting user, not the widget.    |
| `UI-005`  | Web UI    | Each link of the widget is a route of the remote of pulse: history, preferences. |

## Retired identifiers

This file has no retired identifier.
