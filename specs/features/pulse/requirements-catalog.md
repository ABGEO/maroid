---
id: PULSE
title: The check-ins of a person, the catalog and the data
type: requirements
status: approved
created: 2026-11-23
updated: 2026-11-23
approved_by: Temuri
approved_on: 2026-11-23
constrained_by: [OWN, UI]
---

# Requirements: The catalog and the data

[`requirements.md`](requirements.md) holds the problem and the definitions. Each
statement here acts in the deck, for the person who reads the page.

## 5. Functional requirements

### `PULSE-FR-023`

A person must create a question with a type and a prompt. The type is scale, yes or
no, number, choice, text, or location.

**Examples:**

- Normal case: a scale from 1 to 5, "How is your energy?".
- Limit case: a number from 0 to 20, "Coffee today?", the unit "cups", quick values
  0, 1, 2, and 3.
- Unwanted case: a number with a minimum of 10 and a maximum of 5. The deck shows
  that the minimum must be smaller.

### `PULSE-FR-024`

The type of a question must not change after the question holds a value. The
prompt, the unit, the limits, the labels, and the quick values change at any time.

**Why:** A series keeps one meaning. A new type is a new question (`PULSE-FR-025`).

### `PULSE-FR-025`

A person must archive a question and restore it from the archive. An archived
question is not asked, and its values stay. A person deletes a question only while it
holds no value.

**Example:** Nina wants the weight in pounds instead of kilograms. Nina archives
"Weight" and creates "Weight in pounds".

### `PULSE-FR-026`

A question of the type choice must hold 2 to 12 choices. Each choice has a label that
changes at any time. A value of a choice records the choice, not its label. A person
removes a choice only while no value records it.

**Example:** The choice "Coffee" becomes "Espresso". Each earlier value still reads
as the same choice, now shown as "Espresso".

### `PULSE-FR-027`

A person must create, change, and delete a check-in: a name, 1 to 6 local times,
the weekdays, a flag that turns it on or off, and 1 to 20 questions in order. A
deleted check-in leaves its values in place.

**Examples:**

- Normal case: "Gym", at 07:30 on Monday, Wednesday, and Friday, asking the energy.
- Unwanted case: the time 25:00. The deck shows that a time runs from 00:00 to 23:59.

### `PULSE-FR-028`

One question must belong to several check-ins, and its values form one series.

**Example:** The mood question in "Morning", "Midday", "Afternoon", and "Evening" gives four
values a day in one chart.

### `PULSE-FR-029`

The defaults must hold these check-ins and questions. A short name stands for each
question here, and it is no part of the data.

| Check-in  | When            | Questions                                                                              |
| --------- | --------------- | -------------------------------------------------------------------------------------- |
| Morning   | Every day 09:00 | `mood`, `sleep_hours`, `sleep_quality`, `weight`                                       |
| Midday    | Every day 13:00 | `mood`                                                                                 |
| Afternoon | Every day 17:00 | `mood`                                                                                 |
| Evening   | Every day 22:00 | `mood`, `energy`, `alcohol_drinks`, `exercised`, `meditated`, `main_thing`, `location` |
| Weekly    | Sunday 18:00    | `week_rating`, `week_note`                                                             |

`mood`, `sleep_quality`, `energy`, and `week_rating` are scales from 1 to 5.
`sleep_hours` is a number from 0 to 24 hours with quick values 6, 7, 8, and 9.
`weight` is a number from 30 to 250 kg. `alcohol_drinks` is a number from 0 to 30
with quick values 0, 1, 2, and 3. `exercised` and `meditated` are yes or no.
`main_thing` and `week_note` are text. `location` is a location.

### `PULSE-FR-030`

A value must carry the moment of its occurrence, the moment of the answer, and its
source. A value of a round has the source `telegram`.

**Why:** A mood answered at 09:40 belongs to the morning of 09:00. A later importer
writes its own source.

### `PULSE-FR-031`

The deck must list the questions of the person, archived or not, each with its latest
value and its count of values.

### `PULSE-FR-032`

The deck must show the values of one question over a chosen range of days, newest
first, each with the moment of its occurrence in the time zone of the person.

### `PULSE-FR-033`

The deck must draw a chart of a scale or a number, with the mean and the count of
the values of each day. A yes or no draws the count of yes for each day. The days
follow the time zone of the person.

### `PULSE-FR-034`

The deck must show a location value as its latitude and longitude, with a link that
opens the point on a map.

## Retired identifiers

This file has no retired identifier.
