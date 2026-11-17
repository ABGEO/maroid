---
id: ADR-0021
title: The worker runs the tasks of an outbox of the hub
type: adr
status: accepted
created: 2026-11-18
updated: 2026-11-18
decided: 2026-11-18
changes: [ARC-010, OWN-003, OWN-004]
supersedes:
superseded_by:
---

# The worker runs the tasks of an outbox of the hub

## Context

`FILES-FR-009` makes Maroid process the bytes of a file after the upload ends, while the
person works on. `FILES-NFR-001` gives 30 seconds from the end of the upload to the state
`ready`. `FILES-FR-019` and `FILES-NFR-005` add two more pieces of work that no request
carries: the delete of an unclaimed file after 24 hours, and the removal of the bytes of a
deleted file within one hour.

Three rules stand against each path that does this work today:

| Rule      | What it says                                                           | Why the work does not fit                                                                                        |
| --------- | ---------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------- |
| `ARC-010` | `maroid worker` runs the cron jobs and the MQTT subscribers.           | A cron job runs at most one time each minute. A photo then waits up to 60 seconds before the work starts.        |
| `OWN-003` | The table lists each entry point and the acting scope that it carries. | A unit of work that a queued task starts is no entry point of the table, so it carries no acting scope.          |
| `OWN-004` | A shared table holds a row that belongs to nobody.                     | The row of a task names the workspace or the user that owns its file, and the worker that reads it has no scope. |

`FILES-DD-005` selects an outbox: the change of a file and its next task commit in one
transaction, and a worker leases each due task.

## Decision

`maroid worker` gains a worker that runs the tasks of an outbox of the hub. A task names
its owner, and the worker sets that owner as the acting scope before the first statement
of the task. The outbox is a shared table whose row holds identifiers and no content.

## Rationale

The outbox joins the change and its task in one transaction, so no crash leaves a file
without its next step, and no task exists without its change. A lease with
`FOR UPDATE SKIP LOCKED` lets two workers share the table, and no task runs two times at
once. A poll of 2 seconds leaves 28 seconds of `FILES-NFR-001` to the work itself.

The task runs under the policy of its owner, as every other unit of work does. The
worker reads the shared table with no scope, and then reaches a scoped table only through
`database.WithScopeTx`. No policy crosses a scope, and `OWN-006` stays as it is.

The row of a task holds the identifier of the file, the kind of its owner, the
identifier of the owner, the key of its objects, and the action. A reader of the table
learns that a file exists, and nothing of its content. `ADR-0014` faced the same lack of
scope for a job, and chose a run for each workspace. A run for each workspace every 2
seconds costs one transaction for each workspace, also for a workspace that holds no task.

## Alternatives

| Alternative                                                | Why we did not select it                                                                                                       |
| ---------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------ |
| A cron job each minute, which `ADR-0014` already permits   | A photo waits up to 60 seconds before the work starts. `FILES-NFR-001` fails.                                                  |
| An event of the bucket for each new object                 | It binds the hub to the event setup of a file store that other projects share, and it covers neither the expiry nor the purge. |
| A goroutine of `maroid serve http` that runs the work      | It breaks the split of `ARC-010`: a slow decode of 48 megapixels then competes with the requests of every person.              |
| One outbox table for each scope, scoped as the file tables | The worker then needs a run for each workspace and each user to see a task, as in `ADR-0014`, every 2 seconds.                 |

## Consequences

### Rules that change

`ARC-010` changes. The new table:

> | Command             | Runs                                                              |
> | ------------------- | ----------------------------------------------------------------- |
> | `maroid serve http` | The HTTP API and the Telegram webhook.                            |
> | `maroid worker`     | The cron jobs, the MQTT subscribers, and the tasks of the outbox. |

`OWN-003` gains one row in its table:

> | A task of the outbox | The owner of the task, when a user owns its file. | The owner of the task, when a workspace owns its file. |

and one sentence under the table:

> A task of the outbox names its owner. The worker sets it with `OWN-007` before the
> first statement of the task.

`OWN-004` gains one sentence at the end:

> A queue of the hub is a shared table too. Its row names an owner and holds no content,
> and the work that it starts runs in the scope of that owner. See `OWN-003`.

### Specifications to examine

| Document                  | Why                                                                                    |
| ------------------------- | -------------------------------------------------------------------------------------- |
| `features/files/spec.md`  | `FILES-DD-005` waits for this record. Section 3 marks `ARC-010` and `OWN-003` as open. |
| `features/health/spec.md` | Confirm that the readiness of `maroid worker` needs no change for a new worker.        |

### Result

- Positive: a photo starts its work within 2 seconds of the end of the upload.
- Positive: a crash between a change and its task cannot happen, because both commit in
  one transaction.
- Negative: the worker polls the table every 2 seconds, also when it holds no task.
- Negative: a shared table now holds a row that names an owner. A defect in the worker
  that skips `database.WithScopeTx` would act with no scope, and `OWN-006` then hides
  every scoped row from it, so the defect fails closed.
- Neutral: the chart runs `maroid worker` with no `--workers` flag, so it runs the new
  worker with no change.
- Work that follows: the migration of `public.file_tasks`, `file.Worker`, and the change
  of `command/worker.go`, in the build plan of `FILES`.
