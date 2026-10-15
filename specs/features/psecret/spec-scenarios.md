---
id: PSECRET
title: The scenarios of the secrets that a plugin writes
type: spec
status: approved
created: 2026-10-12
updated: 2026-10-12
approved_by: Temuri
approved_on: 2026-10-12
constrained_by: [TST, OWN, TRC]
requirements: features/psecret/requirements.md
---

# Specification: The scenarios of the secrets that a plugin writes

An integration scenario runs against PostgreSQL and OpenBao with the transit keys of
user A, user B, workspace A, and workspace B. "The probe" is a plugin identifier that a
test passes to `NewPluginSecrets`. A unit scenario replaces the database or the cipher
with a fake.

## `PSECRET-SC-001`

**Verifies:** `PSECRET-FR-001`, `PSECRET-FR-003`, `PSECRET-FR-006`
**Layer:** integration

**Given** user A is the acting user.
**When** the probe calls `Put(user, "session", "token-1")`, then `Get(user, "session")`.
**Then** the read returns `token-1`.

## `PSECRET-SC-002`

**Verifies:** `PSECRET-FR-002`
**Layer:** integration

**Given** user A acts in workspace A.
**When** the probe calls `Put(workspace, "session", "token-w")`, then user B acts in
workspace A and the probe calls `Get(workspace, "session")`.
**Then** the read returns `token-w`.

## `PSECRET-SC-003`

**Verifies:** `PSECRET-FR-004`
**Layer:** integration

**Given** user A is the acting user, and no row holds `session` for the probe.
**When** the probe calls `Get(user, "session")`.
**Then** the error satisfies `errors.Is(err, ErrSecretAbsent)`.

## `PSECRET-SC-004`

**Verifies:** `PSECRET-FR-004`
**Layer:** integration

**Given** user A acts in workspace A, and the probe stored `session` for the user scope.
**When** the probe calls `Get(workspace, "session")`.
**Then** the error satisfies `errors.Is(err, ErrSecretAbsent)`.

## `PSECRET-SC-005`

**Verifies:** `PSECRET-FR-005`
**Layer:** integration

**Given** user A is the acting user, and the probe called `PutUntil(user, "login", "a", now + 5 minutes)`.
**When** the probe calls `Put(user, "login", "b")`.
**Then** `Get(user, "login")` returns `b`. `plugin_user_secrets` holds one row for the
name, and its `expires_at` is null.

## `PSECRET-SC-006`

**Verifies:** `PSECRET-FR-007`
**Layer:** integration

**Given** user A is the acting user, and the probe stored `session`.
**When** the probe calls `Delete(user, "session")`.
**Then** `Get(user, "session")` answers `ErrSecretAbsent`.

## `PSECRET-SC-007`

**Verifies:** `PSECRET-FR-007`
**Layer:** integration

**Given** user A is the acting user, and no row holds `session` for the probe.
**When** the probe calls `Delete(user, "session")`.
**Then** the call returns no error.

## `PSECRET-SC-008`

**Verifies:** `PSECRET-FR-008`, `PSECRET-FR-009`
**Layer:** integration

**Given** user A is the acting user.
**When** the probe calls `PutUntil(user, "login", "uuid-1", now + 5 minutes)`, then
`Get(user, "login")` at once.
**Then** the read returns `uuid-1`.

## `PSECRET-SC-009`

**Verifies:** `PSECRET-FR-009`
**Layer:** integration

**Given** user A is the acting user, and the probe called `PutUntil(user, "login", "uuid-1", now + 1 second)`.
**When** the probe calls `Get(user, "login")` 2 seconds later.
**Then** the error satisfies `errors.Is(err, ErrSecretAbsent)`.

## `PSECRET-SC-010`

**Verifies:** `PSECRET-FR-009`
**Layer:** unit

**Given** a store with a fake cipher.
**When** the probe calls `PutUntil(user, "login", "uuid-1", now - 1 second)`.
**Then** the error satisfies `errors.Is(err, ErrSecretInvalid)`. The fake cipher records no call.

## `PSECRET-SC-011`

**Verifies:** `PSECRET-FR-010`
**Layer:** integration

**Given** user A holds three rows for the probe: one expired 1 hour ago, one that
expires in 1 hour, and one with no expiry.
**When** the `plugin_user_secrets_sweep` job runs for user A.
**Then** `plugin_user_secrets` holds the two rows that did not expire.

## `PSECRET-SC-012`

**Verifies:** `PSECRET-FR-010`
**Layer:** integration

**Given** workspace A and workspace B each hold one expired row and one row with no expiry.
**When** the `plugin_workspace_secrets_sweep` job runs for workspace A.
**Then** workspace A holds its row with no expiry. Workspace B holds both of its rows.

## `PSECRET-SC-013`

**Verifies:** `PSECRET-FR-010`
**Layer:** unit

**Given** a cron worker whose lister answers workspaces A, B, and C, and a job of the
hub that declares `CronScopePerWorkspace`.
**When** the worker prepares, then runs the job one time.
**Then** `Prepare` returns no error. The job runs three times, one time with each
workspace as the acting workspace, and with no acting user.

## `PSECRET-SC-014`

**Verifies:** `PSECRET-FR-011`
**Layer:** integration

**Given** user A manages workspace A, and the probe stored `session` with no expiry for workspace A.
**When** user A disables the probe in workspace A and enables it again.
**Then** `Get(workspace, "session")` returns the stored value.

## `PSECRET-SC-015`

**Verifies:** `PSECRET-FR-012`
**Layer:** unit

**Given** a store with a fake cipher, and user A as the acting user.
**When** the probe writes under each name of the table.
**Then** each write gives the answer of the table.

| Name              | Answer             |
| ----------------- | ------------------ |
| `session`         | No error           |
| 64 characters `a` | No error           |
| 65 characters `a` | `ErrSecretInvalid` |
| `Session Token`   | `ErrSecretInvalid` |
| The empty string  | `ErrSecretInvalid` |

## `PSECRET-SC-016`

**Verifies:** `PSECRET-FR-013`
**Layer:** unit

**Given** a store with a fake cipher, and user A as the acting user.
**When** the probe writes `session` with each value of the table.
**Then** each write gives the answer of the table.

| Value                | Answer             |
| -------------------- | ------------------ |
| 16384 characters `ა` | No error           |
| 16385 characters `a` | `ErrSecretInvalid` |
| The empty string     | `ErrSecretInvalid` |

## `PSECRET-SC-017`

**Verifies:** `PSECRET-FR-014`
**Layer:** unit

**Given** a context with an acting workspace and no acting user.
**When** the probe calls `Get`, `Put`, and `Delete` for the user scope.
**Then** each error satisfies `errors.Is(err, ErrSecretScopeAbsent)`. The fake database records no transaction.

## `PSECRET-SC-018`

**Verifies:** `PSECRET-FR-015`
**Layer:** unit

**Given** a context with an acting user and no acting workspace, as a job for each user runs.
**When** the probe calls `Get`, `Put`, and `Delete` for the workspace scope.
**Then** each error satisfies `errors.Is(err, ErrSecretScopeAbsent)`.

## `PSECRET-SC-019`

**Verifies:** `PSECRET-FR-016`, `PSECRET-INV-003`
**Layer:** integration

**Given** user A is the acting user.
**When** the probe calls `Put(user, "session", "plain-token-value")`.
**Then** the `ciphertext` column of the row does not hold `plain-token-value`, and it
starts with `vault:v`.

## `PSECRET-SC-020`

**Verifies:** `PSECRET-FR-017`
**Layer:** unit

**Given** a store with a fake cipher that records each key, and user A acting in workspace A.
**When** the probe writes one value for the user scope and one for the workspace scope.
**Then** the fake cipher records `secret.UserKey(A)`, then `secret.WorkspaceKey(A)`.

## `PSECRET-SC-021`

**Verifies:** `PSECRET-FR-018`
**Layer:** unit

**Given** a stored row, and a fake cipher whose `Decrypt` fails.
**When** the probe calls `Get(user, "session")`.
**Then** the call returns an error. The error does not satisfy `errors.Is(err, ErrSecretAbsent)`.

## `PSECRET-SC-022`

**Verifies:** `PSECRET-FR-019`
**Layer:** unit

**Given** a store whose logger writes to a buffer, a fake cipher whose `Encrypt` fails,
and a second store whose fake database fails the `Upsert`.
**When** the probe calls `Put(user, "session", "plain-token-value")` on each store.
**Then** no error text and no line of the buffer holds `plain-token-value`.

## `PSECRET-SC-023`

**Verifies:** `PSECRET-FR-020`
**Layer:** unit

**Given** the packages of the hub.
**When** the test lists the dependencies of `internal/handler`, `internal/server`, and
`internal/mcpserver` with `go list -deps`.
**Then** no list holds `apps/hub/internal/pluginsecret`.

## `PSECRET-SC-024`

**Verifies:** `PSECRET-NFR-001`
**Layer:** integration

**Given** user A is the acting user, and the probe stored `session` with a value of 1024 characters.
**When** the probe calls `Get(user, "session")` 100 times.
**Then** the 95th percentile of the time from the call to the return of the value is
under 200 milliseconds.

## `PSECRET-SC-025`

**Verifies:** `PSECRET-INV-001`
**Layer:** integration

**Given** user A is the acting user, and the plugin `dev.maroid.probe-a` stored `session`.
**When** the plugin `dev.maroid.probe-b` calls `Get(user, "session")`.
**Then** the error satisfies `errors.Is(err, ErrSecretAbsent)`.

## `PSECRET-SC-026`

**Verifies:** `PSECRET-INV-002`
**Layer:** integration

**Given** the probe stored `session` for user A, and for workspace A.
**When** user B acts in workspace B and the probe reads `session` in each scope.
**Then** each read answers `ErrSecretAbsent`.

## Retired identifiers

This file has no retired identifier.
