---
id: ADR-0027
title: One outbox runs every background task of the hub
type: adr
status: accepted
created: 2026-11-23
updated: 2026-11-23
decided: 2026-11-23
changes: [JOB-009, JOB-010, OWN-003]
supersedes:
superseded_by:
---

# One outbox runs every background task of the hub

## Context

`ADR-0021` gave `maroid worker` an outbox, and `FILES-DD-005` built it as
`public.file_tasks`. Its columns name a file, a scope, an object key, and one of five
actions of a file. The `files` worker leases from it and calls one function for each
action.

The removal of a workspace needs a second kind of task: the destruction of a transit
key. The key names no file and no object. A second table and a
second worker would copy the lease, the retry, the scope, and the poll of the first.
Each later kind of task would copy them again.

## Decision

One table, `public.outbox_tasks`, holds every task of the hub. A row names a type, a
JSON payload, and the owner of the task when it has one. One worker, `outbox`, leases
the due tasks and calls the handler that the type names. The files become five handlers
of it.

## Rationale

The lease, the retry, the owner scope, and the poll live in one place. A new kind of
task adds a type and a handler, and no table and no worker. The owner stays in two
columns and not in the payload, so the worker sets the acting scope before any handler
runs, as `ADR-0021` demands, and no handler can forget it. A unique `dedupe_key`
replaces the query of `EnqueueOnce` on the columns of a file, so any type can ask for
one task at a time.

The behavior of the files stays the same: the same poll of 2 seconds, the same lease of
5 minutes, the same limit of 4 minutes for a task, and the same backoff.

## Alternatives

| Alternative                                     | Why we did not select it                                                                                       |
| ----------------------------------------------- | -------------------------------------------------------------------------------------------------------------- |
| One table and one worker for each kind of task. | Each copies the lease, the retry, the scope, and the poll. A defect in one copy stays in the others.           |
| The owner inside the payload.                   | Each handler sets the scope itself. A handler that forgets it acts with no scope.                              |
| A queue outside PostgreSQL.                     | The change and its task no longer commit in one transaction, which is the reason for the outbox of `ADR-0021`. |

## Consequences

### Rules that change

`JOB` gains two rules:

> **JOB-009.** The hub runs a task that a change starts through one outbox,
> `public.outbox_tasks`. The change and its task commit in one transaction. A row names
> a type, a JSON payload, and the owner of the task: a workspace, a user, or none. A
> payload holds identifiers and no content. The `outbox` worker leases each due task
> with `FOR UPDATE SKIP LOCKED`, sets the owner as the acting scope, and calls the
> handler of the type. A task runs at least once, so every handler is idempotent. A
> failed task waits 2 seconds, then twice as long, up to 5 minutes, with no limit on
> the attempts.
>
> **JOB-010.** A type names a domain of the hub and an action, as `file.purge`. One
> handler serves one type, and the registry rejects a second handler of a type. A task
> whose type has no handler fails and retries, so a worker of an older build leaves it
> for a newer one.

`OWN-003` changes the row of the outbox in its table:

> | A task of the outbox | The owner of the task, when a user owns it. | The owner of the task, when a workspace owns it. |

### Specifications that change

| Document                           | Change                                                                                                                                                                                               |
| ---------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `features/files/spec-decisions.md` | `FILES-DD-005` cites `JOB-009`. The worker is `outbox`, and it reads `outbox.poll_interval` and `outbox.concurrency`.                                                                                |
| `features/files/spec.md`           | Section 4.1 replaces `public.file_tasks`, `model.FileTask`, `repository.FileTask`, and `file.Worker` with the outbox and five handlers. Section 4.2 drops `public.file_tasks`. The keys of 4.3 move. |
| `features/files/spec-scenarios.md` | `FILES-SC-035` names `outbox.concurrency` and `outbox.poll_interval`.                                                                                                                                |

### The design

```sql
-- A queue of the hub, so a shared table: the worker reads it with no scope, and runs
-- each task in the scope of the owner that the row names.
CREATE TABLE public.outbox_tasks
(
    id           UUID        NOT NULL PRIMARY KEY DEFAULT uuidv7(),
    type         TEXT        NOT NULL,
    payload      JSONB       NOT NULL DEFAULT '{}'::jsonb,
    owner_scope  TEXT,
    owner_id     UUID,
    dedupe_key   TEXT,
    run_after    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    locked_until TIMESTAMPTZ,
    attempts     INTEGER     NOT NULL DEFAULT 0,
    last_error   TEXT,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT outbox_tasks_type_check CHECK (type ~ '^[a-z][a-z0-9_]*(\.[a-z][a-z0-9_]*)+$'),
    CONSTRAINT outbox_tasks_owner_scope_check CHECK (owner_scope IN ('workspace', 'user')),
    CONSTRAINT outbox_tasks_owner_check CHECK ((owner_scope IS NULL) = (owner_id IS NULL))
);

CREATE INDEX outbox_tasks_run_after ON public.outbox_tasks (run_after);
CREATE UNIQUE INDEX outbox_tasks_dedupe_key ON public.outbox_tasks (dedupe_key)
    WHERE dedupe_key IS NOT NULL;
```

The migration moves each row of `public.file_tasks` into the new table, then drops the
old one. The type is `file.` followed by the action. The payload holds `file_id` and
`object_key`. The scope and the owner move to `owner_scope` and `owner_id`. A
`process` task takes the dedupe key `file.process:<file id>`, which is the one task
that `EnqueueOnce` guards today. The down migration reverses each step.

| Path                                          | Holds                                                                                                                            |
| --------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------- |
| `apps/hub/internal/outbox/task.go`            | `Task`, `Draft`, `Owner`, and `Handler`, with `Type() string` and `Handle(ctx context.Context, task Task) error`                 |
| `apps/hub/internal/outbox/registry.go`        | `Registry`: one handler for each type                                                                                            |
| `apps/hub/internal/outbox/worker.go`          | `Worker`, named `outbox`: the lease, the scope, the dispatch, the retry. `Prepare` calls `Prepare` of each handler that has one. |
| `apps/hub/internal/repository/outbox_task.go` | `OutboxTask`: `Enqueue`, `Lease`, `Done`, `Retry`, `Drop`                                                                        |
| `apps/hub/internal/file/task.go`              | The five handlers: `file.process`, `file.check`, `file.discard_upload`, `file.expire`, `file.purge`                              |
| `apps/hub/internal/config/config.go`          | `Outbox`, with `poll_interval` (default `2s`) and `concurrency` (default `2`). `Files` drops both.                               |

`Enqueue` takes a draft with a dedupe key with `ON CONFLICT DO NOTHING`, so a second
draft with the same key adds no row while the first task exists.

### Result

- Positive: a new kind of task needs a type and a handler, and nothing else.
- Positive: the scope of a task is set in one place.
- Negative: `files.poll_interval` and `files.concurrency` move to `outbox.*`. A
  configuration that sets the old keys loses them, and the defaults apply.
- Negative: `maroid worker --workers files` becomes `--workers outbox`.
- Neutral: the refactor keeps the behavior of the files. It runs in the fast lane, and
  every scenario of `FILES` passes with no change of its text, apart from the keys of
  `FILES-SC-035`.
- Work that follows: the migration, the package `outbox`, the handlers of the files, and
  the configuration, before the removal of a workspace builds on the outbox.
