---
id: IDENT
title: The user record and the ownership of a row
type: requirements
status: approved
created: 2026-09-11
updated: 2026-09-15
approved_by: Temuri
approved_on: 2026-09-12
constrained_by: [OWN, DAT, SEC, TG, JOB, REP, PLG]
---

# Requirements: The user record and the ownership of a row

## 1. Problem

Maroid knows who is asking and then forgets it. The login gives a Telegram identity,
the access check accepts it, and the request reaches the data with that identity
dropped. Every row belongs to the installation, not to a person.

A second person therefore cannot use it: their first request shows them the data of
the first person. Today each person needs a separate instance.

## 2. Users

- The owner: a second person uses the instance and reads none of the owner's data.
- A household member: their own bills, plants, and contributions, and nobody else's.
- A plugin author: isolation that they get without writing it into each query.

## 3. Out of scope

- Moving a per-user value of a plugin into the database. See `CFG-007`.
- Sharing one record between two people, or a role beyond the ownership of a record.
- Scoping the records of the plugins that exist today. They stay shared for now.
- Self-registration. The owner creates each user record. See `SEC-004`.
- Per-user routing of a notification, and any tool that creates a user record.

## 4. Functional requirements

### `IDENT-FR-001`

Maroid must hold one permanent record for each user that it serves.

**Why:** A scoped record needs an owner that outlives a change of a Telegram profile.

### `IDENT-FR-002`

Maroid must refuse every interaction from a person that holds no active user record,
in the web shell and in the bot.

**Why:** The record is the access list. A person that Maroid does not hold reaches nothing.

### `IDENT-FR-003`

A scoped record that a user creates must belong to that user, and to nobody else.


### `IDENT-FR-004`

A user must read only the records that belong to them, and the shared records.

**Examples:**

- Normal case: two people each hold three plants, and each of them lists three plants.
- Unwanted case: a list shows six plants to both people, or hides the shared list of
  transaction types from one of them.

### `IDENT-FR-005`

A user must not write a record that belongs to another user. A delete is a write.

### `IDENT-FR-006`

Maroid must answer a request for the record of another user exactly as it answers a
request for a record that does not exist.

**Why:** A different answer tells the requester that the record exists.

### `IDENT-FR-007`

A scheduled task that creates a scoped record must state the user that it acts for.

### `IDENT-FR-008`

Every record that exists before this feature must stay a shared record, readable by
every user.

### `IDENT-FR-009`

Maroid must keep the records of a user that the owner blocked.

**Why:** Removing a person from the household must not delete the history of the household.

### `IDENT-FR-010`

Maroid must serve a person whose record the owner created outside the product, at their
next interaction and with no restart.

**Why:** The record appears with no interaction, so no value that the hub holds in
memory can decide the access.

### `IDENT-FR-011`

The owner must block the access of a user without deleting their records.

## 5. Non-functional requirements

### `IDENT-NFR-001`

Read every scoped record as user A, while user B holds scoped records. The count of the
records of user B in the result must be zero.

### `IDENT-NFR-002`

Resolving the acting user must add no more than 10 milliseconds to an HTTP request, at
the 95th percentile, measured from its arrival to its first statement at the database.

## 6. Invariants

### `IDENT-INV-001`

A record names exactly one user, or it is a shared record.

### `IDENT-INV-002`

No unit of work reaches a scoped record without an acting user.

## 7. Constraints from the guidelines

| Rule      | Guideline        | Effect on this feature                                              |
| --------- | ---------------- | ------------------------------------------------------------------- |
| `OWN-002` | Record ownership | The owner creates the record. It is active or blocked.              |
| `OWN-006` | Record ownership | The isolation runs in the database and fails closed.                |
| `SEC-003` | Security         | The token names an identity. The hub maps it to the record.         |
| `SEC-004` | Security         | The user record is the allowlist. The configuration loses its list. |

## 8. Open questions

| #   | Question                                                                                                 | Owner  | Answer                                                                                                                                                      |
| --- | -------------------------------------------------------------------------------------------------------- | ------ | ----------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1   | Which user owns the rows that the utility jobs collect, given that the credentials are global?           | Temuri | Nobody. Those records stay shared. A later feature moves the configuration of a plugin to the user, and then a job runs once for each user that enabled it. |
| 2   | Does the owner create a user record with a command of the hub, or with a statement against the database? | Temuri | With a statement against the database. `EXTID-FR-010` adds the command later, because the first identity needs an invitation.                               |

## Retired identifiers

| ID              | Retired    | Reason                                                                                         |
| --------------- | ---------- | ------------------------------------------------------------------------------------------------ |
| `IDENT-INV-003` | 2026-09-15 | `ADR-0002` took the Telegram identifier off `public.users`. `EXTID-INV-001` succeeds it.       |
