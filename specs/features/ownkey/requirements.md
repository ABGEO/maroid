---
id: OWNKEY
title: The protection key of each user and each workspace
type: requirements
status: approved
created: 2026-11-24
updated: 2026-11-24
approved_by: Temuri
approved_on: 2026-11-24
constrained_by: [OWN, JOB]
---

# Requirements: The protection key of each user and each workspace

## 1. Problem

`PSET-FR-016` protects the secrets of each user and each workspace with a protection
key of their own. Maroid creates no protection key. An administrator creates one by
hand for each invite and for each new workspace. Until they do, every save of a secret
of that user or that workspace fails. A key created by hand refuses its destruction
unless the administrator also allows it, so the removal of `WSREMOVE` cannot destroy it.

## 2. Users

- An administrator: invites a person and adds no step at the secret store.
- A member: creates a workspace and saves its settings with no help from an
  administrator.

## 3. Out of scope

- The removal of a user record, and the destruction of its key. `OWN-002` keeps every
  user record.
- A rotation of a protection key.
- A protection key for any owner of a secret other than a user record or a workspace.
- The creation of a protection key at the moment of a save of a secret.
- A page or a command that lists the protection keys.

## 4. Functional requirements

### `OWNKEY-FR-001`

Maroid must create the protection key of a user record when it creates the user record.

**Examples:**

- Normal case: an administrator invites Nina. Her protection key exists in the secret
  store, and Nina saves a password in the settings of a plugin.
- Limit case: the administrator invites Nina through the command line. Maroid creates
  the key as for an invite from the web shell.

### `OWNKEY-FR-002`

Maroid must create the protection key of a workspace when it creates the workspace.

**Examples:**

- Normal case: Nina creates "Garden". Its protection key exists in the secret store.
- Limit case: the invite of Nina creates her first workspace. Maroid creates the key of
  the workspace and the key of Nina.

### `OWNKEY-FR-003`

Maroid must answer the creation of a user record or a workspace without waiting for the
secret store.

**Why:** A secret store that is down does not block an invite or a new workspace.

**Examples:**

- Normal case: the secret store is ready. Maroid answers, then creates the key.
- Limit case: the secret store is down. Maroid answers, and the key waits for
  `OWNKEY-FR-004`.

### `OWNKEY-FR-004`

Maroid must try the creation of a protection key again until it succeeds, when the
secret store fails.

**Examples:**

- Normal case: the secret store is down for 2 hours. Maroid creates the key after the
  store returns.
- Limit case: the hub restarts before the creation. Maroid creates the key after the
  restart.
- Unwanted case: the secret store refuses the creation five times in a row. Maroid
  tries again.

### `OWNKEY-FR-005`

Maroid must keep a protection key that already exists, with every version of it.

**Why:** An administrator created a key by hand before this feature. A replaced key
loses every secret under it.

**Examples:**

- Normal case: the key of "Home" exists from before. Maroid creates no new key, and
  every secret of "Home" still reads.
- Limit case: Maroid creates the same key twice after a restart. The second attempt
  changes nothing.

### `OWNKEY-FR-006`

Maroid must allow the destruction of every protection key that it creates or keeps.

**Why:** The removal of a workspace destroys its key under `WSREMOVE-FR-008`. A key that
refuses its destruction keeps a backup of the secrets readable.

**Examples:**

- Normal case: Nina removes "Garden". Maroid destroys its key.
- Limit case: an administrator created the key of "Home" by hand and did not allow its
  destruction. Maroid allows it, and a later removal of "Home" destroys the key.

### `OWNKEY-FR-007`

Maroid must create the protection key of every user record and every workspace that
exists before the first start of a hub that holds this feature.

**Examples:**

- Normal case: an instance holds 3 user records and 5 workspaces with no key. Maroid
  creates 8 keys.
- Limit case: an administrator created 2 of those keys by hand. Maroid keeps them under
  `OWNKEY-FR-005` and creates 6.

### `OWNKEY-FR-008`

Maroid must create no protection key for a workspace that it removed.

**Why:** A key of a removed workspace protects nothing, and no later removal destroys
it.

**Examples:**

- Normal case: Nina creates "Test" and removes it 1 second later. No key of "Test"
  exists in the secret store after the limit of `WSREMOVE-NFR-003`.
- Limit case: the secret store is down during the creation and the removal. No key of
  "Test" exists after the store returns.

### `OWNKEY-FR-009`

Maroid must refuse the save of a secret while the protection key of its user or its
workspace does not exist yet. Maroid keeps no part of the save.

**Why:** The owner accepts the short gap between the creation and the key. A second
save after the key exists succeeds.

**Examples:**

- Normal case: Nina creates "Garden" and saves a password 1 second later, before the
  key exists. Maroid refuses the save. Her next save succeeds.
- Unwanted case: Maroid stores the secret with no protection. This must not happen
  under `PSET-FR-015`.

## 5. Non-functional requirements

### `OWNKEY-NFR-001`

The protection key must exist in the secret store 10 seconds after Maroid answers the
creation of its user record or its workspace, while the secret store is ready.
Measure from the answer to the creation to the first successful read of the key.

**Why:** A person saves the settings of a new workspace within a minute.

### `OWNKEY-NFR-002`

The protection key must exist in the secret store 6 minutes after the secret store is
ready again, when the store was down at the creation. Measure from the moment the store
answers its health check as ready.

**Why:** `JOB-009` waits at most 5 minutes between two attempts.

### `OWNKEY-NFR-003`

Maroid must hold no permission to create a key in the secret store under a name other
than the name of the protection key of a user record or a workspace. Verify against the
permissions that the secret store grants to Maroid.

**Why:** A hub that an attacker reaches mints no other key.

## 6. Constraints from the guidelines

| Rule      | Guideline        | Effect on this feature                                                                               |
| --------- | ---------------- | ---------------------------------------------------------------------------------------------------- |
| `JOB-009` | Jobs             | The creation of a key is a task of the outbox. It commits with the record.                           |
| `JOB-010` | Jobs             | The creation of a key adds one type and one handler.                                                 |
| `OWN-002` | Record ownership | Only an administrator creates a user record, and no user record ends. No key of a user is destroyed. |
| `OWN-003` | Record ownership | A task names its owner. The creation of a key acts for no owner, as the destruction.                 |

`ADR-0028` reverses the decision of `PSET-DD-004` and `PSECRET-DD-003` that the hub
creates no key, and changes `WSREMOVE-DD-004`.

## 7. Open questions

| #   | Question                                                                  | Owner  | Answer                                |
| --- | ------------------------------------------------------------------------- | ------ | ------------------------------------- |
| 1   | Does a save before the key exists fail, or create the key at that moment? | Temuri | It fails. See `OWNKEY-FR-009`.        |
| 2   | Do the user records and the workspaces that exist today receive a key?    | Temuri | Yes. See `OWNKEY-FR-007`.             |
| 3   | Does this feature destroy the key of a user?                              | Temuri | No. The key allows it. See section 3. |

Answer every question before the approval. An open question blocks stage 2.

## Retired identifiers

This file has no retired identifier.
