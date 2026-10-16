---
id: GARDEN
title: The care of the next three days
type: spec
status: approved
created: 2026-10-16
updated: 2026-10-16
approved_by: Temuri
approved_on: 2026-10-16
constrained_by: [RES, OWN, UI, TS, TST]
requirements: features/garden/requirements-upcoming.md
---

# Specification: The care of the next three days

## 1. Summary

Every care interval gains `due_in`, the count of UTC days until it becomes due.
`GET /care-intervals` gains the filter `due_within`. The page `/attention` reads
`due_within=3` and shows the upcoming list after the attention list. No table changes,
and no change reaches the hub, `pluginapi`, the reminder, the bot command, or the widget.
The name of the plant on each line of both lists links to the plant page.

## 2. Guideline compliance

| Rule                 | Guideline      | How this specification obeys it                                                      |
| -------------------- | -------------- | ------------------------------------------------------------------------------------ |
| `RES-005`, `RES-006` | REST           | The upcoming list is a page of `GET /care-intervals`. The cursor carries the filter. |
| `OWN-006`, `OWN-008` | Ownership      | The policy keeps the rows of the acting workspace. The filter names no workspace.    |
| `UI-007`             | Web UI         | The page calls `host.api` through `createJasmineApi`.                                |
| `TS-002`             | Frontend style | The page keeps its runes.                                                            |

## 3. Design

### 3.1 Components

Every path is under `plugins/jasmine/`, except `api.yaml`.

| Path                                      | Action | Holds                                                                                                                                                                                                                       |
| ----------------------------------------- | ------ | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `repository/care_interval.go`             | change | `selectCareIntervals` gains `CROSS JOIN LATERAL (SELECT ci.days - counted.days_since AS due_in) upcoming`. `CareIntervalFilter` gains `DueWithin int`, and zero keeps every care interval. `List` applies it (section 3.3). |
| `model/care_interval.go`                  | change | `CareInterval.DueIn int` with the tag `db:"due_in"`.                                                                                                                                                                        |
| `dto/care_interval.go`                    | change | `CareIntervalResponse.DueIn int` with the tag `json:"due_in"`.                                                                                                                                                              |
| `handler/care_interval.go`                | change | `filterDueWithin = "due_within"`. `List` adds it to `page.Options.Filters`. `checkCareFilters` refuses a value that is not an integer from 1 to 365. `careFilter` sets `DueWithin`.                                         |
| `handler/care_test.go`                    | change | `GARDEN-SC-043` to `GARDEN-SC-045`.                                                                                                                                                                                         |
| `handler/measure_test.go`                 | change | `GARDEN-SC-046`.                                                                                                                                                                                                            |
| `ui/src/api/types.ts`                     | change | `CareInterval.due_in: number`. `CareIntervalFilter.due_within?: number`.                                                                                                                                                    |
| `ui/src/api/careIntervals.ts`             | change | `list` passes `due_within`.                                                                                                                                                                                                 |
| `ui/src/pages/attention/Attention.svelte` | change | Section 3.4.                                                                                                                                                                                                                |
| `specs/features/garden/api.yaml`          | change | The parameter `due_within` of `listCareIntervals`. The member `due_in` of `CareInterval`.                                                                                                                                   |

### 3.2 Declarations

**HTTP routes.** The route of `spec.md` section 4.3 changes. It adds no route.

| Method | Path              | Permission    | Change                                                                 | Realizes                          |
| ------ | ----------------- | ------------- | ---------------------------------------------------------------------- | --------------------------------- |
| `GET`  | `/care-intervals` | `plants.read` | The filter `due_within`. The member `due_in` on every item it answers. | `GARDEN-FR-039`, `GARDEN-NFR-004` |

`GET /plants/{plantId}/care-intervals` and the `PUT` answer `due_in` too, since they
answer the same schema.

### 3.3 The filter

`due_in` is `days - days_since`. It is 1 on the day before the care interval becomes
due, 0 on the day it becomes due, and negative after. It is an integer of UTC days, so
the label of `GARDEN-FR-041` needs no date of the browser.

`List` adds this condition to its `WHERE`:

```sql
AND (CAST(:due_within AS integer) IS NULL
     OR (standing.state IS NOT NULL
         AND upcoming.due_in BETWEEN 1 AND CAST(:due_within AS integer)))
```

`standing.state IS NOT NULL` drops the inactive plants. A `due_in` of 1 or more holds
only for `on_schedule`, so the filter needs no state. With `state` as well, both apply:
`state=due&due_within=3` answers an empty page.

| `due_within`                                | Answer                                 |
| ------------------------------------------- | -------------------------------------- |
| Absent                                      | No condition.                          |
| An integer from 1 to 365                    | The condition above.                   |
| `0`, `366`, `-1`, `3.5`, `abc`, or an empty | 400, `/problems/http/request-invalid`. |

The detail is "The parameter due_within must be an integer from 1 to 365."

### 3.4 The page

`load` reads both lists with `Promise.all`: `readAll` over `state: ['due', 'overdue']`,
and `readAll` over `due_within: 3`. A failure of either shows the alert of the page that
exists, "Failed to load the attention list.", with its retry. A "Done" on a line of the
attention list runs `load`, so the two lists stay one read.

After the groups of the attention list, the page shows a `section` with the heading
"Next 3 days". The section shows in every state of `ready`, also when the attention
list is empty.

| Part      | Content                                                                                                                                                          |
| --------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Order     | `due_in` ascending, then the number of `plant.environment.display_id`, then the number of `plant.display_id`, then the index of the type in `MAINTENANCE_TYPES`. |
| A line    | `plant.display_id`, `plant.common_name`, then `typeLabel(type)`, `every formatDays(days)`, then the label.                                                       |
| The label | `due_in` 1 answers "tomorrow". 2 answers "in 2 days". 3 answers "in 3 days". `dueLabel(dueIn: number): string` in `ui/src/lib/format.ts` holds it.               |
| No action | A line has no button and no link that writes.                                                                                                                    |
| Empty     | `<div class="empty-state">No care in the next 3 days.</div>`                                                                                                     |

The name of the plant on a line of either list is a link with the class
`link link-hover font-medium`, as in the plant list. Its `href` is
``host.href(`/plants/${care.plant.id}`)``, which the route `/plants/:id` of `GARDEN-DD-011`
answers. The "Done" button stays outside the link.

The lead of the page reads "The care that is due or overdue, by environment, then the
care of the next 3 days."

## 4. Design decisions

### `GARDEN-DD-015`

**Realizes:** `GARDEN-FR-039`, `GARDEN-NFR-004`
**Decision:** `GET /care-intervals` takes `due_within`, and the database applies it. Every care interval answers `due_in`.
**Rationale:** The workspace of `GARDEN-SC-032` holds 4000 care intervals. A filter in the browser over `state=on_schedule` reads up to 40 pages of 100. The filter reads the few rows of the next 3 days in one page.
**Alternatives:** A filter in the browser. It risks `GARDEN-NFR-004` on the page. A fourth care state `upcoming`. It breaks the three states of `GARDEN-FR-021`. A resource of its own. `RES-002` holds no deviation for it.

### `GARDEN-DD-016`

**Realizes:** `GARDEN-FR-040`, `GARDEN-FR-041`
**Decision:** The route stays sorted by `id`. The page reads every page and sorts. The hub answers `due_in`, and the page builds the label from it.
**Rationale:** `GARDEN-DD-008` sorts the attention list the same way, with no change to `libs/rest/page`. A label from `due_in` follows the UTC count of `GARDEN-FR-019`. A label from a date and the clock of the browser differs between 00:00 and 04:00 at UTC+4.
**Alternatives:** A sort field `due_in` on the route. `libs/rest/page` seeks by `id` alone, so a second sort needs the subquery of `GARDEN-DD-008` for one page. A member `due_on` with a date. The browser then compares it with its own date.

## 5. Scenarios

`spec-scenarios.md` holds `GARDEN-SC-043` to `GARDEN-SC-049`. `SPC-001`.

## Retired identifiers

This file has no retired identifier.
