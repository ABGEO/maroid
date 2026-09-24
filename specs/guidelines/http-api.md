---
id: API
title: The HTTP API
type: guideline
status: active
created: 2026-09-11
updated: 2026-09-23
scope: [apps/hub/internal/server/, apps/hub/internal/handler/, apps/hub/internal/middleware/]
related: [ARC, ERR, PLG, RES, SEC, UI]
---

# The HTTP API

## API-001

The hub serves APIs that `RES-007` names, on one router. The router is chi.

## API-002

The router applies this middleware, in this order:
the flow identifier, the real address, the access log, the recoverer,
`StripSlashes`, and the JSON content type. CORS follows when `cors.enabled` is true.

The flow identifier comes first, so every later middleware and every log record
reaches it. `ERR-006` gives the value, the generator, and the header.

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

A route inside its sunset window carries `Deprecation` and `Sunset`. `RES-011`
gives the window.

**Why:** A client reads `Allow` to learn what the route takes. A header that names
a scheme nobody implements teaches a client nothing.

## API-003

`Z-<number>` names a rule of the standard that `RES-001` binds.

The hub fixes these routes:

| Method and path                          | Serves                                    | Access                 |
| ---------------------------------------- | ----------------------------------------- | ---------------------- |
| `POST /auth/sessions`                    | The start of a sign in                    | Public                 |
| `GET /auth/sessions/self`             | The acting user of the session            | Authenticated          |
| `DELETE /auth/sessions/self`          | The end of a session                      | Public                 |
| `GET /auth/callback`                     | The finish of a flow                      | Public                 |
| `POST /auth/invitation-redemptions`      | The redemption of an invitation           | Public                 |
| `POST /auth/identities`                  | The start of an attach                    | Authenticated          |
| `GET /auth/identities`                   | The external accounts of the acting user  | Authenticated          |
| `DELETE /auth/identities/{provider}`     | The removal of one external account       | Authenticated          |
| `/plugins`                               | The list of loaded plugins                | Authenticated          |
| `/plugins/{id}/api/*`                    | The routes of a plugin                    | Authenticated          |
| `/plugins/{id}/settings*`                | The settings of a user for a plugin       | Authenticated          |
| `/plugins/{id}/ui/*`                     | The assets of a plugin                    | Public. See `SEC-006`. |
| `/telegram/webhook`                      | The Telegram updates                      | Secret token, allowlist |
| `/.well-known/oauth-protected-resource*` | The discovery of the IdP for an MCP client | Public                |
| `/mcp`                                   | The Model Context Protocol tools          | Bearer, MCP audience   |

A path holds no verb, as `Z-141` asks. `RES-002` holds the method, the status,
and the name that `GET /auth/callback` deviates on, and `RES-004` the one address
that a standard derives.

The three routes that start a flow at the IdP answer 202 with the address to
visit, and the deck navigates there. A `GET` cannot serve them, because each one
mints a binding and writes a cookie, which `Z-149` forbids a `GET` to do.

`GET /auth/callback` finishes a sign in, an attach, and a redemption. One client
of the IdP holds one redirect address, so one route finishes all three.

`self` is the pseudo-identifier that `Z-143` names for a resource whose
identifier comes from the session cookie.

`DELETE /auth/sessions/self` is public. A sign out behind an access check
answers 401 for a dead cookie. A second sign out must answer as the first one did.

## API-004

A plugin declares a route as `pluginapi.Route` with a method, a pattern, and a handler.
The hub mounts every route of a plugin under `/plugins/{id}/api`.
A plugin does not choose its own prefix.

**Why:** The prefix comes from the plugin identifier, so two plugins cannot collide.

## API-005

A handler implements `handler.Handler` and registers itself on a `chi.Router`.
A handler function returns an error. `handler.Wrap` logs the error.

## API-006

A response body is JSON. A success response obeys `RES-001`, and an error
response obeys `ERR-001`.

## Retired identifiers

This file has no retired identifier.
