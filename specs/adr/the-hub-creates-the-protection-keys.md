---
id: ADR-0028
title: The hub creates each protection key
type: adr
status: accepted
created: 2026-11-24
updated: 2026-11-24
decided: 2026-11-24
changes: []
supersedes:
superseded_by:
---

# The hub creates each protection key

## Context

`PSET-DD-004` and `PSECRET-DD-003` decide that the hub creates no transit key. An
administrator creates `maroid-user-<id>` and `maroid-workspace-<id>` by hand. The
policy of the hub grants `update` on the paths of encrypt and decrypt and no `create`,
so an encrypt under a missing key answers `permission denied` and creates nothing.

Each invite and each new workspace therefore needs a step at OpenBao. Until the
administrator takes it, every save of a secret of that owner fails. `WSREMOVE-DD-004`
asks the same person to set `deletion_allowed=true`, or the removal of the workspace
cannot destroy the key. `OWNKEY` asks the hub to do both.

## Decision

The hub creates the protection key through a task of the outbox of `ADR-0027`, of the
type `secret_key.create`. The task commits in the transaction that creates the user
record or the workspace. The handler creates the key when it is absent, then sets
`deletion_allowed=true` on it. A key that exists keeps every version.

The policy of the hub grants `update` on `transit/keys/maroid-user-*`, and `read`,
`update`, and `delete` on `transit/keys/maroid-workspace-*`. A write of a key needs
`update`, not `create`, as `OWNKEY-DD-002` records. Each path of the keys admits the
parameter `deletion_allowed` with the value `true` alone. The paths of encrypt and decrypt
keep `update` alone. An encrypt creates its key only with `create` on its path, so a
save of a secret never creates a key.

## Rationale

The outbox commits the task with the record, so a crash after the commit loses no key,
and a store that is down blocks no invite. The creation and the configuration are both
idempotent, so the retry of `JOB-009` is safe.

The reason of `PSET-DD-004` was that a hub that an attacker reaches mints no key. The
prefix keeps that limit for every other name, and `OWNKEY-NFR-003` holds it. Inside the
prefix, `update` also lets the hub rotate a key. The allowed parameters refuse a trim
and every other change of the configuration, so the hub cannot make an old version
unreadable. No path of export or backup enters the policy.

An encrypt that creates its key closes the gap between the commit and the task. It
needs `create` on the path of encrypt, and a defect then mints a key for any name that
it encrypts under. The owner accepts the gap of `OWNKEY-FR-009` instead.

## Alternatives

| Alternative                                                   | Why we did not select it                                                                   |
| ------------------------------------------------------------- | ------------------------------------------------------------------------------------------ |
| Keep the creation by hand.                                    | Each invite and each workspace needs an administrator at OpenBao.                          |
| Create the key in the request, before the commit.             | A store that is down blocks the invite, and a rollback leaves a key with no owner.         |
| Grant `create` on the path of encrypt.                        | A save creates its key, so a defect mints a key for any name under the prefix it encrypts. |
| A key that the administrator creates with `deletion_allowed`. | The removal of `WSREMOVE` still depends on a step by hand for each workspace.              |

## Consequences

### Rules that change

No guideline rule changes. The decisions below change.

### Specifications that change

| Document                    | Change                                                                                                        |
| --------------------------- | ------------------------------------------------------------------------------------------------------------- |
| `features/pset/spec.md`     | `PSET-DD-004` drops "An administrator creates each key" and "it creates no key". It cites `ADR-0028`.         |
| `features/psecret/spec.md`  | `PSECRET-DD-003` drops "The hub creates no key". The alternative cites `ADR-0028`.                            |
| `features/wsremove/spec.md` | `WSREMOVE-DD-004` drops the step by hand for `deletion_allowed` and the alternative "`update` on the prefix". |

### Result

- Positive: an invite and a new workspace need no step at OpenBao.
- Positive: every key allows its destruction, so `WSREMOVE-FR-008` holds for a key
  that an administrator created before.
- Negative: the hub can create and rotate a key under the two prefixes.
- Negative: a save of a secret in the first seconds after the creation fails.
- Negative: an instance changes the policy of the hub before it runs this feature.
- Work that follows: the type `secret_key.create`, its handler, the migration that
  queues a task for each user record and each workspace, and the policy of `testbao`.
