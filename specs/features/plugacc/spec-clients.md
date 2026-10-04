---
id: PLUGACC
title: The administrator and the enablement in the command line, the bot, the scheduler, and the deck
type: spec
status: approved
created: 2026-10-02
updated: 2026-10-04
approved_by: Temuri
approved_on: 2026-10-04
constrained_by: [SEC, OWN, TG, JOB, CLI, UI, TS, SPC, LNG]
requirements: features/plugacc/requirements.md
---

# Specification: The administrator and the enablement in the command line, the bot, the scheduler, and the deck

`spec.md` holds the data model, the routes, and the checks of the hub. This file holds
the four clients that read them. `SPC-001` divides them.

## 1. The command line

| Command              | Flags                                                        | Does                                                      | Realizes         |
| -------------------- | ------------------------------------------------------------ | --------------------------------------------------------- | ---------------- |
| `maroid user invite` | `--first-name`, `--last-name`, `--user`, `--ttl`, `--admin`  | `--admin` marks the record as an administrator, for a new record and for one that `--user` names | `PLUGACC-FR-001` |

`InviteRequest` gains `Administrator bool`, and `auth.Service.Invite` writes the mark
in the transaction that writes the record or the invitation. `CLI-001` keeps its tree,
because the command exists.

## 2. The bot

| Path                                            | Action | Holds                                                        |
| ----------------------------------------------- | ------ | ------------------------------------------------------------ |
| `apps/hub/internal/telegram/command/wrapper.go` | change | `Wrapper.Handle` drops a command of a disabled plugin before the permission. `command.Checks` holds its checks |

**The drop.** `Wrapper.Handle` reads the acting workspace from the context and asks
`EnablementChecker.IsEnabled`. For a plugin that the workspace does not enable it
returns `nil`, writes one line at the level `info`, and sends nothing, which realizes
`PLUGACC-FR-019`. The check runs before the permission of `PERMS`, so a disabled plugin
answers nothing and never names a role.

The menu keeps every command, as `TG-004` gives it. Section 8 of `spec.md` postpones
the menu of each chat.

## 3. The scheduler

`CronScopePerWorkspace` of `IDENT-DD-007` reads its workspaces from
`EnablementService.WorkspacesEnabling`, which answers every workspace that enables the
plugin of the job, so a job runs for those and for no other, which realizes
`PLUGACC-FR-020`. This feature carries out step 15 of `features/ident/spec.md`.

| Path                                  | Action | Holds                                                     |
| ------------------------------------- | ------ | --------------------------------------------------------- |
| `libs/pluginapi/cron.go`              | change | `CronScopePerWorkspace`                                   |
| `apps/hub/internal/registry/cron.go`  | change | `RegisterOf` keeps the plugin of each job. `PluginOf` names it |
| `apps/hub/internal/worker/cron.go`    | change | `runForEachWorkspace`, from `WorkspacesEnabling`          |

A job of the hub names no plugin, so no workspace enables it. A job of the hub that
declares `CronScopePerWorkspace` fails the preparation of the worker with
`ErrCronScopeWithoutPlugin`, in place of a job that never runs.

## 4. The deck

| Path                                                                   | Action | Holds                                                        |
| ---------------------------------------------------------------------- | ------ | ------------------------------------------------------------ |
| `apps/deck/src/routes/(dashboard)/admin/+layout.svelte`                | create | Renders a page of `/admin` for an administrator alone        |
| `apps/deck/src/routes/(dashboard)/admin/users/+page.svelte`            | create | The users: create, invite, block, unblock, mark, and the link to an allowlist |
| `apps/deck/src/routes/(dashboard)/admin/users/[user]/+page.svelte`     | create | One user and their allowlist                                 |
| `apps/deck/src/routes/(dashboard)/admin/workspaces/+page.svelte`       | create | Every workspace, with links to its members and its plugins   |
| `apps/deck/src/routes/(dashboard)/w/[workspace]/plugins/+page.svelte`  | create | The plugins of the workspace, and the switch of each         |
| `apps/deck/src/lib/api/{users,enablements}.ts`                         | create | The routes of `spec.md` section 4.3                          |
| `apps/deck/src/lib/state/plugins.svelte.ts`                            | change | The entries of the plugins of the acting workspace, from `/workspaces/{id}/plugins` |
| `apps/deck/src/lib/navigation/breadcrumbs.ts`                          | change | Names a plugin and a workspace from the page that loaded them |
| `apps/deck/src/routes/(dashboard)/plugins/+page.svelte`                | change | Offers the settings of a plugin that the acting workspace enables alone |
| `libs/api-client/src/problem.ts`                                       | change | `PROBLEM_TYPE.administratorLast`                             |
| `apps/deck/src/lib/components/layout/Sidebar.svelte`                   | change | Names the plugins of the acting workspace alone, and links an administrator to the pages of `/admin` |

The pages under `/admin` render for an administrator alone. `GET /auth/sessions/self`
already answers the person of the session, and gains `is_administrator` for that test.
The hub still guards each route, so a hidden link is a convenience.

The page of the plugins of a workspace lists the catalog of `GET /plugins` and every
plugin that the workspace enables, and marks each enabled one. A manager switches a
plugin of their allowlist on or off, and an administrator switches any plugin, which
realizes `PLUGACC-FR-024`. A manager switches off an enabled plugin that their
allowlist does not hold, and cannot switch it on again.

The sidebar, the breadcrumbs, and the pages of a plugin read the entry of each plugin
from the list of the enablements of the acting workspace, as `PLUGACC-DD-009` gives,
not from `GET /plugins`. A member reaches every enabled plugin whatever their allowlist
holds, which realizes `PLUGACC-FR-017` and `PLUGACC-FR-025`.

An administrator in a workspace that they are not a member of sees no page of a plugin
in the sidebar, and the pages of the workspace say that they manage it as an
administrator, which realizes `PLUGACC-FR-033`. The deck reads the membership from the
list of `GET /workspaces`, and the name of the workspace from the read of
`GET /workspaces/{id}`, so the header and the breadcrumbs name it.

The form of a new user lists every loaded plugin, and the administrator picks the
allowlist there. The picker is disabled while the form marks an administrator, and the
page of an administrator shows the allowlist disabled, with a note that an
administrator reaches every plugin, which realizes `PLUGACC-FR-035`. The form sends `allowed_plugins` with the record, which realizes
`PLUGACC-FR-032`. The page of one user changes the names, the mark of an administrator, the block, and
the allowlist, which realizes `PLUGACC-FR-034`, `PLUGACC-FR-029`, `PLUGACC-FR-030`,
`PLUGACC-FR-005`, `PLUGACC-FR-006`, and `PLUGACC-FR-008`.

An administrator reaches every action of `PLUGACC-FR-023` from the pages of `/admin`,
and the members and the plugins of any workspace through the pages of `/w/{id}`, which
admit an administrator for those two pages as `PLUGACC-DD-003` gives. The address of
an invitation shows one time, with a control that copies it.

## Retired identifiers

| ID               | Retired    | Reason |
| ---------------- | ---------- | ------ |
| PLUGACC-DD-007   | 2026-10-04 | The owner postponed the menu of each chat (`PLUGACC-FR-031`, `PLUGACC-NFR-002`). Its later design takes new identifiers. |
| PLUGACC-DD-008   | 2026-10-04 | The owner postponed the menu of each chat (`PLUGACC-FR-031`, `PLUGACC-NFR-002`). Its later design takes new identifiers. |
