---
id: ADR-0022
title: A table of a plugin references a file of the hub
type: adr
status: accepted
created: 2026-11-18
updated: 2026-11-18
decided: 2026-11-18
changes: [DAT-003]
supersedes:
superseded_by:
---

# A table of a plugin references a file of the hub

## Context

`FILES-FR-022` makes Maroid refuse to delete a file while a row of the consumer
references it. The consumer is a plugin, and its rows live in the schema of the plugin.
The file lives in `public.workspace_files` or `public.user_files`, which the hub owns.

`DAT-003` reads:

> A plugin does not read a table and does not write a table in a different schema.
> A plugin gets data from another component through `pluginapi.Host`.

`OWN-005` already makes one exception: the scope columns `workspace_id` and `user_id`
of a table of a plugin reference `public.workspaces` and `public.users`. No rule permits
a reference to any other table of `public`.

Without a reference, the hub cannot see which rows of which schema name a file. A plugin
then deletes its row and the file in two transactions, and a failure between them
leaves either a row that names a deleted file, or a file that no row names.

## Decision

A table of a plugin references a file of the hub with a foreign key on
`public.workspace_files (id)` or `public.user_files (id)`, with `ON DELETE RESTRICT`.
The reference is the only contact of the plugin with those tables. The plugin reads,
claims, and deletes a file through `Host.Files`.

## Rationale

The database refuses the delete of a referenced file in every schema, in the same
statement, with no code in the hub that knows the plugin. `ARC-008` holds.

A foreign key reads no row and writes no row on behalf of the plugin, so the aim of
`DAT-003` holds: a plugin corrupts no data of the hub. `RESTRICT` keeps the delete of a
file from reaching a row of a plugin, so no action of the hub removes data of a plugin.

PostgreSQL checks a foreign key past row level security. The reference therefore needs
no scope, and the plugin cannot learn of a file of another workspace through it. A
plugin that stores an identifier of a file of another scope finds out at `Claim`, which
answers `ErrFileAbsent` for any file outside the plugin and the acting scope. The expiry
of an unclaimed file meets the foreign key, and `FILES-DD-009` turns that refusal into a
claim, so a lost claim heals.

## Alternatives

| Alternative                                                           | Why we did not select it                                                                                                                         |
| --------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------ |
| A list of references that the hub keeps, written through `Host.Files` | The row of the plugin and the reference commit in two transactions. A failure between them leaves a reference that names no row, or the reverse. |
| A copy of each file table in the schema of each plugin                | The hub then needs a rule for each schema to find a file. `ARC-008` fails.                                                                       |
| `ON DELETE CASCADE` from the file to the row of the plugin            | A delete in the hub removes data of a plugin with no call of that plugin.                                                                        |
| No reference, and a plugin that deletes its file before its row       | A failure after the file delete leaves a row that names nothing, and the gallery shows a broken image.                                           |

## Consequences

### Rules that change

`DAT-003` changes. The new text:

> A plugin does not read a table and does not write a table in a different schema.
> A plugin gets data from another component through `pluginapi.Host`.
>
> A table of a plugin references a table of `public` with a foreign key in two cases
> only: a scope column, which `OWN-005` gives, and a column that names a file, on
> `public.workspace_files (id)` or `public.user_files (id)` with `ON DELETE RESTRICT`.
> The plugin claims the file through `Host.Files` before it trusts the identifier.
>
> **Why:** A foreign key reads and writes nothing for the plugin. It lets the database
> refuse the delete of a file that a row of a plugin still names.

### Specifications to examine

| Document                          | Why                                                                              |
| --------------------------------- | -------------------------------------------------------------------------------- |
| `features/files/spec.md`          | `FILES-DD-009` waits for this record. Section 3 marks `DAT-003` as open.         |
| `features/garden/requirements.md` | The gallery of the garden plugin is the first table that takes such a reference. |

### Result

- Positive: no row of a plugin names a deleted file, and the hub holds no rule for one plugin.
- Positive: a lost claim heals at the expiry, because the reference proves the claim.
- Negative: a plugin whose rows stay after the hub unloads it keeps its files for ever,
  because nothing deletes its rows.
- Negative: a migration of a plugin depends on a table of `public`. `DAT-011` already
  runs the core migrations first.
- Work that follows: the migrations of `FILES`, and the first reference in the gallery of
  the garden plugin.
