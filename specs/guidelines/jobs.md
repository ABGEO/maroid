---
id: JOB
title: Background jobs
type: guideline
status: active
created: 2026-09-11
updated: 2026-11-23
scope:
  [
    apps/hub/internal/worker/,
    apps/hub/internal/outbox/,
    apps/hub/internal/command/worker.go,
    plugins/*/job/
  ]
related: [ARC, PLG, MQT, OWN]
---

# Background jobs

A job runs with no request, so it declares the scope that it runs in. See `OWN-009`.

## JOB-001

A background worker implements `worker.Worker` with `Name`, `Prepare`, `Start`, and `Stop`.
The name is stable and lowercase, for example `cron` or `mqtt`.

## JOB-002

The hub prepares every worker before it starts any worker.

**Why:** A configuration error must stop the process before a job runs.

## JOB-003

`Start` blocks until the context is cancelled.
`Stop` finishes inside the deadline of the context that it receives.

## JOB-004

`maroid worker --workers cron,mqtt` selects the workers.
The value `all` runs every worker.

## JOB-005

A plugin declares a cron job as `pluginapi.CronJob`.
The metadata gives an identifier and a schedule.
`Run` receives a context and returns an error.

## JOB-006

The scheduler skips a run when the previous run of the same job is still active.

**Why:** A slow job must not run against itself.

## JOB-007

The worker logs an error that a job returns. The error does not stop the scheduler
and does not stop another job.

## JOB-008

A registry rejects a second job with an identifier that it already holds.

## JOB-009

The hub runs a task that a change starts through one outbox, `public.outbox_tasks`.
The change and its task commit in one transaction. A row names a type, a JSON payload,
and the owner of the task: a workspace, a user, or none. A payload holds identifiers
and no content. The `outbox` worker leases each due task with `FOR UPDATE SKIP LOCKED`
every `outbox.poll_interval`, runs at most `outbox.concurrency` at one time, sets the
owner as the acting scope, and calls the handler of the type. A task runs at least
once, so every handler is idempotent. A failed task waits 2 seconds, then twice as
long, up to 5 minutes, with no limit on the attempts. `ADR-0027`.

**Why:** The change and its task cannot part, and the lease, the retry, and the scope
live in one place for every kind of task.

## JOB-010

A type names a domain of the hub and an action, as `file.purge`. One handler serves
one type, and the registry rejects a second handler of a type. A task whose type has no
handler fails and retries, so a worker of an older build leaves it for a newer one.

## Retired identifiers

This file has no retired identifier.
