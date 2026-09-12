---
id: SEC
title: Authentication and authorization
type: guideline
status: active
created: 2026-09-11
updated: 2026-09-11
scope: [apps/hub/internal/auth/, apps/hub/internal/middleware/, .keys/]
related: [ARC, API, TG, CFG, OWN]
---

# Authentication and authorization

## SEC-001

Identity comes from Telegram. The OIDC issuer is the Telegram OAuth service.
Maroid holds no password.

Maroid holds a local user record, and that record owns the data. `OWN-001` gives it.
The record carries no credential.

**Why:** The household already has Telegram. A second account is a cost with no value.
A row still needs an owner that a Telegram profile change cannot move.

## SEC-002

The hub issues its own JWT after the OIDC callback.
The algorithm is RS256. The hub signs with the RSA key pair that `jwt.private_key`
and `jwt.public_key` name. The default lifetime is 168 hours.

Verification requires the RS256 method, the issuer, an issued-at claim, and an
expiry claim.

## SEC-003

The subject claim of the JWT is the Maroid user identifier, as `OWN-001` gives it.

**Why:** The token names the owner of the data, with no mapping step.

## SEC-004

The user record is the allowlist. A request that carries an identity with no active
user record gets status 401. The configuration holds no list of identifiers.
The hub checks the record on each request, so a block ends a live token.

**Why:** The record that owns the data also grants the access. One list decides both.

## SEC-005

The token travels in the cookie `maroid_token`, or in the `Authorization` header
with the `Bearer` prefix. The cookie comes first.

## SEC-006

`/plugins/{id}/ui/*` serves the assets of a plugin with no authentication.

This is a known gap, not a decision. A plugin user interface must hold no secret
in its bundle. Every request for data still passes `SEC-004`.

## SEC-007

The Telegram webhook accepts a request only from a network that
`telegram.webhook.allowed_networks` holds. Other requests get status 403.

## Retired identifiers

This file has no retired identifier.
