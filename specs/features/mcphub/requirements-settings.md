---
id: MCPHUB
title: The settings of a plugin over the Model Context Protocol
type: requirements
status: approved
created: 2026-09-21
updated: 2026-09-21
approved_by: Temuri
approved_on: 2026-09-21
constrained_by: [SEC, OWN, LOG, ARC]
---

# Requirements: The settings of a plugin over the Model Context Protocol

`requirements.md` holds the server, the identity, and the tools that a plugin
declares. This file holds the settings of a plugin. `SPC-001` divides the two,
because `requirements.md` reached the size that `LNG-013` gives.

## 1. Problem

A plugin that reads an external account needs the credential of the acting user.
That person fills it in the deck, and in no other place. `PSET` gave the field,
the protection, and the two routes that the deck calls.

An agent reaches none of it. It reports which plugin the hub loaded, and it
reads no field of any plugin. It cannot tell a plugin that waits for a setting
from a plugin that runs, and it changes no value that a person asks it to
change.

## 2. Users

- The owner: asks an agent to read the settings of a plugin, and to change a
  value that is no secret.
- A Maroid user who is not the owner: reads and changes their own settings the
  same way, and reaches the settings of no other person.

## 3. Out of scope

- The settings of one user that another user reads or changes.
- A history of the changes of a setting.
- A tool that removes every stored value of a plugin in one call.
- A report of each plugin whose settings are absent. An agent reads the settings
  schema and the stored values of one plugin, and derives it.
- A path that fills a secret outside the deck.

## 4. Functional requirements

### `MCPHUB-FR-015`

The hub must report the settings schema of a plugin that an MCP client names.

**Why:** An agent needs the name, the kind, and the requirement of each field
before it sends one value.

**Examples:**

- Normal case: the call names a plugin that declares three fields. The report
  names all three, with the kind of each.
- Unwanted case: the call names a plugin that declares no settings schema. The
  hub answers a failure that names the plugin.

### `MCPHUB-FR-016`

The hub must report the settings that the acting user stored for a plugin that
an MCP client names.

**Why:** An agent that changes one field keeps every other field. It reads the
stored values to know which field holds a value. `PSET-FR-004` and `PSET-FR-005`
govern what that report holds, so a secret stays unreadable.

**Examples:**

- Normal case: the acting user stored two of three fields. The report names the
  two values, and it marks the third as empty.
- Limit case: the acting user stored no value. The report holds no value, and it
  is no failure.
- Unwanted case: another user stored a value for the same plugin. The report
  holds no value of that user.

### `MCPHUB-FR-017`

The hub must store the settings that an MCP client sends for the acting user.

**Why:** The owner corrects a wrong value in the session that found it.
`MCPHUB-FR-019` bounds which field a save changes.

**Examples:**

- Normal case: the call names one field. The hub stores that field, and it keeps
  every other stored field. See `PSET-FR-006`.
- Limit case: the call names a value of 4097 characters. The hub stores nothing.
- Unwanted case: the call names a field that the settings schema does not
  declare. The hub stores nothing.

### `MCPHUB-FR-018`

The hub must name each field that caused it to reject a save from an MCP client.

**Why:** An agent corrects a rejected save with no further question to the
person, when the failure names the field and the reason. `PSET-FR-009` gives the
same report to the deck.

### `MCPHUB-FR-019`

The hub must reject a save from an MCP client that changes a secret field.

**Why:** A credential that an agent sends passes the MCP client, the transcript
of that client, and the model that reads it. The deck gives that value a path
that starts at the person and ends at the hub.

**Examples:**

- Normal case: the call names each field that is no secret field. The hub stores
  them.
- Limit case: the call names a secret field, and the value is the mask. The hub
  stores every other field, and it keeps the stored secret. See `PSET-FR-006`.
- Unwanted case: the call names a secret field, and the value is a credential.
  The hub stores nothing, and the failure names that field.
- Limit case: the settings schema requires a secret field, and the acting user
  stored no value for it. The hub stores nothing. See `PSET-FR-008`.

## 5. Constraints from the guidelines

| Rule      | Guideline         | Effect on this feature                                                                     |
| --------- | ----------------- | ------------------------------------------------------------------------------------------ |
| `ARC-008` | Architecture      | The two tools read the registry that holds the settings schema of each plugin. They name no plugin. |
| `SEC-004` | Security          | The user record grants the call. The verification that `MCPHUB-FR-002` gives applies with no change. |
| `OWN-003` | Record ownership  | `MCPHUB-INV-001` gives the call one acting user. The settings of that user answer it.      |
| `OWN-005` | Record ownership  | A save names no user. The policy sets the owner of the row.                                |
| `OWN-006` | Record ownership  | A call with no acting user reads no row and writes no row.                                 |
| `LOG-008` | Logging           | An MCP client can send a credential that `MCPHUB-FR-019` rejects. No log line holds that value. |

## 6. Open questions

| #   | Question                                                                                        | Owner  | Answer                                                                                                                             |
| --- | ----------------------------------------------------------------------------------------------- | ------ | ------------------------------------------------------------------------------------------------------------------------------------ |
| 1   | Does a tool store a secret field, or does a person type a secret in the deck only?              | Temuri | A person types it in the deck. `MCPHUB-FR-019` states it. A setup that needs a credential ends in the deck, and no credential reaches the MCP client. |
| 2   | Does one call report the settings of every plugin, or of the one plugin that the call names?    | Temuri | The one plugin that the call names. `MCPHUB-FR-015` and `MCPHUB-FR-016` state it.                                                   |

## Retired identifiers

This file has no retired identifier.
