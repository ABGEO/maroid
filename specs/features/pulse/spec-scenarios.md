---
id: PULSE
title: The check-ins of a person, the scenarios
type: spec
status: approved
created: 2026-11-23
updated: 2026-11-23
approved_by: Temuri
approved_on: 2026-11-23
constrained_by: [TST, OWN, TRC]
requirements: features/pulse/requirements.md
---

# Specification: The scenarios of pulse

`spec.md` holds the design. This file holds the scenarios.

Every integration scenario runs against the container of `testdb.Start`, with the
migrations of the hub and of pulse. Nina, in `Asia/Tbilisi` (UTC+4), turned pulse on
unless a scenario says otherwise. Beka never turned it on. Both are members of
workspace H, which enables pulse. A fake `message.Messenger` records each send, edit,
and release, and gives each message the identifier `m1`, `m2`, and so on. The clock
is fixed. A time with no zone is local to Nina. A name in code, `mood` for example,
stands for the copy of that default question of `spec-data.md` that Nina holds, and
`<mood>` stands for its `id`.

## 6. Scenarios

### `PULSE-SC-001`

**Verifies:** `PULSE-FR-001`
**Layer:** integration

**When** Nina, with no row of settings, sends `PUT /settings` `{enabled: true}` with
`If-Match: "0"`, then turns pulse off, then on again.
**Then** Nina holds 5 check-ins and 12 questions after the first turn-on, and the
same counts after the third request. `seeded_at` keeps its first value.

### `PULSE-SC-002`

**Verifies:** `PULSE-FR-001`, `PULSE-NFR-002`
**Layer:** integration

**When** `tick` runs for Beka at 09:00.
**Then** the schema holds no row of Beka, and the fake records no call.

### `PULSE-SC-003`

**Verifies:** `PULSE-FR-002`
**Layer:** integration

**Given** "Morning" of 09:00 is offered as `m1`.
**When** Nina turns pulse off at 09:10, and `tick` runs at 13:00.
**Then** the round is expired, `m1` reads `Morning check-in: expired.`, and the tick
of 13:00 sends nothing. The questions, the check-ins, and the values stay.

### `PULSE-SC-004`

**Verifies:** `PULSE-FR-003`
**Layer:** integration

**When** Nina sends `/pause 2026-12-01` on 2026-11-28, and `tick` runs at 09:00 of
each day to 2026-12-01.
**Then** `paused_until` is 2026-11-30T20:00:00Z, no round fires on 28, 29, or 30
November, and "Morning" fires at 09:00 on 1 December. With `/pause 3d` at 10:00 on
2026-11-28, `paused_until` is 10:00 on 2026-12-01, and "Morning" of that day never
fires.

### `PULSE-SC-005`

**Verifies:** `PULSE-FR-003`
**Layer:** unit

**When** Nina sends `/pause yesterday`, `/pause 2025-01-01`, and `/pause 400d`, then
`/resume` during a pause.
**Then** each of the first three answers the format, and `paused_until` does not
change. `/resume` answers `Check-ins resumed.` and clears it.

### `PULSE-SC-006`

**Verifies:** `PULSE-FR-004`
**Layer:** manual

**When** Nina, with the zone `Asia/Tbilisi`, and Zura, with no zone, open each page
of pulse.
**Then** each page of Nina shows `Asia/Tbilisi` in its header. Each page of Zura shows
"The schedule uses UTC" with a link to the profile, and shows each time in UTC.

### `PULSE-SC-007`

**Verifies:** `PULSE-FR-005`
**Layer:** unit

**Given** "Weekly" at 18:00 on Sunday, in `Asia/Tbilisi`.
**When** `schedule.Latest` runs at 2026-11-29T14:00:30Z, a Sunday, and at
2026-11-30T14:00:30Z.
**Then** the first answers 2026-11-29T14:00:00Z. The second answers none.

### `PULSE-SC-008`

**Verifies:** `PULSE-FR-005`
**Layer:** integration

**Given** "Morning" fired at 09:00 Tbilisi on 2026-11-24.
**When** Nina changes the zone to `Europe/Berlin` at 10:00 Tbilisi, and `tick` runs
each minute to 2026-11-25T08:01:00Z.
**Then** "Morning" fires at 2026-11-24T08:00:00Z and at 2026-11-25T08:00:00Z, each
09:00 Berlin, and at no other moment.

### `PULSE-SC-009`

**Verifies:** `PULSE-FR-006`
**Layer:** unit

**When** the first `tick` after a stop from 12:50 runs at 13:20.
**Then** "Midday" of 13:00 fires.

### `PULSE-SC-010`

**Verifies:** `PULSE-FR-006`
**Layer:** integration

**Given** "Gym" at 07:30 and "Run" and "Bike" at 08:00, with no tick since 07:00.
**When** `tick` runs at 08:00.
**Then** "Bike" fires, then "Run", each with its own start prompt. "Gym" does not
fire, and its `fires_after` is 07:30.

### `PULSE-SC-011`

**Verifies:** `PULSE-FR-007`
**Layer:** unit

**When** the first `tick` after a stop from 08:30 runs at 10:15.
**Then** "Morning" of 09:00 does not fire. At 10:00 the next day nothing fires either.
At 09:00 the next day "Morning" fires.

### `PULSE-SC-012`

**Verifies:** `PULSE-FR-008`
**Layer:** integration

**When** Nina creates "Tea" at 09:00 by `POST /checkins` at 10:00, and at 10:05 moves
"Midday" from 13:00 to 10:00 by `PUT`.
**Then** neither fires before the next day. "Tea" fires at 09:00 the next day.

### `PULSE-SC-013`

**Verifies:** `PULSE-FR-009`
**Layer:** unit

**When** `schedule.Resolve` runs for 02:30 in `Europe/Berlin` on 2027-03-28 and on
2027-10-31, and `tick` runs each minute of 2027-10-31 from 00:00 to 04:00 UTC for a
check-in at 02:30.
**Then** the first answers 2027-03-28T01:30:00Z, which is 03:30 local. The second
answers 2027-10-31T00:30:00Z. The check-in fires one time on 2027-10-31.

### `PULSE-SC-014`

**Verifies:** `PULSE-FR-010`
**Layer:** integration

**Given** "Morning" of 09:00 started, and its question `weight` waits as `m5` with a
claim.
**When** `tick` runs at 13:00.
**Then** the round of 09:00 and its prompts are expired, `m1` reads
`Morning check-in: expired.`, `m5` reads `Weight (kg) expired`, the claim of `m5` is
released, no value exists for the round, and "Midday" offers its start prompt.

### `PULSE-SC-015`

**Verifies:** `PULSE-INV-003`
**Layer:** integration

**When** two `tick` runs for Nina at 09:00:10 and 09:00:20 overlap in two processes.
**Then** one round of "Morning" for 09:00 exists, and the fake records one start
prompt.

### `PULSE-SC-016`

**Verifies:** `PULSE-NFR-001`
**Layer:** unit, manual

**When** `tick` runs at 08:59:59 and at 09:01:59.
**Then** the first sends nothing. The second sends the start prompt of 09:00. Manual:
on the compose stack, the start prompt of a check-in set two minutes ahead arrives
within two minutes after its time.

### `PULSE-SC-017`

**Verifies:** `PULSE-FR-011`
**Layer:** integration

**When** "Evening" fires, a check-in with one question fires, and a check-in whose
every question is archived falls due.
**Then** the first sends `Evening check-in, 7 questions.` with Start `start` and
Decline `decline`. The second reads `1 question.` The third fires no round.

### `PULSE-SC-018`

**Verifies:** `PULSE-FR-012`
**Layer:** integration

**When** Nina presses Decline on `m1`.
**Then** the round is declined, `m1` reads `Morning check-in: declined.`, and the fake
records no other send.

### `PULSE-SC-019`

**Verifies:** `PULSE-FR-013`, `PULSE-FR-014`, `PULSE-FR-030`
**Layer:** integration

**When** Nina presses Start on `m1` at 09:05, then 4 on the question `m2` at 09:40.
**Then** `m1` reads `Morning check-in, 4 questions. Started.`, `m2` asks
`How is your mood? (1 bad, 5 great)`, then reads `How is your mood? 4`, and `m3` asks
`How many hours did you sleep? (h)`. One value holds the `mood` question, `scale`, 4, `occurred_at`
09:00, `answered_at` 09:40, and the source `telegram`.

### `PULSE-SC-020`

**Verifies:** `PULSE-FR-015`, `PULSE-FR-016`
**Layer:** integration

**When** Nina starts "Morning" and presses Skip on each of the 4 questions.
**Then** each question reads `<prompt> skipped`, the round is done, `m1` reads
`Morning check-in: done.`, no value exists, and no message follows the last edit.

### `PULSE-SC-021`

**Verifies:** `PULSE-FR-017`
**Layer:** unit

**When** `round.Markup` builds a scale from 1 to 5 with labels, a scale from 1 to 10,
a yes or no, and a choice of 3.
**Then** the first holds one row `v:1` to `v:5` and a row Skip. The second holds two
rows of 5. The third holds Yes `y`, No `n`, and Skip. The fourth holds two rows of
`c:<choice id>`, two then one, and Skip.

### `PULSE-SC-022`

**Verifies:** `PULSE-FR-018`
**Layer:** integration

**Given** `weight` waits, from 30 to 250 kg.
**When** Nina replies "900", then "72,5". On a second round, Nina replies "250". On
`sleep_hours`, Nina presses the quick value 8.
**Then** "900" sends `Send a number from 30 to 250, or press Skip.` and the claim
stays. "72,5" records 72.5, and the claim is released. "250" records 250. The press
records 8.

### `PULSE-SC-023`

**Verifies:** `PULSE-FR-019`
**Layer:** integration

**When** Nina replies 1001 characters to `main_thing`, then 1000.
**Then** the first sends `Send at most 1000 characters, or press Skip.`. The second
records the text.

### `PULSE-SC-024`

**Verifies:** `PULSE-FR-020`
**Layer:** integration

**Given** `location` waits as `m8`, sent with the keyboard Send location and Skip.
**When** Nina sends "home", then shares 41.7151, 44.8271. On a second round, Nina
sends the text "Skip".
**Then** "home" sends `Share a location, or press Skip.`. The location records the
latitude and the longitude, and sends `Location saved.` with the removal of the
keyboard. "Skip" records nothing and sends `Skipped.` with the removal.

### `PULSE-SC-025`

**Verifies:** `PULSE-FR-021`, `PULSE-INV-002`
**Layer:** integration

**When** Nina presses 3 on the mood question of a round that expired yesterday, and
presses 4 twice in parallel on a waiting mood question.
**Then** the first press answers `This check-in is closed.` and records nothing. One
press of the pair records 4, the other answers the closed notice, and one value
exists for the prompt.

### `PULSE-SC-026`

**Verifies:** `PULSE-FR-022`
**Layer:** unit

**When** pulse sends a scale, a yes or no, a choice, a number, a text, and a location
question.
**Then** the first three carry no `ClaimReplyUntil`. The last three carry a claim 29
days ahead.

### `PULSE-SC-027`

**Verifies:** `PULSE-FR-023`
**Layer:** integration

**When** Nina posts a scale from 1 to 5, "How is your energy?"; a number from 0 to 20,
"Coffee today?", unit "cups", quick values 0 to 3; and a number from 10 to 5.
**Then** the first two answer 201, each with a new `id`. The third answers 422 at
`/config/max`. A scale from 1 to 12 answers 422 at `/config/max`. A number with
`low_label` answers 400 `member-unknown`.

### `PULSE-SC-028`

**Verifies:** `PULSE-FR-024`
**Layer:** integration

**Given** `weight` holds one value.
**When** Nina puts `weight` with the type `scale`, then with the prompt "Weight today".
**Then** the first answers 409 `question-has-values`. The second answers 200.

### `PULSE-SC-029`

**Verifies:** `PULSE-FR-025`
**Layer:** integration

**Given** `weight` holds values, and "Coffee today?" holds none.
**When** Nina archives `weight`, "Morning" fires, Nina restores `weight` from the
archive, and Nina deletes `weight` and "Coffee today?".
**Then** the round reads `3 questions.`, and its values of `weight` stay. The delete
of `weight` answers 409 `question-has-values`. The delete of "Coffee today?" answers 204.

### `PULSE-SC-030`

**Verifies:** `PULSE-FR-026`
**Layer:** integration

**Given** "What did you drink?", a choice of "Coffee" with the `id` C and "Tea" with
the `id` T, and one value that records C.
**When** Nina puts the question with C labelled "Espresso" and T; then with T and
"Water" with no `id`; then with C and "Water" with no `id`; then with C alone; then
with 13 choices.
**Then** the rename answers 200, and the value still records C, shown as "Espresso".
T and "Water" answer 409 `choice-in-use` with C in `detail`. C and "Water" answer 200,
delete T, and give "Water" a new `id`. C alone and thirteen answer 422.

### `PULSE-SC-031`

**Verifies:** `PULSE-FR-027`
**Layer:** integration

**When** Nina posts "Gym" at 07:30 on 1, 3, and 5 with `energy`; a check-in at
"25:00"; one with 7 times; one with 21 questions; then deletes "Morning", which holds
values.
**Then** "Gym" answers 201. The next three answer 422. The delete answers 204, and the
values of "Morning" stay.

### `PULSE-SC-032`

**Verifies:** `PULSE-FR-028`
**Layer:** integration

**When** Nina answers `mood` in "Morning", "Midday", "Afternoon", and "Evening" of one
day.
**Then** `GET /questions/<mood>/values` answers four values, and the day of
`GET /questions/<mood>/days` holds the count 4.

### `PULSE-SC-033`

**Verifies:** `PULSE-FR-029`
**Layer:** unit

**When** `catalog.Defaults` answers the catalog.
**Then** it holds the five check-ins, times, and weekdays of `PULSE-FR-029`, and its
12 questions carry the types, the limits, the quick values, and the prompts of
`spec-data.md`.

### `PULSE-SC-034`

**Verifies:** `PULSE-FR-030`
**Layer:** integration

**When** Nina answers "Evening" of 22:00 at 00:40 the next day.
**Then** each value holds `occurred_at` 22:00 of the first day and `answered_at` 00:40.

### `PULSE-SC-035`

**Verifies:** `PULSE-FR-031`
**Layer:** integration

**Given** 3 values of `mood`, the last 4, 1 value of `weight`, 72.5, and an archived
question with no value.
**When** Nina reads `GET /questions` and `GET /questions?archived=true`.
**Then** `mood` reads `value_count` 3 and `latest` 4, `weight` reads 1 and 72.5, and
the archived question reads 0 and no `latest`.

### `PULSE-SC-036`

**Verifies:** `PULSE-FR-032`
**Layer:** integration

**Given** 150 values of `mood` over 40 days.
**When** Nina reads `GET /questions/<mood>/values?from=<day 11>&to=<day 40>&limit=100`.
**Then** the pages hold the values of those 30 local days, newest first, 100 on the
first page with a `next` link.

### `PULSE-SC-037`

**Verifies:** `PULSE-FR-033`
**Layer:** integration, manual

**Given** `mood` 3 at 23:30 Tbilisi on 24 November, 19:30 UTC, and 5 at 09:00 on 25
November; `exercised` yes, no, and yes on three days.
**When** Nina reads the days of `mood` and of `exercised`.
**Then** 24 November holds the count 1 and the mean 3, and 25 November the count 1
and the mean 5. `exercised` holds `yes_count` 1, 0, and 1. A range of 367 days
answers 422. Manual: the Data page draws the columns and the line.

### `PULSE-SC-038`

**Verifies:** `PULSE-FR-034`
**Layer:** manual

**When** Nina opens `location` on the Data page.
**Then** each value shows `41.71510, 44.82710` and a link to OpenStreetMap at that
point, which opens in a new tab.

### `PULSE-SC-039`

**Verifies:** `PULSE-INV-001`
**Layer:** integration

**Given** Nina holds rows in every table, and Beka acts in workspace H.
**When** Beka reads each `GET` route, `GET /rounds` too, and a transaction with Beka as the acting user
counts the rows of each table.
**Then** `GET /settings` answers `enabled` false, every other route answers an empty
page or 404 for an `id` of Nina, and each count is 0.

### `PULSE-SC-040`

**Verifies:** `PULSE-INV-004`
**Layer:** integration

**When** a statement updates the `question_id` of a value of Nina, then its type.
**Then** each raises `check_violation`, and the value keeps both.

### `PULSE-SC-041`

**Verifies:** `PULSE-DD-020`
**Layer:** integration

**When** Nina sends a body of 64 KiB and one byte to `PUT /settings`,
`POST /questions`, and `POST /checkins`.
**Then** each answers 413 `content-too-large`, and no row changes.

### `PULSE-SC-042`

**Verifies:** `PULSE-FR-035`, `PULSE-FR-036`
**Layer:** integration

**Given** the defaults of Nina in Asia/Tbilisi on 24 November. Morning fires at 09:00,
and Nina answers 3, 7, and 4, then skips Weight. Midday fires at 13:00. Afternoon
fires at 17:00, which expires Midday, and Nina declines Afternoon.
**When** Nina reads `GET /rounds`.
**Then** the page holds Afternoon `declined`, Midday `expired`, and Morning `done`, in
that order. Morning holds its four questions in order: three with a value, and Weight
`skipped`. The question of Midday is `expired`, and the question of Afternoon is
`pending`.

### `PULSE-SC-043`

**Verifies:** `PULSE-FR-037`
**Layer:** integration

**Given** the rounds of `PULSE-SC-042`.
**When** Nina reads `GET /rounds` with `checkin_id` of Morning, then with `state`
`expired`, then with both, then with `state` `lost`, then with `checkin_id` `morning`.
**Then** the answers hold Morning alone, Midday alone, and no round. The last two
answer 400.

### `PULSE-SC-044`

**Verifies:** `PULSE-FR-035`, `PULSE-FR-036`
**Layer:** manual

**Given** the rounds of `PULSE-SC-042`.
**When** Nina opens History, opens each round, and then opens History from Morning on
Check-ins.
**Then** Morning reads 3 of 4 answered, with "3 of 5", "7 h", "4 of 5", and
"Skipped". Midday reads "Expired", and Afternoon reads "Not asked" with no count. The
last page lists Morning alone, with Morning chosen in the filter.

## Retired identifiers

This file has no retired identifier.
