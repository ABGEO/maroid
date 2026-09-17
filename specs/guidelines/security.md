---
id: SEC
title: Authentication and authorization
type: guideline
status: active
created: 2026-09-11
updated: 2026-09-17
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
