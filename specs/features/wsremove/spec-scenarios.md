---
id: WSREMOVE
title: The scenarios of the removal of a workspace
type: spec
status: approved
created: 2026-11-23
updated: 2026-11-23
approved_by: Temuri
approved_on: 2026-11-23
constrained_by: [TST, OWN, TRC]
requirements: features/wsremove/requirements.md
---

# Specification: The scenarios of the removal of a workspace

`spec.md` holds the design. This file holds the scenarios. `SPC-001` divides them.

Every integration scenario runs against the container that `testdb.Start` gives, with
the migrations of the hub applied. Unless a scenario says otherwise, five active user
records exist: Ana, Beka, Gio, Nino, and Admin, an administrator. Workspace H has Ana as
`manager`, Beka as `editor`, and Gio as `viewer`. Workspace S has Nino as its only
member, a `manager`. Admin is a member of nothing.

## `WSREMOVE-SC-001`

**Verifies:** `WSREMOVE-FR-001`
**Layer:** integration

**Given** workspace S, and a second workspace of Nino.
**When** Nino sends `DELETE /workspaces/{S}`.
**Then** the answer is 204. `GET /workspaces/{S}` answers 404, and `GET /workspaces`
of Nino names the second workspace alone.

## `WSREMOVE-SC-002`

**Verifies:** `WSREMOVE-FR-001`
**Layer:** integration

**Given** workspace S is the only workspace of Nino.
**When** Nino removes S.
**Then** the answer is 204, and `GET /workspaces` of Nino answers an empty page.

## `WSREMOVE-SC-003`

**Verifies:** `WSREMOVE-FR-001`
**Layer:** integration

**Given** workspace H.
**When** Gio, then Beka, sends `DELETE /workspaces/{H}`.
**Then** each answer is 403 `permission-denied`, with the permission `workspace.remove`
and the lowest role `manager`. H holds three members.

## `WSREMOVE-SC-004`

**Verifies:** `WSREMOVE-FR-002`, `WSREMOVE-FR-003`
**Layer:** integration

**Given** workspace H, with a notification provider and a setting of a plugin.
**When** Ana removes H.
**Then** the answer is 409 `members-remain` with `other_members` 2. H, its members, its
provider, and its setting stay. `public.outbox_tasks` holds no new row.

## `WSREMOVE-SC-005`

**Verifies:** `WSREMOVE-FR-002`
**Layer:** integration

**Given** workspace S, and Ana added to S as a second `manager`.
**When** Nino removes S.
**Then** the answer is 409 `members-remain` with `other_members` 1.

## `WSREMOVE-SC-006`

**Verifies:** `WSREMOVE-FR-002`
**Layer:** integration

**Given** workspace S.
**When** Admin adds Beka to S while Nino removes S, in two orders: the addition holds
its transaction open while the removal starts, then the removal holds its lock while the
addition starts.
**Then** in the first order, the removal answers `members-remain` and S holds Beka. In
the second order, the removal answers 204, and the addition answers 404 `not-found`.

## `WSREMOVE-SC-007`

**Verifies:** `WSREMOVE-FR-004`
**Layer:** integration

**Given** workspace H.
**When** Admin sends `DELETE /workspaces/{H}`.
**Then** the answer is 204. `GET /workspaces/{H}` answers 404 to Ana, Beka, and Gio.

## `WSREMOVE-SC-008`

**Verifies:** `WSREMOVE-FR-004`
**Layer:** integration

**Given** a workspace whose only member is Admin, as `manager`.
**When** Admin removes it.
**Then** the answer is 204.

## `WSREMOVE-SC-009`

**Verifies:** `WSREMOVE-FR-005`
**Layer:** integration

**Given** workspace S holds one row in each table of `public` that section 4.2 of
`spec.md` lists, and a second workspace holds one row in each too.
**When** Nino removes S.
**Then** each table holds no row of S, and the row of the second workspace in each
table stays.

## `WSREMOVE-SC-010`

**Verifies:** `WSREMOVE-FR-005`, `OWN-006`, `DAT-003`
**Layer:** integration

**Given** the schema `probe`, migrated from `testdata` and loaded by no plugin, with
two tables that a workspace scopes under the cascade of `OWN-005`: `probe.places`, and
`probe.photos`, which names `probe.places` with no action and a row of
`public.workspace_files` with `ON DELETE RESTRICT`. Workspace S disables every plugin.
S holds a place, a file, and a photo of the place that names the file.
**When** Nino removes S, from a transaction that sets no `app.workspace_id`.
**Then** the answer is 204. `probe.places`, `probe.photos`, and
`public.workspace_files` hold no row of S.

## `WSREMOVE-SC-011`

**Verifies:** `WSREMOVE-FR-005`
**Layer:** integration

**Given** the schema `stale`, migrated from `testdata`, with a table that names
`public.workspaces` with no action, and one row of S in it. S holds a setting and a
file.
**When** Nino removes S.
**Then** the answer is 500 `internal`. S, its setting, its file, and the row of `stale`
stay. `public.outbox_tasks` holds no new row.

## `WSREMOVE-SC-012`

**Verifies:** `WSREMOVE-FR-006`
**Layer:** integration

**Given** Nino holds a setting of a plugin of the user, a plugin secret of the user, a
file of the user, and a Telegram chat.
**When** Nino removes S.
**Then** each of the four rows stays.

## `WSREMOVE-SC-013`

**Verifies:** `WSREMOVE-FR-007`, `WSREMOVE-NFR-002`
**Layer:** integration

**Given** `upload_ttl` of 15 minutes. S holds a ready file F1, and a pending file F2
with its tasks `check` and `expire`.
**When** Nino removes S at the moment T.
**Then** `public.outbox_tasks` holds one `file.purge` for F1 and one for F2, each owned
by S, with the key of `keyOf` and a `run_after` of T plus 16 minutes. It holds no
`file.check` and no `file.expire` of S.

## `WSREMOVE-SC-014`

**Verifies:** `WSREMOVE-FR-010`, `WSREMOVE-NFR-002`
**Layer:** unit

**Given** a due `file.purge` of a file of a removed workspace, and a stub file store
that fails the delete three times, then succeeds.
**When** the `outbox` worker runs its cycles.
**Then** the task stays after each failure, with a `run_after` that grows and never
passes 5 minutes after the failure. After the fourth attempt, the stub holds no object
under the key, and the table holds no task.

## `WSREMOVE-SC-015`

**Verifies:** `WSREMOVE-FR-008`, `WSREMOVE-NFR-003`
**Layer:** integration

**Given** OpenBao from `testbao`, with the key `maroid-workspace-<S>` and
`deletion_allowed=true`, and a setting of S that the key protects.
**When** Nino removes S at the moment T, and the `outbox` worker runs.
**Then** within 30 seconds of T, a read of `transit/keys/maroid-workspace-<S>`
answers no data, and `public.outbox_tasks` holds no task of the type
`secret_key.destroy`.

## `WSREMOVE-SC-016`

**Verifies:** `WSREMOVE-FR-008`
**Layer:** integration

**Given** OpenBao from `testbao`, and no key `maroid-workspace-<S>`.
**When** the `outbox` worker runs the `secret_key.destroy` of S.
**Then** the task is done, and the worker logs no error.

## `WSREMOVE-SC-017`

**Verifies:** `WSREMOVE-FR-009`
**Layer:** integration

**Given** a stub file store and a stub cipher, each of which fails every call. S holds a
file and a secret.
**When** Nino removes S.
**Then** the answer is 204. Neither stub records a call.

## `WSREMOVE-SC-018`

**Verifies:** `WSREMOVE-FR-010`, `WSREMOVE-NFR-003`
**Layer:** unit

**Given** a due `secret_key.destroy`, and a stub `KeyDestroyer` that fails two times,
then succeeds.
**When** one worker runs a cycle, stops, and a second worker runs the next cycles.
**Then** `attempts` reads 1, then 2, and `run_after` waits 2 seconds, then 4. The second
worker destroys the key on the third attempt, and the table holds no task. No wait passes
5 minutes.

## `WSREMOVE-SC-019`

**Verifies:** `WSREMOVE-FR-011`
**Layer:** manual

**Given** the deck on the settings page of "Old flat", as its only manager.
**When** the person opens the removal and types "Old flats", then "old flat", then
"Old flat".
**Then** "Remove" stays disabled for the first two, and turns enabled for the third.

## `WSREMOVE-SC-020`

**Verifies:** `WSREMOVE-FR-012`, `WSREMOVE-FR-014`
**Layer:** manual

**Given** the deck, a person with "Home" and "Old flat", on the settings page of
"Old flat".
**When** the person removes "Old flat", then creates a third workspace and removes it,
then removes "Home".
**Then** the first removal opens "Home". The last removal shows the page of
`WSPACE-FR-024`. The settings page of a workspace in which the person is an editor shows
no panel.

## `WSREMOVE-SC-021`

**Verifies:** `WSREMOVE-FR-013`, `WSREMOVE-FR-003`
**Layer:** manual

**Given** the deck as Admin, on the list of every workspace, and the deck as Ana on the
settings page of H.
**When** Ana removes H, then Admin removes H.
**Then** Ana reads "Remove the 2 other members first." with a link to the members. The
removal of Admin succeeds, and the row of H leaves the list.

## `WSREMOVE-SC-022`

**Verifies:** `WSREMOVE-FR-015`
**Layer:** integration

**Given** Gio is a member of H and of a second workspace W. The chat of Gio selected H.
**When** Admin removes H, and Gio sends a command of the bot.
**Then** the row of the chat stays with no selected workspace, and the command runs in W
under `WSPACE-FR-017`.

## `WSREMOVE-SC-023`

**Verifies:** `WSREMOVE-NFR-001`
**Layer:** integration

**Given** workspace H, and a read of a plugin route of H by Beka that answers 200.
**When** Admin removes H, and Beka sends the same read right after the answer of 204.
**Then** the read answers 404 `not-found`.

## `WSREMOVE-SC-024`

**Verifies:** `WSREMOVE-INV-001`
**Layer:** integration

**Given** the migrations of one schema, applied with `testdb`.
**When** `RequireWorkspaceCascade` reads `pg_constraint` for each foreign key of the
schema that references `public.workspaces`.
**Then** each key deletes with `CASCADE`, except `public.telegram_chats`, whose key
sets `NULL`. The hub runs it for `public`, and each plugin of section 4.2 of `spec.md`
runs it for its own schema.

## Retired identifiers

This file has no retired identifier.
