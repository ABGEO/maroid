---
id: PULSE
title: The check-ins of a person
type: requirements
status: approved
created: 2026-11-23
updated: 2026-11-24
approved_by: Temuri
approved_on: 2026-11-23
constrained_by: [OWN, JOB, TG, PLG, DAT, UI, SEC]
---

# Requirements: The check-ins of a person

The statements divide into four files:

| File                                                 | Holds                                                     |
| ---------------------------------------------------- | --------------------------------------------------------- |
| `requirements.md`                                    | The problem, the turn-on, the pause, and the schedule     |
| [`requirements-round.md`](requirements-round.md)     | The round on Telegram: the start prompt and the questions |
| [`requirements-catalog.md`](requirements-catalog.md) | The questions, the check-ins, the defaults, and the data  |
| [`requirements-widget.md`](requirements-widget.md)   | The widget of today on the home page                      |

## 1. Problem

A person wants to know how their mood, their sleep, and their habits change over
weeks and months. A note in a diary needs the person to remember it, and nobody
remembers at 22:00 every day. A short question on the phone at the right hour, with
a button to press, gets an answer in five seconds. The answers then need one place,
where the person reads a series over time.

## 2. Users

- A person answers the questions of a check-in on Telegram, at the hours they chose.
- The same person chooses the questions and the hours in the deck, and reads their
  values there.

Every record of pulse belongs to one person. Another member of a workspace never
reads it.

## 3. Out of scope

- A correction of a value: undo, change, or an answer for an earlier time.
- A value outside a check-in, for example a `/track` command.
- Access to the values through a tool of an agent.
- The city, the country, or the weather of a location.
- A restore of the defaults. A person recreates a deleted default by hand.
- An import or an export, for example from Swarm, JetLovers, or Monica.
- A reminder for a check-in that the person did not answer.
- A check-in that a workspace shares between its members.

## 4. Definitions

| Term       | Meaning                                                                                                                             |
| ---------- | ----------------------------------------------------------------------------------------------------------------------------------- |
| question   | One thing that pulse asks, with a type and a prompt. For example "How is your mood?", a scale from 1 to 5.                          |
| check-in   | A named list of questions, with the local times and the weekdays at which pulse asks them. For example "Morning", daily at 09:00.   |
| occurrence | One local time of a check-in on one day. "Morning" at 09:00 on 2026-11-24 is one occurrence.                                        |
| round      | One occurrence that fired: its start prompt, its questions, and its state. A round is offered, started, done, declined, or expired. |
| value      | One answer to one question of one round, with the moment of the occurrence and the source.                                          |
| default    | A question or a check-in that pulse ships. The first turn-on copies each one to the person, who then owns and edits the copy.       |

## 5. Functional requirements

### `PULSE-FR-001`

A person must turn pulse on before any check-in fires for them. The first turn-on
must copy every default question and every default check-in to the person.

**Why:** A person who only opens the page must not start to receive messages.

**Examples:**

- Normal case: Nina turns pulse on. Nina holds the five default check-ins, and
  "Morning" fires at the next 09:00.
- Limit case: Nina turns pulse off, and on again. Nothing is copied a second time.
- Unwanted case: Beka never turned pulse on. No check-in fires for Beka.

### `PULSE-FR-002`

A person must turn pulse off. While pulse is off, no check-in fires, and every round
that waits expires at the turn-off. The questions, the check-ins, and the values stay.

### `PULSE-FR-003`

A person must pause pulse until a date or for a duration, from the deck or with the
command `/pause`. The command `/resume`, or the deck, ends the pause. No occurrence
fires during the pause, and an occurrence of the pause never fires later.

**Examples:**

- Normal case: `/pause 2026-12-01`. Nothing fires until 00:00 of 2026-12-01 in the
  time zone of the person.
- Limit case: `/pause 3d` at 10:00. Nothing fires until 10:00 three days later.
- Unwanted case: `/pause yesterday`. The bot answers how to write a date or a
  duration, and nothing changes.

### `PULSE-FR-004`

Every page of pulse must show the time zone that the schedule uses. With no time
zone on the profile, the page must say that the schedule uses UTC, and link to the
profile.

### `PULSE-FR-005`

A check-in must fire at each of its local times, on each of its weekdays, in the time
zone of the person. A check-in with no weekday fires every day.

**Examples:**

- Normal case: "Weekly" at 18:00 on Sunday, for a person in `Asia/Tbilisi`, fires at
  14:00 UTC each Sunday.
- Limit case: "Morning" fired at 09:00 in `Asia/Tbilisi`. At 10:00 Tbilisi time,
  Nina changes the time zone to `Europe/Berlin`, where it is 07:00. "Morning" fires
  again at 09:00 Berlin time the same day, and at 09:00 Berlin time each day after.

### `PULSE-FR-006`

Only the latest due occurrence of a person must fire. An earlier occurrence that the
person missed never fires. Two occurrences due at the same minute both fire, the
earlier time first, then by the name of the check-in.

**Why:** Three questions about a mood of four hours ago carry no information.

**Examples:**

- Normal case: the hub was down from 12:50 to 13:20. At 13:20 "Midday" of 13:00 fires.
- Limit case: the hub was down from 12:00 to 17:05. "Midday" never fires.
  "Afternoon" of 17:00 fires.

### `PULSE-FR-007`

An occurrence more than 60 minutes in the past must never fire.

**Why:** A morning check-in that arrives at noon asks the wrong question.

**Example:** The hub was down from 08:30 to 10:15. "Morning" of 09:00 never fires.

### `PULSE-FR-008`

A new check-in, a turned-on check-in, and a check-in whose times change must fire
first at their next occurrence after the change.

**Example:** At 10:00 Nina adds a check-in at 09:00. It fires first the next day.

### `PULSE-FR-009`

An occurrence in a skipped hour of a change of daylight saving time must fire later
by the length of the gap. An occurrence in a repeated hour must fire one time, at its
first moment.

**Why:** The deck reads a local time in a gap the same way, so a page and the
schedule agree on the moment.

**Examples:**

- Limit case: in `Europe/Berlin` on 2027-03-28, an occurrence at 02:30 fires at 03:30.
- Limit case: in `Europe/Berlin` on 2027-10-31, an occurrence at 02:30 fires one
  time, at 00:30 UTC.

### `PULSE-FR-010`

When a round fires, every earlier round of the person that is still offered or
started must expire. An expired round records no value for a question that waited.

## 6. Non-functional requirements

### `PULSE-NFR-001`

The start prompt of an occurrence must reach Telegram within 2 minutes after its
local time, while the worker runs. The measurement starts at the minute of the
occurrence and ends at the call to Telegram.

### `PULSE-NFR-002`

For a person who never turned pulse on, a check of the schedule must write no record
and call no external service.

**Why:** The check runs every minute for every active user of the instance.

## 7. Invariants

### `PULSE-INV-001`

A question, a check-in, a round, and a value of a person are visible to that person
alone, in every workspace.

### `PULSE-INV-002`

One question of one round holds at most one value.

### `PULSE-INV-003`

One occurrence fires at most one round, whatever the count of workers and restarts.

### `PULSE-INV-004`

The question and the type of a value never change after the value exists.

## 8. Constraints from the guidelines

| Rule      | Guideline    | Effect on this feature                                                               |
| --------- | ------------ | ------------------------------------------------------------------------------------ |
| `OWN-004` | Ownership    | Every record is personal, so every table is scoped to a user.                        |
| `OWN-009` | Ownership    | The check runs for each active user. It reads no workspace.                          |
| `JOB-006` | Jobs         | A slow check skips the next minute. An occurrence still fires inside its 60 minutes. |
| `TG-003`  | Telegram     | `/pause` and `/resume` run in a chat that selected a workspace that enables pulse.   |
| `TG-007`  | Telegram     | A round reaches the person through the plugin messages of `TGMSG`.                   |
| `TG-008`  | Telegram     | A typed answer reaches pulse through a reply claim. A conversation comes first.      |
| `PLG-009` | Plugin model | A round lives in the database, so a restart loses nothing.                           |
| `DAT-010` | Data         | Every moment is stored in UTC. The schedule converts with the time zone of a person. |
| `UI-005`  | Web UI       | The pages are routes of the remote of the plugin.                                    |
| `SEC-004` | Security     | A blocked user is no active user, so no check-in fires for them.                     |

## 9. Open questions

| #   | Question                                           | Owner  | Answer                                                  |
| --- | -------------------------------------------------- | ------ | ------------------------------------------------------- |
| 1   | Who receives check-ins?                            | Temuri | A person who turned pulse on. `PULSE-FR-001`.           |
| 2   | Does the start prompt name the check-in and count? | Temuri | Yes. No closing message: the start prompt shows "done". |
| 3   | Does a choice store its label or a stable key?     | Temuri | A stable identifier, with an editable label.            |
| 4   | Is a round with every question skipped done?       | Temuri | Done.                                                   |

## Retired identifiers

This file has no retired identifier.
