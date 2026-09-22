---
id: API
title: The HTTP API
type: guideline
status: active
created: 2026-09-11
updated: 2026-09-22
scope: [apps/hub/internal/server/, apps/hub/internal/handler/, apps/hub/internal/middleware/]
related: [ARC, ERR, PLG, SEC, UI]
---

# The HTTP API

## API-001

The hub serves one HTTP API. The router is chi.

## API-002

The router applies this middleware, in this order:
the request identifier, the real address, the access log, the recoverer,
`StripSlashes`, and the JSON content type. CORS follows when `cors.enabled` is true.

The request identifier comes first, so every later middleware and every log record
reaches it. `ERR-006` gives the value and the generator.

The real address and the recoverer are of the hub, not of chi. `SEC-009` gives the
reason for the first, and `ERR-001` for the second. The access log is
`go-chi/httplog`, and `LOG-010` gives what it writes.

## API-007

A 405 names the methods that the route holds, in the `Allow` header, because
RFC 9110 asks every 405 for one. Chi builds that header in its own responder and
drops it as soon as a router sets one of its own, so the router of the hub builds
it again.

A 401 of a web route carries no `WWW-Authenticate` header. RFC 9110 asks for one,
and no scheme of that header describes the cookie that `SEC-005` gives. `/mcp` is
the one route that carries it, because a bearer token has a scheme.

**Why:** A client reads `Allow` to learn what the route takes. A header that names
a scheme nobody implements teaches a client nothing.

## API-003

The path prefixes are fixed:

| Prefix                    | Serves                                    | Access                 |
| ------------------------- | ----------------------------------------- | ---------------------- |
| `/auth`, `/auth/callback` | The login flow                            | Public                 |
| `/auth/invite`            | The redemption of an invitation           | Public                 |
| `/auth/me`                | The current user                          | Authenticated          |
| `/auth/link`              | The start of an attach                    | Authenticated          |
| `/auth/identities*`       | The external accounts of the current user | Authenticated          |
| `/auth/logout`            | The end of a session                      | Public                 |
| `/plugins`                | The list of loaded plugins                | Authenticated          |
| `/plugins/{id}/api/*`     | The routes of a plugin                    | Authenticated          |
| `/plugins/{id}/settings*` | The settings of a user for a plugin       | Authenticated          |
| `/plugins/{id}/ui/*`      | The assets of a plugin                    | Public. See `SEC-006`. |
| `/telegram/webhook`       | The Telegram updates                      | Network allowlist      |
| `/ping`                   | The health check                          | Public                 |
| `/.well-known/oauth-protected-resource*` | The discovery of Dex for an MCP client | Public      |
| `/mcp`                    | The Model Context Protocol tools          | Bearer, MCP audience   |

`/auth/callback` finishes a sign in, an attach, and a redemption. One client of Dex
holds one redirect address, so one route finishes all three.

`/auth/logout` is public. A sign out behind an access check answers 401 for a dead
cookie. A second sign out must answer as the first one did.

## API-004

A plugin declares a route as `pluginapi.Route` with a method, a pattern, and a handler.
The hub mounts every route of a plugin under `/plugins/{id}/api`.
A plugin does not choose its own prefix.

**Why:** The prefix comes from the plugin identifier, so two plugins cannot collide.

## API-005

A handler implements `handler.Handler` and registers itself on a `chi.Router`.
A handler function returns an error. `handler.Wrap` logs the error.

## API-006

A response body is JSON. An error response obeys `ERR-001`.

## Retired identifiers

This file has no retired identifier.
