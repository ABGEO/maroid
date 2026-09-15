---
id: ADR-0002
title: Dex as the authorization server
type: adr
status: accepted
created: 2026-09-15
updated: 2026-09-16
decided: 2026-09-15
changes: [SEC-001, SEC-002, SEC-003, OWN-001, OWN-003, API-003, GLO-user,
  GLO-identity, GLO-provider, GLO-external-account, GLO-invitation]
supersedes:
superseded_by:
---

# Dex as the authorization server

## Context

Maroid binds a person to one Telegram account. `public.users` carries
`telegram_id BIGINT NOT NULL UNIQUE`. The HTTP callback reads the `id` claim of
Telegram, and the bot resolves an update by the same number. One column holds one
external account, so a person holds one external account.

The hub also mints its own RS256 token after the callback. It reads the key pair
from `.keys/`, and `jwt.private_key` names the file. The hub is an issuer and a
resource server at the same time.

Dex federates several providers behind one OpenID Connect endpoint. Three of its
properties decide the shape of this decision.

| Property of Dex                                                                                      | Consequence for Maroid                                        |
| ---------------------------------------------------------------------------------------------------- | --------------------------------------------------------------- |
| A token carries the connector and the upstream user identifier in the `federated_claims` claim.      | One pair of values names one external account.                  |
| The `sub` claim encodes the upstream user identifier and the connector, so `userIDKey` changes it.   | A row that joins on `sub` orphans when a connector changes.     |
| A connector accepts every account that its upstream provider accepts.                                | Authentication at Dex is not permission at Maroid.              |

## Decision

Dex becomes the authorization server. The hub becomes its consumer: the hub verifies
a token that Dex issued against the JWKS of Dex, and signs no token of its own.

One row in `public.identities` maps one pair of `federated_claims` to one user record.
That row is the only path from an external account to a user record. Telegram becomes
one connector among many.

## Rationale

The user record stays the allowlist that `SEC-004` gives, and the hub reads it on
every request. A block therefore ends a live token within one request. That property
is the reason the hub can stop signing its own token, and keep revocation.

`federated_claims` is the join key, not `sub`. A subject that changes with the
configuration of a connector orphans every row that names it.

One verification path serves the browser today and an agent later. A second token
format costs a second path and keeps the key material in `.keys/`.

The identity carries the profile of a provider, because two providers give two names
for one person. The user record carries the name that the owner chose.

## Alternatives

| Alternative                                             | Why we did not select it                                                                            |
| ------------------------------------------------------- | --------------------------------------------------------------------------------------------------- |
| Keep the Telegram OIDC service as the issuer.           | One connector only. A second provider needs a second flow inside the hub.                            |
| Hold the provider and the external account on `users`.  | One person then holds one external account. A second one needs a second user record, and the rows of a person split across both. |
| Join on the `sub` claim of Dex.                         | `sub` changes when a connector changes, and every identity then names nobody.                        |
| Keep the token of the hub and read Dex at the callback. | Two token formats to verify when an agent arrives, and `.keys/` stays.                               |
| Let an unknown external account create a user record.   | Any Google account then reaches Maroid. `SEC-004` states the opposite.                               |

## Consequences

### Rules that change

`SEC-001` names Dex as the issuer. The new text:

> Identity comes from Dex. Dex is the authorization server, and the hub is its
> consumer. Telegram is one connector of Dex, and it is not the only one.
> Maroid holds no password.
>
> Maroid holds a local user record, and that record owns the data. `OWN-001` gives it.
> The record carries no credential.
>
> **Why:** One person holds several external accounts. A further connector costs one
> entry in the configuration of Dex, and no flow inside the hub.

`SEC-002` loses the signature and the key pair. The new text:

> The hub signs no token. It verifies a token that Dex issued, against the JWKS of
> Dex, and it holds the key set in memory.
>
> A verification checks the signature, the issuer, the audience, an issued-at claim,
> and an expiry claim. The audience is the client identifier of the hub.
>
> **Why:** One issuer means one verification path. The browser today and an agent
> later present a token of the same shape.

`SEC-003` loses the Maroid user identifier as the subject. The new text:

> The subject claim of the token belongs to Dex. The hub stores it nowhere.
>
> The hub reads the `federated_claims` claim. That claim names the connector and the
> user identifier at the upstream provider. One row in `public.identities` maps the
> pair to the Maroid user identifier.
>
> **Why:** The subject of Dex changes when the configuration of a connector changes.
> A stored subject then names nobody.

`OWN-001` loses the Telegram user identifier as the natural key. The new text:

> The hub holds one record for each person, in the table `public.users`. The record
> is permanent, and it carries the name that the owner gave it.
>
> One row in `public.identities` binds one external account to one user record. That
> row is the natural key, and `SEC-003` reads it. One user record holds several of
> them. One external account belongs to at most one user record.

`OWN-003` changes two rows of its table. The new table:

> | Entry point       | Source of the acting user                                            |
> | ----------------- | -------------------------------------------------------------------- |
> | An HTTP request   | The identity that `federated_claims` names. See `SEC-003`.           |
> | A Telegram update | The identity of the Telegram connector for the sender of the update. |
> | A cron job        | The user that the job declares. See `OWN-009`.                       |
>
> One resolver reads `public.identities` for the first two rows. A connector that
> arrives later adds no path.

`API-003` gains three prefixes. The new rows:

> | Prefix              | Serves                                    | Access        |
> | ------------------- | ----------------------------------------- | ------------- |
> | `/auth/link`        | The start of an attach                    | Authenticated |
> | `/auth/invite`      | The redemption of an invitation           | Public        |
> | `/auth/identities*` | The external accounts of the current user | Authenticated |

`/auth/callback` finishes a sign in, an attach, and a redemption. One client of Dex
holds one redirect address, so one route finishes all three.

`GLO-user` loses the Telegram user identifier. The glossary gains `GLO-identity`,
`GLO-provider`, `GLO-external-account`, `GLO-invitation`, `GLO-attach`, and
`GLO-detach`.

`SEC-004`, `SEC-005`, and `OWN-002` do not change. `SEC-004` carries the whole gate
after this decision, because no other check limits who reaches Maroid. `SEC-005` keeps
the cookie `maroid_token`, and the token inside it comes from Dex. `OWN-002` keeps the
owner as the only creator of a user record.

The `scope` field of `guidelines/security.md` loses `.keys/`.

### Configuration

`jwt.issuer`, `jwt.private_key`, `jwt.public_key`, and `jwt.token_expiry` go. The
`OIDC` block points at Dex. `CFG-005` already keeps a client secret out of the file.

### Migrations

Four migrations follow. Three of them create one shared table each: the identity, the
authorization flow, and the invitation. Each one states its reason under `OWN-004`,
because the resolver reads the table before an acting user exists.

The fourth reshapes `public.users`. It writes one identity for each row that holds a
Telegram identifier. It splits `display_name` into `first_name` and `last_name` at
the first space, then drops the four columns that go. The extract runs before the
drop, because `telegram_id` is the only place that holds the external account of
that person.

### Constraints on the identity provider

Two settings stop being a choice of the owner.

| Setting                                                       | Reason                                                                                          |
| --------------------------------------------------------------- | ------------------------------------------------------------------------------------------------- |
| The Telegram connector exposes the numeric Telegram user identifier as its connector user identifier. | The bot reads that number from an update. A different value gives one person two identities and breaks `EXTID-FR-017`. |
| The lifetime of a token matches the lifetime that `EXTID-NFR-002` gives. | The hub renews nothing, so the token alone decides when a person signs in again.                 |

### Specifications to examine

| Document                             | Why                                                                                  |
| ------------------------------------ | -------------------------------------------------------------------------------------- |
| `features/ident/requirements.md`     | `IDENT-INV-003` names the Telegram identifier. It retires, and `EXTID-INV-001` succeeds it. |
| `features/ident/spec.md`             | Section 4.2, `IDENT-DD-005`, and the design decision on the subject claim all name the Telegram identifier. |
| `features/ident/spec-scenarios.md`   | The scenario that inserts a duplicate Telegram identifier.                            |

`IDENT-FR-002`, `IDENT-FR-010`, and `IDENT-FR-011` stay true word for word. The
resolution of the acting user changes under them, and the statements do not.

### Result

- Positive: one person holds many external accounts, and each one reaches the same data.
- Positive: the hub holds no signing key and no key rotation of its own.
- Positive: a token that an agent presents later needs no second verification path.
- Negative: a sign in now depends on a second service. A sign in fails when Dex does not answer.
- Negative: the session lifetime moves into the configuration of Dex.
- Work that follows: the feature `EXTID`, then the capability that a plugin asks for, then the scopes and MCP, in that order.
