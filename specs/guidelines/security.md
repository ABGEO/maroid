---
id: SEC
title: Authentication and authorization
type: guideline
status: active
created: 2026-09-11
updated: 2026-10-08
scope: [apps/hub/internal/auth/, apps/hub/internal/middleware/]
related: [ARC, API, TG, CFG, OWN, RES]
---

# Authentication and authorization

## SEC-001

Identity comes from Dex. Dex is the authorization server, and the hub is its
consumer. Telegram is one provider of Dex, and it is not the only one.

Maroid stores no password and no password hash. An administrator sets the
password of a local account through the hub. The hub hashes it with bcrypt in
memory, sends the hash to Dex, and logs neither value.

Maroid holds a local user record. The record carries no credential. A workspace
owns the data, and `OWN-010` gives it.

**Why:** One person holds several external accounts. A further provider costs one
form in the deck, and no flow inside the hub.

## SEC-002

The hub signs no token. It verifies a token that Dex issued, against the JWKS of
Dex, and it holds the key set in memory.

A verification checks the signature, the issuer, the audience, an issued-at claim,
and an expiry claim. The audience is the client identifier of the entry point
that receives the token. The web shell and an MCP client each authenticate
against their own client identifier of Dex.

**Why:** One issuer means one verification path. The browser today and an agent
later present a token of the same shape.

## SEC-003

The subject claim of the token belongs to Dex. The hub stores it nowhere.

The hub reads the `federated_claims` claim. That claim names the connector and the
user identifier at the upstream provider. One row in `public.identities` maps the
pair to the Maroid user identifier.

A provider keeps its identifier for its life. A provider of the OIDC kind also
keeps its issuer and its `userIDKey`. Deleting a provider deletes its identities.

**Why:** The subject of Dex changes when the configuration of a connector changes.
A stored subject then names nobody.

## SEC-004

The user record is the allowlist. A request that carries an identity with no active
user record gets status 401. The configuration holds no list of identifiers.
The hub checks the record on each request, so a block ends a live token.

**Why:** The record that owns the data also grants the access. One list decides both.

## SEC-005

The hub reads the credential of a web request from the session cookie. The hub
reads no `Authorization` header on that path.

The session cookie holds the access token that the IdP issued for the hub. It
holds no identity token.

`/mcp` is the one entry point that reads a bearer header. `API-003` names it, and
`SEC-002` gives the verification.

**Why:** One credential path needs no precedence rule. The attributes that
`SEC-008` gives then hold for every request, because a cookie is the only path.

## SEC-006

`/plugins/{id}/ui/*` serves the assets of a plugin with no authentication.

This is a known gap, not a decision. A plugin user interface must hold no secret
in its bundle. Every request for data still passes `SEC-004`.

## SEC-007

The Telegram webhook accepts a request only from a network that
`telegram.webhook.allowed_networks` holds. Other requests get status 403.

`SEC-009` gives the address that this rule reads. A request that carries none is
refused, because the guard fails closed.

## SEC-008

Every cookie that the hub sets carries these attributes:

| Attribute   | Value     |
| ----------- | --------- |
| Name prefix | `__Host-` |
| `Secure`    | Set       |
| `HttpOnly`  | Set       |
| `Path`      | `/`       |
| `Domain`    | Absent    |
| `SameSite`  | `Lax`     |

The name of a cookie is a constant in the code. The configuration does not hold it.

`SameSite` is load bearing, and not a hardening default alone. A browser stores
a `Lax` cookie from a `POST` only when the request is same site. The three
routes of `API-003` that start a flow set a cookie on such a `POST`.

**Why:** The hub and the deck answer at two hosts of one domain. Without the
prefix a sibling host writes a cookie that the hub reads. A name that no
deployment changes needs no check at the start.

## SEC-009

The hub resolves the address of a caller with a `ClientIPFrom` middleware of chi,
which puts it in the context. Code reads it with `middleware.GetClientIPAddr`,
and never from `RemoteAddr` or from a header of its own.

| `server.trusted_proxies` | Middleware                  | Reads                                     |
| -------------------------- | ----------------------------- | ------------------------------------------- |
| Empty, the default       | `ClientIPFromRemoteAddr`    | The peer of the connection.               |
| One or more CIDR         | `ClientIPFromXFF`           | `X-Forwarded-For`, from the right, past every address that a trusted network holds. |

`RealIP` of chi is deprecated and does not serve this rule. It rewrites
`RemoteAddr` from `X-Forwarded-For`, `X-Real-IP` or `True-Client-IP` sent by any
peer, so a caller chooses the address that a guard then reads.

A guard fails closed. `SEC-007` refuses a request that carries no resolved address.

A deployment behind a proxy sets the key, or the allowlist of the webhook reads the
address of the proxy and refuses every update. `chart/values.yaml` carries it.

**Why:** The allowlist of the webhook is an access decision, and a header that any
caller can set is no guard at all. The middleware of chi also folds a v4 mapped
IPv6 address, strips a zone, and merges a repeated header, so no second spelling
of a trusted address walks past the check.

## SEC-010

A route that a third-party service calls verifies a secret. The hub registers
that secret with the service, and compares it before it reads the body. A request
that carries no matching secret gets status 401.

The guideline of the service names the header that carries it.

A network allowlist does not replace the secret. An allowlist answers where a
request comes from, and a secret answers who sent it. Where both guard one route,
the allowlist runs first, and a request that fails it gets 403 and no read of the
secret.

**Why:** An address is a weaker claim than a secret. A service that publishes the
ranges it sends from lets any caller inside those ranges past an allowlist.

## SEC-011

A user record is an administrator or not. An administrator creates and blocks a
user record, and sets the plugin allowlist of a user. An administrator manages every
workspace of the instance through the management API: its members and its
enablements. An administrator manages the providers of the instance, and sets the
password of a local account. The plugin allowlist does not limit an administrator. An administrator
reads the rows of a workspace only as a member of it. The CLI makes the first
administrator, and gives it a local account.

**Why:** The person who runs the instance is not entitled to the pension of each
person on it.

## SEC-012

A plugin serves a workspace only where it is enabled. A plugin starts disabled
in every workspace. A manager enables a plugin that the plugin allowlist of the
manager holds. `SEC-011` gives the access of an administrator. A member uses every plugin
that the workspace enables, and the hub reads no plugin allowlist of the member.

A plugin that is disabled in the acting workspace is absent from it. Its routes
answer `not-found`, a call to its MCP tools in that workspace answers `not-found`,
its Telegram commands are absent from the menu of the chat, and its jobs skip the
workspace. The list of MCP tools names no workspace, so it stays one list for every
person.

## SEC-013

A plugin declares each permission that it checks, and names the lowest role
that holds it. The hub declares its own permissions the same way. The hub
prefixes a permission of a plugin with the plugin identifier.

Every route, MCP tool, and Telegram command that acts in a workspace declares
one permission. The hub checks the role of the acting user in the acting
workspace against it before it calls the handler. A route of the hub that acts
in no workspace declares none. A member whose role does not hold the permission
gets `permission-denied`. A plugin whose entry declares no permission, or a
permission that the plugin does not declare, fails to load.

**Why:** A check that the hub runs is a check that no handler forgets. A load
that fails names the gap before any person meets it.

## Retired identifiers

This file has no retired identifier.
