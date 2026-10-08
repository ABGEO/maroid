---
id: ADR-0011
title: A repository of the hub comes from the transaction
type: adr
status: accepted
created: 2026-10-08
updated: 2026-10-09
decided: 2026-10-09
changes: [REP-003]
supersedes:
superseded_by:
---

# A repository of the hub comes from the transaction

## Context

`REP-003` makes a repository take `*sqlx.Tx`, and the caller creates it inside the
transaction. The scope of `REP` is `plugins/*/repository/`, so the rule never bound
the hub.

The hub used two shapes. Three repositories took the transaction: `TelegramChat`,
`Idempotency`, and `PluginSettings`, each over a table that a policy scopes. Eight
held the pool and came from the container as a dependency: `User`, `Identity`,
`Invitation`, `AuthFlow`, `Workspace`, `WorkspaceMember`, `WorkspacePlugin`, and
`AllowedPlugin`. The compliance table of `IDENT` gives the reason for the pool: each
operation is one statement on the request path.

The second shape took a `tx` parameter on some methods and none on others. A write
that spans the hub and Dex then needed a hook that ran service code inside a
transaction that the repository owned, which `IDPROV` added as `beforeCommit`.

## Decision

Every repository of the hub takes `*sqlx.Tx` from its constructor, as a plugin does. A
service, a handler, or a middleware holds `*sqlx.DB` and creates each repository inside
`database.WithTx`, `database.WithScopeTx`, or `database.FetchTx`. The container offers
no repository.

## Rationale

One shape for every repository. A reader of a service sees each write of one
transaction in one block, and a write that also calls Dex commits after Dex answers
with no hook.

A method of a repository no longer chooses between the pool and a transaction, so no
read lands outside the transaction of the write beside it.

A test of a service runs against the database, as every test of a repository already
does. The two tests that faked a repository now drive a real row: the resolver reads
an identity, and a trigger refuses the write of an invitation.

## Alternatives

| Alternative                                                          | Why we did not select it                                                                                 |
| -------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------- |
| Keep the pool repositories and their `tx` parameters.                | Two shapes, and a hook for each write that crosses to an external service.                                |
| Convert `Identity` alone, which `IDPROV` uses.                       | Four repositories from the transaction and seven on the pool. The mix stays.                              |
| A second constructor from the transaction beside the one from the pool. | A third shape.                                                                                         |

## Consequences

### Rules that change

`REP` gains `apps/hub/internal/repository/` in its `scope` field.

`REP-003` becomes:

> A repository takes `*sqlx.Tx` from its constructor. It does not take `*sqlx.DB`
> and it does not take `pluginapi.PluginDB`. No method of it takes a transaction.
>
> A plugin creates the repository inside `PluginDB.WithTx`. The hub creates it inside
> `database.WithTx`, `database.WithScopeTx`, or `database.FetchTx`. A repository is
> never a dependency that a constructor takes.
>
> **Why:** `DAT-004` sets the search path on the transaction of a plugin, and `OWN-007`
> sets the scope on the transaction of the hub. A repository that holds the pool would
> read outside both, and a write that spans two repositories would span two
> transactions.

### Cost

A read on the request path runs `BEGIN`, the statement, and `COMMIT`: three round trips
in place of one. The resolver of the acting user and the middleware of the workspace
each take one transaction. `IDENT-NFR-002` gives 10 milliseconds at the 95th percentile
for both, and a round trip on the network of the deployment is under 1 millisecond.
`IDENT-SC-012` measures it again after this change.

### Specifications to examine

| Document                        | Why                                                                                 |
| ------------------------------- | ----------------------------------------------------------------------------------- |
| `features/ident/spec.md`        | The row of `REP-003` gives the pool as the shape of the hub.                        |
| `features/extid/spec.md`        | `IdentityRepository.Attach` takes a transaction.                                    |
| `features/wspace/spec.md`       | `workspace.Middleware` takes `WorkspaceMemberRepository`.                           |
| `features/plugacc/spec.md`      | `AdmitAdministrator` takes `WorkspaceRepository`, and `WorkspaceAccess` holds `Members`. |
| `features/mcphub/spec.md`       | `tools.NewListWorkspaces` takes a repository.                                       |
| `features/idprov/spec.md`       | `Detach` and `DeleteByProvider` take a transaction.                                 |

### Result

- Positive: one shape of repository in the whole repository.
- Positive: a write that spans the database and Dex reads as one block, with no hook.
- Negative: two more round trips for each read on the request path.
- Negative: a test of a service needs Docker, where two of them used a fake.
