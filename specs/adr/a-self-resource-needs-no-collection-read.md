---
id: ADR-0015
title: A self resource needs no collection read
type: adr
status: accepted
created: 2026-10-12
updated: 2026-10-12
decided: 2026-10-12
changes: [RES-002]
supersedes:
superseded_by:
---

# A self resource needs no collection read

## Context

`PARKACC` names the one connection of the acting user `/connections/self`, and its one
pending connection `/pending-connections/self`. `Z-134` makes each segment plural, and
`self` names the record that the session selects, as `DELETE /auth/sessions/self` does.

`Z-143` asks that every intermediate path resolve. A `GET` of `/connections` would list
at most one record, the one that `/connections/self` already answers. `RES-002` exempts
five paths from `Z-143` today, and no rule covers a `self` record.

## Decision

A collection whose only member a client reaches is the `self` record of the acting user
answers no `GET`. `RES-002` names the case.

## Rationale

The list would repeat one record, and a client that reads it learns nothing more. A
route of zero value is a route to test, document, and keep.

## Alternatives

| Alternative | Why we did not select it |
| ----------- | ------------------------ |
| A `GET` of the collection that answers a page of zero or one item | It repeats `/connections/self`, and `RES-005` makes it carry a cursor that never moves. |
| A singular segment, `/connection` | `Z-134` asks for the plural, and `RES-002` holds no deviation for a plugin route. |

## Consequences

`RES-002` gains this text in the `Z-143` row, after the list of the five paths:

> A collection whose only member a client reaches is the `self` record of the acting
> user answers no `GET` either, `/connections` of a plugin for example. `ADR-0015`.

- Positive: a plugin adds a `self` resource with one route for each operation.
- Negative: a client cannot discover the `self` record from its collection.
- Work that follows: none. `PARKACC` already uses the form.
