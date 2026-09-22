---
id: LOG
title: Logging
type: guideline
status: active
created: 2026-09-11
updated: 2026-09-22
scope: [apps/hub/internal/logger/, apps/, libs/, plugins/]
related: [ERR, LIF, SEC, LNG]
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

The access record of `LOG-010` is the one exception. Its library builds the message
and offers no option to change it.

## LOG-008

Never log a token, a password, a secret, or a full request body that can hold one.

## LOG-009

A log record of an HTTP request carries the `request` attribute. It holds the
identifier that `ERR-006` gives, as the bare UUID and without the URN prefix.

**Why:** A person reports the identifier that the response carries. The attribute
is the way back to the cause.

## LOG-010

The hub writes one record for each request that it answers. `go-chi/httplog`
writes it, through the logger that `LOG-001` gives, so the record carries the
attribute of `LOG-009` like every other:

```json
{
  "msg": "GET /ping => HTTP 200 (3µs)",
  "component": "middleware",
  "middleware": "access",
  "http.request.method": "GET",
  "url.path": "/ping",
  "http.response.status_code": 200,
  "event.duration": 3000,
  "request": "01a0c611-c3d9-710d-84de-7df920aa9a5f"
}
```

| Setting              | Value        | Reason                                                     |
| ---------------------- | -------------- | ------------------------------------------------------------ |
| `Schema`             | `SchemaECS`  | One name for each attribute. `SchemaOTEL` replaces it when a deployment collects traces. |
| `RecoverPanics`      | `false`      | It answers a bare 500, and `ERR-001` asks every failure for a problem. The recoverer of the hub answers instead. |
| `LogRequestBody`     | Absent       | A body of the hub carries a token and a secret. `LOG-008`.  |
| `LogResponseBody`    | Absent       | The same reason.                                            |
| `Level`              | `Debug`      | The logger holds the level of the configuration, so this one filters nothing of its own. |

The message of this record is the one exception to `LOG-007`. `httplog` builds it
by concatenation, as `GET /plugins => HTTP 200 (1.2ms)`, and no option changes it.
Every attribute of the record still carries the value that the message repeats.

**Why:** One library writes the access record, and one field moves every attribute
of it to the OpenTelemetry names. A hand written middleware carries neither.

## Retired identifiers

This file has no retired identifier.
