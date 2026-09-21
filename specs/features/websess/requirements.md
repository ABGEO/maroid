---
id: WEBSESS
title: The session of a person at the web shell
type: requirements
status: approved
created: 2026-09-21
updated: 2026-09-21
approved_by: Temuri
approved_on: 2026-09-21
constrained_by: [SEC, API, UI, CFG]
---

# Requirements: The session of a person at the web shell

## 1. Problem

The hub puts the identity token of the IdP in a cookie, and three faults follow.

The identity token describes a person to the client that signed them in. The hub
uses it as the credential for every call instead. Maroid accepts a value that the
IdP issued for another purpose, and the value that the IdP issued for this purpose
reaches nothing.

The cookie carries a name that a browser enforces nothing about. The hub answers at
one host, and the deck answers at a sibling host of the same domain. Any host under
that domain writes a cookie of that name, and the hub reads it.

A person cannot sign out. The deck offers no control, and the hub offers no address.
The credential stays in the browser until it expires. A second person at the same
machine opens the deck and reads the records of the first person.

## 2. Users

| Person           | Need                                                                     |
| ---------------- | ------------------------------------------------------------------------ |
| A person at the deck | Sign in one time, work, and sign out at the end. Read no credential. |
| The owner        | A person who leaves a shared machine stops reaching the records.         |
| A plugin author  | One acting user for each request, with no change to a plugin.            |

## 3. Out of scope

- A store in the hub that holds the session of a person. The browser holds it.
- A renewal of the credential. A person signs in again when it expires.
- The check of the origin of a request against a forged request. A later feature.
- The end of the session of the person at the IdP. See `WEBSESS-FR-006`.
- The refusal of a credential that a person held before a sign out. Nothing
  records the sign out, so the credential works until it expires.
- The bearer token of an MCP client. `/mcp` keeps the path that `API-003` gives it.

## 4. Functional requirements

### `WEBSESS-FR-001`

The hub must put the access token of the IdP in the session cookie.

**Why:** The identity token describes a person. A call to Maroid needs the value
that the IdP issued for that call.

**Examples:**

- Normal case: a person signs in. The session cookie holds the access token.
- Unwanted case: the session cookie holds the identity token.

### `WEBSESS-FR-002`

A browser must drop a cookie of the hub when a host other than the hub sets it.

**Why:** The hub and the deck share one domain. A sibling host writes a cookie of
the name that the hub reads, and the hub then serves the request of that host.

**Examples:**

- Normal case: the hub sets the session cookie. The browser returns it to the hub.
- Unwanted case: a sibling host of the hub sets a cookie of that name. The browser
  holds no such cookie for the hub.

### `WEBSESS-FR-003`

A browser must send a cookie of the hub over an encrypted connection only.

**Why:** A credential in clear text on the network belongs to the network.

### `WEBSESS-FR-004`

A script in a page must not read a cookie of the hub.

**Why:** One flaw in one page of the deck, or in one plugin, otherwise hands the
credential of a person to the author of that flaw.

### `WEBSESS-FR-005`

The hub must resolve the acting user of a web request from the session cookie alone.

**Why:** A page that sets a request header otherwise chooses its own credential.
The hub trusts the value that it set, and no other.

**Examples:**

- Normal case: a request carries the session cookie. The hub answers with the
  records of that person.
- Unwanted case: a request carries a token in a header and no session cookie. The
  hub answers 401.
- Limit case: a request carries a session cookie and a header that names another
  person. The hub answers for the person that the cookie names.

### `WEBSESS-FR-006`

The hub must end the session of a person when the person asks for it.

The session at the IdP survives. A sign in that follows reaches the deck with no
question from the IdP.

**Why:** A second person at the same machine reads the records of the first person
today. The end of the session at Maroid stops that.

**Examples:**

- Normal case: a person signs out. The next request of that browser gets 401.
- Limit case: a person signs out a second time. The hub answers as it answered the
  first time, and reports no failure.
- Unwanted case: a sign out leaves the session cookie in the browser.

### `WEBSESS-FR-007`

The hub must send the person to a target that the configuration allows after a
sign out.

**Why:** The target comes from the request. An unchecked target sends the person
to a foreign host, with the name of Maroid in the address that they clicked.

**Examples:**

- Normal case: a sign out names the address of the deck. The person lands there.
- Unwanted case: a sign out names a foreign host. The hub refuses the target.

### `WEBSESS-FR-008`

The deck must offer a control that signs the person out.

**Why:** `WEBSESS-FR-006` gives the operation. Nothing reaches it without a control.

## 5. Non-functional requirements

### `WEBSESS-NFR-001`

A sign out must remove the session cookie from the browser in one request. Measure
from the request that the control sends to the answer of the hub.

**Why:** A sign out that needs a second request leaves a window in which the
credential is still in the browser.

### `WEBSESS-NFR-002`

The hub must refuse a request that carries a cookie of a name that the hub no
longer sets. Measure at the first deployment that follows the change.

**Why:** The cookie changes name. A browser holds the old one for as long as its
lifetime, and the old one must grant nothing.

## 6. Invariants

### `WEBSESS-INV-001`

The browser of a person holds at most one session cookie for Maroid.

### `WEBSESS-INV-002`

The session cookie holds a value that the IdP issued. The hub puts no other value
in it.

## 7. Constraints from the guidelines

| Rule      | Guideline            | Effect on this feature                                                                   |
| --------- | -------------------- | ---------------------------------------------------------------------------------------- |
| `SEC-002` | Security             | The hub verifies the access token against the key set of the IdP, as it verifies the identity token today. |
| `SEC-004` | Security             | A sign out does not change the user record. The record stays the allowlist.               |
| `SEC-005` | Security             | The rule names the cookie and gives the header as a second path. `WEBSESS-FR-001`, `WEBSESS-FR-002`, and `WEBSESS-FR-005` all contradict it. The rule changes, so this feature needs an ADR. |
| `API-003` | The HTTP API         | The prefix table gains the address of the sign out.                                       |
| `UI-001`  | The web user interface | The deck carries the control that `WEBSESS-FR-008` gives. A plugin serves no full page. |
| `CFG-003` | Configuration        | The hub validates the configuration at the start. The name of a cookie that fails `WEBSESS-FR-002` stops the hub there. |
| `CFG-007` | Configuration        | The list of the targets that `WEBSESS-FR-007` checks is a value of the installation.      |

`EXTID-NFR-002` stays true. A person signs in no more often than one time in seven
days, and this feature does not change that interval.

## Retired identifiers

This file has no retired identifier.
