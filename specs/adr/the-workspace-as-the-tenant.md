---
id: ADR-0008
title: The workspace as the tenant
type: adr
status: accepted
created: 2026-10-01
updated: 2026-10-01
decided: 2026-10-01
changes: [OWN-001, OWN-002, OWN-003, OWN-004, OWN-005, OWN-006, OWN-007, OWN-008,
  OWN-009, OWN-010, OWN-011, SEC-001, SEC-011, SEC-012, SEC-013, API-003, API-004,
  CFG-007, ERR-003, RES-002, RES-005, RES-009, SPC-003, UI-008, PLG-010,
  GLO-acting-user, GLO-scoped-table, GLO-shared-table, GLO-scoped-record,
  GLO-shared-record, GLO-setting, GLO-invitation, GLO-workspace, GLO-member,
  GLO-workspace-role, GLO-acting-workspace, GLO-administrator,
  GLO-plugin-allowlist, GLO-enablement, GLO-permission]
supersedes:
superseded_by:
---

# The workspace as the tenant

## Context

`OWN-005` gives every scoped row one user. `OWN-006` shows the row to that user
and to nobody else. The rule came from `IDENT`, before Maroid had any notion of
a permission.

Three needs do not fit it:

| Need                                                                                  | Why one user per row fails                                                    |
| ------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------- |
| Two or more people maintain one garden: its environments and every plant in them.     | The row has one user, and the policy hides it from the second person.          |
| A household reads one electricity bill.                                                | Each member fetches and stores a copy, and a per-user job fetches it once for each member. |
| A person holds a different level of access in each place: writes in one, reads in another. | A user has no level. The policy answers yes or no for the whole row.          |

A pension contribution stays private to one person. Sharing is a choice of the
people who hold the data, not a property of the plugin.

The list problem decides where access lives. An engine outside the database
answers "may this person see this row". It cannot answer "which rows may this
person see" inside a query. Row level security can, if the rule is in
PostgreSQL. Access to a row must therefore be a fact that a policy reads.

No plugin table carries `user_id` today. `IDENT-FR-008` keeps every plugin table
shared, and no plugin declares `CronScopePerUser`. Two hub tables are scoped:
`public.plugin_settings` and `public.idempotency_keys`.

Maroid also holds no role today. Any active user reaches every loaded plugin,
and a user record is created only from the CLI.

## Decision

A workspace owns every row of a plugin. A row carries `workspace_id`, and the
policy of the table shows it to the acting workspace alone. Users join a
workspace as members, each with one fixed role: manager, editor, or viewer.

Each unit of work acts in one workspace. The hub resolves it, checks that the
acting user is a member of it, checks that the plugin is enabled in it, and
checks the role against the permission that the entry declares. The hub then
calls the plugin.

## Rationale

**One policy for every plugin.** `workspace_id = app.workspace_id` is the whole
rule. A plugin writes no share table, no membership join, and no inheritance. An
environment and its plants sit in one workspace, so access to one is access to
the other.

**Shared ownership needs no owner.** A member who leaves takes nothing with
them, and nothing is left without a holder. The household keeps the bill.

**Private data needs no special kind.** A workspace with one member is private.
One kind of workspace keeps one code path. The hub creates the first workspace
of a user with the user record, so a new user reaches Maroid with a place to
work. The workspace carries no kind. Its manager keeps it private or invites
others later.

**One acting workspace.** Every write needs a target workspace anyway. One
workspace per unit of work keeps the policy an equality check that an index
serves, and keeps the view of a person on one place at a time.

**A path segment carries the workspace.** `Z-183` puts the business intent of
an operation in the path, the query, the method, and the content, and keeps a
header for generic context. The person chooses the workspace for each request,
so it is business intent. `Z-183` lists `X-Tenant-ID`, but the gateway derives
that value from the OAuth token: one credential names one tenant. One session of
Maroid reaches every workspace of its user, so no token names the workspace.

The workspace is a resource, and `Z-143` puts a sub-resource under the path of
its parent. Each intermediate path resolves: `/workspaces/{workspaceId}/plugins`
answers the enablements, and `/workspaces/{workspaceId}/plugins/{id}` answers
one of them. A route of a plugin sits two levels below the workspace, within the
three that `Z-147` allows, while the plugin keeps its own resources flat.

`Z-145` offers a top level path to a resource with a globally unique identifier,
and every row carries a UUID. The policy of `OWN-006` needs the workspace before
the query finds the row, so a path with no workspace finds nothing.

`{workspaceId}` is the UUID of the workspace, as `DAT-009` gives every row. A
rename changes no address. The deck shows the name.

The address of a page in the deck carries the workspace too. Two tabs show two
workspaces, and a link opens the same workspace for every member.

**Fixed roles keep one vocabulary.** A plugin declares a permission and names
the lowest role that holds it. The roles form one order: viewer, editor,
manager. A person learns three words once and reads the same words in every
plugin.

**The check is one lookup and one comparison.** The model needs the role of the
member and the lowest role of the permission. A policy engine is not required
by this decision. A later ADR records whether Casbin or an embedded OPA earns a
place, after a spike against this model.

**The hub checks membership, and the policy fails closed.** The hub sets
`app.workspace_id` only after it finds the membership. A transaction that sets
no workspace sees no row. A cron run of a plugin acts for a workspace and for no
user, so the policy reads the workspace alone. A job of the hub that reaches a
table that a user scopes runs for each user instead.

**The administrator manages the instance, not the data.** An administrator
creates a user and sets the plugin allowlist of a user. Two facts follow with no rule
of their own: the plugin allowlist does not limit an administrator, and the management
API reaches every workspace of the instance. An administrator therefore enables
any plugin in any workspace. An administrator reads the rows of a workspace only
as a member of it.

## Alternatives

| Alternative | Why we did not select it |
| ----------- | ------------------------ |
| Keep one user per row and add a permission engine. | The list problem. A shared row stays hidden from the second person, whatever the engine answers. |
| Groups as principals, and a share table in each plugin. | Each plugin writes its own share table and its own policy. A row still has one owner, who keeps it after they leave the household. |
| One share table in the hub for every entity of every plugin. | No foreign key reaches across schemas. Each plugin still joins the table in each policy, and inheritance needs a join for each level. |
| A relationship service: OpenFGA, SpiceDB, Ory Keto. | A second service to deploy and to keep in sync. The list problem remains: the database still needs the facts. |
| A personal kind of workspace, apart from a shared kind. | Two kinds of workspace, two code paths, and a rule that forbids a second member in one of them. |
| Every workspace of a person at once in each request. | A write still names one workspace. The policy becomes `= ANY()`, and a page mixes the rows of several places. |
| The workspace in a request header, or a current workspace in the session. | `Z-183` keeps business intent out of a header. A current workspace in the session makes two tabs of one person interfere, and a link loses its workspace. |
| A slug of the workspace in the path. | A rename breaks every stored link, or the hub keeps every old slug. A slug also needs a uniqueness rule across the instance. |
| The workspace in the access token. | One token then names one workspace. A switch needs a new sign in, and an MCP client reaches one workspace for each connection. |
| Custom roles in each workspace. | A role editor in the deck, and a different vocabulary in each workspace. |
| Roles that each plugin defines. | A different vocabulary in each plugin. A manager cannot read the access of a member across plugins. |

## Consequences

### Rules that change

`OWN-002` names the administrator as the creator. The new text:

> The hub creates no user record from an interaction. An administrator creates it
> first. The record is active or blocked, and no third state exists. The hub deletes
> no record, because a delete orphans every row that names it. `SEC-004` is the gate.

`OWN-003` gains the acting workspace. The new text:

> Every unit of work carries one acting user, one acting workspace, or both. Work
> without an acting workspace reaches no table that a workspace scopes. Work without
> an acting user reaches no table that a user scopes.
>
> | Entry point       | Acting user                                                       | Acting workspace                                    |
> | ----------------- | ----------------------------------------------------------------- | --------------------------------------------------- |
> | An HTTP request   | The identity that `federated_claims` names. See `SEC-003`.        | The `{workspaceId}` segment of the path.            |
> | A Telegram update | The identity of the Telegram connector for the sender.            | The workspace that the chat selected.               |
> | An MCP tool call  | The identity that `federated_claims` names. See `SEC-002`.        | The `workspace` argument of the call.               |
> | A cron job        | The user of the run, for a job for each user. See `OWN-009`.     | The workspace of the run, for a job for each workspace. See `OWN-009`. |
>
> When a unit of work carries both, the hub checks that the acting user is a member
> of the acting workspace before it sets the workspace. A user who is not a member
> gets `not-found`.

`OWN-004` names three scopes. The new text:

> A table is scoped to a workspace, scoped to a user, or shared, and the migration
> states which. A table of a plugin is scoped to a workspace or shared. A table that
> a user scopes belongs to the hub. It holds a value that belongs to one person in
> every workspace: an idempotency key, a setting that a plugin declares personal.
> A shared table holds a row that belongs to nobody: `public.users`, a reference
> table, `schema_migrations`. A new table of a plugin is scoped to a workspace. A
> shared table gives its reason in a comment.

`OWN-005` gives the column of each scope. The new text:

> A table that a workspace scopes carries this column:
>
> ```sql
> workspace_id UUID NOT NULL
>     DEFAULT NULLIF(current_setting('app.workspace_id', true), '')::uuid
>     REFERENCES public.workspaces (id)
> ```
>
> A table that a user scopes carries the same column for `user_id`, with the setting
> `app.user_id` and the reference `public.users (id)`.
>
> An insert names neither column. The session gives the value, and each reference is
> an exception to `DAT-003`. A unique constraint starts with the scope column.

`OWN-006` gives the policy of a workspace. The new text:

> A scoped table enables row level security and forces it:
>
> ```sql
> ALTER TABLE plants ENABLE ROW LEVEL SECURITY;
> ALTER TABLE plants FORCE ROW LEVEL SECURITY;
>
> CREATE POLICY plants_workspace_isolation ON plants
>     USING (workspace_id = NULLIF(current_setting('app.workspace_id', true), '')::uuid)
>     WITH CHECK (workspace_id = NULLIF(current_setting('app.workspace_id', true), '')::uuid);
> ```
>
> A table that a user scopes uses the same policy on `user_id` and `app.user_id`.
> `FORCE` applies the policy to the role that owns the table. A session that sets
> no value for the scope sees no row.
>
> **Why:** The policy fails closed. A missing setting hides the data.

`OWN-007` sets both values. The new text:

> A transaction sets the acting user and the acting workspace before its first
> statement, each as a bind parameter, and each one only when the unit of work
> carries it:
>
> ```go
> _, err = tx.ExecContext(ctx, "SELECT set_config('app.workspace_id', $1, true)", workspaceID)
> ```
>
> The third argument keeps the setting inside the transaction, because the pool
> reuses the connection.
>
> Two functions set them, and no other place does: `database.WithScopeTx` for the
> hub, and `PluginDB.WithTx` for a plugin, with the search path. See `DAT-004`.

`OWN-008` names both columns. The new text:

> A repository writes no filter on `workspace_id` or `user_id`, and sets no value
> for either. `OWN-006` does both.

`OWN-009` runs a job for each workspace. The new text:

> A cron job declares its scope. A shared job runs one time, with no acting
> workspace and no acting user, and reaches only a shared table. A job for each
> workspace runs one time for each workspace that enables the plugin, and the
> scheduler sets the acting workspace of each run. A job for each user runs one
> time for each active user, and the scheduler sets the acting user of each run.
> Only a job of the hub runs for each user, because a table that a user scopes
> belongs to the hub.

`OWN-010` is new:

> A workspace is one row in `public.workspaces`. It holds the records of a plugin
> that its members share. Every workspace is of one kind. The hub creates the
> first workspace of a user in the same transaction as the user record, with that
> user as its manager. Any active user creates a further workspace and becomes its
> first manager. A user with no workspace reaches no plugin. The hub deletes no workspace, because a delete
> orphans every row that names it.

`OWN-011` is new:

> One row in `public.workspace_members` binds one user to one workspace with one
> role: `manager`, `editor`, or `viewer`. The roles form that order, from the most
> access to the least. A manager adds an existing user record as a member, removes
> a member, and changes a role. A workspace keeps at least one manager, so the
> last manager neither leaves nor loses the role.

`SEC-001` moves the data to the workspace. The second paragraph becomes:

> Maroid holds a local user record. The record carries no credential. A workspace
> owns the data, and `OWN-010` gives it.

`SEC-011` is new:

> A user record is an administrator or not. An administrator creates and blocks a
> user record, and sets the plugin allowlist of a user. An administrator manages every
> workspace of the instance through the management API: its members and its
> enablements. The plugin allowlist does not limit an administrator. An administrator
> reads the rows of a workspace only as a member of it. The CLI makes the first
> administrator.
>
> **Why:** The person who runs the instance is not entitled to the pension of each
> person on it.

`SEC-012` is new:

> A plugin serves a workspace only where it is enabled. A plugin starts disabled
> in every workspace. A manager enables a plugin that the plugin allowlist of the
> manager holds. `SEC-011` gives the access of an administrator.
> A member uses every plugin that the workspace enables, and the hub reads no
> plugin allowlist of the member.
>
> A plugin that is disabled in the acting workspace is absent from it. Its routes
> answer `not-found`, a call to its MCP tools in that workspace answers `not-found`,
> its Telegram commands are absent from the menu of the chat, and its jobs skip the
> workspace. The list of MCP tools names no workspace, so it stays one list for every
> person.

`SEC-013` is new:

> A plugin declares each permission that it checks, and names the lowest role
> that holds it. The hub declares its own permissions the same way. The hub
> prefixes a permission of a plugin with the plugin identifier.
>
> Every route, MCP tool, and Telegram command that acts in a workspace declares
> one permission. The hub checks the role of the acting user in the acting
> workspace against it before it calls the handler. A route of the hub that acts
> in no workspace declares none. A member whose role does not hold the permission
> gets `permission-denied`. A plugin whose entry declares no permission, or a
> permission that the plugin does not declare, fails to load.
>
> **Why:** A check that the hub runs is a check that no handler forgets. A load
> that fails names the gap before any person meets it.

`API-003` moves the routes of a plugin under the workspace. The changed rows:

> | Method and path                                         | Serves                                         | Access                 |
> | ------------------------------------------------------- | ---------------------------------------------- | ---------------------- |
> | `/workspaces*`                                          | The workspaces, the members, and the enablements | Authenticated        |
> | `/workspaces/{workspaceId}/plugins/{id}/api/*`          | The routes of a plugin                         | Member, enabled        |
> | `/workspaces/{workspaceId}/plugins/{id}/settings*`      | The settings of a plugin                       | Member, enabled        |
> | `/users*`                                               | The user records and the plugin allowlists     | Administrator          |

`API-004` follows. Its second sentence becomes:

> The hub mounts every route of a plugin under `/workspaces/{workspaceId}/plugins/{id}/api`.

`/plugins` stays, and lists the plugins that the hub loaded. `/plugins/{id}/ui/*`
stays, because an asset belongs to no workspace.

The glossary changes six terms and gains eight:

| ID                      | Change   | Meaning                                                                                |
| ----------------------- | -------- | -------------------------------------------------------------------------------------- |
| `GLO-scoped-table`      | Changed  | A table whose every row belongs to one workspace or to one user, with row level security. |
| `GLO-shared-table`      | Changed  | A table whose rows belong to no workspace and to no user.                              |
| `GLO-scoped-record`     | Changed  | A record that belongs to exactly one workspace or exactly one user.                    |
| `GLO-shared-record`     | Changed  | A record that belongs to nobody. Every user reads it.                                  |
| `GLO-acting-user`       | Changed  | The user that one unit of work runs for. A request, an update, a tool call, and the run of a job for each user each have exactly one. Any other cron run has none. |
| `GLO-setting`           | Changed  | One value that a user or a workspace stores for one field of one plugin. The plugin declares which. |
| `GLO-workspace`         | New      | The place that owns the records of a plugin. Its members share them.                    |
| `GLO-member`            | New      | A user who belongs to a workspace, with one workspace role.                             |
| `GLO-workspace-role`    | New      | The access of a member: manager, editor, or viewer.                                    |
| `GLO-acting-workspace`  | New      | The workspace that one unit of work runs in.                                           |
| `GLO-administrator`     | New      | A user who manages the users, the plugin allowlists, and every workspace of the instance.     |
| `GLO-plugin-allowlist`  | New      | The plugins that one user may enable in a workspace that they manage.                  |
| `GLO-enablement`        | New      | The fact that one plugin serves one workspace.                                         |
| `GLO-permission`        | New      | One action that the hub or a plugin checks, with the lowest workspace role that holds it. |

The role of a workspace is `manager` and not `owner`, because `GLO-owner` names
the maintainer of the repository. The term is `plugin allowlist` and not
`allowlist`, because `SEC-004` and `SEC-007` each use the word for a gate.

### Rules that follow

These rules change to agree with the rules above. Each change carries no new
decision.

| Rule                     | Change                                                                  |
| ------------------------ | ----------------------------------------------------------------------- |
| `OWN-001`                | An administrator gives the name of a user record.                       |
| `CFG-007`                | A value of one workspace or one person lives in a scoped record.        |
| `ERR-003`                | A row of another workspace, a workspace of no membership, and a disabled plugin each answer `not-found`. The registry gains `permission-denied`, status 403, for `SEC-013`. |
| `RES-002`                | The `Z-110` row and the `Z-135` row name the new prefix. The `Z-143` row stays, because `/plugins/{id}` still leads to the assets of `SEC-006`. |
| `RES-005`                | A collection answers the rows of the acting workspace.                  |
| `RES-009`                | `uid` stays, because the permission of `SEC-013` is no scope of the token. |
| `SPC-003`                | A route, an MCP tool, and a Telegram command declare a permission. A cron job declares where it runs. |
| `UI-008`                 | The client of a plugin user interface reaches the acting workspace.     |
| `PLG-010`                | Loading a plugin is not enablement.                                     |
| `GLO-invitation`         | An administrator issues an invitation.                                  |

The introductions of `JOB`, `REP`, and `TG` follow too. `CLI-001` gains the
command that makes an administrator in the feature that names it. `PLG-006`
already gives the parts of the permission capability.

### Specifications that follow

Each feature below changed to agree with this decision. A statement that keeps its
subject keeps its identifier. A build plan step that the code has not met yet is
open.

| Document            | Change                                                                                  |
| ------------------- | ----------------------------------------------------------------------------------------- |
| `features/ident/`   | A row belongs to the acting workspace. A job runs for each workspace or for each user. `IDENT-SC-013` and `IDENT-SC-014` are new. |
| `features/pset/`    | `PSET-FR-021`: a plugin declares the scope of each field. One table for each scope, one key for each workspace, and the routes under `/workspaces/{workspaceId}`. `PSET-DD-012` and `PSET-DD-013` are new. |
| `features/mcphub/`  | `MCPHUB-FR-020` to `MCPHUB-FR-023` and `MCPHUB-INV-003`: the workspace list tool, the `workspace` argument, and its three checks. `MCPHUB-DD-021` to `MCPHUB-DD-023` are new. |
| `features/pcap/`    | The `permissions` capability, the permission of each item, and the workspace segment of each path. |
| `features/extid/`   | An administrator issues an invitation. `EXTID-FR-018`: the command creates the first workspace of the record. |
| `features/apifmt/`  | The example routes move under `/workspaces/{workspaceId}`, and a collection answers the rows of the acting workspace. |

### Migrations

The hub creates four shared tables: the workspaces, the members, the plugin allowlists,
and the enablements. Each states its reason under `OWN-004`, because the hub reads
them before it sets an acting workspace. `public.users` gains the administrator
flag.

An existing user record gets one workspace, with that user as its manager. A
setting moves to that workspace or stays with the user, as its plugin declares.
`public.idempotency_keys` stays scoped to a user.

Each plugin moves its tables from shared to scoped to a workspace in its own
feature. The existing rows of the plugin go to one workspace that the feature
names.

### Result

- Positive: one policy for every table of every plugin, and no sharing code in any plugin.
- Positive: a shared record has one copy, and a job fetches it one time.
- Positive: the access of a person is three words in every plugin.
- Negative: a person shares a whole workspace or nothing. Two gardens with two
  sets of people need two workspaces.
- Negative: every route of a plugin moves under `/workspaces/{workspaceId}`. Every
  shipped client and every plugin user interface changes its paths.
- Negative: `libs/pluginapi` changes, and every plugin rebuilds. `BLD-004`.
- Open: an MQTT message names a device and no workspace. A plugin that stores it
  needs a shared table from the device to the workspace. The feature that scopes
  that plugin decides.
- Work that follows: the engine spike and its ADR, the workspace feature, the
  feature for the administrator and the plugin allowlist, the feature for the permission
  capability, then one feature for each plugin that scopes its tables.
