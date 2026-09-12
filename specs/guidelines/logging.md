---
id: LOG
title: Logging
type: guideline
status: active
created: 2026-09-11
updated: 2026-09-11
scope: [apps/hub/internal/logger/, apps/, libs/, plugins/]
related: [LIF, SEC, LNG]
---

# Logging

## LOG-001

The logger is `log/slog`. The hub builds one logger at the start.
A plugin receives it from `Host.Logger()`. A plugin creates no logger of its own.

## LOG-002

The output goes to stdout.
The format is JSON, or text when `logger.format` says so.
The handler adds the source position only when `env` is `dev`.

**Why:** A container collects stdout. A file needs a volume and rotation.

## LOG-003

A component adds two attributes with `With`: `component`, and a second attribute
that names the instance.

```go
logger.With(
    slog.String("component", "worker"),
    slog.String("worker", "cron"),
)
```

The values in use are `command`, `handler`, `middleware`, `migrator`, `worker`,
`notifier-dispatcher`, and `telegram-updates-handler`.

## LOG-004

A plugin logger adds `plugin`, `plugin_version`, and `plugin_api_version`.

## LOG-005

An error is an attribute, `slog.Any("error", err)`.
Never put an error inside the message text.

**Why:** A collector can filter and count an attribute. It cannot parse a sentence.

## LOG-006

Use `InfoContext`, `ErrorContext`, or the matching variant when a context exists.

## LOG-007

A message is lowercase. It names the event, not the value.
It obeys `LNG`.

## LOG-008

Never log a token, a password, a secret, or a full request body that can hold one.

## Retired identifiers

This file has no retired identifier.
