---
id: PSECRET
title: The secrets that a plugin writes
type: requirements
status: approved
created: 2026-10-12
updated: 2026-10-12
approved_by: Temuri
approved_on: 2026-10-12
constrained_by: [PLG, ARC, OWN, CFG, EXT, LOG]
---

# Requirements: The secrets that a plugin writes

## 1. Problem

A plugin that signs in to an external service gets a session token back. The token is
not a setting: no person types it, and the plugin replaces it when the service ends it.
Today the plugin has two places for such a value, and both are wrong. A secret field
makes the person copy the token by hand. A table of the plugin holds the token in a form
that a reader of the database reads. `PSET` puts a protected value that no settings
record holds out of its scope.

The same plugin also holds a short state between two calls of one sign in: the service
sends a code to a phone, and the plugin needs the identifier of that sign in when the
person types the code.

## 2. Users

- A plugin author: stores a token and a short sign in state under a name, and writes no
  cryptography.
- A household member: connects an external account one time, and no reader of the
  database reads the token.
- An administrator: destroys the protection of one person or one workspace, and every
  value of that scope becomes unreadable.

## 3. Out of scope

- A route, a page, or a command through which a person reads, lists, or deletes a plugin secret.
- A plugin secret that one plugin reads for another plugin.
- A read of the plugin secret of a user who is not the acting user, or of a workspace
  that is not the acting workspace.
- A conditional write that replaces a value only when it holds an expected value.
- A limit on the count of the plugin secrets of one plugin.
- A history of the values of a plugin secret.
- A deletion of plugin secrets when a workspace disables the plugin.

## 4. Functional requirements

### `PSECRET-FR-001`

A plugin must write a plugin secret under a name for the acting user.

**Why:** The session token of a personal account belongs to one person in every workspace.

### `PSECRET-FR-002`

A plugin must write a plugin secret under a name for the acting workspace.

**Why:** A household that shares one external account shares its token.

### `PSECRET-FR-003`

A plugin must read a plugin secret by its name and its scope.

### `PSECRET-FR-004`

Maroid must report a plugin secret as absent when the scope holds no plugin secret under that name.

**Why:** A plugin that finds no token asks the person to connect. A failure asks for nothing.

**Examples:**

- Normal case: the plugin reads `session` for a user who never connected. The read reports absent.
- Unwanted case: the plugin wrote `session` for the user scope and reads `session` for
  the workspace scope. The read reports absent.

### `PSECRET-FR-005`

A write under a name that the scope already holds must replace the value and the expiry.

### `PSECRET-FR-006`

A read that starts after a write returns must return the value of that write.

**Why:** The plugin stores the token and uses it in the next call of the same request.

### `PSECRET-FR-007`

A plugin must delete a plugin secret by its name and its scope.

**Examples:**

- Normal case: the external service rejects the token. The plugin deletes `session`. The next read reports absent.
- Limit case: the plugin deletes a name that the scope does not hold. Maroid reports no failure.

### `PSECRET-FR-008`

A plugin must give a plugin secret an expiry at the write, or no expiry.

**Why:** A pending sign in ends after minutes. A session token ends when the service rejects it.

### `PSECRET-FR-009`

Maroid must report a plugin secret as absent after its expiry.

**Examples:**

- Normal case: the plugin writes `login` with an expiry 5 minutes ahead and reads it after 4 minutes. The read returns the value.
- Limit case: the plugin reads `login` 1 second after the expiry. The read reports absent.
- Unwanted case: the plugin gives an expiry in the past. Maroid rejects the write.

### `PSECRET-FR-010`

Maroid must delete a plugin secret no later than 24 hours after its expiry.

**Why:** An expired value that nobody reads is ciphertext that a backup keeps for no reason.

### `PSECRET-FR-011`

Maroid must keep a plugin secret with no expiry until the plugin deletes or replaces it.

**Examples:**

- Normal case: a manager disables the plugin and enables it again. The plugin reads the same token.
- Normal case: the owner deploys a new version of the plugin. The plugin reads the token that the old version wrote.

### `PSECRET-FR-012`

Maroid must reject a write whose name does not match the name grammar.
A name holds 1 to 64 characters: lowercase letters, digits, `.`, `_`, and `-`.

**Examples:**

- Normal case: `session`. Maroid stores it.
- Limit case: a name of 65 characters. Maroid stores nothing.
- Unwanted case: `Session Token`. Maroid stores nothing.

### `PSECRET-FR-013`

Maroid must reject a write whose value is empty or holds more than 16384 characters.

**Why:** A token of a service fits in this limit. A larger value is a file, not a credential.

### `PSECRET-FR-014`

Maroid must fail a call for the user scope when the unit of work carries no acting user.

### `PSECRET-FR-015`

Maroid must fail a call for the workspace scope when the unit of work carries no acting workspace.

**Examples:**

- Normal case: a Telegram update in a chat that selected a workspace. The call for the workspace scope succeeds.
- Unwanted case: a job for each user calls the workspace scope. The call fails.

### `PSECRET-FR-016`

Maroid must store a plugin secret so that a reader of the database cannot read the value.

### `PSECRET-FR-017`

Maroid must protect the plugin secrets of one user, and of one workspace, with the
protection of that user or that workspace.

**Why:** An administrator who destroys the protection of one person or one household
makes every value of that scope unreadable with one action. `PSET-FR-016` gives that protection.

### `PSECRET-FR-018`

Maroid must fail the read of a plugin secret when it cannot remove the protection.

**Why:** An empty token becomes a rejected call at the external service, and the cause stays invisible.

### `PSECRET-FR-019`

Maroid must write a plugin secret to no log.

### `PSECRET-FR-020`

Maroid must offer no route that returns a plugin secret to a client.

**Why:** The plugin is the only reader. A route is a path that a stolen session reads.

## 5. Non-functional requirements

### `PSECRET-NFR-001`

A plugin must read one plugin secret in no more than 200 milliseconds, at the 95th
percentile. Measure from the call of the plugin to the return of the value.

**Why:** Each command of a plugin reads the token before its first external call.

## 6. Invariants

### `PSECRET-INV-001`

A plugin reaches no plugin secret that another plugin wrote.

### `PSECRET-INV-002`

The plugin secret of one user reaches no other user. The plugin secret of one workspace
reaches no other workspace.

### `PSECRET-INV-003`

The database holds the value of a plugin secret in a protected form only.

## 7. Constraints from the guidelines

| Rule      | Guideline         | Effect on this feature                                                       |
| --------- | ----------------- | ---------------------------------------------------------------------------- |
| `PLG-007` | Plugin model      | A plugin reaches a plugin secret through the host.                           |
| `ARC-008` | Architecture      | The hub holds no rule that names one plugin or one name of a plugin secret.  |
| `OWN-003` | Record ownership  | Each call reads the acting user or the acting workspace of its unit of work. |
| `OWN-004` | Record ownership  | A plugin secret belongs to one user or to one workspace.                     |
| `OWN-006` | Record ownership  | The isolation of a plugin secret runs in the database and fails closed.      |
| `CFG-007` | Configuration     | The credential of one person stays out of the configuration file.            |
| `EXT-005` | External services | A failure of the secret store is an error, never an absent value.            |
| `LOG-008` | Logging           | No log holds a plugin secret.                                                |

## 8. Open questions

| #   | Question                                                           | Owner  | Answer                                                       |
| --- | ------------------------------------------------------------------ | ------ | ------------------------------------------------------------ |
| 1   | What happens to an expired plugin secret?                          | Temuri | A read reports it absent. Maroid deletes it within 24 hours. |
| 2   | Does a workspace that disables the plugin lose its plugin secrets? | Temuri | No. Settings and the rows of a plugin stay too.              |
| 3   | Does a person see or delete a plugin secret outside the plugin?    | Temuri | No. The plugin offers its own disconnect.                    |

## Retired identifiers

This file has no retired identifier.
