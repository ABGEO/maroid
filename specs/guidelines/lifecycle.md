---
id: LIF
title: Process lifecycle
type: guideline
status: active
created: 2026-09-11
updated: 2026-09-11
scope: [apps/hub/internal/app/, apps/hub/internal/depresolver/, apps/hub/internal/command/]
related: [ARC, DEP, CLI, JOB, LOG]
---

# Process lifecycle

`DEP` gives the container that builds a dependency. This file gives the order in
which the process starts and stops.

## LIF-003

The start order is fixed:

1. Read and validate the configuration.
2. Build the logger.
3. Load the enabled plugins.
4. Build the command tree.
5. Run the selected command.

Step 3 comes before step 4 because a plugin adds a command. See `PLG-010`.

## LIF-004

A process that runs more than one long-lived service runs them in an `errgroup`.
The first error cancels the group.

## LIF-005

A cancelled context starts the shutdown.
The process stops the services in the reverse order of the start.

## LIF-006

A shutdown step runs with a context that the cancellation does not reach,
and with a timeout of 10 seconds.

```go
ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), shutdownTimeout)
```

**Why:** The context that triggered the shutdown is already cancelled.
A step that uses it cannot finish its work.

## LIF-007

`http.ErrServerClosed` and `context.Canceled` mean a clean stop.
Do not report them as a failure.

## LIF-008

A shutdown step logs its own error and does not stop the next step.
`Close` joins the errors with `errors.Join`.

**Why:** One resource that fails to close must not leak the others.

## Retired identifiers

| ID        | Retired    | Reason              |
| --------- | ---------- | ------------------- |
| `LIF-001` | 2026-09-11 | Moved to `DEP-001`. |
| `LIF-002` | 2026-09-11 | Moved to `DEP-005`. |
