---
id: PULSE
title: The check-ins of a person, the widget of today
type: spec
status: approved
created: 2026-11-24
updated: 2026-11-24
approved_by: Temuri
approved_on: 2026-11-24
constrained_by: [PLG, OWN, UI, TST]
requirements: features/pulse/requirements-widget.md
---

# Specification: The widget of today

`spec.md` holds the design of pulse. This file holds the widget "Today", its design
decisions, and its scenarios. `WIDGET` holds the contract of a widget and the card
in the deck.

## 2. Coverage

| Requirement    | Where this specification realizes it                                |
| -------------- | ------------------------------------------------------------------- |
| `PULSE-FR-038` | Section 4.3, `PULSE-SC-045`, `PULSE-SC-053`                         |
| `PULSE-FR-039` | `Round.ListBetween`, `PULSE-SC-045`, `PULSE-SC-047`, `PULSE-SC-048` |
| `PULSE-FR-040` | `schedule.Upcoming`, `PULSE-DD-021`, `PULSE-SC-045`, `PULSE-SC-049` |
| `PULSE-FR-041` | `PULSE-DD-021`, `PULSE-SC-050`                                      |
| `PULSE-FR-042` | Load step 5, `PULSE-SC-045`                                         |
| `PULSE-FR-043` | The texts, `PULSE-SC-045`, `PULSE-SC-046`                           |
| `PULSE-FR-044` | `PULSE-DD-022`, `PULSE-SC-046`                                      |
| `PULSE-FR-045` | `PULSE-DD-023`, `PULSE-SC-045`, `PULSE-SC-047`, `PULSE-SC-053`      |
| `PULSE-FR-046` | Load step 7, `PULSE-SC-045`                                         |
| `PULSE-FR-047` | Load step 1, `PULSE-SC-051`                                         |
| `PULSE-FR-048` | Load step 6, `PULSE-DD-024`, `PULSE-SC-052`                         |
| `PULSE-FR-049` | Load step 6, `PULSE-SC-052`                                         |

## 3. Guideline compliance

| Rule      | Guideline    | How this specification obeys it                                                         |
| --------- | ------------ | --------------------------------------------------------------------------------------- |
| `PLG-006` | Plugin model | The widget uses the existing capability `pluginapi.WidgetPlugin`, so no new registry.   |
| `OWN-003` | Ownership    | The hub loads the widget in the request of the acting user, with their zone.            |
| `OWN-007` | Ownership    | Every read runs inside `PluginDB.WithTx`, which sets the acting user.                   |
| `OWN-008` | Ownership    | `Round.ListBetween` writes no filter on `user_id`. The policy narrows the rows.         |
| `UI-005`  | Web UI       | Each link names a route of the remote: `/history`, `/history/:checkin`, `/preferences`. |
| `TST-004` | Testing      | Each scenario names its layer.                                                          |

## 4. Design

### 4.1 Components

Every path is under `plugins/pulse/`.

| Path                          | Change | Holds                                                                                          |
| ----------------------------- | ------ | ---------------------------------------------------------------------------------------------- |
| `widget/doc.go`               | create | The package comment                                                                            |
| `widget/today.go`             | create | `NewToday(db pluginapi.PluginDB, now func() time.Time) *pluginapi.FuncWidget`, and the texts   |
| `widget/today_test.go`        | create | `PULSE-SC-045` to `PULSE-SC-052`                                                               |
| `schedule/occurrence.go`      | change | `Upcoming(checkin model.Checkin, zone *time.Location, floor, now time.Time) []time.Time`       |
| `schedule/occurrence_test.go` | change | The unit cases of `Upcoming`                                                                   |
| `repository/round.go`         | change | `ListBetween(ctx context.Context, from, to time.Time) ([]model.Round, error)` in the interface |
| `model/settings.go`           | change | `PauseLayout = "2 Jan 2006, 15:04"`, which `/pause` and the widget both read                   |
| `telegram/command/pause.go`   | change | Reads `model.PauseLayout` in place of `pausedShown`                                            |
| `main.go`                     | change | `Widgets()`, and the assertion of `pluginapi.WidgetPlugin`                                     |

`Widgets()` answers `[]pluginapi.Widget{widget.NewToday(p.db, p.now)}`, so a test
fixes the clock as for `job.Tick`.

`schedule.Upcoming` answers the occurrences of the check-in on the local day of `now`
in the zone, each at a moment `m` with `m > now` and `m > floor`, oldest first. A day
that the weekdays of the check-in exclude answers none. Each moment comes from
`schedule.Resolve`, so a skipped and a repeated hour read as the tick reads them
(`PULSE-FR-009`).

`Round.ListBetween` answers the rounds with `from <= occurrence_at < to`, ordered by
`occurrence_at, checkin_name`, with `roundColumns`.

### 4.3 Declarations

**Widgets.**

| Key     | Title | Kind   | Size | Permission     | Realizes       |
| ------- | ----- | ------ | ---- | -------------- | -------------- |
| `today` | Today | `list` | `m`  | `checkins.use` | `PULSE-FR-038` |

**Capabilities.** `widgets` joins the list of `spec.md`.

### 4.4 Flow

`Load(ctx)` runs one `PluginDB.WithTx` and writes nothing:

1. `Settings.Get`. `sql.ErrNoRows`, or `enabled` false: answer
   `NotSetUpAnswer("Turn pulse on to receive check-ins.", "/preferences")`
   (`PULSE-FR-047`).
2. Let `zone = pluginapi.ActingTimeZone(ctx)`, `now = now()`, `from` the local
   midnight of `now` in `zone`, and `to` the local midnight one day later.
3. `Round.ListBetween(from, to)`, then `Prompt.ListOf` of their identifiers. Each
   round becomes an item (`PULSE-FR-039`).
4. `Checkin.ListDue`. For each check-in, `schedule.Upcoming(checkin, zone, floor, now)`
   with `floor = max(fires_after, paused_until)`. Each moment becomes an item
   (`PULSE-FR-040`, `PULSE-FR-041`).
5. Sort the items by the moment, then by the name of the check-in (`PULSE-FR-042`).
6. No item: answer `EmptyAnswer` with the pause text while `paused_until > now`
   (`PULSE-FR-048`), or with "No check-in today." (`PULSE-FR-049`).
7. Answer `ListAnswer` with the items and `More: "/history"` (`PULSE-FR-046`).

**The texts.** The label of each item is the name of the check-in. The meta line is
the local time of the occurrence, `15:04` in `zone`, a middle dot, then:

| Source                 | Text                  |
| ---------------------- | --------------------- |
| An upcoming occurrence | `Upcoming`            |
| A round `offered`      | `Waiting to start`    |
| A round `started`      | `<n> of <m> answered` |
| A round `done`         | `Done`                |
| A round `declined`     | `Declined`            |
| A round `expired`      | `Expired`             |

`<n>` counts the prompts of the round in the state `answered`, and `<m>` is
`QuestionCount` (`PULSE-FR-044`). The words of a state are those of the history page.

**The link.** An item links to `/history/<checkin_id>`. A round whose check-in was
deleted holds no `checkin_id`, and links to `/history` (`PULSE-FR-045`).

### 4.5 Errors

| Condition                    | Behavior                                                          |
| ---------------------------- | ----------------------------------------------------------------- |
| A read fails on the database | `Load` answers the wrapped error. The card shows `WIDGET-FR-020`. |

## 5. Design decisions

### `PULSE-DD-021`

**Realizes:** `PULSE-FR-040`, `PULSE-FR-041`
**Decision:** The widget computes each upcoming occurrence from `Checkin.ListDue` and
`schedule.Upcoming`, with the floor of the tick, less its 60 minutes:
`max(fires_after, paused_until)`. An occurrence earlier than `now` never becomes an
item. Only a round shows the past.
**Rationale:** The widget and the tick read the same check-ins, the same flag, the
same pause, and the same `Resolve`, so the widget calls "upcoming" only what the tick
will fire. A check-in with no active question never fires, and `ListDue` already
leaves it out. The read costs one query for at most a few dozen check-ins.
**Alternatives:** A table of the next occurrences: a write at each change of a
check-in, of the pause, and of the zone. The same rule in SQL: a second copy of the
rule of the skipped and the repeated hour.

### `PULSE-DD-022`

**Realizes:** `PULSE-FR-044`
**Decision:** The widget counts the answered prompts from `Prompt.ListOf` over the
rounds of the day. `roundColumns` does not change.
**Rationale:** `ListOf` exists and reads every prompt of a day in one query: six
rounds of seven questions is 42 rows. A column in `roundColumns` adds a subquery to
every read of a round, the tick included, for one consumer.
**Alternatives:** `answered_count` in `roundColumns`: the cost above. A query for each
round: six reads in place of one.

### `PULSE-DD-023`

**Realizes:** `PULSE-FR-045`
**Decision:** An item links to the page route `/history/:checkin`, which exists. A
round with no check-in links to `/history`.
**Rationale:** The route already narrows History to one check-in. An upcoming
occurrence has no round, so a link to a round would not cover it.
**Alternatives:** A route for one round: a new behavior of History, outside the
requirements.

### `PULSE-DD-024`

**Realizes:** `PULSE-FR-048`
**Decision:** The pause text is `Check-ins paused until %s.` with
`model.PauseLayout`, the text that `/pause` answers.
**Rationale:** The person reads one sentence for one state, on Telegram and in the
deck. A pause reaches 366 days ahead, so the year stays in the text.
**Alternatives:** A short form with no year: a pause of a year reads as next week.

## 6. Scenarios

Each integration scenario uses `pulsetest.Schema`, the fixed clock, and the people
of `spec-scenarios.md`. 24 November 2026 is a Tuesday. Nina turned pulse on at 08:00
with the defaults. The tick runs only at the moments a scenario names.

### `PULSE-SC-045`

**Verifies:** `PULSE-FR-038`, `PULSE-FR-039`, `PULSE-FR-040`, `PULSE-FR-042`,
`PULSE-FR-043`, `PULSE-FR-045`, `PULSE-FR-046`
**Layer:** integration

**Given** at 08:00 Nina adds the check-in Water at 13:00 with the question `mood`. The tick
runs at 09:00, and Nina answers Morning. The tick runs at 13:00, which fires Midday
and Water, and Nina declines Midday.
**When** the widget loads at 14:00.
**Then** the meta is key `today`, title Today, `list`, `m`, `checkins.use`. The
answer is `ready` with these items, and `More` `/history`:

| Label     | Meta                       | Link                   |
| --------- | -------------------------- | ---------------------- |
| Morning   | `09:00 · Done`             | `/history/<Morning>`   |
| Midday    | `13:00 · Declined`         | `/history/<Midday>`    |
| Water     | `13:00 · Waiting to start` | `/history/<Water>`     |
| Afternoon | `17:00 · Upcoming`         | `/history/<Afternoon>` |
| Evening   | `22:00 · Upcoming`         | `/history/<Evening>`   |

Weekly does not show.

### `PULSE-SC-046`

**Verifies:** `PULSE-FR-043`, `PULSE-FR-044`
**Layer:** integration

**Given** the tick runs at 22:00 and fires Evening, with 7 questions.
**When** the widget loads at 22:01, then Nina starts the round, answers `mood`,
`energy`, and `alcohol_drinks`, skips `exercised`, and the widget loads again.
**Then** the meta line of Evening reads `22:00 · Waiting to start`, then
`22:00 · 3 of 7 answered`.

### `PULSE-SC-047`

**Verifies:** `PULSE-FR-039`, `PULSE-FR-045`
**Layer:** integration

**Given** Nina turned pulse on at 08:00 on 23 November. The tick runs at 22:00 on
23 November, which fires Evening, then at 09:00 and 13:00 on 24 November. Morning
expires Evening of 23 November, and Midday expires Morning. Nina then deletes the
check-in Midday.
**When** the widget loads at 14:00.
**Then** the items are Morning `09:00 · Expired`, Midday `13:00 · Waiting to start`
with the link `/history`, then Afternoon and Evening as upcoming. Evening of
23 November does not show.

### `PULSE-SC-048`

**Verifies:** `PULSE-FR-039`, `PULSE-FR-040`
**Layer:** integration

**Given** the tick runs at 09:00, and Nina answers Morning.
**When** the widget loads at 10:30 in `America/Chicago`, where it is 00:30 on
24 November.
**Then** the items are Morning, Midday, Afternoon, and Evening, each `Upcoming` at
09:00, 13:00, 17:00, and 22:00 in Chicago. No round shows: the round of Morning fell
at 23:00 on 23 November there.

### `PULSE-SC-049`

**Verifies:** `PULSE-FR-040`
**Layer:** integration

**Given** the tick runs at 09:00, and Nina answers Morning. At 14:00 Nina pauses
pulse until 18:00.
**When** the widget loads at 14:00, then Nina turns off Evening and it loads again.
**Then** the first answer holds Morning, then Evening at `22:00 · Upcoming`.
Afternoon of 17:00 does not show. The second answer holds Morning alone.

### `PULSE-SC-050`

**Verifies:** `PULSE-FR-041`
**Layer:** integration

**Given** the tick runs at 09:00. At 10:00 Nina adds the check-in Water at 09:00.
**When** the widget loads at 10:00, then at 13:01 with no tick at 13:00.
**Then** neither answer holds Water. The second answer holds Morning, Afternoon, and
Evening, and not Midday.

### `PULSE-SC-051`

**Verifies:** `PULSE-FR-047`, `PULSE-INV-001`
**Layer:** integration

**Given** the rounds of `PULSE-SC-045`.
**When** Beka loads the widget at 14:00, then Nina turns pulse off and loads it.
**Then** both answers are `not set up`, with "Turn pulse on to receive check-ins."
and the link `/preferences`. Beka reads no round of Nina.

### `PULSE-SC-052`

**Verifies:** `PULSE-FR-048`, `PULSE-FR-049`
**Layer:** integration

**When** the widget loads in each of these states. Each state starts from the
defaults of Nina.

1. At 08:30, after Nina paused pulse at 08:00 until 2026-12-01.
2. At 08:30, after Nina turned off every check-in.
3. At 09:30, after the tick ran at 09:00, Nina answered Morning, and Nina paused pulse
   until 2026-12-01.

**Then** the answers are `empty` with "Check-ins paused until 1 Dec 2026, 00:00.",
then `empty` with "No check-in today.", then `ready` with Morning alone.

### `PULSE-SC-053`

**Verifies:** `PULSE-FR-038`, `PULSE-FR-045`
**Layer:** manual

**Given** the state of `PULSE-SC-045` on the compose stack, in workspace H.
**When** Nina opens the home page of H, then presses Morning on the widget.
**Then** the widget Today shows the five items. History opens with Morning chosen in
the filter.

### Unit cases of `schedule.Upcoming`

`PULSE-SC-054` verifies `PULSE-FR-040` and `PULSE-FR-009`, layer unit. Each case
lists the check-in, the zone, the floor, `now`, and the moments it answers:

| Check-in            | Zone            | Floor            | Now              | Answers                    |
| ------------------- | --------------- | ---------------- | ---------------- | -------------------------- |
| 09:00, 22:00, daily | `Asia/Tbilisi`  | 24 Nov 09:00     | 24 Nov 14:00     | 22:00                      |
| 18:00, Sunday       | `Asia/Tbilisi`  | 24 Nov 08:00     | 24 Nov 14:00     | none, a Tuesday            |
| 17:00, 22:00, daily | `Asia/Tbilisi`  | 24 Nov 18:00     | 24 Nov 14:00     | 22:00, a pause until 18:00 |
| 02:30, daily        | `Europe/Berlin` | 2027-03-28 00:00 | 2027-03-28 01:00 | 03:30, the skipped hour    |
| 02:30, daily        | `Europe/Berlin` | 2027-10-31 00:00 | 2027-10-31 01:00 | 00:30 UTC, once            |

## 7. Build plan

| #   | Step                                                                 | Realizes                         | Done |
| --- | -------------------------------------------------------------------- | -------------------------------- | ---- |
| 1   | `schedule.Upcoming`, with `PULSE-SC-054` first                       | `PULSE-FR-040`                   | [x]  |
| 2   | `Round.ListBetween`, `model.PauseLayout`, and `/pause` reads it      | `PULSE-FR-039`, `PULSE-FR-048`   | [ ]  |
| 3   | `widget.NewToday`, with `PULSE-SC-045` to `PULSE-SC-052` first       | `PULSE-FR-038` to `PULSE-FR-049` | [ ]  |
| 4   | `Widgets()` in `main.go`, the build of the `.so`, and `PULSE-SC-053` | `PULSE-FR-038`, `PULSE-FR-045`   | [ ]  |

## Retired identifiers

This file has no retired identifier.
