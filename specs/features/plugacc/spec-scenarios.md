---
id: PLUGACC
title: The scenarios of the administrator, the allowlist, and the enablement
type: spec
status: approved
created: 2026-10-02
updated: 2026-10-05
approved_by: Temuri
approved_on: 2026-10-05
constrained_by: [TST, OWN, TRC]
requirements: features/plugacc/requirements.md
---

# Specification: The scenarios of the administrator, the allowlist, and the enablement

`spec.md` and `spec-clients.md` hold the design. This file holds the scenarios.
`SPC-001` divides them.

Every integration scenario runs against the container that `testdb.Start` gives, with
the migrations of the hub applied. Unless a scenario says otherwise, Zura is an
administrator. Workspace H has Ana as `manager`, Beka as `editor`, and Gio as `viewer`.
Zura is no member of H. The hub loaded the probe plugin P, with a route, a setting, an
MCP tool, a command of the bot, and a job for each workspace, and the plugin Q. The
allowlist of Ana holds P.

## `PLUGACC-SC-001`

**Verifies:** `PLUGACC-FR-001`
**Layer:** integration

**Given** a database with no administrator, and the record of Ana.
**When** an operator runs `maroid user invite --first-name Nino --admin`, then
`maroid user invite --user <the identifier of Ana> --admin`.
**Then** the record of Nino and the record of Ana are both administrators, and each
command prints the address of an invitation.

## `PLUGACC-SC-002`

**Verifies:** `PLUGACC-FR-002`
**Layer:** integration

**Given** a blocked user record, Levan.
**When** Zura reads `GET /users`, then `GET /users/{Levan}`.
**Then** the page holds every record with its status and its mark, Levan as `blocked`
and Zura as an administrator, and the second answer carries Levan with an `ETag`.

## `PLUGACC-SC-003`

**Verifies:** `PLUGACC-FR-003`, `PLUGACC-FR-004`
**Layer:** integration

**Given** Zura.
**When** Zura creates the user "Nino Beridze", then asks a new invitation for Nino.
**Then** the first answers 201 with the record and the address of an invitation, and
Nino is the manager of a new workspace. The second answers 201 with a second address,
and both invitations stay valid.

## `PLUGACC-SC-004`

**Verifies:** `PLUGACC-FR-005`, `PLUGACC-FR-006`
**Layer:** integration

**Given** Beka holds a session.
**When** Zura sets the status of Beka to `blocked`, Beka sends a request, Zura sets the
status back to `active`, and Beka sends a request.
**Then** the first request of Beka answers 401, and the second one answers 200. Every
record of H stays.

## `PLUGACC-SC-005`

**Verifies:** `PLUGACC-FR-007`, `PLUGACC-FR-029`, `PLUGACC-FR-030`, `PLUGACC-INV-003`
**Layer:** integration

**Given** Zura, the only administrator.
**When** Zura takes her own mark, then blocks her own record, then marks Ana, then Ana
and Zura each take the mark of the other at the same moment.
**Then** the first two answer 409 `administrator-last` and change nothing. Ana becomes
an administrator. Of the last two, one answers 200 and the other 409, and one active
administrator remains.

## `PLUGACC-SC-006`

**Verifies:** `PLUGACC-FR-008`
**Layer:** integration

**Given** the allowlist of Beka is empty.
**When** Zura adds P and Q to it, adds P again, adds `dev.maroid.none`, then removes Q.
**Then** the first two answer 201, the repeat answers 200 with the same row, the third
answers 422 with the pointer `/plugin_id`, and the allowlist of Beka holds P alone.

## `PLUGACC-SC-007`

**Verifies:** `PLUGACC-FR-009`
**Layer:** integration

**Given** H enables P, and workspace G of Nino enables nothing.
**When** Zura reads `GET /workspaces?scope=all`.
**Then** the page holds H with three members and P, and G with one member and no
plugin.

## `PLUGACC-SC-008`

**Verifies:** `PLUGACC-FR-010`, `PLUGACC-INV-002`
**Layer:** integration

**Given** H enables P and holds two records of P.
**When** Zura reads the members of H, changes the role of Gio to `manager`, then calls
the route of P in H and reads the settings of P in H.
**Then** the first two succeed. The last two answer 404, and Zura reads no record and
no setting of H.

## `PLUGACC-SC-009`

**Verifies:** `PLUGACC-FR-011`, `PLUGACC-FR-012`, `PLUGACC-FR-017`
**Layer:** integration

**Given** H, which enables nothing.
**When** Gio reads the plugins of H, Ana enables P, Ana enables Q, and Gio reads again.
**Then** the first read is empty. P answers 201. Q answers 403 with the permission
`plugin-allowlist` and names Q. The second read holds P alone.

## `PLUGACC-SC-010`

**Verifies:** `PLUGACC-FR-013`
**Layer:** integration

**Given** H, and an empty allowlist of Zura.
**When** Zura enables Q in H.
**Then** the answer is 201, and every member of H reaches Q.

## `PLUGACC-SC-011`

**Verifies:** `PLUGACC-FR-014`, `PLUGACC-FR-015`, `PLUGACC-FR-027`
**Layer:** integration

**Given** H enables P and Q, holds two records of P, and a setting of P.
**When** Ana disables P, Zura disables Q, Gio calls the route of P, and Ana enables P
again and reads its records and its setting.
**Then** both disables answer 204, the call of Gio answers 404, and after the enable
Ana reads the two records and the setting as they were.

## `PLUGACC-SC-012`

**Verifies:** `PLUGACC-FR-016`
**Layer:** integration

**Given** Ana enabled P in H.
**When** Zura removes P from the allowlist of Ana, and Gio calls the route of P.
**Then** H still enables P, and the call of Gio runs.

## `PLUGACC-SC-013`

**Verifies:** `PLUGACC-FR-018`, `PLUGACC-INV-001`
**Layer:** integration

**Given** H does not enable P.
**When** Ana calls the route of P, the settings of P, the MCP tool of P with H, and a
route that no plugin declares.
**Then** the two routes answer the same 404 as the route that no plugin declares, and
the tool answers the protocol error of an unknown tool. P runs zero times.

## `PLUGACC-SC-014`

**Verifies:** `PLUGACC-FR-019`
**Layer:** unit

**Given** the chat of Gio selected H, and H does not enable P.
**When** Gio sends the command of P.
**Then** the bot sends nothing, the command runs zero times, and the log holds one line
at the level `info`.

## `PLUGACC-SC-016`

**Verifies:** `PLUGACC-FR-020`
**Layer:** unit

**Given** the job of P declares `CronScopePerWorkspace`, H enables P, and G does not.
**When** the scheduler fires the job one time.
**Then** the job runs one time, with H as the acting workspace.

## `PLUGACC-SC-017`

**Verifies:** `PLUGACC-FR-021`, `PLUGACC-FR-028`
**Layer:** integration

**Given** the hub loaded P and Q, and the allowlist of Ana holds P.
**When** Ana reads `GET /plugins`, then Zura reads it.
**Then** Ana reads P alone, and Zura reads P and Q.

## `PLUGACC-SC-018`

**Verifies:** `PLUGACC-FR-022`
**Layer:** integration

**Given** Ana, who is no administrator.
**When** Ana calls each route of `/users`, `GET /workspaces?scope=all`, and enables Q in
G of Nino.
**Then** the routes of `/users` and `scope=all` answer 403 with the permission
`administration`, and the enable in G answers 404, because Ana is no member of G.

## `PLUGACC-SC-019`

**Verifies:** `PLUGACC-FR-011`, `PLUGACC-FR-026`
**Layer:** integration

**Given** a database after the migrations of `PERMS`, with two workspaces and three
user records.
**When** the migrations of this feature run.
**Then** no workspace enables a plugin, every allowlist is empty, and no record is an
administrator.

## `PLUGACC-SC-020`

**Verifies:** `PLUGACC-NFR-001`
**Layer:** integration

**Given** H enables P, and Gio holds a session.
**When** Ana disables P, and Gio calls the route of P right after the disable answers.
**Then** the call answers 404. Repeat 100 times, and every one answers 404.

## `PLUGACC-SC-021`

**Verifies:** `PLUGACC-FR-025`
**Layer:** manual

**Given** H enables P, and G enables Q. Ana is a member of both.
**When** Ana opens H, then switches to G.
**Then** the sidebar names P in H and Q in G, and never both.

## `PLUGACC-SC-022`

**Verifies:** `PLUGACC-FR-023`, `PLUGACC-FR-024`
**Layer:** manual

**Given** Zura and Ana.
**When** Zura creates a user with P on the allowlist, copies the invitation, blocks and unblocks the user, marks
and unmarks them, edits their allowlist, and changes the members and the plugins of H,
all from the deck. Ana switches P off and on in H from the deck.
**Then** each action succeeds from the deck. Ana sees no page under `/admin`.

## `PLUGACC-SC-023`

**Verifies:** `PLUGACC-FR-032`
**Layer:** integration

**Given** Zura is an administrator, and the hub loaded P and Q.
**When** Zura creates Nina with `allowed_plugins` [P], then creates Levan with
`allowed_plugins` [Q, `dev.maroid.none`].
**Then** the first creation answers 201, and the allowlist of Nina holds P alone. The
second answers 422 with the pointer `/allowed_plugins/1`, and no record of Levan, no
workspace, and no invitation exist.

## `PLUGACC-SC-024`

**Verifies:** `PLUGACC-FR-017`, `PLUGACC-FR-021`
**Layer:** integration

**Given** Zura enables Q in H. Gio is a member of H, and his allowlist is empty.
**When** Gio reads `GET /workspaces/{H}/plugins` and `GET /plugins`.
**Then** the enablements answer Q with `plugin` that carries its version and its
capabilities. `GET /plugins` answers no Q. A route of Q in H answers Gio.

## `PLUGACC-SC-025`

**Verifies:** `PLUGACC-FR-025`, `PLUGACC-FR-033`
**Layer:** manual

**Given** Zura enables Q in H. Gio is a member of H with an empty allowlist. Zura is no
member of H.
**When** Gio opens H in the deck, then Zura opens the plugins of H.
**Then** the sidebar of Gio names Q and opens its pages. The sidebar of Zura names no
page of a plugin, the header and the breadcrumbs name H, and the page says that Zura
manages H as an administrator.

## `PLUGACC-SC-026`

**Verifies:** `PLUGACC-FR-034`
**Layer:** integration

**Given** Zura is an administrator, and Beka holds the first name "Bkea" and the last
name "Kapanadze".
**When** Zura sends `PATCH /users/{Beka}` with `first_name` "Beka", then with
`last_name` "".
**Then** the first answer carries "Beka" and the last name stays. The second answer
carries no last name. The identities and the workspaces of Beka stay. Ana, who is no
administrator, receives permission-denied for the same change.

## `PLUGACC-SC-027`

**Verifies:** `PLUGACC-FR-035`
**Layer:** integration

**Given** Zura and Levan are administrators, and Ana is not.
**When** Zura adds P to the allowlist of Levan, removes P from it, and creates Nina as
an administrator with `allowed_plugins` [P].
**Then** each answers 409 with `/problems/hub/administrator-allowlist`, and no record of
Nina exists. Zura adds P to the allowlist of Ana, and it answers 201.

## `PLUGACC-SC-028`

**Verifies:** `PLUGACC-FR-018`, `PLUGACC-INV-001`
**Layer:** unit

**Given** the acting workspace does not enable P, and P declares settings.
**When** an agent calls `get_plugin_settings` and `save_plugin_settings` for P.
**Then** each answers the failure of a plugin that declares no settings, as for a plugin
that the hub did not load, and the settings service reads and writes nothing.

## `PLUGACC-SC-029`

**Verifies:** `PLUGACC-FR-021`, `PLUGACC-FR-028`
**Layer:** integration

**Given** the hub loaded P and Q, and the allowlist of Ana holds P.
**When** Ana calls `list_plugins` over MCP, then Zura, an administrator, calls it.
**Then** Ana reads P alone, as `GET /plugins` answers her. Zura reads P and Q.

## `PLUGACC-SC-030`

**Verifies:** `PLUGACC-FR-036`
**Layer:** manual

**Given** H enables Q, Q declares a required setting that holds no value in H, and Gio
holds an empty allowlist.
**When** Gio, Ana, and Zura each open the plugins of H in the deck.
**Then** Gio and Ana each read Q in use, marked as missing a value, with the link to its
settings. Ana also reads P to turn on, and Gio reads no list to turn on. Zura reads Q in
use and P to turn on, with no mark and no link to the settings. The sidebar of Gio links no list of plugins outside
H.

## `PLUGACC-SC-031`

**Verifies:** `PLUGACC-FR-037`
**Layer:** manual

**Given** H enables Q.
**When** Zura opens `/admin/plugins`, then Ana opens the same address.
**Then** Zura reads P with no workspace, and Q with H. Ana sees no page under `/admin`.

## Retired identifiers

| ID               | Retired    | Reason |
| ---------------- | ---------- | ------ |
| PLUGACC-SC-015   | 2026-10-04 | The owner postponed the menu of each chat (`PLUGACC-FR-031`, `PLUGACC-NFR-002`). Its later design takes new identifiers. |
