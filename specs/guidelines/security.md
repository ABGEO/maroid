---
id: SEC
title: Authentication and authorization
type: guideline
status: active
created: 2026-09-11
updated: 2026-09-21
scope: [apps/hub/internal/auth/, apps/hub/internal/middleware/]
related: [ARC, API, TG, CFG, OWN]
---

# Authentication and authorization

## SEC-001

Identity comes from Dex. Dex is the authorization server, and the hub is its
consumer. Telegram is one connector of Dex, and it is not the only one.
Maroid holds no password.

Maroid holds a local user record, and that record owns the data. `OWN-001` gives it.
The record carries no credential.

**Why:** One person holds several external accounts. A further connector costs one
entry in the configuration of Dex, and no flow inside the hub.

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

The name of a cookie is a constant in the code. The configuration does not hold it.

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

## Retired identifiers

This file has no retired identifier.
