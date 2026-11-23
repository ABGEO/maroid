---
id: ADR-0026
title: A workspace ends with every record it holds
type: adr
status: accepted
created: 2026-11-23
updated: 2026-11-23
decided: 2026-11-23
changes: [OWN-005, OWN-010, SEC-011, ERR-003]
supersedes:
superseded_by:
---

# A workspace ends with every record it holds

## Context

`WSREMOVE-FR-001` and `WSREMOVE-FR-004` let a manager and an administrator remove a
workspace. `WSREMOVE-FR-005` deletes every record that the workspace scopes, in the hub
and in every plugin, also in a plugin that the instance no longer loads.

Four rules stand against it:

| Rule      | What it says                                                                     | Why the removal does not fit                                                                   |
| --------- | -------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------- |
| `OWN-010` | The hub deletes no workspace, because a delete orphans every row that names it.  | The removal deletes the workspace.                                                             |
| `OWN-005` | The scope column references `public.workspaces (id)` with no action on a delete. | A delete of the workspace fails while any row names it, and the hub cannot reach every schema. |
| `SEC-011` | An administrator manages the members and the enablements of every workspace.     | The administrator also removes a workspace with members in it.                                 |
| `ERR-003` | The registry of the problem types of the hub.                                    | `WSREMOVE-FR-002` refuses a removal while other members remain, and no type names the refusal. |

## Decision

A scope column of a workspace references `public.workspaces (id)` with
`ON DELETE CASCADE`, so one delete of the row of the workspace deletes every record of
it in one transaction. The hub deletes a workspace only through the removal of
`WSREMOVE`.

## Rationale

The cascade reaches every schema with no list of the plugins, so a plugin that the
instance no longer loads loses its rows too. A check on PostgreSQL 18 confirmed three
facts. The cascade runs as the owner of each table, so `FORCE ROW LEVEL SECURITY` hides
no row from it, and the delete needs no acting workspace. A row of a plugin that names a
file with `ON DELETE RESTRICT`, as `DAT-003` gives, blocks no cascade, because both rows
go in the same statement. A delete of that file alone is still refused.

The delete of the workspace row takes `FOR UPDATE` on it. An insert that names the
workspace takes `FOR KEY SHARE` for its foreign key, so a write of a plugin and a
removal never interleave: the write lands first and the cascade deletes it, or the
write fails on the foreign key.

## Alternatives

| Alternative                                                                            | Why we did not select it                                                                                                                  |
| -------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------- |
| The hub reads the catalog for every column that references a workspace and deletes it. | The hub then orders the deletes across the foreign keys inside each plugin. A plugin that adds a key between two tables breaks the order. |
| A capability of each plugin that deletes its own rows.                                 | A plugin that the instance no longer loads keeps its rows, which `WSREMOVE-FR-005` forbids.                                               |
| A flag that hides a removed workspace.                                                 | `WSREMOVE-FR-005` demands a delete. A flag keeps every record and every file.                                                             |

## Consequences

### Rules that change

`OWN-005` changes. The column:

> ```sql
> workspace_id UUID NOT NULL
>     DEFAULT NULLIF(current_setting('app.workspace_id', true), '')::uuid
>     REFERENCES public.workspaces (id) ON DELETE CASCADE
> ```

and one sentence after the column:

> A column of a workspace cascades, so the removal of a workspace deletes its rows. A
> column of a user cascades on nothing, because `OWN-002` deletes no user record.

`OWN-010` changes its last sentence to:

> The hub deletes a workspace only when a manager or an administrator removes it, and
> the delete takes every row that names the workspace with it. See `OWN-005`.

`SEC-011` changes its third sentence to:

> An administrator manages every workspace of the instance through the management API:
> its members, its enablements, and its removal.

`ERR-003` gains one row in the table of the hub:

> | `members-remain` | 409 | The workspace holds other members. |

### Requirements that change

| Document                          | Change                                                                                          |
| --------------------------------- | ----------------------------------------------------------------------------------------------- |
| `features/wspace/requirements.md` | Section 3 drops "Deleting or archiving a workspace. `OWN-010` keeps every workspace."           |
| `features/perms/requirements.md`  | The table of `PERMS-FR-005` gains the row "Remove the workspace", with the lowest role manager. |

### Specifications to examine

| Document                  | Why                                                                                              |
| ------------------------- | ------------------------------------------------------------------------------------------------ |
| `features/wspace/spec.md` | Section 4.2 says that no foreign key cascades. The removal adds the cascade.                     |
| `features/pset/spec.md`   | `PSET-DD-004` says that the hub creates no key. The hub still creates none, and now deletes one. |
| `features/files/spec.md`  | A purge of `FILES` now also starts from a removal.                                               |

### Result

- Positive: a plugin that the instance no longer loads loses the rows of a removed
  workspace, with no code of the plugin.
- Positive: the delete is one statement in one transaction. A failure deletes nothing.
- Negative: every plugin that holds a table of a workspace writes one migration that
  replaces the foreign key of the scope column.
- Negative: a defect that deletes a row of `public.workspaces` deletes every record of
  it. Only the removal of `WSREMOVE` writes that statement.
- Work that follows: the migrations of the hub and of each plugin, in the build plan of
  `WSREMOVE`.
