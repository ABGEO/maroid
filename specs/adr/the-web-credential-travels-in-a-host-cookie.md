---
id: ADR-0004
title: The web credential travels in a host cookie
type: adr
status: accepted
created: 2026-09-21
updated: 2026-09-21
decided: 2026-09-21
changes: [SEC-005, SEC-008, API-003]
supersedes:
superseded_by:
---

# The web credential travels in a host cookie

## Context

`SEC-005` gives the credential of a web request two paths and one name:

> The token travels in the cookie `maroid_token`, or in the `Authorization` header
> with the `Bearer` prefix. The cookie comes first.

Three facts stand against that rule.

**The hub and the deck answer at two hosts of one domain.** The name `maroid_token`
carries no prefix, so a browser enforces nothing about it. Any host under that
domain writes a cookie of that name, and the hub reads it.

**The header is a second credential path.** `auth.Middleware` reads the cookie,
then falls back to the header. Two sources need a precedence rule, and the header
carries none of the attributes that a browser enforces on a cookie. No client sends
one: `libs/api-client` sends the cookie and nothing else.

**The cookie holds the identity token.** `ADR-0002` put it there, because the hub
verified one token shape and the identity token was the one the callback held. The
identity token describes a person to the client that signed them in. It is not the
value that the IdP issues for a call to Maroid.

`MCPHUB-DD-001` later established that the IdP mints the access token as one signed
token with the client as the audience, the same shape as the identity token. `/mcp`
verifies an access token that way today, against the key set that `SEC-002` gives.

`WEBSESS-FR-001`, `WEBSESS-FR-002`, and `WEBSESS-FR-005` each contradict `SEC-005`.

## Decision

The hub reads the credential of a web request from a cookie and from nothing else.
The cookie holds the access token, and its name carries the `__Host-` prefix.
`/mcp` stays the one entry point that reads a bearer header.

## Rationale

The prefix moves the guard into the browser. A browser refuses a `__Host-` cookie
that carries a domain, that carries a path other than `/`, or that arrives over an
unencrypted connection. A sibling host of the hub therefore writes nothing that the
hub reads. No code in the hub can enforce that, because the hub sees one cookie
header and not the host that wrote it.

`HttpOnly` denies a script the value of the token. A flaw in a page then reaches
the hub for as long as the script runs, and the attacker carries no token away to
use from another place. The prefix and this attribute carry the security of this
decision.

The removal of the header carries little of it, and this record states the limit. A
client outside a browser sets a cookie as readily as a header, so a stolen token
reaches the hub either way. A script in the page needs no header, because a browser
attaches the cookie to the request that the script makes. One case remains: a token
that leaked to a log, to an address, or to a referrer, replayed from a page that the
configuration allows. The rest is one credential path in place of two, with no
precedence rule and one branch fewer. `TokenFromContext` has no caller today, so the
change reaches one function.

The access token costs no second verification path. The IdP signs it with the hub
as the audience, and `SEC-002` already checks the signature, the issuer, the
audience, and the expiry against the key set in memory. `/mcp` runs that check on an
access token now.

`SEC-004` carries revocation, and this decision does not touch it. The hub reads the
user record on each request, so a blocked record still ends a live credential in one
request.

## Alternatives

| Alternative                                                        | Why we did not select it                                                                                                        |
| ------------------------------------------------------------------ | --------------------------------------------------------------------------------------------------------------------------------- |
| Keep the header as a second path, and rename the cookie only.      | The rename carries the security of this decision, and this alternative keeps it. It also keeps a second credential source that no client sends.   |
| Keep the header behind a setting that names a development machine. | It keeps a branch in the middleware and a path that a deployment can switch on by mistake. Bruno reads a cookie.                 |
| Keep the identity token in the cookie.                             | The hub keeps a credential that the IdP issued to describe a person, and the value that the IdP issued for the call reaches nothing. |
| Hold the credential in a store of the hub, and give the browser an opaque value. | It adds a store, a deadline, and a second component on the path of every request. No statement asks for one yet.    |
| Bind the cookie to the host in the code of the hub.                | The hub cannot see which host wrote a cookie. The browser is the only place that holds the fact.                                 |
| Hold the cookie name in the configuration, and check it at the start. | It adds a key that no deployment changes, and a wrong value then stops the hub of a person who never asked for the knob.      |

## Consequences

### Rules that change

`SEC-005` loses the header and names the token. The new text:

> The hub reads the credential of a web request from the session cookie. The hub
> reads no `Authorization` header on that path.
>
> The session cookie holds the access token that the IdP issued for the hub. It
> holds no identity token.
>
> `/mcp` is the one entry point that reads a bearer header. `API-003` names it, and
> `SEC-002` gives the verification.
>
> **Why:** One credential path needs no precedence rule. The attributes that
> `SEC-008` gives then hold for every request, because a cookie is the only path.

`SEC-008` is new. It holds the attributes of every cookie, not of the session cookie
alone, because the flow of a sign in sets a second one:

> Every cookie that the hub sets carries these attributes:
>
> | Attribute   | Value     |
> | ----------- | --------- |
> | Name prefix | `__Host-` |
> | `Secure`    | Set       |
> | `HttpOnly`  | Set       |
> | `Path`      | `/`       |
> | `Domain`    | Absent    |
>
> The name of a cookie is a constant in the code. The configuration does not hold it.
>
> **Why:** The hub and the deck answer at two hosts of one domain. Without the
> prefix a sibling host writes a cookie that the hub reads. A name that no
> deployment changes needs no check at the start.

`API-003` gains one row in its table:

> | `/auth/logout` | The end of a session | Public |

The rule gains one sentence under the table:

> `/auth/logout` is public. A sign out behind an access check answers 401 for a dead
> cookie. A second sign out must answer as the first one did.

`SEC-002`, `SEC-004`, and `SEC-006` do not change.

### Specifications to examine

| Document                        | Why                                                                                   |
| ------------------------------- | --------------------------------------------------------------------------------------- |
| `features/extid/spec.md`        | `EXTID-DD-006` puts the identity token in `maroid_token`. The decision supersedes it. Section 4 names the cookie. |
| `features/mcphub/spec.md`       | `MCPHUB-DD-001` stays true. Confirm that `/mcp` reads no cookie.                       |
| `features/websess/spec.md`      | The specification that this ADR unblocks. It names the cookie and the address of the sign out. |

### Migration

No migration of the database. One migration of the browsers: the cookie changes
name, so every signed-in browser holds a value that the hub stops reading. Every
person signs in one more time at the first deployment that carries the change.

### Result

- Positive: a sibling host of the hub writes no cookie that the hub reads.
- Positive: one credential path, with no precedence rule between two sources.
- Positive: the hub holds the value that the IdP issued for a call to Maroid.
- Negative: every live session ends at the deployment. Each person signs in again.
- Negative: a request by hand needs a cookie. A bearer header reaches the web
  routes no longer.
- Negative: the decision stops no client outside a browser. A stolen token reaches
  the hub from one that sets a cookie.
- Work that follows: the specification of `WEBSESS`, then the code.
