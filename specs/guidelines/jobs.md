---
id: JOB
title: Background jobs
type: guideline
status: active
created: 2026-09-11
updated: 2026-09-11
scope: [apps/hub/internal/worker/, apps/hub/internal/command/worker.go, plugins/*/job/]
related: [ARC, PLG, MQT, OWN]
---

# Background jobs

A job runs with no request, so it declares the user that it acts for. See `OWN-009`.

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

## Retired identifiers

This file has no retired identifier.
