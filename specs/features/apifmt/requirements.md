---
id: APIFMT
title: The answer that a client reads
type: requirements
status: approved
created: 2026-09-24
updated: 2026-09-24
approved_by: Temuri
approved_on: 2026-09-24
constrained_by: [RES, ERR, DAT, OWN]
---

# Requirements: The answer that a client reads

This file holds the body of an answer: the spelling of a member, the page of a
collection, the cursor, the order, and the value of a point in time.
`requirements-platform.md` holds the routes, the stored address, the migrations,
the published documents, and the headers. `SPC-001` divides the two, because one
file reached the size that `LNG-013` gives.

## 1. Problem

`ADR-0005` gave the failure half of the API one shape. No decision gave the
success half one, so each handler chose.

## 2. Users

| Person                      | Needs                                                               |
| --------------------------- | --------------------------------------------------------------------- |
| A person who deploys Maroid | One shape for every answer, so that a client they write keeps working. |
| The person at the deck      | A page that loads when a collection grows past a screen.             |
| The owner                   | An answer that matches the standard, so that a reviewer carries no rules in their head. |

## 3. Out of scope

- An order that a client chooses. `APIFMT-FR-007` gives the reason.
- The routes, the stored address, the migrations, the published documents, and
  the headers. `requirements-platform.md` holds each one.

## 4. Definitions

`GLO-page` gives the page. `GLO-cursor` gives the cursor.

## 5. Functional requirements

### `APIFMT-FR-001`

The hub must spell every member of an answer with one convention.

**Why:** A client reads one rule and needs no table of exceptions. The deck reads
five members that the hub spells another way, and renders none of them.

**Examples:**

- Normal case: an answer names the moment of the last write as its column does.
- Limit case: a member that a third-party service names keeps the spelling of
  that service, because Maroid does not choose it.
- Limit case: a key that a person stored keeps the spelling that they gave it.
- Unwanted case: two answers of one API spell one idea two ways.

### `APIFMT-FR-002`

The hub must answer every collection as a page.

**Why:** An array grows no member. A page carries the links, and whatever a later
decision adds, with no change that breaks a client.

**Examples:**

- Normal case: a collection of two rows answers a page that holds two items.
- Limit case: a collection of no rows answers a page with an empty array.
- Unwanted case: a collection answers a bare array, and the next added member
  breaks every reader.

### `APIFMT-FR-003`

The hub must give each page the cursor that reaches the next page.

**Why:** A client that counts rows itself reads one row twice when a write
arrives between two reads, and misses one when a write removes a row.

**Examples:**

- Normal case: a client follows the cursor from one page to the next.
- Limit case: the last page carries no cursor, and a client stops.
- Unwanted case: a client builds a cursor itself and reads a page that no answer
  offered.

### `APIFMT-FR-004`

When a collection has no upper bound, the hub must answer one page of it.

When a collection has an upper bound, the hub must answer every item in one page.

**Why:** The deployment bounds the loaded plugins, and the providers that the IdP
federates bound the external accounts of one person. A second page of either
answers a question that nobody asks.

**Examples:**

- Normal case: a request for the plants of a person answers the first page.
- Limit case: a request for the loaded plugins answers every one, and carries no
  cursor.
- Unwanted case: a request for the loaded plugins names a page, and the route
  answers a failure.

### `APIFMT-FR-005`

When a request carries a stale cursor, the hub must answer a failure that names
a stale cursor.

**Why:** A client that cannot tell a cursor it must retire from a cursor it built
wrong retries forever, or abandons a collection that works.

**Examples:**

- Normal case: a cursor built under one set of filters, sent with another set,
  answers the failure that names a stale cursor.
- Limit case: a cursor that Maroid did not produce answers the failure that
  names an invalid request.
- Unwanted case: both answer one failure, and a client cannot choose between a
  retry and a report.

### `APIFMT-FR-006`

The hub must answer a collection in the one order that the route declares.

**Why:** A client that reads two pages of one collection needs the order to hold
between them.

**Examples:**

- Normal case: every page of a collection follows the order that the route names.
- Limit case: a route that declares no order answers the order of the identifier.
- Unwanted case: a route answers two requests in two orders, and a client that
  reads every page reads one row twice.

### `APIFMT-FR-007`

When a request names a field that the route does not declare, the hub must
answer a failure that names that field.

**Why:** A client learns which field a route takes from the failure, and not from
a document that it has to find. No route declares a field today, so every named
field answers the failure, and a route that declares one later needs no change to
this statement.

**Examples:**

- Normal case: a request names the creation time, the route declares no field,
  and the failure names the creation time.
- Limit case: a route declares one field, and a request that names it answers the
  collection in that order.
- Unwanted case: the route ignores the field, and a client believes it applied.

### `APIFMT-FR-008`

The hub must answer every point in time in UTC.

**Why:** A client converts once, for the person in front of it. A server that
answers its own zone gives every client a rule that no answer states.

**Examples:**

- Normal case: one moment answers one value to a reader in any zone.
- Limit case: a value that a third-party service sends in another zone converts
  before Maroid answers it.
- Unwanted case: one row answers two values on two servers.

### `APIFMT-FR-009`

The hub must give one meaning to a member that holds no value.

**Why:** A client that must tell an absent member from a member holding nothing
carries a rule for each route. One meaning removes that rule.

**Examples:**

- Normal case: a record with no species answers the same way every time it has
  none.
- Limit case: a collection with no rows answers an empty array, and never
  nothing.
- Unwanted case: one route omits a member and another answers it holding nothing,
  for one condition.

### `APIFMT-FR-010`

The hub must name a failure with an identifier that reads the same on every
deployment.

**Why:** A client compares one constant, and a translation table keys on it. An
identifier built from the address of a deployment differs on each one, so a
client that reads two deployments matches neither.

**Examples:**

- Normal case: two deployments answer one condition with one identifier.
- Limit case: a plugin names a failure of its own domain, and the identifier
  carries the identifier of that plugin.
- Unwanted case: an identifier resolves against the address of the request, and
  one condition reads two ways.

## 6. Non-functional requirements

### `APIFMT-NFR-001`

The time to answer one page must not grow with the position of that page.
Measure from the request to the last byte of the answer. Use a collection of ten
thousand rows. Compare page 1 against page 100. The second must not exceed the
first by more than one fifth.

**Why:** A route that counts past every row it already answered grows slower on
each page. A client that reads a long collection then waits longer each time.

### `APIFMT-NFR-002`

A route that answers one page of an unbounded collection must bound that page.
When a request names no page size, the hub must answer the default. When a
request names a size above the ceiling, the hub must answer a failure. `RES-005`
holds both numbers, and neither reaches a bounded collection. Measure at the
answer body.

**Why:** A default bounds the answer for a client that names nothing. A ceiling
bounds it for a client that names too much.

## 7. Invariants

### `APIFMT-INV-001`

A client that reads every page of a collection which no write changes reads each
item exactly once, at all times.

**Why:** `APIFMT-FR-003` exists to give this property. A client that cannot rely
on it counts rows itself.

### `APIFMT-INV-002`

Every member that Maroid names carries one spelling convention, at all times.

**Why:** No route of Maroid is exempt. `RES-003` gives the two surfaces that the
convention does not reach: a key that a person stored, and a name that a
third-party service chose.

## 8. Constraints from the guidelines

| Rule      | Guideline        | Effect on this feature                                                   |
| --------- | ---------------- | -------------------------------------------------------------------------- |
| `RES-001` | REST conventions | The standard decides each shape. This feature chooses none of them.      |
| `RES-002` | REST conventions | Holds every deviation. This feature introduces none, and an ADR adds a row if one appears. |
| `RES-003` | REST conventions | Gives the two surfaces that `APIFMT-INV-002` does not reach.             |
| `RES-005` | REST conventions | Gives the members of the page, the two page sizes, and the order that a route declares. |
| `RES-006` | REST conventions | Gives the cursor, which a client passes back and never builds.           |
| `ERR-001` | Errors           | Every failure of this feature answers a problem.                         |
| `ERR-002` | Errors           | Gives the form of the identifier that `APIFMT-FR-010` demands.           |
| `ERR-003` | Errors           | Holds `cursor-stale`, which `APIFMT-FR-005` names.                       |
| `DAT-009` | Data             | Every row carries an identifier that sorts by time, which `APIFMT-NFR-001` rests on. |
| `DAT-010` | Data             | Stores a point in time in UTC, which `APIFMT-FR-008` answers.            |
| `OWN-006` | Ownership        | A collection answers the rows of the acting user, and a page counts those rows only. |

## 9. Open questions

This file has no open question.

## Retired identifiers

This file has no retired identifier.
