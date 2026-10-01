---
id: WSPACE
title: The workspaces, their members, and the workspace that a unit of work acts in
type: requirements
status: approved
created: 2026-10-01
updated: 2026-10-02
approved_by: Temuri
approved_on: 2026-10-01
constrained_by: [OWN, SEC, ERR, API, TG]
---

# Requirements: The workspaces, their members, and the workspace that a unit of work acts in

## 1. Problem

`ADR-0008` made the workspace the owner of every record of a plugin. No workspace
exists yet. A person cannot create one, cannot add the other people of the
household to it, and cannot say which one a request acts in. Two people who share a
garden or an electricity bill each hold a copy today, or share nothing.

## 2. Users

- A person alone: one workspace, and nobody else reads its records.
- A household member: creates a workspace for the household, adds the others, and
  reads the same bills as they do.
- A gardener: a member of the garden of a friend and of their own home, each kept apart.

## 3. Out of scope

- The administrator, the plugin allowlist, and the enablement of a plugin. `PLUGACC`
  gives them.
- The workspace roles, every permission, and the rule that keeps a manager. `PERMS`
  gives them, and it releases together with this feature.
- The first workspace of a new user record. `EXTID-FR-018` gives it.
- Deleting or archiving a workspace. `OWN-010` keeps every workspace.
- Moving a record from one workspace to another.
- Adding a person who holds no user record.
- A limit on the count of workspaces or of members.
- A history of the changes to a membership.

## 4. Functional requirements

### `WSPACE-FR-001`

An active user must create a workspace with a name of 1 to 64 characters.

**Examples:**

- Normal case: a person creates the workspace "Home".
- Limit case: a name of 64 characters. Maroid creates the workspace.
- Unwanted case: an empty name, or a name of 65 characters. Maroid creates nothing
  and names the field.

### `WSPACE-FR-002`

Maroid must make the user who creates a workspace its first member.

**Why:** A workspace that nobody belongs to reaches nobody.

### `WSPACE-FR-003`

A user must list the workspaces that they are a member of.

**Examples:**

- Normal case: a member of two workspaces reads both.
- Unwanted case: the list names a workspace that the user is not a member of.

### `WSPACE-FR-004`

A manager must rename a workspace, within the limits of `WSPACE-FR-001`.

**Why:** Two workspaces can share a name. Each person sees their own list only.

### `WSPACE-FR-005`

A manager must list the name of every active user record.

**Why:** A manager picks a new member from that list.

### `WSPACE-FR-006`

A manager must add an active user record as a member.

**Examples:**

- Normal case: a manager adds a person.
- Unwanted case: the person is already a member. Maroid changes nothing and says so.
- Unwanted case: the user record is blocked. Maroid adds nothing.

### `WSPACE-FR-008`

A manager must remove another member.

### `WSPACE-FR-009`

A member must leave a workspace.

### `WSPACE-FR-011`

A member must list the members of the workspace.

**Why:** A member knows who else reads the records of the place.

### `WSPACE-FR-012`

Maroid must keep every record of a workspace when a member leaves or a manager removes
one.

**Why:** The household keeps its bills when a person moves out.

**Examples:**

- Normal case: one of two members leaves. The other reads every record.
- Unwanted case: the last manager leaves. `PERMS-FR-008` refuses it, so a workspace with
  records always keeps a member who reaches them.

### `WSPACE-FR-013`

A request to Maroid must name the workspace that it acts in.

**Why:** One session reaches every workspace of its user. See `OWN-003`.

### `WSPACE-FR-014`

Maroid must answer a request that names a workspace of which the user is no member
exactly as it answers a request that names no workspace that exists.

**Why:** A different answer tells the requester that the workspace exists.

### `WSPACE-FR-015`

A person must select the workspace of a Telegram chat.

### `WSPACE-FR-016`

Maroid must keep the selected workspace of a chat until the person selects another.

### `WSPACE-FR-017`

Maroid must select the only workspace of a person for a chat that selected none.

**Why:** A person in one workspace never meets the question.

### `WSPACE-FR-018`

Maroid must ask the person to select a workspace when the chat selected none and the
person is a member of two or more. Maroid runs the command only after the selection.

**Examples:**

- Normal case: a person in two workspaces sends a command in a new chat. The bot
  names both and runs nothing.
- Unwanted case: the bot runs the command in one of the two that it picked.

### `WSPACE-FR-019`

Maroid must clear the selected workspace of a chat when the person stops being a
member of it.

### `WSPACE-FR-020`

The web shell must show the name of the acting workspace on every page that reads
the records of one.

### `WSPACE-FR-021`

A person must switch the web shell to another of their workspaces in at most 2 clicks
from any page.

### `WSPACE-FR-022`

The address of a page of the web shell must name its workspace.

**Why:** A link that one member sends opens the same workspace for the other member.
Two tabs show two workspaces.

### `WSPACE-FR-023`

The web shell must open the workspace that the person acted in last in that browser,
after a sign in.

**Why:** The browser keeps the choice, so Maroid stores nothing for it.

**Examples:**

- Normal case: a person left the deck on "Garden". The next sign in in the same
  browser opens "Garden".
- Limit case: the person signs in from a second browser for the first time. The web
  shell opens the first of their workspaces in the order of `WSPACE-FR-003`.
- Unwanted case: the person is no longer a member of "Garden". The web shell opens
  the first of their workspaces in the order of `WSPACE-FR-003`, and shows nothing of
  "Garden".

### `WSPACE-FR-024`

The web shell must tell a person with no workspace to create one, and show no page of
a plugin.

### `WSPACE-FR-025`

A person must reach every action of `WSPACE-FR-001`, `WSPACE-FR-004`,
`WSPACE-FR-006`, `WSPACE-FR-008`, `WSPACE-FR-009`, and `WSPACE-FR-011` from the web
shell.

**Why:** The deck is the place where a person meets the workspace first. The bot and
an agent reach the records, and the deck shapes the place that holds them.

### `WSPACE-FR-026`

Maroid must give every user record that exists before this feature one workspace,
with that user as its member.

**Why:** `ADR-0008` moves every record of that user into it.

## 5. Non-functional requirements

### `WSPACE-NFR-001`

A member that another member removes must reach no record of the workspace in the
first request after Maroid answers the removal. Measure at the response to that request.

**Why:** A removal that waits for a session to end keeps a person inside.

## 6. Invariants

### `WSPACE-INV-002`

A user holds at most one membership in one workspace.

## 7. Constraints from the guidelines

| Rule      | Guideline        | Effect on this feature                                                      |
| --------- | ---------------- | ----------------------------------------------------------------------------- |
| `OWN-003` | Record ownership | `WSPACE-FR-013` and `WSPACE-FR-015` give the acting workspace of each entry point. |
| `OWN-010` | Record ownership | Any active user creates a workspace. No workspace is deleted.               |
| `OWN-011` | Record ownership | A membership binds one user to one workspace. `PERMS` adds the role, and the two release together. |
| `SEC-004` | Security         | A blocked user record reaches no workspace.                                 |
| `SEC-011` | Security         | An administrator changes the members of every workspace. `PLUGACC` gives that access. |
| `SEC-013` | Security         | `PERMS` declares the permission of each route. The two release together.   |
| `ERR-003` | Errors           | `WSPACE-FR-014` answers `not-found`.                                         |
| `API-003` | HTTP API         | The routes of this feature sit under `/workspaces`.                         |

## 8. Open questions

| #   | Question                                                                                        | Owner  | Answer |
| --- | ----------------------------------------------------------------------------------------------- | ------ | ------ |
| 1   | Does `WSPACE-FR-023` remember the last workspace for the person on every device, or for each browser alone? | Temuri | For each browser alone. Maroid stores nothing for it. |
| 2   | Who renames a workspace and changes its members before `PERMS` adds the roles?    | Temuri | Any member. Every member is equal until `PERMS`.      |
| 3   | May the last member leave?                                                        | Temuri | Yes. The records stay, and nobody reaches them.       |
| 4   | How does this feature meet `SEC-013` and `OWN-011`, which name roles and permissions? | Temuri | It releases together with `PERMS`. No guideline changes. |

Answer every question before the approval. An open question blocks stage 2.

## Retired identifiers

| ID               | Retired    | Reason                                                     |
| ---------------- | ---------- | ---------------------------------------------------------- |
| `WSPACE-FR-007`  | 2026-10-01 | A change of a role moved to `PERMS`, which owns the roles. |
| `WSPACE-FR-010`  | 2026-10-01 | The last manager rule moved to `PERMS`.                    |
| `WSPACE-INV-001` | 2026-10-01 | The last manager rule moved to `PERMS`.                    |
