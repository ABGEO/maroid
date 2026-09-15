---
id: CLI
title: The command line
type: guideline
status: active
created: 2026-09-11
updated: 2026-09-15
scope: [apps/hub/cmd/, apps/hub/internal/command/, apps/hub/internal/commander/]
related: [LIF, PLG, JOB, ARC, OWN]
---

# The command line

The library is cobra.

## CLI-001

The root command is `maroid`. The tree is:

```
maroid serve http       The HTTP API and the Telegram webhook.
maroid worker           The background workers. See JOB-004.
maroid migrate up       The migrations. See DAT-005.
maroid user invite      A user record and an invitation for it. See OWN-002.
```

## CLI-002

A command implements `commander.Commander`, which gives `Command() *cobra.Command`.

## CLI-003

A command struct holds the resolver, and a logger with the attributes
`component=command` and `command=<name>`. See `LOG-003`.

## CLI-004

`PreRunE` validates and prepares. `RunE` performs the work.
A command returns an error. A command does not call `os.Exit`.

**Why:** The root command owns the exit code and the cleanup.

## CLI-005

A plugin adds a command through `CommandRegistry`.
The registry rejects a duplicate name. See `PLG-011`.

## CLI-006

A command takes the context from cobra with `cmd.Context()`.
It passes that context to everything it starts.

## Retired identifiers

This file has no retired identifier.
