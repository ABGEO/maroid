---
id: PLUGACC
title: The administrator, the plugin allowlist, and the enablement of a plugin
type: requirements
status: approved
created: 2026-10-02
updated: 2026-10-05
approved_by: Temuri
approved_on: 2026-10-05
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
  on from the day they add the person, and manages the members and the plugins of any
  workspace.
- A manager: turns on the plugins that the workspace needs, from their allowlist.
- A member: sees and uses every plugin that the workspace turned on, whoever turned it
  on, and no other.

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

### `PLUGACC-FR-032`

An administrator must set the plugin allowlist of a user record when they create it.

**Why:** Some plugins hold sensitive records. The administrator decides which plugins a
person can turn on before the person signs in for the first time.

**Examples:**

- Normal case: an administrator creates a record for Nina with the garden plugin on
  her allowlist. Nina turns the garden plugin on in her first workspace.
- Limit case: an administrator creates a record with an empty allowlist. `PLUGACC-FR-008`
  adds a plugin later.
- Unwanted case: the allowlist names a plugin that the hub did not load. Maroid refuses
  it, names the plugin, and creates no record.

### `PLUGACC-FR-034`

An administrator must change the first name and the last name of a user record.

**Why:** A person who received an invitation with a misspelled name, or no name, keeps
the record. A new record for them would lose their workspaces.

**Examples:**

- Normal case: an administrator corrects "Nnia" to "Nina". The record keeps its
  identities, its workspaces, and its allowlist.
- Limit case: an administrator clears the last name. The record holds no last name.

### `PLUGACC-FR-035`

Maroid must refuse a change to the plugin allowlist of an administrator, at the creation
of the record and after it.

**Why:** An administrator turns on any loaded plugin, as `PLUGACC-FR-013` gives. An
allowlist on their record would suggest a limit that does not hold.

**Examples:**

- Unwanted case: an administrator creates a record that is an administrator, with the
  garden plugin on its allowlist. Maroid refuses it and creates no record.
- Unwanted case: an administrator adds the garden plugin to the allowlist of another
  administrator. Maroid refuses it.
- Limit case: a person with an allowlist becomes an administrator. The allowlist stays,
  has no effect while the mark holds, and returns to effect when the mark goes.

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

A member must read every plugin that the workspace enables, whatever their own plugin
allowlist holds.

**Why:** The workspace acts on its enabled plugins, whoever enabled them. The allowlist
decides who turns a plugin on, not who uses it. `PLUGACC-FR-016` gives the same rule.

**Examples:**

- Normal case: Ana enables the garden plugin in H. Gio, a member of H with an empty
  allowlist, reads the garden plugin in H and uses it.
- Normal case: an administrator enables the pension plugin in H. Every member of H reads
  it, and no allowlist of a member holds it.

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

A user must read, as the plugins that they can turn on, the loaded plugins that their
plugin allowlist holds, and no other.

**Why:** A person sees what they can turn on, and nothing that they never can.
`PLUGACC-FR-017` gives the plugins that a workspace already uses.

### `PLUGACC-FR-028`

An administrator must read every loaded plugin.

### `PLUGACC-FR-022`

Maroid must refuse every action of `PLUGACC-FR-002` to `PLUGACC-FR-010`,
`PLUGACC-FR-013`, `PLUGACC-FR-027`, `PLUGACC-FR-029`, `PLUGACC-FR-030`,
`PLUGACC-FR-032`, and `PLUGACC-FR-034` from a user who is no administrator.

### `PLUGACC-FR-023`

An administrator must reach every action of `PLUGACC-FR-002` to `PLUGACC-FR-010`,
`PLUGACC-FR-013`, `PLUGACC-FR-027`, `PLUGACC-FR-029`, `PLUGACC-FR-030`,
`PLUGACC-FR-032`, and `PLUGACC-FR-034` from the web shell.

### `PLUGACC-FR-024`

A manager must enable and disable a plugin of their workspace from the web shell.

### `PLUGACC-FR-025`

The navigation of the web shell must name only the plugins that the acting workspace
enables.

### `PLUGACC-FR-036`

The web shell must show the plugins of a workspace on one page of that workspace. Every
member must read there the plugins that the workspace enables. A person who can turn a
plugin on in the workspace must also read there the plugins of their allowlist that the
workspace does not enable.

**Why:** A person uses a plugin inside a workspace and turns it on inside a workspace. A
list outside a workspace shows a member nothing that they use.

**Examples:**

- Normal case: H enables Q. Gio, a viewer of H with an empty allowlist, opens the plugins
  of H. He reads Q, and no plugin to turn on.
- Normal case: H enables Q, and the allowlist of Ana, a manager of H, holds P. Ana reads Q
  as in use, and P as a plugin to turn on.
- Limit case: H enables no plugin. Gio reads that H uses no plugin.
- Unwanted case: Gio opens a list of plugins that names no workspace, and reads no plugin
  while H enables Q.

### `PLUGACC-FR-037`

The web shell must show an administrator every loaded plugin on one page that names no
workspace, with the workspaces that enable each plugin.

**Why:** An administrator sets the allowlists and enables a plugin in any workspace. They
need the whole catalog in one place.

**Examples:**

- Normal case: the hub loaded P and Q, and H enables Q. Zura reads P with no workspace,
  and Q with H.
- Unwanted case: Ana, who is no administrator, reaches the page.

### `PLUGACC-FR-033`

The web shell must offer an administrator no page of a plugin in a workspace that they
are not a member of.

**Why:** `PLUGACC-INV-002` closes the records of that workspace to the administrator. A
page that the hub answers with not-found tells them nothing.

**Examples:**

- Normal case: an administrator opens the plugins of the workspace of John. The page
  switches each plugin, and the navigation names no page of a plugin.
- Normal case: the same administrator opens a workspace that they are a member of. The
  navigation names each plugin that it enables.

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
| 4   | Does a member use a plugin that their own allowlist does not hold?               | Temuri | Yes. The workspace acts on its enabled plugins, whoever enabled them. `PLUGACC-FR-017`. |
| 5   | Does a removal from an allowlist disable the plugin where the person turned it on? | Temuri | No. The plugin stays on, and an administrator disables it by hand. `PLUGACC-FR-016`, `PLUGACC-FR-027`. |
| 6   | Does an administrator keep the members and the plugins of any workspace?         | Temuri | Yes. `PLUGACC-FR-010`, `PLUGACC-FR-013`, `PLUGACC-FR-027`. |

Answer every question before the approval. An open question blocks stage 2.

## Retired identifiers

This file has no retired identifier.
