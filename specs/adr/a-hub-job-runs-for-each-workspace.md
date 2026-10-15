---
id: ADR-0014
title: A job of the hub runs for each workspace
type: adr
status: accepted
created: 2026-10-12
updated: 2026-10-12
decided: 2026-10-12
changes: [OWN-009]
supersedes:
superseded_by:
---

# A job of the hub runs for each workspace

## Context

`PSECRET-FR-010` makes Maroid delete an expired plugin secret within 24 hours. A plugin
secret of a workspace lives in a table that a workspace scopes, and `OWN-006` forces its
policy. A shared run carries no acting workspace, so it sees no row of that table.

`OWN-009` lets a job run for each workspace only when a plugin owns the job: the job runs
for each workspace that enables its plugin. A job of the hub names no plugin, and the cron
worker refuses it with `ErrCronScopeWithoutPlugin` (`features/plugacc/spec-clients.md`,
section 3). The hub therefore holds no way to reach every row of a table that a workspace
scopes.

## Decision

A job of the hub that declares the scope for each workspace runs one time for each
workspace of the instance, with that workspace as the acting workspace.

## Rationale

The run reaches each table that a workspace scopes through the same policy as every
other unit of work. No policy crosses a scope, and `OWN-006` stays as it is. The cron
worker already runs a job for each user over every active user, so the change adds the
same loop for the second scope.

## Alternatives

| Alternative                                                                       | Why we did not select it                                                                                       |
| --------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------- |
| A second policy that lets a shared run delete an expired row of any scope         | It is the first policy that crosses a scope. It changes `OWN-006`, and a defect in it reaches every workspace. |
| A delete of the expired rows of a scope at each read and each write in that scope | A scope that no unit of work touches keeps its expired rows for ever. `PSECRET-FR-010` fails.                  |
| A role of the database that bypasses the policy for the job                       | It adds a second connection and a second credential to the hub for one delete.                                 |

## Consequences

`OWN-009` changes. The new text:

> A cron job declares its scope. A shared job runs one time, with no acting workspace
> and no acting user, and reaches only a shared table. A job of a plugin for each
> workspace runs one time for each workspace that enables the plugin. A job of the hub
> for each workspace runs one time for each workspace of the instance. The scheduler
> sets the acting workspace of each run. A job for each user runs one time for each
> active user, and the scheduler sets the acting user of each run. A job of a plugin
> runs for each user only when it writes a table that a user scopes.

- Positive: the hub reaches every table that a workspace scopes with no new policy.
- Negative: one run of such a job costs one transaction for each workspace, also for a
  workspace that holds no row.
- Work that follows: `PSECRET` removes `ErrCronScopeWithoutPlugin` and changes the cron
  worker. The last paragraph of section 3 of `features/plugacc/spec-clients.md` no
  longer holds, and `PSECRET` replaces it.
