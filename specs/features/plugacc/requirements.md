---
id: PLUGACC
title: The administrator, the plugin allowlist, and the enablement of a plugin
type: requirements
status: approved
created: 2026-10-02
updated: 2026-10-02
approved_by: Temuri
approved_on: 2026-10-02
constrained_by: [SEC, OWN, ERR, EXT, TG, JOB, ARC, CLI]
---

# Requirements: The administrator, the plugin allowlist, and the enablement of a plugin

## 1. Problem

Every loaded plugin serves every workspace. A pension plugin that one person
installed for themselves appears in the workspace of the household, and nobody can
turn it off there. A manager cannot choose which plugins their workspace uses.

The person who runs the instance adds a user, issues an invitation, and blocks a user
from the command line or with a statement against the database. No page of the web
shell does any of it, and nothing limits which plugins a user turns on.

## 2. Users

- An administrator: adds and blocks people, decides which plugins each person can turn
  on, and fixes a workspace that lost its way.
- A manager: turns on the plugins that the workspace needs, from their allowlist.
- A member: sees and uses the plugins that the workspace turned on, and no other.

## 3. Out of scope

- Deleting a user record. `OWN-002` keeps every record.
- An allowlist for one member inside one workspace. The workspace decides for every
  member.
- A history of the changes to an allowlist or to an enablement.
- A limit on the count of plugins that a workspace turns on.

## 4. Functional requirements

### `PLUGACC-FR-001`

The command that creates a user record or issues an invitation for one must mark the
record as an administrator when the person who runs it asks for it.

**Why:** An instance needs its first administrator before any page can make one. The
command reaches an existing record too, so an instance that holds users and no
administrator gets one.

### `PLUGACC-FR-002`

An administrator must list every user record, with its status and whether it is an
administrator.

### `PLUGACC-FR-003`

An administrator must create a user record and obtain its invitation from the web
shell.

**Why:** `EXTID-FR-010` gives the same action on the command line.

### `PLUGACC-FR-004`

An administrator must obtain a new invitation for a user record that already exists.

### `PLUGACC-FR-005`

An administrator must block a user record.

### `PLUGACC-FR-006`

An administrator must unblock a user record.

### `PLUGACC-FR-007`

Maroid must refuse a change that leaves the instance with no active administrator.

**Why:** Only an administrator reaches the pages of this feature. With none left, the
command line is the only way back.

**Examples:**

- Normal case: two administrators, and one of them takes the mark from themselves.
  Maroid accepts it.
- Unwanted case: the only administrator takes the mark from themselves. Maroid
  refuses it.
- Unwanted case: the only administrator blocks their own record. Maroid refuses it.

### `PLUGACC-FR-008`

An administrator must set the plugin allowlist of a user.

**Examples:**

- Normal case: an administrator puts the electricity plugin and the garden plugin on the
  allowlist of a person.
- Limit case: an empty allowlist. The person turns on no plugin in any workspace.
- Unwanted case: the allowlist names a plugin that the hub did not load. Maroid refuses
  it and names the plugin.

### `PLUGACC-FR-009`

An administrator must list every workspace of the instance, with its members and the
plugins that it turns on.

### `PLUGACC-FR-010`

An administrator must perform every action of a manager on the members of any
workspace, whether or not they are a member of it.

**Why:** A workspace whose only manager left the household needs a new manager, and no
member can name one. `PERMS-FR-008` still holds for the change.

### `PLUGACC-FR-029`

An administrator must mark another user record as an administrator.

### `PLUGACC-FR-030`

An administrator must take the mark of an administrator from a user record.

### `PLUGACC-FR-011`

A plugin must start disabled in every workspace.

### `PLUGACC-FR-012`

A manager must enable a plugin that their plugin allowlist holds.

**Examples:**

- Normal case: the allowlist of the manager holds the garden plugin. The manager turns
  it on, and every member reaches it.
- Unwanted case: the allowlist of the manager does not hold the pension plugin. Maroid
  refuses it and names the plugin.

### `PLUGACC-FR-013`

An administrator must enable any loaded plugin in any workspace.

### `PLUGACC-FR-014`

A manager must disable a plugin in their workspace.

### `PLUGACC-FR-027`

An administrator must disable a plugin in any workspace.

### `PLUGACC-FR-015`

Maroid must show every record and every setting of a plugin again when a workspace
enables it after a disable.

**Why:** A household that turns a plugin off for one month keeps its history.

### `PLUGACC-FR-016`

Maroid must keep a plugin enabled when the plugin allowlist of the manager who enabled
it no longer holds it.

**Why:** The allowlist decides who turns a plugin on, not who uses it. `SEC-012`.

### `PLUGACC-FR-017`

A member must list the plugins that the workspace enables.

### `PLUGACC-FR-018`

Maroid must answer a web route or a function for an agent of a plugin that the acting
workspace does not enable exactly as it answers one that does not exist.

**Why:** A different answer tells the caller that the plugin serves another workspace.

### `PLUGACC-FR-019`

Maroid must drop a command of the bot of a plugin that the selected workspace does not
enable, with no answer, and run nothing.

**Why:** The bot already drops an update from a sender with no user record in silence.
`SEC-012` makes the command absent, and an absent command answers nothing.

### `PLUGACC-FR-031`

The menu of commands of a chat must name no command of a plugin that its selected
workspace does not enable.

**Why:** `SEC-012` removes the command from the chat, so the menu offers nothing that
the bot drops.

**Examples:**

- Normal case: a person selects the garden workspace. The menu of the chat shows the
  commands of the garden plugin and not the commands of the parking plugin.
- Normal case: a manager turns the parking plugin on. The menu of every chat that
  selected the workspace gains its commands.
- Limit case: a chat selected no workspace. The menu names the commands of the hub
  alone.

### `PLUGACC-FR-020`

A scheduled task of a plugin must run for each workspace that enables the plugin, and
for no other.

### `PLUGACC-FR-021`

A user must read the loaded plugins that their plugin allowlist holds, and no other.

**Why:** A person sees what they can turn on, and nothing that they never can.

### `PLUGACC-FR-028`

An administrator must read every loaded plugin.

### `PLUGACC-FR-022`

Maroid must refuse every action of `PLUGACC-FR-002` to `PLUGACC-FR-010`,
`PLUGACC-FR-013`, `PLUGACC-FR-027`, `PLUGACC-FR-029`, and `PLUGACC-FR-030` from a user
who is no administrator.

### `PLUGACC-FR-023`

An administrator must reach every action of `PLUGACC-FR-002` to `PLUGACC-FR-010`,
`PLUGACC-FR-013`, `PLUGACC-FR-027`, `PLUGACC-FR-029`, and `PLUGACC-FR-030` from the web
shell.

### `PLUGACC-FR-024`

A manager must enable and disable a plugin of their workspace from the web shell.

### `PLUGACC-FR-025`

The navigation of the web shell must name only the plugins that the acting workspace
enables.

### `PLUGACC-FR-026`

Every workspace that exists before this feature must enable no plugin, and every user
record must hold an empty plugin allowlist.

**Why:** The owner chose to start closed. An administrator turns each plugin on.

## 5. Non-functional requirements

### `PLUGACC-NFR-001`

A plugin that a workspace disables must answer as absent in the first request after
Maroid answers the change. Measure at the response to that request.

### `PLUGACC-NFR-002`

Maroid must send the new menu of a chat to Telegram within 10 seconds of the change of
its selection or of an enablement of its workspace. Measure at the hub, from the
response to the change to the answer of Telegram to the menu.

**Why:** The menu of a chat that lags shows a command that the bot drops.

## 6. Invariants

### `PLUGACC-INV-001`

A plugin serves no workspace that does not enable it.

### `PLUGACC-INV-002`

An administrator reads no record of a workspace that they are not a member of.

### `PLUGACC-INV-003`

An instance that holds an administrator holds at least one active administrator.

## 7. Constraints from the guidelines

| Rule      | Guideline        | Effect on this feature                                                        |
| --------- | ---------------- | ------------------------------------------------------------------------------- |
| `SEC-011` | Security         | The administrator, and the access to every workspace without its records.     |
| `SEC-012` | Security         | The enablement, the allowlist, the answer of a disabled plugin, and the menu of a chat. |
| `SEC-004` | Security         | A blocked record reaches nothing at its next request.                         |
| `OWN-002` | Record ownership | No user record is deleted. A block keeps every record.                        |
| `ERR-003` | Errors           | A disabled plugin answers `not-found`.                                         |
| `JOB-005` | Jobs             | A job of a plugin runs for each workspace that enables it.                    |
| `ARC-008` | Architecture     | The hub names no plugin in a rule of the enablement.                          |
| `CLI-001` | Command line     | `maroid user invite` gains the mark of an administrator.                      |
| `TG-004`  | Telegram         | The hub registers the commands of a chat for that chat, beside the default scope. |

## 8. Open questions

| #   | Question                                                                         | Owner  | Answer |
| --- | -------------------------------------------------------------------------------- | ------ | ------ |
| 1   | What do the workspaces and the users that exist before this feature receive?    | Temuri | Nothing. `PLUGACC-FR-026`. |
| 2   | How does a user become an administrator?                                         | Temuri | The command line makes the first one, `PLUGACC-FR-001`. An administrator marks the others, `PLUGACC-FR-029`. |
| 3   | Does a command of the bot of a disabled plugin leave the menu?                   | Temuri | Yes, as `SEC-012` gives. A use of it gets no answer. |

Answer every question before the approval. An open question blocks stage 2.

## Retired identifiers

This file has no retired identifier.
