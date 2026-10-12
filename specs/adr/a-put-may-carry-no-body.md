---
id: ADR-0018
title: A put of a self resource may carry no body
type: adr
status: accepted
created: 2026-10-13
updated: 2026-10-13
decided: 2026-10-13
changes: [RES-002]
supersedes:
superseded_by:
---

# A put of a self resource may carry no body

## Context

`GWPACC` connects the water account of a workspace with `PUT /connections/self`. The
hub derives the whole connection from the settings of the workspace, so the request
carries no representation. `Z-148` describes a `PUT` that sends the representation of
the resource. A repeat of the connect leaves the workspace connected, which `Z-149`
asks of a `PUT`. `RES-002` names no deviation for a `PUT` with no body.

## Decision

A `PUT` of a `self` resource whose state the hub derives from stored data carries no
body. `RES-002` names the case.

## Rationale

The client holds nothing to send, and a body that repeated the stored settings would
let a request bypass them. The method still tells a client that a repeat is safe.

## Alternatives

| Alternative | Why we did not select it |
| ----------- | ------------------------ |
| `POST /connections` | A repeat has no new effect, so `POST` tells the client that a retry is unsafe when it is safe. |
| A `PUT` with an empty object `{}` | A body with no member is a representation in name alone, and every client must send it. |

## Consequences

`RES-002` gains a row for `Z-148`: "A `PUT` of a `self` resource whose state the hub
derives from stored data carries no body, `PUT /connections/self` of a plugin for
example. The repeat leaves the same state, as `Z-149` asks. `ADR-0018`."

- Positive: the connect of a plugin is idempotent and carries no secret in the request.
- Negative: a generic client cannot learn the representation from the request.
- Work that follows: none. `GWPACC` already uses the form.
