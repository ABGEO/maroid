---
id: OWN
title: Record ownership
type: guideline
status: active
created: 2026-09-11
updated: 2026-10-09
scope: [apps/hub/, libs/pluginapi/, plugins/*/db/, plugins/*/repository/]
related: [DAT, SEC, REP, TG, JOB, PLG]
---

# Record ownership

Maroid serves more than one person. This guideline gives the user record, the
workspace, the acting user, the acting workspace, and the isolation between them.
`GLO-user` defines a user, and `GLO-workspace` defines a workspace.

## OWN-001

The hub holds one record for each person, in the table `public.users`. The record
is permanent, and it carries the name that an administrator gave it.

One row in `public.identities` binds one external account to one user record. That
row is the natural key, and `SEC-003` reads it. One user record holds several of
them. One external account belongs to at most one user record.

## OWN-002

The hub creates no user record from an interaction. An administrator creates it
first. The record is active or blocked, and no third state exists. The hub deletes
no record, because a delete orphans every row that names it. `SEC-004` is the gate.

## OWN-003

Every unit of work carries one acting user, one acting workspace, or both. Work
without an acting workspace reaches no table that a workspace scopes. Work without
an acting user reaches no table that a user scopes.

| Entry point       | Acting user                                                | Acting workspace                         |
| ----------------- | ---------------------------------------------------------- | ---------------------------------------- |
| An HTTP request   | The identity that `federated_claims` names. See `SEC-003`. | The `{workspaceId}` segment of the path. |
| A Telegram update | The identity of the Telegram connector for the sender.     | The workspace that the chat selected.    |
| An MCP tool call  | The identity that `federated_claims` names. See `SEC-002`. | The `workspace` argument of the call.    |
| A cron job        | The user of the run, for a job for each user. See `OWN-009`. | The workspace of the run, for a job for each workspace. See `OWN-009`. |

When a unit of work carries both, the hub checks that the acting user is a member
of the acting workspace before it sets the workspace. A user who is not a member
gets `not-found`.

## OWN-004

A table is scoped to a workspace, scoped to a user, or shared, and the migration
states which. A table that a user scopes holds a value that belongs to one person
in every workspace: an idempotency key, a setting that a plugin declares personal,
the records of a personal account that a plugin collects. A shared table holds a
row that belongs to nobody: `public.users`, a reference table, `schema_migrations`.
A new table of a plugin is scoped to a workspace, unless each row belongs to one
person. A table that a user scopes, and a shared table, give the reason in a comment.

## OWN-005

A table that a workspace scopes carries this column:

```sql
workspace_id UUID NOT NULL
    DEFAULT NULLIF(current_setting('app.workspace_id', true), '')::uuid
    REFERENCES public.workspaces (id)
```

A table that a user scopes carries the same column for `user_id`, with the setting
`app.user_id` and the reference `public.users (id)`.

An insert names neither column. The session gives the value, and each reference is
an exception to `DAT-003`. A unique constraint starts with the scope column.

## OWN-006

A scoped table enables row level security and forces it:

```sql
ALTER TABLE plants ENABLE ROW LEVEL SECURITY;
ALTER TABLE plants FORCE ROW LEVEL SECURITY;

CREATE POLICY plants_workspace_isolation ON plants
    USING (workspace_id = NULLIF(current_setting('app.workspace_id', true), '')::uuid)
    WITH CHECK (workspace_id = NULLIF(current_setting('app.workspace_id', true), '')::uuid);
```

A table that a user scopes uses the same policy on `user_id` and `app.user_id`.
`FORCE` applies the policy to the role that owns the table. A session that sets
no value for the scope sees no row.

**Why:** The policy fails closed. A missing setting hides the data.

## OWN-007

A transaction sets the acting user and the acting workspace before its first
statement, each as a bind parameter, and each one only when the unit of work
carries it:

```go
_, err = tx.ExecContext(ctx, "SELECT set_config('app.workspace_id', $1, true)", workspaceID)
```

The third argument keeps the setting inside the transaction, because the pool
reuses the connection.

Two functions set them, and no other place does:

| Function                | Sets them for                                    |
| ----------------------- | ------------------------------------------------ |
| `database.WithScopeTx`  | The hub, on a table that `public` holds.         |
| `PluginDB.WithTx`       | A plugin, with the search path. See `DAT-004`.   |

## OWN-008

A repository writes no filter on `workspace_id` or `user_id`, and sets no value
for either. `OWN-006` does both.

## OWN-009

A cron job declares its scope. A shared job runs one time, with no acting
workspace and no acting user, and reaches only a shared table. A job for each
workspace runs one time for each workspace that enables the plugin, and the
scheduler sets the acting workspace of each run. A job for each user runs one
time for each active user, and the scheduler sets the acting user of each run.
A job of a plugin runs for each user only when it writes a table that a user scopes.

## OWN-010

A workspace is one row in `public.workspaces`. It holds the records of a plugin
that its members share. Every workspace is of one kind. The hub creates the
first workspace of a user in the same transaction as the user record, with that
user as its manager. Any active user creates a further workspace and becomes its
first manager. A user with no workspace reaches no plugin. The hub deletes no
workspace, because a delete orphans every row that names it.

## OWN-011

One row in `public.workspace_members` binds one user to one workspace with one
role: `manager`, `editor`, or `viewer`. The roles form that order, from the most
access to the least. A manager adds an existing user record as a member, removes
a member, and changes a role. A workspace keeps at least one manager, so the
last manager neither leaves nor loses the role.

## Retired identifiers

This file has no retired identifier.
