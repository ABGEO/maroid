---
id: ADR-0003
title: An MCP tool call as a second entry point of the IdP
type: adr
status: accepted
created: 2026-09-17
updated: 2026-09-17
decided: 2026-09-17
changes: [SEC-002, OWN-003]
supersedes:
superseded_by:
---

# An MCP tool call as a second entry point of the IdP

## Context

The MCPHUB feature exposes tools over the Model Context Protocol. An MCP client
is a native or a command line application. It cannot hold a client secret, so
it cannot authenticate as the web shell does.

`SEC-002` ties the audience of a verified token to one client identifier: the
hub's own. `OWN-003` names two entry points that resolve an acting user, an
HTTP request and a Telegram update. Neither statement admits an MCP tool call.

The IdP mints its access token the same way it mints an ID token, a signed
JWT. An MCP client authenticates against a second, public client identifier of
the IdP, with no secret and PKCE. The hub verifies that token with the key set
of the IdP, the same path `SEC-002` already gives.

## Decision

`SEC-002` admits a second client identifier, dedicated to an MCP client. An MCP
tool call becomes a third entry point in `OWN-003`, and it resolves the acting
user through the same identity resolver as an HTTP request.

## Rationale

The `SEC-002` verification path stays one path: the signature, the issuer, the
audience, the issued-at claim, and the expiry claim, checked against the key
set of the IdP. Only the expected audience differs by entry point. A second
verification path would duplicate that logic and the defects that come with it.

The identity resolver of `OWN-003` already turns a federated identity into an
acting user, and `SEC-004` already rejects an identity with no active user
record. An MCP tool call reuses both, so a tool that reaches a scoped table
inherits `OWN-005` through `OWN-008` for free, in this iteration or the next.

## Alternatives

| Alternative                                          | Why we did not select it                                                                                                    |
| ------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------------- |
| Route an MCP client through the hub's own client identifier | An MCP client would need the client secret of the hub. A native or a command line application cannot keep one confidential. |
| A second verification path for an MCP tool call      | Duplicates the check that `SEC-002` already gives, and a defect in one path would not appear in the other.                  |
| Leave a tool call outside `OWN-003`, with no acting user | A tool that reaches a scoped table would set no acting user, and `OWN-006` would return zero rows for every caller.        |

## Consequences

- Positive: one verification path serves the web shell and an MCP client. A
  tool that touches a scoped table works with no further change to `OWN`.
- Positive: an MCP client authenticates with no client secret, so a native or a
  command line application holds no confidential value.
- Negative: the IdP now holds two client identifiers for the hub to track, and
  a rotation touches both.
- Work that follows: `SEC-002` and `OWN-003` are corrected.
  `specs/features/mcphub/requirements.md` is written.
- Examined: `ident/spec.md` states that section 4.4 resolves the acting user
  at three entry points, and `extid/spec.md` states in `EXTID-DD-012` that
  `OWN-003` names two entry points that resolve an identity through
  `auth.IdentityResolver`. Both describe the code as it stands today, with no
  MCP tool call in it, so neither is wrong yet. The specification of MCPHUB
  raises each count by one, and adds the flow of the MCP tool call to
  `ident/spec.md` section 4.4, the same way `EXTID` added the flow of the
  Telegram update.
