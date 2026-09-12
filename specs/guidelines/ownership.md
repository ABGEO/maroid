---
id: OWN
title: Record ownership
type: guideline
status: active
created: 2026-09-11
updated: 2026-09-11
scope: [apps/hub/, libs/pluginapi/, plugins/*/db/, plugins/*/repository/]
related: [DAT, SEC, REP, TG, JOB, PLG]
---

# Record ownership

Maroid serves more than one person. This guideline gives the user record,
the acting user, and the isolation between them. `GLO-user` defines a user.

## OWN-001

The hub holds one record for each person, in the table `public.users`.
The Telegram user identifier is the natural key. The record is permanent.

## OWN-002

The hub creates no user record from an interaction. The owner creates it first. The
record is active or blocked, and no third state exists. The hub deletes no record,
because a delete orphans every row that names it. `SEC-004` is the gate.

## OWN-003

Every unit of work carries one acting user. Work without one reaches no scoped table.

| Entry point       | Source of the acting user                      |
| ----------------- | ---------------------------------------------- |
| An HTTP request   | The subject claim of the token. See `SEC-003`. |
| A Telegram update | The Telegram user identifier in the update.    |
| A cron job        | The user that the job declares. See `OWN-009`. |

## OWN-004

A table is scoped or shared, and the migration states which.
A scoped table holds a row that belongs to one user. A shared table holds a row that
belongs to nobody: `public.users`, a reference table, `schema_migrations`.
A new table is scoped. A shared table gives its reason in a comment, and a table that
waits to be scoped names the feature that will scope it.

## OWN-005

A scoped table carries this column:

```sql
user_id UUID NOT NULL
    DEFAULT NULLIF(current_setting('app.user_id', true), '')::uuid
    REFERENCES public.users (id)
```

An insert does not name `user_id`. The session gives the value, and this reference is
the one exception to `DAT-003`.
A unique constraint starts with `user_id`, because two people share a natural key.

## OWN-006

A scoped table enables row level security and forces it:

```sql
ALTER TABLE plants ENABLE ROW LEVEL SECURITY;
ALTER TABLE plants FORCE ROW LEVEL SECURITY;

CREATE POLICY plants_user_isolation ON plants
    USING (user_id = NULLIF(current_setting('app.user_id', true), '')::uuid)
    WITH CHECK (user_id = NULLIF(current_setting('app.user_id', true), '')::uuid);
```

`FORCE` applies the policy to the role that owns the table, and the hub connects as that
role. A session that sets no `app.user_id` sees no row.

**Why:** The policy fails closed. A missing setting hides the data.

## OWN-007

A transaction sets the acting user before its first statement, as a bind parameter:

```go
_, err = tx.ExecContext(ctx, "SELECT set_config('app.user_id', $1, true)", userID)
```

The third argument keeps the setting inside the transaction, because the pool
reuses the connection. `PluginDB.WithTx` is the only place that sets it. See `DAT-004`.

## OWN-008

A repository writes no filter on `user_id` and sets no value for it. `OWN-006` does both.

## OWN-009

A cron job that reaches a scoped table declares the user of each run, and the scheduler
sets it. A job that reaches only a shared table declares none.

## Retired identifiers

This file has no retired identifier.
