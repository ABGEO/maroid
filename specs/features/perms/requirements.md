---
id: PERMS
title: The workspace roles, and the permission that every action needs
type: requirements
status: approved
created: 2026-10-02
updated: 2026-10-02
approved_by: Temuri
approved_on: 2026-10-02
constrained_by: [OWN, SEC, ERR, PLG, ARC, TG]
---

# Requirements: The workspace roles, and the permission that every action needs

## 1. Problem

`WSPACE` makes every member of a workspace equal. A gardener who only waters the
plants of a friend can also rename the garden, remove the friend, and change the
credential of every plugin. A household cannot give a guest read access alone.

`OWN-011` gives three roles and `SEC-013` gives a permission to every action, and no
part of Maroid holds either today. A plugin cannot say that its delete needs more than
its list.

## 2. Users

- A manager: decides who joins and with which role, and keeps the workspace in hand.
- An editor: changes the records of the place, and changes nobody's access.
- A viewer: reads the records, and changes nothing.
- A plugin author: names what each action needs once, and writes no check.

## 3. Out of scope

- A role that a workspace or a plugin defines. The three roles are fixed.
- A permission that depends on the record: an editor who changes their own records
  and no other.
- The administrator and the enablement of a plugin. `PLUGACC` gives them.
- A scheduled task and a device message. Neither has an acting user, so neither has a
  role.
- A menu of the bot that differs by role.
- A history of the changes to a role.

## 4. Functional requirements

### `PERMS-FR-001`

Every membership must hold exactly one role: manager, editor, or viewer.

### `PERMS-FR-002`

A role must hold every permission that a role below it holds. The order is manager,
editor, viewer.

**Why:** A person learns one ladder, and a higher role never loses an action.

### `PERMS-FR-003`

Maroid must make the user who creates a workspace its manager.

### `PERMS-FR-004`

Maroid must make every member that exists before this feature a manager.

**Why:** Every member was equal before this feature, and a manager keeps every action
they had.

### `PERMS-FR-005`

The roles must reach these actions of the hub, and no role below the one named:

| Action                                                    | Lowest role |
| --------------------------------------------------------- | ----------- |
| Read the workspace, its members, and their roles          | viewer      |
| Leave the workspace                                       | viewer      |
| Read the settings of a plugin                             | viewer      |
| Change the settings of a plugin                           | editor      |
| Rename the workspace                                      | manager     |
| List the user records to add                              | manager     |
| Add a member, remove a member, or change a role           | manager     |

**Examples:**

- Normal case: an editor changes the account number of the electricity bill.
- Unwanted case: a viewer renames the workspace. Maroid refuses it.

### `PERMS-FR-006`

A manager must give a role to a member that they add.

**Examples:**

- Normal case: a manager adds a person as an editor.
- Limit case: the manager names no role. Maroid refuses the change and names the
  missing role.

### `PERMS-FR-007`

A manager must change the role of a member.

### `PERMS-FR-008`

Maroid must refuse a change that leaves a workspace with no manager.

**Examples:**

- Normal case: two managers, and one of them leaves. Maroid accepts it.
- Unwanted case: the only manager leaves, even as the only member. Maroid refuses it.
- Unwanted case: the only manager changes their own role to editor. Maroid refuses it.
- Unwanted case: two managers each change the role of the other to viewer at the same
  moment. Maroid accepts one and refuses the other.

### `PERMS-FR-009`

A user must read their role in each workspace that they are a member of.

### `PERMS-FR-010`

A member must read the role of every member of the workspace.

### `PERMS-FR-011`

A plugin must declare each permission that it checks, with the lowest role that holds
it.

**Why:** The author of the plugin knows that a delete needs more than a list.

### `PERMS-FR-012`

Every action of a plugin that acts in a workspace must name one permission that the
plugin declares. An action is a web route, a function for an agent, or a command of
the bot.

### `PERMS-FR-013`

Maroid must refuse to load a plugin that holds an action with no permission, or with a
permission that the plugin does not declare.

**Why:** A plugin that forgets a permission fails at the start, before any person
meets the gap.

### `PERMS-FR-014`

A permission of one plugin must grant no action of another plugin.

**Examples:**

- Normal case: two plugins each declare "write". Each one guards its own actions.
- Unwanted case: a role that holds "write" of one plugin reaches the "write" of the
  other.

### `PERMS-FR-015`

Maroid must refuse an action when the role of the acting user in the acting workspace
does not hold its permission.

**Examples:**

- Normal case: an editor deletes a plant through a route that needs editor.
- Unwanted case: a viewer deletes a plant from the web shell. Maroid refuses it.
- Unwanted case: a viewer asks an agent to delete a plant. Maroid refuses the call.
- Unwanted case: a viewer sends the command of the bot that deletes a plant. Maroid
  refuses it and runs nothing.

### `PERMS-FR-016`

Maroid must name the permission and the lowest role that holds it, when it refuses an
action under `PERMS-FR-015`.

**Why:** A member asks the right person for the right role.

### `PERMS-FR-017`

Maroid must report to a member the permissions that their role holds in the acting
workspace.

**Why:** The web shell and the page of a plugin hide a control that the role does not
reach. Maroid still checks every action.

### `PERMS-FR-018`

The web shell must show no control for an action that the role of the person does not
reach.

### `PERMS-FR-019`

A manager must change the role of a member from the web shell.

### `PERMS-FR-020`

The form of the web shell that adds a member must start on the role viewer.

**Why:** Viewer is the least access, so a manager who does not change the form grants
the least.

## 5. Non-functional requirements

### `PERMS-NFR-001`

A member whose role a manager lowers must lose the actions of the old role in the first
request after Maroid answers the change. Measure at the response to that request.

### `PERMS-NFR-002`

The check of a permission must add no more than 2 milliseconds to a request at the
95th percentile, measured from the read of the membership to the start of the action.

**Why:** The check runs on every action of every plugin.

## 6. Invariants

### `PERMS-INV-001`

Every workspace has at least one manager.

## 7. Constraints from the guidelines

| Rule      | Guideline        | Effect on this feature                                                    |
| --------- | ---------------- | --------------------------------------------------------------------------- |
| `OWN-011` | Record ownership | The three roles, their order, and the last manager rule.                  |
| `SEC-013` | Security         | Every action that acts in a workspace names one permission. The hub checks it. |
| `ERR-003` | Errors           | A refusal under `PERMS-FR-015` answers `permission-denied`.               |
| `PLG-006` | Plugin model     | The declaration of the permissions is a capability.                       |
| `PLG-007` | Plugin model     | A plugin declares its permissions through the plugin interface.           |
| `ARC-008` | Architecture     | The hub names no permission of a plugin.                                  |
| `TG-003`  | Telegram         | A command of a plugin names its permission in its metadata.               |

`WSPACE` releases together with this feature, as `WSPACE-DD-005` gives. At the
approval of this document, `WSPACE-FR-004`, `WSPACE-FR-005`, `WSPACE-FR-006`, and
`WSPACE-FR-008` take the actor of `PERMS-FR-005`, and the limit case of
`WSPACE-FR-012` and `WSPACE-SC-008` follow `PERMS-FR-008`.

## 8. Open questions

| #   | Question                                                                         | Owner  | Answer |
| --- | -------------------------------------------------------------------------------- | ------ | ------ |
| 1   | Can the last member leave a workspace?                                           | Temuri | No. `PERMS-FR-008` and `OWN-011` keep a manager in every workspace. |
| 2   | Does a command of the bot that the role does not reach leave the menu?           | Temuri | No. It stays, and the use answers under `PERMS-FR-016`. |
| 3   | Which role does a new member get?                                                | Temuri | The one that the manager picks. The form starts on viewer. |

Answer every question before the approval. An open question blocks stage 2.

## Retired identifiers

This file has no retired identifier.
