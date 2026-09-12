---
id: TST
title: Testing
type: guideline
status: active
created: 2026-09-11
updated: 2026-09-11
scope: [apps/, libs/, plugins/]
related: [GO, TS, PRC]
---

# Testing

> **The repository has no test today.**
> This guideline gives the target. The first feature that runs through the full
> lane builds the test setup. Until then, each rule applies to the new code only.

## TST-001

A test cites the scenario identifier that it verifies, in a comment.

```go
// NOTIF-SC-003: A second enqueue with the same business key adds no row.
func TestEnqueueIsIdempotent(t *testing.T) {
```

## TST-002

Write the test before the code. Run the test. Confirm that it fails for the correct reason.

**Why:** A test that never failed proves nothing.

## TST-003

Do not change a test and do not change a scenario to make a test pass.
A test that looks wrong means the specification is wrong or the code is wrong.
Correct the specification first. See `PRC-001`.

## TST-004

A scenario declares its layer: `unit`, `integration`, or `manual`.
A non-functional requirement needs a scenario that measures the number in it.

## Retired identifiers

This file has no retired identifier.
