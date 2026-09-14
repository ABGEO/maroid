---
id: API
title: The HTTP API
type: guideline
status: active
created: 2026-09-11
updated: 2026-09-14
scope: [apps/hub/internal/server/, apps/hub/internal/handler/, apps/hub/internal/middleware/]
related: [ARC, PLG, SEC, UI]
---

# The HTTP API

## API-001

The hub serves one HTTP API. The router is chi.

## API-002

The router applies this middleware, in this order:
`RealIP`, `Logger`, `Recoverer`, `StripSlashes`, and the JSON content type.
CORS follows when `cors.enabled` is true.

## API-003

The path prefixes are fixed:

| Prefix                    | Serves                              | Access                 |
| ------------------------- | ----------------------------------- | ---------------------- |
| `/auth`, `/auth/callback` | The login flow                      | Public                 |
| `/auth/me`                | The current user                    | Authenticated          |
| `/plugins`                | The list of loaded plugins          | Authenticated          |
| `/plugins/{id}/api/*`     | The routes of a plugin              | Authenticated          |
| `/plugins/{id}/settings*` | The settings of a user for a plugin | Authenticated          |
| `/plugins/{id}/ui/*`      | The assets of a plugin              | Public. See `SEC-006`. |
| `/telegram/webhook`       | The Telegram updates                | Network allowlist      |
| `/ping`                   | The health check                    | Public                 |

## API-004

A plugin declares a route as `pluginapi.Route` with a method, a pattern, and a handler.
The hub mounts every route of a plugin under `/plugins/{id}/api`.
A plugin does not choose its own prefix.

**Why:** The prefix comes from the plugin identifier, so two plugins cannot collide.

## API-005

A handler implements `handler.Handler` and registers itself on a `chi.Router`.
A handler function returns an error. `handler.Wrap` logs the error.

## API-006

A response body is JSON.
An error response carries the HTTP status and a JSON body that names the reason.

## Retired identifiers

This file has no retired identifier.
