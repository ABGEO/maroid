---
id: ADR-0023
title: A plugin deletes a file in its own transaction
type: adr
status: accepted
created: 2026-11-18
updated: 2026-11-18
decided: 2026-11-18
changes: [DAT-003]
supersedes:
superseded_by:
---

# A plugin deletes a file in its own transaction

## Context

`GARDEN-FR-053` makes Maroid delete the file of each photo that a delete of a photo, of
a moment, or of a plant removes. `ADR-0022` gives the row of the photo a foreign key on
`public.workspace_files (id)` with `ON DELETE RESTRICT`.

`PluginFiles.Delete` runs in a transaction of the hub. While the transaction of the
plugin holds the row, the foreign key refuses the delete of the file. So the plugin
calls `Delete` after its commit, and a failure between the commit and the call leaves a
claimed file that no row names. `FILES-FR-019` deletes an unclaimed file only, so
nothing ever deletes it.

The plugin and the hub share one database. `DAT-004` puts `public` on the search path of
every transaction of a plugin.

## Decision

`PluginFiles.Delete` becomes `Delete(ctx context.Context, tx *sqlx.Tx, ids ...string) error`.
Through the transaction of the plugin, it deletes each file row of the plugin in the
acting scope and writes a `purge` task for it. The plugin deletes its own rows first.

## Rationale

The file rows, the `purge` tasks, and the rows of the plugin commit together or roll
back together. The foreign key checks at the end of each statement, and the transaction
already removed the row of the plugin, so the delete of the file passes. A plugin that
calls `Delete` while a row still names the file meets SQLSTATE `23503`, which answers
`ErrFileReferenced` at once, and the plugin rolls back.

After the commit, `Get` no longer answers the file, and the worker of `ADR-0021` removes
the bytes with the `purge` task that exists today. The change adds no task action and no
migration.

The hub runs the SQL, not the plugin. The plugin names its files through `Host.Files`,
as `DAT-003` demands. The policy of the acting scope and the condition on the plugin
limit the delete to a file of the plugin in that scope.

A plugin that holds no row of the file, such as a form that drops an upload, calls
`Delete` inside its own `PluginDB.WithTx`. So one method serves every consumer.

## Alternatives

| Alternative                                                                       | Why we did not select it                                                                                  |
| --------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------- |
| `Delete` in a transaction of the hub, after the commit of the plugin              | A failure between the commit and the call leaves a claimed file that nothing deletes.                     |
| A new `Release` that writes a `release` task, and the worker deletes the file row | It needs a new task action and a migration, and `Get` answers the file until the worker runs.             |
| `Release` next to the old `Delete`                                                | The old `Delete` refuses every referenced file and loses a file after a failure. Two methods invite that. |
| A trigger function of the hub that the migration of a plugin attaches             | The delete then hides in a migration, and the handler shows no trace of it.                               |
| An outbox table and a cron job in each plugin                                     | Each plugin with files repeats the same table and job.                                                    |

## Consequences

### Rules that change

`DAT-003` changes. The last sentence of the second paragraph becomes:

> The plugin claims the file through `Host.Files` before it trusts the identifier, and
> deletes the file through `Host.Files` in the transaction that removes the reference.

### Specifications to examine

| Document                                 | Why                                                                                                                          |
| ---------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------- |
| `features/files/spec.md`                 | `PluginFiles.Delete` in section 4.1 and the row of `Delete` in section 4.5.                                                  |
| `features/files/spec-decisions.md`       | `FILES-DD-009`: `Delete` runs in the transaction of the consumer.                                                            |
| `features/files/spec-scenarios.md`       | A commit deletes the file and purges it. A rollback keeps both. A row that still names the file answers `ErrFileReferenced`. |
| `features/garden/requirements-photos.md` | `GARDEN-FR-053` is the first consumer.                                                                                       |

`FILES-FR-021` and `FILES-FR-022` keep their text.

### Result

- Positive: no deleted row leaves a file in the file store, after a failure too.
- Positive: no new task action, no migration, and one `Delete` for every consumer.
- Negative: `PluginFiles` takes a `*sqlx.Tx`, so a call of the host joins the
  transaction of the plugin. `PluginDB` already exposes `*sqlx.Tx`.
- Negative: a change of `libs/pluginapi` rebuilds every plugin (`BLD-004`).
- Work that follows: the change of `FILES` above, then the gallery of the garden plugin.
