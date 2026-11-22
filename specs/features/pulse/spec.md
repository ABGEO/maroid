---
id: PULSE
title: The check-ins of a person
type: spec
status: approved
created: 2026-11-23
updated: 2026-11-23
approved_by: Temuri
approved_on: 2026-11-23
constrained_by: [PLG, DAT, OWN, JOB, TG, UI, RES, PERMS, TGMSG, BLD]
requirements: features/pulse/requirements.md
---

# Specification: The check-ins of a person

| File                                     | Holds                                                     |
| ---------------------------------------- | --------------------------------------------------------- |
| `spec.md`                                | The components, the declarations, the errors, the plan    |
| [`spec-data.md`](spec-data.md)           | The tables of the schema `pulse`                          |
| [`spec-schedule.md`](spec-schedule.md)   | The tick, the time math, the turn-on, and the pause       |
| [`spec-round.md`](spec-round.md)         | The states of a round, its messages, a press, and a reply |
| [`spec-scenarios.md`](spec-scenarios.md) | The scenarios                                             |
| [`api.yaml`](api.yaml)                   | The routes                                                |

## 1. Summary

A new plugin `dev.maroid.pulse` asks a person short questions on Telegram at the
local times of their check-ins, and records each answer as a value of the question. A
job runs every minute for each user and fires the latest due check-in. The round
goes through the plugin messages of `TGMSG`. The deck pages of the plugin edit the
questions and the check-ins and show the values.

## 2. Coverage

| Requirement                    | Where this specification realizes it                                         |
| ------------------------------ | ---------------------------------------------------------------------------- |
| `PULSE-FR-001`                 | `spec-schedule.md` turn-on, `PULSE-DD-001`, `PULSE-SC-001`                   |
| `PULSE-FR-002`                 | `spec-schedule.md` turn-on, `spec-round.md` expiry, `PULSE-SC-003`           |
| `PULSE-FR-003`                 | `spec-schedule.md` pause and tick step 3, `PULSE-SC-004`, `PULSE-SC-005`     |
| `PULSE-FR-004`                 | Section 4.3 pages, `host.timeZone`, `PULSE-SC-006`                           |
| `PULSE-FR-005`                 | `spec-schedule.md` time math, `PULSE-DD-003`, `PULSE-SC-007`, `PULSE-SC-008` |
| `PULSE-FR-006`                 | Tick step 4, `PULSE-DD-004`, `PULSE-DD-005`, `PULSE-SC-009`, `PULSE-SC-010`  |
| `PULSE-FR-007`                 | Tick step 3, `PULSE-DD-005`, `PULSE-SC-011`                                  |
| `PULSE-FR-008`                 | `fires_after`, `PULSE-DD-004`, `PULSE-SC-012`                                |
| `PULSE-FR-009`                 | `schedule.Resolve`, `PULSE-DD-006`, `PULSE-SC-013`                           |
| `PULSE-FR-010`                 | Tick step 5, `spec-round.md` expiry, `PULSE-SC-014`                          |
| `PULSE-FR-011`                 | `spec-round.md` messages, `PULSE-SC-017`                                     |
| `PULSE-FR-012`                 | `spec-round.md` press, `PULSE-SC-018`                                        |
| `PULSE-FR-013`, `PULSE-FR-014` | `spec-round.md` press, `PULSE-DD-008`, `PULSE-SC-019`                        |
| `PULSE-FR-015`, `PULSE-FR-016` | `spec-round.md` press, `PULSE-SC-020`                                        |
| `PULSE-FR-017`                 | `spec-round.md` messages, `PULSE-SC-021`                                     |
| `PULSE-FR-018`                 | `spec-round.md` reply, `PULSE-DD-010`, `PULSE-SC-022`                        |
| `PULSE-FR-019`                 | `spec-round.md` reply, `PULSE-SC-023`                                        |
| `PULSE-FR-020`                 | `spec-round.md` reply, `PULSE-DD-011`, `PULSE-SC-024`                        |
| `PULSE-FR-021`                 | `spec-round.md` press, `PULSE-DD-009`, `PULSE-SC-025`                        |
| `PULSE-FR-022`                 | `PULSE-DD-010`, `PULSE-SC-026`                                               |
| `PULSE-FR-023`                 | `questions`, `api.yaml` `QuestionInput`, `PULSE-SC-027`                      |
| `PULSE-FR-024`                 | `PULSE-DD-014`, `PULSE-SC-028`                                               |
| `PULSE-FR-025`                 | `archived_at`, `PULSE-DD-014`, `PULSE-SC-029`                                |
| `PULSE-FR-026`                 | `choices`, `PULSE-DD-013`, `PULSE-SC-030`                                    |
| `PULSE-FR-027`                 | `checkins`, `checkin_questions`, `PULSE-SC-031`                              |
| `PULSE-FR-028`                 | `PULSE-DD-012`, `PULSE-SC-032`                                               |
| `PULSE-FR-029`                 | `catalog/defaults.go`, `PULSE-SC-033`                                        |
| `PULSE-FR-030`                 | `question_values`, `PULSE-SC-019`, `PULSE-SC-034`                            |
| `PULSE-FR-031`                 | `GET /questions`, `PULSE-SC-035`                                             |
| `PULSE-FR-032`                 | `GET /questions/{id}/values`, `PULSE-SC-036`                                 |
| `PULSE-FR-033`                 | `GET /questions/{id}/days`, `PULSE-DD-015`, `PULSE-SC-037`                   |
| `PULSE-FR-034`                 | The data page, `PULSE-DD-016`, `PULSE-SC-038`                                |
| `PULSE-NFR-001`                | `PULSE-DD-003`, `PULSE-SC-016`                                               |
| `PULSE-NFR-002`                | Tick step 1, `PULSE-SC-002`                                                  |
| `PULSE-INV-001`                | Section 4.2 scope, `PULSE-SC-039`                                            |
| `PULSE-INV-002`                | `question_values_prompt_key`, `PULSE-DD-009`, `PULSE-SC-025`                 |
| `PULSE-INV-003`                | `rounds_occurrence_key`, `PULSE-DD-007`, `PULSE-SC-015`                      |
| `PULSE-INV-004`                | `question_values_frozen`, `PULSE-DD-012`, `PULSE-SC-040`                     |

## 3. Guideline compliance

| Rule      | Guideline    | How this specification obeys it                                                   |
| --------- | ------------ | --------------------------------------------------------------------------------- |
| `PLG-003` | Plugin model | The identifier is `dev.maroid.pulse`, the schema `pulse`.                         |
| `PLG-007` | Plugin model | Pulse reaches the database, the messenger, and the bot through `pluginapi.Host`.  |
| `PLG-009` | Plugin model | Every round and every claim lives in the database. Nothing waits in memory.       |
| `DAT-001` | Data         | Eight tables in the schema `pulse`. `spec-data.md`.                               |
| `DAT-009` | Data         | Every table has `id UUID DEFAULT uuidv7()`. An insert names no `id`.              |
| `DAT-010` | Data         | Every moment is `TIMESTAMPTZ`. A local time of a check-in is `TIME` with no date. |
| `OWN-004` | Ownership    | Every table is scoped to a user, with the reason in a comment.                    |
| `OWN-006` | Ownership    | Each table forces row level security on `user_id`.                                |
| `OWN-008` | Ownership    | No repository filters by `user_id`.                                               |
| `OWN-009` | Ownership    | `tick` runs for each user. `spec-schedule.md`.                                    |
| `JOB-005` | Jobs         | `job.Tick` implements `pluginapi.CronJob`.                                        |
| `JOB-006` | Jobs         | The 60-minute window absorbs a skipped minute. `PULSE-DD-005`.                    |
| `TG-003`  | Telegram     | `/pause` and `/resume` are `pluginapi.TelegramCommand` with a permission.         |
| `TG-007`  | Telegram     | A round sends through `message.Messenger`. Pulse names no chat.                   |
| `TG-008`  | Telegram     | A button question claims no reply. `PULSE-DD-010`.                                |
| `UI-005`  | Web UI       | The four pages are routes of the remote.                                          |
| `UI-007`  | Web UI       | The pages reach the hub through `@maroid/api-client`.                             |
| `RES-001` | REST         | `api.yaml` follows the routes of `GARDEN`: pages, `ETag`, `If-Match`, problems.   |
| `BLD-003` | Build        | The user interface builds before the shared object.                               |

## 4. Design

### 4.1 Components

Every path is under `plugins/pulse/`, unless it says otherwise. Every file is new.

| Path                                      | Holds                                                                                                  |
| ----------------------------------------- | ------------------------------------------------------------------------------------------------------ |
| `main.go`, `embed.go`, `go.mod`           | `Meta`, `New`, the capabilities of section 4.3, `uiAssets`                                             |
| `db/migrations/`                          | The eight tables of `spec-data.md`                                                                     |
| `model/`                                  | `Settings`, `Question`, `ScaleConfig`, `NumberConfig`, `Choice`, `Checkin`, `Round`, `Prompt`, `Value` |
| `repository/`                             | One repository for each table, and `Values` for the summary and the days                               |
| `catalog/defaults.go`, `catalog/seed.go`  | The defaults of `PULSE-FR-029`, `Seed(ctx, tx)` for the first turn-on                                  |
| `schedule/occurrence.go`                  | `Latest(c model.Checkin, zone *time.Location, floor, now time.Time) (time.Time, bool)`, `Resolve`      |
| `settings/service.go`                     | `Service.Get`, `Service.Put`, `Service.Pause`, `Service.Resume`                                        |
| `round/service.go`                        | `Service.Fire`, `Start`, `Decline`, `Answer`, `Skip`, `Expire`                                         |
| `round/text.go`, `round/parse.go`         | The texts and the markup of `spec-round.md`, the parse of a reply                                      |
| `job/tick.go`                             | `Tick`, the `pluginapi.CronJob`                                                                        |
| `telegram/receiver.go`                    | `Receiver`, the `pluginapi.TelegramMessagePlugin` over `round.Service`                                 |
| `telegram/command/pause.go`, `resume.go`  | `/pause` and `/resume`                                                                                 |
| `handler/`                                | `settings.go`, `question.go`, `checkin.go`, `value.go`, `problem.go`                                   |
| `permission/permission.go`                | `CheckinsUse = "checkins.use"`                                                                         |
| `ui/`                                     | The remote: `src/pages/{data,checkins,questions,settings}`, `src/api/`                                 |
| `go.work.example` (root)                  | `./plugins/pulse`                                                                                      |
| `config.yaml`, `chart/values.yaml` (root) | The entry `pulse.so`, commented in the chart as jasmine is                                             |

Pulse reads no configuration. A tick slower than one minute breaks `PULSE-NFR-001`,
so its schedule is no setting. `round.Service` takes `message.Messenger` from
`Host.TelegramMessenger()`, and
`now func() time.Time` so a test fixes the clock. `job.Tick` takes the same clock.

### 4.3 Declarations

**HTTP routes.** Each path starts with
`/workspaces/{workspaceId}/plugins/dev.maroid.pulse/api`. Each one names the
permission `checkins.use`. `api.yaml` holds the bodies and the status codes.

| Method   | Path                     | Access         | Realizes                                       |
| -------- | ------------------------ | -------------- | ---------------------------------------------- |
| `GET`    | `/settings`              | `checkins.use` | `PULSE-FR-001` to `PULSE-FR-003`               |
| `PUT`    | `/settings`              | `checkins.use` | `PULSE-FR-001` to `PULSE-FR-003`               |
| `GET`    | `/questions`             | `checkins.use` | `PULSE-FR-023`, `PULSE-FR-025`, `PULSE-FR-031` |
| `POST`   | `/questions`             | `checkins.use` | `PULSE-FR-023`, `PULSE-FR-026`                 |
| `GET`    | `/questions/{id}`        | `checkins.use` | `PULSE-FR-023`, `PULSE-FR-031`                 |
| `PUT`    | `/questions/{id}`        | `checkins.use` | `PULSE-FR-024` to `PULSE-FR-026`               |
| `DELETE` | `/questions/{id}`        | `checkins.use` | `PULSE-FR-025`                                 |
| `GET`    | `/questions/{id}/values` | `checkins.use` | `PULSE-FR-032`, `PULSE-FR-034`                 |
| `GET`    | `/questions/{id}/days`   | `checkins.use` | `PULSE-FR-033`                                 |
| `GET`    | `/checkins`              | `checkins.use` | `PULSE-FR-027`                                 |
| `POST`   | `/checkins`              | `checkins.use` | `PULSE-FR-027`, `PULSE-FR-008`                 |
| `GET`    | `/checkins/{id}`         | `checkins.use` | `PULSE-FR-027`                                 |
| `PUT`    | `/checkins/{id}`         | `checkins.use` | `PULSE-FR-027`, `PULSE-FR-008`                 |
| `DELETE` | `/checkins/{id}`         | `checkins.use` | `PULSE-FR-027`                                 |

**Cron jobs.**

| Identifier | Schedule    | Acts for  | Behavior                      | Realizes                                          |
| ---------- | ----------- | --------- | ----------------------------- | ------------------------------------------------- |
| `tick`     | `* * * * *` | Each user | Fires the latest due check-in | `PULSE-FR-005` to `PULSE-FR-010`, `PULSE-NFR-001` |

**Telegram commands.** Scope: the private chat. Permission: `checkins.use`.

| Command   | Argument                        | Answer                                      | Realizes       |
| --------- | ------------------------------- | ------------------------------------------- | -------------- |
| `/pause`  | `YYYY-MM-DD`, `<n>d`, or `<n>h` | `Check-ins paused until 1 Dec 2026, 00:00.` | `PULSE-FR-003` |
| `/resume` | none                            | `Check-ins resumed.`                        | `PULSE-FR-003` |

A wrong argument answers `Send /pause with a date, 2026-12-01, or a duration, 3d or
12h.` With pulse off, both answer `Pulse is off. Turn it on in the deck.`

**Permissions.**

| Name           | Lowest role | Realizes        |
| -------------- | ----------- | --------------- |
| `checkins.use` | `viewer`    | `PULSE-INV-001` |

**Capabilities.** `migrations`, `ui`, `api`, `cron`, `permissions`,
`telegram_commands`, and `telegram_messages`.

**Pages.** `UIManifest.Routes`, in this order:

| Path         | Label     | Shows                                                                        | Realizes                         |
| ------------ | --------- | ---------------------------------------------------------------------------- | -------------------------------- |
| `/data`      | Data      | The questions, the values of one, and its chart over 7, 30, or 365 days      | `PULSE-FR-031` to `PULSE-FR-034` |
| `/checkins`  | Check-ins | The check-ins, and the form of one: name, times, weekdays, ordered questions | `PULSE-FR-027`                   |
| `/questions` | Questions | The questions, the archive, and the form of one                              | `PULSE-FR-023` to `PULSE-FR-026` |
| `/settings`  | Settings  | Turn on or off, pause until a date                                           | `PULSE-FR-001` to `PULSE-FR-003` |

Every page reads the zone of the schedule from `host.timeZone` of
`@maroid/plugin-sdk`, which the profile fills, and shows it in its header. With
`undefined`, the header reads "The schedule uses UTC" and links to the profile
(`PULSE-FR-004`). A page then passes `UTC` to the time helpers, not the zone of the
browser, so a page and the schedule agree. While pulse is off, every page but
Settings shows a panel that links to Settings. The chart is `ColumnLineChart` of
`@maroid/ui`: the columns are the count of each day, the line is the mean, or the
count of yes for a yes or no. A time shows through `formatMoment` with that zone.

### 4.5 Errors

| Condition                                                      | Behavior                                     | Problem type, under `/problems/dev.maroid.pulse/` |
| -------------------------------------------------------------- | -------------------------------------------- | ------------------------------------------------- |
| A body breaks a limit of `api.yaml`                            | 422, one item for each member                | the shared `invalid`                              |
| A change of the type of a question with a value                | 409                                          | `question-has-values`                             |
| A delete of a question with a value                            | 409                                          | `question-has-values`                             |
| A change that leaves out a choice that a value records         | 409, `detail` names the choice               | `choice-in-use`                                   |
| A choice `id` that the question does not hold                  | 422 at `/config/choices/<n>/id`              | the shared `invalid`                              |
| A check-in that names an archived or unknown question          | 422 at `/question_ids/<n>`                   | the shared `invalid`                              |
| `paused_until` in the past or more than 366 days ahead         | 422 at `/paused_until`                       | the shared `invalid`                              |
| A send of a start prompt is refused, unreachable or no account | The round expires. A warning names the user  | none                                              |
| An edit or a release fails in an expiry                        | An error log names the round. Expiry goes on | none                                              |
| A tick fails on the database                                   | The job answers the error (`JOB-007`)        | none                                              |

A log record names the user, the check-in, and the round. It never carries a value.

## 5. Design decisions

`spec-data.md` holds `PULSE-DD-012` to `PULSE-DD-014` and `PULSE-DD-018`, `spec-schedule.md` holds
`PULSE-DD-003` to `PULSE-DD-007`, and `spec-round.md` holds `PULSE-DD-008` to
`PULSE-DD-011`.

### `PULSE-DD-001`

**Realizes:** `PULSE-FR-001`, `PULSE-FR-002`
**Decision:** A person turns pulse on with `PUT /settings`, and the first turn-on
seeds the defaults. The tick serves a person whose row says `enabled`.
**Rationale:** A job for each user runs for every active user and reads no
enablement of a workspace (`OWN-009`). The row is the consent, and it needs no change
of the hub.
**Alternatives:** A seed at the first visit: a person who only looks gets messages. A
gate on the enablement of a workspace: a new host service and an ADR.

### `PULSE-DD-002`

**Realizes:** `PULSE-INV-001`
**Decision:** One permission, `checkins.use`, at `viewer`, guards every route and
command.
**Rationale:** Each record belongs to the person who acts, so a role in the workspace
grants nothing over another person. A viewer keeps a diary of their own.
**Alternatives:** A read and a write permission: an editor rule over records that no
other member reads.

### `PULSE-DD-015`

**Realizes:** `PULSE-FR-033`
**Decision:** `GET /questions/{id}/days` groups in SQL by
`(occurred_at AT TIME ZONE <zone>)::date`, with the zone of
`pluginapi.ActingTimeZone(ctx)`, over at most 366 days.
**Rationale:** A year of four values a day is 1460 rows. One grouped read answers 366
rows, and the day of a value follows the zone of the person, not the browser.
**Alternatives:** The browser groups the page of values: 15 pages of 100 for a year.

### `PULSE-DD-016`

**Realizes:** `PULSE-FR-034`
**Decision:** A location links to
`https://www.openstreetmap.org/?mlat=<lat>&mlon=<lon>#map=16/<lat>/<lon>`, which
opens in a new tab.
**Rationale:** A link loads nothing into the deck and needs no key. The page shows
the coordinates with five decimals.
**Alternatives:** An embedded map: tiles from a third party on every load of the page.

### `PULSE-DD-017`

**Realizes:** `PULSE-FR-001`
**Decision:** With no row of settings, `GET /settings` answers `enabled` false with
the entity tag `"0"`, and `PUT /settings` with `If-Match: "0"` creates the row.
**Rationale:** The page reads and writes one resource, with `If-Match` on every write
as `RES` asks, and a read writes nothing.
**Alternatives:** A `POST` to create the row: a second route for one boolean.

## 7. Build plan

| #   | Step                                                                        | Realizes                                                           | Done |
| --- | --------------------------------------------------------------------------- | ------------------------------------------------------------------ | ---- |
| 1   | The module, `Meta`, the migrations, the models, the repositories            | `PULSE-INV-001` to `PULSE-INV-004`                                 | [ ]  |
| 2   | The catalog, the seed, and the settings service                             | `PULSE-FR-001`, `PULSE-FR-002`, `PULSE-FR-029`                     | [ ]  |
| 3   | `schedule.Latest` and `Resolve`, with the unit tests first                  | `PULSE-FR-005` to `PULSE-FR-009`                                   | [ ]  |
| 4   | `round.Service`, the receiver, and `job.Tick`                               | `PULSE-FR-010` to `PULSE-FR-022`, `PULSE-NFR-001`, `PULSE-NFR-002` | [ ]  |
| 5   | `/pause` and `/resume`                                                      | `PULSE-FR-003`                                                     | [ ]  |
| 6   | The routes of `api.yaml`                                                    | `PULSE-FR-023` to `PULSE-FR-034`                                   | [ ]  |
| 7   | The pages Settings, Questions, and Check-ins                                | `PULSE-FR-001` to `PULSE-FR-004`, `PULSE-FR-023` to `PULSE-FR-028` | [ ]  |
| 8   | The page Data                                                               | `PULSE-FR-031` to `PULSE-FR-034`                                   | [ ]  |
| 9   | `go.work.example`, `config.yaml`, the chart, and a run on the compose stack | `PULSE-NFR-001`                                                    | [ ]  |

## 8. Out of scope for this specification

Nothing. Section 3 of the requirements holds what pulse does not do.

## Retired identifiers

This file has no retired identifier.
