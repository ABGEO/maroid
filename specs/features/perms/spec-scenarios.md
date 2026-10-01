---
id: PERMS
title: The scenarios of the workspace roles and the permissions
type: spec
status: approved
created: 2026-10-02
updated: 2026-10-02
approved_by: Temuri
approved_on: 2026-10-02
constrained_by: [TST, OWN, TRC]
requirements: features/perms/requirements.md
---

# Specification: The scenarios of the workspace roles and the permissions

`spec.md` holds the design. This file holds the scenarios. `SPC-001` divides them.

Every integration scenario runs against the container that `testdb.Start` gives, with
the migrations of the hub applied. Unless a scenario says otherwise, workspace H has
Ana as `manager`, Beka as `editor`, and Gio as `viewer`, and Nino is a member of
nothing. A probe plugin declares `notes.read` at viewer and `notes.write` at editor,
a route for each, an MCP tool for each, and a command of the bot for each.

## `PERMS-SC-001`

**Verifies:** `PERMS-FR-001`
**Layer:** integration

**Given** workspace H.
**When** a statement inserts a membership with the role `owner`, then one with no role.
**Then** both inserts fail, and H keeps three memberships.

## `PERMS-SC-002`

**Verifies:** `PERMS-FR-002`
**Layer:** unit

**Given** the three roles and the three lowest roles.
**When** `Role.Holds` runs for each of the nine pairs.
**Then** it answers true for manager against all three, for editor against editor and
viewer, and for viewer against viewer alone.

## `PERMS-SC-003`

**Verifies:** `PERMS-FR-003`
**Layer:** integration

**Given** Nino.
**When** Nino creates "Home", and an administrator invites a new person.
**Then** Nino is the `manager` of "Home", and the invited record is the `manager` of
its first workspace.

## `PERMS-SC-004`

**Verifies:** `PERMS-FR-004`
**Layer:** integration

**Given** a database after the migrations of `WSPACE`, with two members in one
workspace.
**When** the migration of this feature runs, and a statement then inserts a membership
that names no role.
**Then** both members are `manager`, and the insert fails.

## `PERMS-SC-005`

**Verifies:** `PERMS-FR-005`
**Layer:** integration

**Given** workspace H.
**When** Gio reads H and its members, saves the settings of the probe plugin, and
renames H. Beka saves the settings, renames H, and reads the candidates. Ana renames H.
**Then** Gio reads both and gets 403 for the save and the rename. Beka saves, and gets
403 for the rename and the candidates. Ana renames.

## `PERMS-SC-006`

**Verifies:** `PERMS-FR-006`
**Layer:** integration

**Given** workspace H.
**When** Ana adds Nino as `editor`, then adds a fifth person with no role.
**Then** Nino is an editor. The second answers 422 with the pointer `/role`, and adds
nobody.

## `PERMS-SC-007`

**Verifies:** `PERMS-FR-007`
**Layer:** integration

**Given** workspace H.
**When** Ana changes the role of Gio to `editor`, then Beka changes the role of Gio to
`manager`.
**Then** the first answers 200, and Gio is an editor. The second answers 403, and Gio
stays an editor.

## `PERMS-SC-008`

**Verifies:** `PERMS-FR-008`, `PERMS-INV-001`
**Layer:** integration

**Given** workspace H, workspace S with Nino as its only member and manager, and
workspace K with two managers, Ana and Beka.
**When** Ana leaves H, Ana changes her own role in H to `editor`, Nino leaves S, then
Ana and Beka each change the role of the other in K to `viewer` at the same moment.
**Then** the first three answer 409 `manager-last` and change nothing. Of the two in
K, one answers 200 and the other 409, and K keeps one manager.

## `PERMS-SC-009`

**Verifies:** `PERMS-FR-009`, `PERMS-FR-010`
**Layer:** integration

**Given** Gio is also the manager of workspace G.
**When** Gio reads `GET /workspaces`, then the members of H.
**Then** the first answer names G with `manager` and H with `viewer`. The second names
Ana as `manager`, Beka as `editor`, and Gio as `viewer`.

## `PERMS-SC-010`

**Verifies:** `PERMS-FR-011`
**Layer:** integration

**Given** the probe plugin.
**When** the hub loads it.
**Then** the registry holds `dev.maroid.probe:notes.read` at viewer and
`dev.maroid.probe:notes.write` at editor.

## `PERMS-SC-011`

**Verifies:** `PERMS-FR-012`, `PERMS-FR-013`
**Layer:** integration

**Given** three copies of the probe plugin: one whose route names no permission, one
whose command of the bot names `notes.delete`, which it does not declare, and one
whose MCP tool names `notes.read` and declares two permissions under one name.
**When** the hub loads each one.
**Then** each load fails, the log names the plugin and the entry, and no route, tool,
or command of the three remains registered.

## `PERMS-SC-012`

**Verifies:** `PERMS-FR-014`
**Layer:** integration

**Given** two probe plugins, P and Q. P declares `notes.write` at editor, and Q
declares `notes.write` at manager.
**When** Beka calls the write route of P, then the write route of Q.
**Then** the first runs, and the second answers 403 with `dev.maroid.q:notes.write`.

## `PERMS-SC-013`

**Verifies:** `PERMS-FR-015`, `PERMS-FR-016`
**Layer:** integration

**Given** workspace H, and the chat of Gio selected H.
**When** Gio calls the write route, the write tool, and the write command of the probe
plugin, and Beka calls the write route.
**Then** the route answers 403 `permission-denied` with `permission` set to
`dev.maroid.probe:notes.write` and `required_role` set to `editor`. The tool answers a
result with `isError` that names both. The bot answers a text that names both. None of
the three runs. The call of Beka runs.

## `PERMS-SC-014`

**Verifies:** `PERMS-FR-017`
**Layer:** integration

**Given** workspace H.
**When** Gio reads `GET /workspaces/{H}`, then Beka reads it.
**Then** the list of Gio holds `notes.read` of the probe and `workspace.read`, and no
write. The list of Beka adds `notes.write` and `settings.write`, and holds no
`members.write`.

## `PERMS-SC-015`

**Verifies:** `PERMS-FR-018`
**Layer:** manual

**Given** Gio, a viewer in H.
**When** Gio opens the members page, the settings of the probe plugin, and its page.
**Then** Gio sees no control to add, remove, or change a member, no save of the
settings, and no write control on the page of the probe plugin.

## `PERMS-SC-016`

**Verifies:** `PERMS-FR-019`, `PERMS-FR-020`
**Layer:** manual

**Given** Ana, the manager of H.
**When** Ana opens the form that adds a member, then changes the role of Gio from the
members page.
**Then** the form starts on viewer, and the role of Gio changes.

## `PERMS-SC-017`

**Verifies:** `PERMS-NFR-001`
**Layer:** integration

**Given** Beka holds a session.
**When** Ana changes the role of Beka to `viewer`, and Beka calls the write route right
after the change answers.
**Then** the call answers 403. Repeat 100 times, and every one answers 403.

## `PERMS-SC-018`

**Verifies:** `PERMS-NFR-002`
**Layer:** integration

**Given** a hub with the probe plugin and a warm connection pool.
**When** 1000 calls of the read route arrive.
**Then** the time from the read of the membership to the start of the handler is 2
milliseconds or less at the 95th percentile.

## Retired identifiers

This file has no retired identifier.
