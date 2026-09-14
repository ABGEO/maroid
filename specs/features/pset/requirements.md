---
id: PSET
title: The settings of a user for a plugin
type: requirements
status: approved
created: 2026-09-13
updated: 2026-09-14
approved_by: Temuri
approved_on: 2026-09-14
constrained_by: [CFG, PLG, OWN, ARC, API, EXT, PRC]
---

# Requirements: The settings of a user for a plugin

## 1. Problem

A plugin that reads a bill needs the credential of the person who owns the bill.
That credential lives in the configuration file today, and the file holds one copy for
the process. The plugin serves one person. `IDENT` gave every row an owner and made a
job run for each user, and the job still reads one credential. `CFG-007` names the move
that this feature makes.

A credential in the database is a credential that a reader of the database reads.
A backup, a console, and a log each hold a copy. Maroid holds no means to make a stored
value unreadable today. The key pair in `.keys/` signs a token, and it protects nothing
that a user stores.

## 2. Users

- A household member: enters their own credential one time, and no reader of the database reads it.
- The owner: adds a second person with no second instance and no change in the configuration file.
- The owner: removes the credentials of one person with no change for the other people.
- A plugin author: declares the fields one time, and writes no cryptography.

## 3. Out of scope

- The form in the deck that renders a settings schema.
- Moving the value of a person out of the configuration of a plugin that exists today.
- A field kind beyond the four that `PSET-FR-002` gives.
- An owner that reads or edits the settings of another user.
- A history of the changes of a setting.
- A settings record that a user fills through the bot.
- A job run restricted to the users that hold a settings record.
- A command that replaces the protection of a user.
- A record of each read of a secret.
- A protected value that no settings record holds.

## 4. Functional requirements

### `PSET-FR-001`

A plugin must declare the fields that a user fills for it.

**Why:** Maroid validates a value for every plugin, and `ARC-008` forbids a rule that names one plugin.

### `PSET-FR-002`

A field must declare one kind. The kinds are a free text, a secret, a true or false value, and a choice from a fixed list.

### `PSET-FR-003`

A user must store a value for a field that a plugin declares.

### `PSET-FR-004`

Maroid must return the stored settings to the user that stored them, except each secret.

**Why:** A reader that receives a credential can leak it, and no reader needs it.

### `PSET-FR-005`

Maroid must tell the user whether a secret field holds a value.

### `PSET-FR-006`

Maroid must keep the stored secret of a field when the save names no value for that field.

**Why:** A user changes one field, and the sender holds no other secret to send back.

### `PSET-FR-007`

A user must remove the stored value of a field.

### `PSET-FR-008`

Maroid must reject a save when a value does not match the settings schema.

**Examples:**

- Normal case: every required field holds a value that the schema permits. Maroid stores them.
- Limit case: a value of 4097 characters. Maroid stores nothing.
- Unwanted case: the save leaves a required field empty. Maroid stores nothing.
- Unwanted case: the save names a field that the schema does not declare. Maroid stores nothing.

### `PSET-FR-009`

Maroid must name each field that caused it to reject a save.

### `PSET-FR-010`

Maroid must return no field that the current settings schema does not declare.

### `PSET-FR-011`

Maroid must remove a stored value that the current settings schema does not declare, at the next save.

### `PSET-FR-012`

A plugin must read the settings of the acting user during a run.

### `PSET-FR-013`

Maroid must report the settings of the acting user as absent when a required field holds no value.

**Why:** A record that nobody filled and one that a new field made incomplete give one condition.

### `PSET-FR-014`

Maroid must report no failure when a plugin ends a run because the settings of the acting user are absent.

**Why:** A job runs for each active user at each tick, and few users hold a record.

### `PSET-FR-015`

Maroid must store a secret so that a reader of the database cannot read it.

### `PSET-FR-016`

Maroid must protect the secrets of one user with a protection that applies to no other user.

**Why:** The owner destroys the protection of one person with no change for the other people.

### `PSET-FR-017`

Maroid must return a secret that it stored before the owner replaced the protection of that user.

**Why:** A replacement that loses a stored credential is a replacement that nobody runs.

### `PSET-FR-018`

Maroid must not start when it cannot reach the service that protects a secret.

**Why:** A credential that Maroid cannot read becomes a job that fails one month later.

**Examples:**

- Normal case: the service answers at the start. Maroid starts and loads every plugin.
- Unwanted case: the service answers nothing. Maroid stops and names the service.
- Unwanted case: the service refuses the credentials of Maroid. Maroid stops and names the refusal.

### `PSET-FR-019`

Maroid must fail the read of a settings record when it cannot read a stored secret.

**Why:** An empty credential becomes a rejected login at the external service, and the
cause stays invisible. See `EXT-005`.

### `PSET-FR-020`

Maroid must write a secret to no log.

## 5. Non-functional requirements

### `PSET-NFR-001`

Maroid must give a plugin the value that a user saved more than 1 second earlier.
Measure from the response of the save to the start of the run.

**Why:** A user that corrects a wrong credential must see the next run use it.

### `PSET-NFR-002`

A plugin must read the settings of one user in no more than 200 milliseconds, at the 95th
percentile. Measure from the call of the plugin to the return of the values.

## 6. Invariants

### `PSET-INV-001`

The settings of one user reach no other user.

### `PSET-INV-002`

The database holds a secret in a protected form only.

## 7. Constraints from the guidelines

| Rule      | Guideline         | Effect on this feature                                                      |
| --------- | ----------------- | ----------------------------------------------------------------------------- |
| `CFG-004` | Configuration     | This feature adds no file in the repository that holds a secret.            |
| `CFG-007` | Configuration     | The value of one person leaves the configuration file.                      |
| `PLG-006` | Plugin model      | The settings schema is a capability: an interface, a registry, a registrar. |
| `PLG-007` | Plugin model      | A plugin reads a setting through the host.                                  |
| `OWN-003` | Record ownership  | The run that reads a setting carries one acting user.                       |
| `OWN-006` | Record ownership  | The isolation of a setting runs in the database and fails closed.           |
| `API-006` | The HTTP API      | A rejected save answers with a body that names each field.                  |
| `EXT-005` | External services | A failure of the protection service is an error, never an empty result.     |
| `PRC-002` | Process           | `PSET-FR-018` is the requirement that permits a new external service.       |

## 8. Open questions

| #   | Question                                                                                                 | Owner  | Answer                                                                                 |
| --- | ---------------------------------------------------------------------------------------------------------- | ------ | ---------------------------------------------------------------------------------------- |
| 1   | Does the owner destroy the protection of a user with a command of Maroid, or with a call to the service? | Temuri | With a call to the service. `IDENT` set the precedent for the user record.   |
| 2   | Does a plugin that declares no required field read an absent settings record as an empty one?            | Temuri | Yes. `PSET-FR-013` reports absent only when a required field holds no value. |

## Retired identifiers

This file has no retired identifier.
