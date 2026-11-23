---
id: WSREMOVE
title: The removal of a workspace and of every record it holds
type: requirements
status: approved
created: 2026-11-23
updated: 2026-11-23
approved_by: Temuri
approved_on: 2026-11-23
constrained_by: [OWN, SEC, DAT, ERR, API]
---

# Requirements: The removal of a workspace and of every record it holds

## 1. Problem

No workspace ends today. A person who moves out of a flat, or who made a workspace
to try a plugin, keeps it in their list. Its records, its files, and its plugin secrets
stay in Maroid, and no person can delete them. `OWN-010` forbids the delete today.

## 2. Users

- A manager who is the last member: removes a workspace that nobody needs.
- A manager of a shared workspace: removes the other members first, then the workspace.
- An administrator: removes any workspace of the instance, with its members in it.

## 3. Out of scope

- An archive, a soft delete, or a restore of a removed workspace.
- An export of the records before the removal.
- A removal through the Telegram bot or an MCP tool.
- A notice to the members of a workspace that an administrator removes.
- The removal of a user record. `OWN-002` keeps every user record.
- A history of the removals.
- The delete of the records from a backup of the database.

## 4. Functional requirements

### `WSREMOVE-FR-001`

A manager must remove a workspace in which they are the only member.

**Examples:**

- Normal case: Nina is the only member of "Old flat". Nina removes it.
- Limit case: "Old flat" is the last workspace of Nina. Maroid removes it.
- Unwanted case: a viewer asks to remove the workspace. Maroid refuses it under
  `PERMS-FR-015`.

### `WSREMOVE-FR-002`

Maroid must refuse a removal by a manager while the workspace holds another member.

**Why:** The other member reads the records that the removal deletes. A manager removes
each member first, so each member leaves the workspace before its records go.

**Examples:**

- Normal case: "Home" has three members. The manager asks to remove it. Maroid refuses
  it and deletes nothing.
- Limit case: the other member is a second manager. Maroid refuses the removal.
- Unwanted case: one manager adds a member while the other manager removes the
  workspace at the same moment. Maroid accepts one of the two and refuses the other.

### `WSREMOVE-FR-003`

Maroid must name the count of the other members when it refuses a removal under
`WSREMOVE-FR-002`.

**Why:** The manager knows how many members to remove before the next attempt.

### `WSREMOVE-FR-004`

An administrator must remove a workspace that holds members, whether or not the
administrator is a member of it.

**Why:** A workspace whose members left the household, or that a person abandoned,
needs no manager to end it.

**Examples:**

- Normal case: an administrator removes "Garden", which holds two members. Neither
  member reaches it after the removal.
- Limit case: the administrator is the only member. Maroid removes it as under
  `WSREMOVE-FR-001`.

### `WSREMOVE-FR-005`

Maroid must delete every record of the hub and of every plugin that the workspace
scopes, when it removes the workspace.

**Why:** A removal is a delete. No record stays hidden behind a flag.

**Examples:**

- Normal case: the removal deletes the plants, the photos, the notification providers,
  the settings of every plugin, and every enablement of the workspace.
- Limit case: the workspace disabled the garden plugin, which still holds its plants
  under `PLUGACC`. The removal deletes the plants.
- Limit case: the instance no longer loads a plugin that holds rows of the workspace.
  The removal deletes those rows.
- Unwanted case: a plugin fails to delete its rows of the workspace. Maroid deletes no
  record and answers the removal with an error.

### `WSREMOVE-FR-006`

Maroid must keep every record that a user scopes when it removes a workspace.

**Why:** `OWN-004` binds such a record to a person in every workspace, not to the
workspace that is removed.

**Examples:**

- Normal case: Nina removes "Old flat". The check-ins and the pension contributions of
  Nina stay.

### `WSREMOVE-FR-007`

Maroid must delete every file of the workspace from the file store.

### `WSREMOVE-FR-008`

Maroid must delete every plugin secret of the workspace from the secret store.

### `WSREMOVE-FR-009`

Maroid must answer the removal without waiting for the file store or the secret store.

**Why:** A store that fails does not keep a removed workspace alive.

**Examples:**

- Normal case: both stores are ready. Maroid answers the removal, then deletes the
  files and the secrets.
- Limit case: the secret store is down. Maroid answers the removal, and the secrets
  wait for `WSREMOVE-FR-010`.

### `WSREMOVE-FR-010`

Maroid must try the delete of a file or a plugin secret again until it succeeds, when
the file store or the secret store fails.

**Examples:**

- Normal case: the file store is down for 2 hours. Maroid deletes the files after the
  store returns.
- Limit case: the hub restarts before the delete. Maroid deletes the files after the
  restart.
- Unwanted case: the file store refuses the delete five times in a row. Maroid tries
  again.

### `WSREMOVE-FR-011`

The web shell must ask the person to type the name of the workspace before it sends a
removal.

**Why:** A removal cannot be undone. The typed name stops a removal of the wrong
workspace.

**Examples:**

- Normal case: the person types "Old flat". The web shell sends the removal.
- Unwanted case: the person types "Old flats". The web shell sends nothing.

### `WSREMOVE-FR-012`

A manager must reach the action of `WSREMOVE-FR-001` from the web shell.

### `WSREMOVE-FR-013`

An administrator must reach the action of `WSREMOVE-FR-004` from the web shell.

### `WSREMOVE-FR-014`

The web shell must open another workspace of the person after they remove the acting
workspace.

**Examples:**

- Normal case: Nina removes "Old flat". The web shell opens "Home", the first of her
  workspaces in the order of `WSPACE-FR-003`.
- Limit case: Nina holds no other workspace. The web shell shows the page of
  `WSPACE-FR-024`.

### `WSREMOVE-FR-015`

Maroid must clear the selected workspace of every Telegram chat that selected the
removed workspace.

**Examples:**

- Normal case: Gio is a member of "Garden" and "Home". The chat of Gio selected
  "Garden". An administrator removes "Garden". The next command of Gio runs in "Home"
  under `WSPACE-FR-017`.

## 5. Non-functional requirements

### `WSREMOVE-NFR-001`

A member of a removed workspace must reach no record of it in the first request after
Maroid answers the removal. Measure at the response to that request.

**Why:** A second tab, an agent, or a bot chat of another member still names the
workspace.

### `WSREMOVE-NFR-002`

The file store must hold no bytes of a file of a removed workspace 1 hour after the
removal. Measure from the answer to the removal, or from the moment the file store is
ready again, whichever is later.

**Why:** `FILES-NFR-005` gives the same limit to a deleted file.

### `WSREMOVE-NFR-003`

The secret store must hold no plugin secret of a removed workspace 1 hour after the
removal. Measure from the answer to the removal, or from the moment the secret store is
ready again, whichever is later.

## 6. Invariants

### `WSREMOVE-INV-001`

No record of the hub or of a plugin names a workspace that does not exist.

## 7. Constraints from the guidelines

| Rule      | Guideline        | Effect on this feature                                                                   |
| --------- | ---------------- | ---------------------------------------------------------------------------------------- |
| `OWN-010` | Record ownership | It forbids the delete of a workspace. An ADR changes it before the code.                 |
| `SEC-011` | Security         | An administrator changes the members and the enablements. The same ADR adds the removal. |
| `OWN-004` | Record ownership | A table that a user scopes keeps its rows. See `WSREMOVE-FR-006`.                        |
| `OWN-011` | Record ownership | The last manager rule applies to a workspace that exists. A removal ends the workspace.  |
| `SEC-013` | Security         | The removal by a manager needs a permission of the hub that only a manager holds.        |
| `DAT-003` | Data             | A row of a plugin that names a file blocks the delete of the file. The row goes first.   |
| `ERR-003` | Errors           | The refusal of `WSREMOVE-FR-002` needs a problem type of the hub.                        |
| `API-003` | HTTP API         | The route of this feature sits under `/workspaces`.                                      |

The same ADR removes "Deleting or archiving a workspace" from section 3 of `WSPACE`,
and adds the removal to the table of `PERMS-FR-005` with the lowest role manager.

## 8. Open questions

| #   | Question                                                                                                                                                                | Owner  | Answer                                           |
| --- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------ | ------------------------------------------------ |
| 1   | A plugin that the instance no longer loads still holds rows of the workspace in its schema. Does the removal delete them, or does Maroid refuse the removal until then? | Temuri | The removal deletes them. See `WSREMOVE-FR-005`. |

Answer every question before the approval. An open question blocks stage 2.

## Retired identifiers

This file has no retired identifier.
