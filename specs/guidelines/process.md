---
id: PRC
title: Process
type: guideline
status: active
created: 2026-09-11
updated: 2026-09-12
scope: [.]
related: [TRC, LNG, TST, GIT]
---

# Process

The narrative guide is `specs/README.md`. This file holds the binding rules.

## PRC-001

Every change to the behavior that a user can see starts with a requirement statement.
The requirement comes before the specification. The specification comes before the code.

**Why:** The code is a result. The intent must exist in a document that a person
and an agent can both read.

## PRC-002

A technology choice needs a requirement that demands it.
Do not add a technology to the guidelines before the requirement exists.

**Why:** A technology without a requirement is a cost without a reason.

## PRC-003

Every document obeys `LNG`. Every identifier obeys `TRC`.

## Retired identifiers

| ID        | Retired    | Reason              |
| --------- | ---------- | ------------------- |
| `PRC-004` | 2026-09-11 | Moved to `GIT-001`. |
