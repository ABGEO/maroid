---
id: WSPACE
title: The scenarios of the workspaces and their members
type: spec
status: approved
created: 2026-10-01
updated: 2026-10-02
approved_by: Temuri
approved_on: 2026-10-02
constrained_by: [TST, OWN, TRC]
requirements: features/wspace/requirements.md
---

# Specification: The scenarios of the workspaces and their members

`spec.md` and `spec-clients.md` hold the design. This file holds the scenarios.
`SPC-001` divides them.

Every integration scenario runs against the container that `testdb.Start` gives, with
the migrations of the hub applied. Unless a scenario says otherwise, four active user
records exist: Ana, Beka, Gio, and Nino. Workspace H has Ana as `manager`, Beka as
`editor`, and Gio as `viewer`, which `PERMS` gives. Nino is a member of nothing.

## `WSPACE-SC-001`

**Verifies:** `WSPACE-FR-001`, `WSPACE-FR-002`
**Layer:** integration

**Given** Nino.
**When** Nino creates "Home", then a workspace with a name of 64 characters, then one
with an empty name, then one with a name of 65 characters.
**Then** the first two answer 201, and Nino is the one member of each. The last two
answer 422 with the pointer `/name`, and no row appears.

## `WSPACE-SC-002`

**Verifies:** `WSPACE-FR-003`
**Layer:** integration

**Given** Gio is also the one member of workspace G.
**When** Gio reads `GET /workspaces`.
**Then** the page holds G and H, and no other workspace.

## `WSPACE-SC-003`

**Verifies:** `WSPACE-FR-004`
**Layer:** integration

**Given** workspace H.
**When** Ana renames it to "Flat", then renames it with a name of 65 characters, then
renames it with an `If-Match` from before the first rename.
**Then** the first answers 200 with "Flat", the second 422 with the pointer `/name`,
and the third 412. The name stays "Flat".

## `WSPACE-SC-004`

**Verifies:** `WSPACE-FR-005`
**Layer:** integration

**Given** a fifth user record, Levan, who is blocked.
**When** Ana reads the candidates of H.
**Then** Ana reads Nino and nobody else.

## `WSPACE-SC-005`

**Verifies:** `WSPACE-FR-006`, `WSPACE-INV-002`
**Layer:** integration

**Given** Levan, who is blocked.
**When** Ana adds Nino, then adds Nino again, then adds Levan.
**Then** the first answers 201, the second 409 `member-exists`, and the third 422 with
the pointer `/user_id`. H holds one row for Nino.

## `WSPACE-SC-006`

**Verifies:** `WSPACE-FR-008`, `WSPACE-FR-009`, `WSPACE-FR-012`
**Layer:** integration

**Given** H holds three rows of a scoped test table, and one of them came from Beka.
**When** Ana removes Beka, then Gio leaves.
**Then** both answer 204, H has Ana alone, and the three rows stay in H.

## `WSPACE-SC-007`

**Verifies:** `WSPACE-FR-011`
**Layer:** integration

**Given** workspace H.
**When** Gio reads the members of H, then reads the membership of Ana.
**Then** the page holds Ana, Beka, and Gio, and the second answer carries Ana with an
`ETag`.

## `WSPACE-SC-008`

**Verifies:** `WSPACE-FR-012`
**Layer:** integration

**Given** workspace G with Ana as `manager` and Gio as `viewer`, and two rows of a
scoped test table, one of them from Gio.
**When** Gio leaves G.
**Then** the answer is 204, Ana is the one member of G, Ana reads both rows, and
`GET /workspaces` of Gio no longer names G.

## `WSPACE-SC-009`

**Verifies:** `WSPACE-FR-013`
**Layer:** integration

**Given** Ana is also the one member of G, and a probe plugin whose route lists the rows
of its scoped table. H holds two rows and G holds one.
**When** Ana calls the route under `/workspaces/{H}`, then under `/workspaces/{G}`.
**Then** the first answer holds the two rows of H, and the second the one row of G.

## `WSPACE-SC-010`

**Verifies:** `WSPACE-FR-014`
**Layer:** integration

**Given** workspace H.
**When** Nino reads `/workspaces/{H}`, then a UUID that no workspace holds, then the
text `abc` in place of a UUID.
**Then** the three answers carry 404 and the type `/problems/http/not-found`, and
their bodies differ in the flow identifier alone.

## `WSPACE-SC-011`

**Verifies:** `WSPACE-FR-026`
**Layer:** integration

**Given** a database before the migration, with Ana, and a user record with no first
name.
**When** the migration runs.
**Then** two workspaces exist: "Ana" with Ana as its member, and "Workspace" with the
second record as its member. The down migration removes both tables.

## `WSPACE-SC-012`

**Verifies:** `WSPACE-NFR-001`
**Layer:** integration

**Given** Beka holds a session.
**When** Ana removes Beka from H, and Beka sends a request under `/workspaces/{H}`
right after the removal answers.
**Then** the request of Beka answers 404. Repeat 100 times, and every one answers 404.

## `WSPACE-SC-013`

**Verifies:** `WSPACE-FR-015`, `WSPACE-FR-016`
**Layer:** integration

**Given** Ana is a member of H and of G, in a chat that selected nothing.
**When** Ana sends `/workspace`, taps G, and then sends a command of a plugin twice.
**Then** the bot answers two buttons, the tap stores G for the chat, and both runs of
the command carry G as the acting workspace.

## `WSPACE-SC-014`

**Verifies:** `WSPACE-FR-017`
**Layer:** unit

**Given** Gio, a member of H alone, in a chat that selected nothing.
**When** Gio sends a command of a plugin.
**Then** the command runs with H as the acting workspace, and the chat stores H.

## `WSPACE-SC-015`

**Verifies:** `WSPACE-FR-018`
**Layer:** unit

**Given** Ana, a member of H and of G, in a chat that selected nothing.
**When** Ana sends a command of a plugin.
**Then** the bot answers "Pick a workspace first" with the buttons of H and G, and the
command runs zero times.

## `WSPACE-SC-016`

**Verifies:** `WSPACE-FR-019`
**Layer:** integration

**Given** Beka selected H in a chat, and is also a member of G and of K.
**When** Ana removes Beka from H, and Beka sends a command of a plugin.
**Then** the chat stores no selection, the bot asks Beka to pick between G and K, and
the command runs zero times.

## `WSPACE-SC-017`

**Verifies:** `WSPACE-FR-020`, `WSPACE-FR-021`
**Layer:** manual

**Given** Ana on a page of a plugin in H, and a member of G.
**When** Ana reads the header, clicks the switcher, and clicks G.
**Then** the header names H before the clicks, the second click opens the overview of
G, and the header names G.

## `WSPACE-SC-018`

**Verifies:** `WSPACE-FR-022`
**Layer:** manual

**Given** Ana and Beka, members of H, and Ana a member of G.
**When** Ana copies the address of a page of H and sends it to Beka. Ana opens G in a
second tab.
**Then** Beka opens the same page of H. Each tab of Ana shows its own workspace after a
reload.

## `WSPACE-SC-019`

**Verifies:** `WSPACE-FR-023`
**Layer:** manual

**Given** Ana left the deck on G in one browser.
**When** Ana signs in again in that browser, then in a second browser for the first
time, then Ana leaves G and signs in again in the first browser.
**Then** the first sign in opens G. The second opens the first workspace of
`WSPACE-SC-002`. The third opens the first workspace and shows nothing of G.

## `WSPACE-SC-020`

**Verifies:** `WSPACE-FR-024`
**Layer:** manual

**Given** Nino, a member of nothing.
**When** Nino signs in.
**Then** the deck shows the form that creates a workspace, the sidebar shows no
plugin, and `/w/{H}` answers the page of a missing workspace.

## `WSPACE-SC-021`

**Verifies:** `WSPACE-FR-025`
**Layer:** manual

**Given** Ana and Gio, members of H.
**When** Ana creates a workspace, renames it, adds Nino, removes Nino, and reads the
members, all from the deck. Gio leaves H from the deck.
**Then** each action succeeds from the deck.

## Retired identifiers

This file has no retired identifier.
