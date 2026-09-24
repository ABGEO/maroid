---
id: APIFMT
title: The platform that carries the API
type: requirements
status: approved
created: 2026-09-24
updated: 2026-09-24
approved_by: Temuri
approved_on: 2026-09-24
constrained_by: [RES, ERR, API, DAT, CFG, SEC, TG, LOG, BLD, OWN]
---

# Requirements: The platform that carries the API

`requirements.md` holds the body of an answer. This file holds what carries it:
the name of a route, the address that a link points at, the order of a migration,
the published document, and the headers that a client reads. `SPC-001` divides
the two, because one file reached the size that `LNG-013` gives.

## 1. Problem

Four routes name an action, and one answers a word that no client reads. Maroid
gives an address to three readers and builds each one differently, so the address
it registers for the bot carries no scheme.

No document describes the API to a reader outside the repository, and no check
refuses a route that departs from the standard. No record of a failure joins to
the request that caused it, and no answer says how long a shared cache
keeps it.

## 2. Users

| Person                      | Needs                                                                 |
| --------------------------- | ----------------------------------------------------------------------- |
| A person who deploys Maroid | One document to read, and an address that reaches their deployment.   |
| A person who reports a fault | One value that finds every record of the request that failed.        |
| The owner                   | A check that refuses a departure, so that a review carries no rules.  |

## 3. Out of scope

- The design of a health check. `APIFMT-FR-012` removes the route that answers a
  fixed word, and a later decision designs the replacement.
- A second address for one deployment. `APIFMT-FR-013` gives the reason.
- The body of an answer. `requirements.md` holds it.

## 4. Definitions

| Term             | Meaning                                                                    |
| ---------------- | ---------------------------------------------------------------------------- |
| external address | The address that reaches one deployment from outside it, with the scheme.  |

`GLO-api-document` gives the published document.

## 5. Functional requirements

### `APIFMT-FR-011`

The hub must name a resource in every path.

**Why:** A person who writes a client reads the path and the method, and learns
what the route does from both. A path that names an action says nothing that the
method does not say.

**Examples:**

- Normal case: the route that ends a session names the session, and takes the
  method that removes one.
- Limit case: a route whose address an external standard derives keeps that
  address, and `RES-004` names the route.
- Unwanted case: two paths name one action with two words.

### `APIFMT-FR-012`

The hub must remove the route that answers a fixed word.

**Why:** No probe of any deployment reads that route, and no client calls it. A
route that nobody reads still carries a contract from the day a client exists.
`ADR-0006` exempts this removal from the window that `RES-011` gives, because no
document of the API has reached a reader yet.

### `APIFMT-FR-013`

The hub must build every external address from one stored value.

That value must carry the scheme. One deployment answers at one address.

**Why:** Maroid gives an address to three readers and builds each one
differently. The address that it registers for the bot carries no scheme. The
address that it reports for discovery carries a scheme that the code fixes, so a
deployment on another scheme reports an address that reaches nobody.

**Examples:**

- Normal case: a link in a page, the address registered for the bot, and the
  address in the discovery document name one host and one scheme.
- Limit case: a deployment behind a proxy answers the address that an outside
  reader reaches, and not the address that the proxy sends.
- Unwanted case: the stored value changes, and one reader keeps the old address.
- Unwanted case: a deployment answers at a second address, and a link built for
  the first reaches a reader of the second.

### `APIFMT-FR-014`

The hub must store one instant for a point in time, whatever zone the server
runs in.

**Why:** A trigger writes the moment of the last write through a conversion that
drops the zone, and the column keeps a zone. Two servers in two zones then store
two instants for one moment, and `APIFMT-FR-008` cannot answer one value.

**Examples:**

- Normal case: two servers in two zones store one instant for one write.
- Limit case: a column that holds a day and no moment keeps that day.
- Unwanted case: the answer of a row moves when the deployment moves.

### `APIFMT-FR-015`

The hub must apply every migration of its own before any migration of a plugin.

**Why:** A migration of the hub creates the function that a trigger of a plugin
calls. A plugin that runs first finds no function, and no rule fixes the order
today.

**Examples:**

- Normal case: a first start applies the migrations of the hub, then those of
  each plugin.
- Limit case: a start with no plugin applies the migrations of the hub alone.
- Unwanted case: two starts of one build apply the migrations in two orders.

### `APIFMT-FR-016`

Maroid must publish one document for each API that it serves.

**Why:** A person who deploys Maroid writes a client, and a client reads one
document. A description spread over the feature that added each route reaches
nobody outside the repository.

**Examples:**

- Normal case: a reader takes one document and calls every route of one API.
- Limit case: an API whose routes no feature declares still publishes a document.
- Unwanted case: a reader assembles a description from several files.

### `APIFMT-FR-017`

The build must refuse a document that breaks the standard.

**Why:** A rule that only a reviewer checks goes stale. The register of
departures and the check must agree, and a departure that no register names must
stop the build.

**Examples:**

- Normal case: a document that obeys every rule builds.
- Limit case: a document that departs from a rule the register names builds.
- Unwanted case: a document departs from a rule that no register names, and the
  build passes.

### `APIFMT-FR-018`

The hub must join every record of one request under one identifier.

The hub must take that identifier from the request when the request carries one.

**Why:** A person reports a failure and names one value. One search over the log
then reaches the line that holds the cause.

**Examples:**

- Normal case: a person reports the value that an answer carried, and one search
  finds every record of that request.
- Limit case: a request that carries no identifier gets one that the hub makes.
- Unwanted case: a caller sends a value that the hub cannot store, and the hub
  answers a failure instead of making its own.

### `APIFMT-FR-019`

When a client writes a record that changed after the client read it, the hub
must refuse the write.

**Why:** Two people editing one record otherwise lose the first edit with no
sign. A client that reads, edits and writes needs to learn that the record moved.

**Examples:**

- Normal case: a client reads a record, writes it back, and the write lands.
- Limit case: a client writes with no knowledge of the version, and the hub takes
  the write.
- Unwanted case: two clients read one record, both write, and the second write
  replaces the first with no failure.

### `APIFMT-FR-020`

When a client repeats a write under the key of an earlier write, the hub must
answer the earlier result and must create no second record.

**Why:** A client that loses the answer to a write cannot tell a write that
failed from one that succeeded. Repeating it must be safe.

**Examples:**

- Normal case: a client repeats a write it did not see answered, and one record
  exists.
- Limit case: a client repeats a write under a new key, and a second record
  exists.
- Unwanted case: a retry after a timeout creates a duplicate.

### `APIFMT-FR-021`

The hub must say, for each route, how long a reader keeps the answer.

**Why:** `OWN-006` filters a collection by the person who asked, so an answer
that a shared cache keeps reaches the wrong reader.

**Examples:**

- Normal case: a route that answers the rows of one person says that no shared
  cache keeps the answer.
- Limit case: a route that answers the same bytes to everyone names a period.
- Unwanted case: a route says nothing, and a proxy decides.

## 6. Constraints from the guidelines

| Rule      | Guideline        | Effect on this feature                                                   |
| --------- | ---------------- | -------------------------------------------------------------------------- |
| `RES-004` | REST conventions | Names the one address that `APIFMT-FR-011` does not reach.               |
| `RES-005` | REST conventions | Builds a link from the value that `APIFMT-FR-013` stores.                |
| `RES-007` | REST conventions | Names the documents that `APIFMT-FR-016` publishes, and the block each one carries. |
| `RES-008` | REST conventions | Gives the merge and the check that `APIFMT-FR-017` demands.              |
| `RES-011` | REST conventions | Gives the window before a route goes, and the exemption that `APIFMT-FR-012` takes. |
| `ERR-006` | Errors           | Gives the identifier of `APIFMT-FR-018`, its source, and its bound.      |
| `API-003` | The HTTP API     | Fixes the route table that `APIFMT-FR-011` realizes, and the route that `APIFMT-FR-012` removes. |
| `API-007` | The HTTP API     | Holds the headers that `APIFMT-FR-019` and `APIFMT-FR-021` add.          |
| `DAT-010` | Data             | Gives the column and the trigger that `APIFMT-FR-014` corrects.          |
| `DAT-011` | Data             | Gives the order that `APIFMT-FR-015` demands.                            |
| `CFG-003` | Configuration    | The value of `APIFMT-FR-013` validates at the start, and the hub stops without it. |
| `SEC-006` | Security         | The assets of a plugin stay public, and `APIFMT-FR-011` does not reach them. |
| `TG-002`  | Telegram         | The value of `APIFMT-FR-013` feeds the webhook registration.             |
| `LOG-009` | Logging          | Carries the identifier of `APIFMT-FR-018` on every record.               |
| `BLD-001` | Build            | Holds the command that `APIFMT-FR-017` runs.                             |
| `OWN-006` | Ownership        | Filters every collection by the acting user, which `APIFMT-FR-021` answers for. |

## 7. Open questions

This file has no open question.

## Retired identifiers

This file has no retired identifier.
