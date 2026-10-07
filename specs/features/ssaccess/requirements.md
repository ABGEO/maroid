---
id: SSACCESS
title: The access of Maroid to the secret store
type: requirements
status: approved
created: 2026-10-07
updated: 2026-10-07
approved_by: Temuri
approved_on: 2026-10-07
constrained_by: [LOG, LIF]
---

# Requirements: The access of Maroid to the secret store

## 1. Problem

Maroid logs in to the secret store one time, at the start. The store grants that
access for a limited time. When the access ends, each save of a secret and each run
that reads a secret fails until a person restarts Maroid. The owner met this failure
in production.

The readiness report counts the store as up during the failure. `HEALTH-FR-006` reads
the state of the store, not the access of Maroid. `PSET-FR-018` proves the access at
the start only.

## 2. Users

| Person                      | Needs                                                              |
| --------------------------- | ------------------------------------------------------------------ |
| A household member          | A saved secret and a plugin run that work on any day, with no restart. |
| A person who deploys Maroid | No restart to cure an ended access, and the cause when access fails. |
| An orchestrator             | A readiness answer that fails while Maroid cannot use the store.   |

## 3. Out of scope

- The first access at the start. `PSET-FR-018` covers it.
- A login credential that the store accepts for a limited count of logins or a
  limited time. Maroid presents the same credential at each login.
- A replacement of the login credential while Maroid runs.
- A second attempt at a save or a read that failed while Maroid held no store access.
- The end of the store access when Maroid stops.

## 4. Definitions

`GLO-secret-store`, `GLO-store-access`, and `GLO-readiness` give the terms of this feature.

## 5. Functional requirements

### `SSACCESS-FR-001`

Maroid must hold a store access for as long as it runs, with no restart.

**Why:** A store access ends after a time that the store sets. An end with no
successor fails each save and each run that touches a secret.

**Examples:**

- Normal case: Maroid runs past the end of its first store access. A member saves a
  secret, and a plugin reads it.
- Limit case: Maroid runs past the longest time that the store lets one access last.
  A member saves a secret, and a plugin reads it.

### `SSACCESS-FR-002`

Maroid must gain a new store access before the current one ends.

**Why:** A save that arrives between the end and the replacement fails.

**Examples:**

- Normal case: a member saves a secret every second across the end of a store access.
  Every save succeeds.

### `SSACCESS-FR-003`

Maroid must keep running when it cannot gain a store access after the start.

**Why:** A restart meets the same refusal, and the readiness answer of
`SSACCESS-FR-006` already reports the failure.

**Examples:**

- Unwanted case: the store answers nothing for one hour. Maroid keeps running, and
  each save of a secret fails with an error.
- Unwanted case: the store refuses the login credential. Maroid keeps running.

### `SSACCESS-FR-004`

Maroid must try again to gain a store access until the store grants one.

**Examples:**

- Normal case: the store answers nothing for 10 minutes, then answers. Maroid holds a
  store access with no restart.

### `SSACCESS-FR-005`

Maroid must log each failed attempt to gain a store access, with the reason that the
store gave.

**Why:** A person who deploys Maroid reads the cause without a search of the store.

**Examples:**

- Normal case: the store refuses the login credential. The log names the refusal.
- Normal case: the store answers nothing. The log names the store and the time limit.

### `SSACCESS-FR-006`

The hub must count the secret store as failed in a readiness answer while the hub holds
no store access.

**Why:** A store that refuses the hub answers the network, and it protects no secret
for the hub.

**Examples:**

- Normal case: the hub holds a store access. The secret store counts as up.
- Unwanted case: the store access of the hub ended, the hub cannot gain a new one, and
  the secret store counts as up.

## 6. Non-functional requirements

### `SSACCESS-NFR-001`

Maroid must hold a store access no more than 60 seconds after the store answers again.
Measure from the first answer of the store to the first readiness answer that counts
the secret store as up.

**Why:** Each save and each run that touches a secret fails until then.

## 7. Constraints from the guidelines

| Rule      | Guideline | Effect on this feature                                                     |
| --------- | --------- | -------------------------------------------------------------------------- |
| `LOG-008` | Logging   | The log of `SSACCESS-FR-005` holds no store access and no login credential. |
| `LIF-005` | Lifecycle | Maroid stops trying to gain a store access when the shutdown starts.       |
| `LIF-006` | Lifecycle | The stop of the attempts fits inside the 10 seconds of a shutdown step.    |

## 8. Open questions

| #   | Question                                                                 | Owner  | Answer                                                                 |
| --- | ------------------------------------------------------------------------ | ------ | ---------------------------------------------------------------------- |
| 1   | Does Maroid stop when it cannot gain a store access after the start?     | Temuri | No. It keeps running and tries again. See `SSACCESS-FR-003`.           |
| 2   | Does this feature or `HEALTH` own the readiness answer for a lost access? | Temuri | This feature. `SSACCESS-FR-006`. `HEALTH` stays unchanged.             |

## Retired identifiers

This file has no retired identifier.
