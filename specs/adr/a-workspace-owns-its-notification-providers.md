---
id: ADR-0013
title: A workspace owns its notification providers
type: adr
status: accepted
created: 2026-10-11
updated: 2026-10-11
decided: 2026-10-11
changes:
  [
    NTF-001,
    NTF-002,
    NTF-004,
    NTF-005,
    NTF-006,
    NTF-007,
    NTF-008,
    SPC-003,
    LOG-003,
    GIT-004,
    GLO-notification,
    GLO-notification-topic,
    GLO-notification-provider,
    GLO-notification-provider-kind,
    GLO-destination,
    GLO-notification-route,
    GLO-unrouted
  ]
supersedes:
superseded_by:
---

# A workspace owns its notification providers

## Context

`NTF` builds one dispatcher for the process from the configuration file. A URL defines
a transport (`NTF-005`), a channel lists transports with a fallback (`NTF-004`), and a
plugin names a channel (`NTF-002`). `ADR-0008` made the workspace the tenant, and
`CFG-007` moves a value of one workspace to a scoped record. The channel of today is a
value of the installation, so every workspace shares one bot and one chat. `GARDEN`
question 2 deferred this to a rework. `NOTIF` is that rework.

The facts of the code today:

| Fact                                                                                                     | Consequence                                                       |
| -------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------- |
| The configuration holds three transports. Each repeats one bot token with another thread.                | A credential and a destination are two things with two lifetimes. |
| A channel name is a string that a plugin hardcodes or reads from its configuration.                      | The hub cannot list what a plugin sends.                          |
| `libs/notifierapi` holds two interfaces and two types. Only `libs/pluginapi` and the plugins import it.  | A second module adds a version to bump and no isolation.          |
| `libs/notifier` holds the dispatcher, the registry, and the Telegram transport. Only the hub imports it. | `PLG-007` already keeps a plugin away from it.                    |
| `PSET` gives a JSON Schema from a Go struct, a form in the deck, and a transit key for each workspace.   | A notification provider kind reuses all three.                    |

## Decision

A workspace stores its notification providers and its notification routes in scoped
records, and the hub delivers a notification through each enabled notification route of
its notification topic. A plugin declares its notification topics as a capability. The
types of `libs/notifierapi` move into `libs/pluginapi`, the code of `libs/notifier` moves
into `apps/hub/internal/notification`, and both modules end.

## Rationale

A notification provider holds the credential, and a notification route holds the
destination. One bot token then lives in one record, and each notification topic picks its
own chat and thread. Today three transports repeat one token for this reason.

A declared notification topic lets the deck list what a manager routes, and lets the hub
tell a fault of the plugin (an undeclared identifier) from a choice of the manager (an
unrouted notification topic).

Every notification route of a notification topic receives the notification. The primary
and fallback lists of `NTF-004` existed because one channel served everyone. A manager
who wants a second place adds a second notification route.

A notification provider kind declares two settings schemas: one for the notification
provider, one for the destination. The hub then validates, renders, and protects both with
the code of `PSET`, and a new kind adds no route and no page.

The kinds stay in the hub. A kind that a plugin adds would need a capability, and a plugin
would then receive a credential of a workspace that it did not declare.

The two modules end because the hub is the only consumer of `libs/notifier`, and
`libs/pluginapi` is the only consumer of `libs/notifierapi` besides the plugins.
`BLD-004` already rebuilds every plugin when `libs/pluginapi` changes.

## Alternatives

| Alternative                                                         | Why we did not select it                                                                   |
| ------------------------------------------------------------------- | ------------------------------------------------------------------------------------------ |
| Keep the channels in the configuration, one set for each workspace. | A restart for each change, and the bot token stays in plain text.                          |
| A notification provider that holds the destination too.             | The token repeats in each record, as it repeats in the configuration today.                |
| A named destination record between a provider and a route.          | One more page and one more table for a reuse that one household does not need.             |
| Keep the primary and fallback lists on a notification route.        | A fallback only matters when one route serves everyone. Two routes cover the case.         |
| A durable queue with a retry.                                       | Each attachment then lives in the database. No job sends more than one notification a day. |
| A notification provider or route that a user owns.                  | Doubles the records, the policies, and the pages. No plugin sends to one person today.     |
| The bot of the hub as the default notification provider.            | Every workspace then sends through the bot of the instance.                                |
| Keep `libs/notifier` as a library.                                  | One consumer. The scheme registry and the URL parser have no use after this decision.      |

## Consequences

### Rules that change

`NTF-001` becomes:

> A plugin declares each notification topic that it sends, with an identifier, a name,
> and an optional description. The declaration is a capability, as `PLG-006` gives.
> A plugin sends a notification through `Host.Notifier()`, which gives a `pluginapi.Notifier`.

`NTF-002` becomes:

> A plugin names a notification topic, never a notification provider or a destination:
> `Notify(ctx, topicID, msg)`. The hub returns an error for a notification topic that the
> plugin does not declare.
>
> **Why:** The manager changes where a notification goes in the deck. The plugin does not change.

`NTF-003` keeps its text. `Message` and `Attachment` move to `libs/pluginapi`.

`NTF-004` becomes:

> The hub delivers a notification through each enabled notification route of its
> notification topic in the acting workspace, when the notification provider is enabled.
> A failure of one notification route does not stop the others. The call returns an
> error only when every notification route that the hub tried failed.
>
> The hub logs and drops a notification of an unrouted notification topic, or of a unit
> of work with no acting workspace. The call returns no error.

`NTF-005` is retired. A URL defines nothing. `NTF-007` replaces it.

`NTF-006` becomes:

> A new notification provider kind registers in the kind registry of the hub.
> The registry rejects an identifier that it already holds, and rejects a nil kind.
> A plugin adds no notification provider kind.

`NTF-007` is new:

> A notification provider kind declares two settings schemas: the fields of a
> notification provider and the fields of a destination. The field kinds are those of
> `PSET-FR-002`. The kind validates a credential against its external service before
> the hub stores it.

`NTF-008` is new:

> A notification provider and a notification route are scoped records of a workspace.
> A secret field of a notification provider takes the protection of its workspace.
> The configuration holds none of them. See `CFG-007`.

The frontmatter of `notifications.md` changes `scope` to
`[apps/hub/internal/notification/, libs/pluginapi/]` and `related` to `[PLG, PSET, OWN, CFG]`.

`SPC-003` changes the row "Notification channel" to:

> | Notification topic | Identifier, Name, Sent when, Realizes | `NTF-001`, `NTF-002` |

`LOG-003` replaces the component `notifier-dispatcher` with `notification`.

`GIT-004` changes the example of the `<lib>` row from `notifier` to `rest`.

### Glossary

`GLO-notification` becomes: "A message that a plugin sends to a workspace through the
notification routes of one notification topic."

The terms of section 4 of `features/notif/requirements.md` join the glossary:
`GLO-notification-topic`, `GLO-notification-provider`, `GLO-notification-provider-kind`,
`GLO-destination`, `GLO-notification-route`, and `GLO-unrouted`.

### Results

- Positive: each workspace holds its own bot, chat, and thread, and changes them with no restart.
- Positive: the bot token gets the protection of `PSET`.
- Positive: the hub lists every notification topic, so a manager sees what is unrouted.
- Positive: two Go modules end.
- Negative: a notification of every workspace stops at the deploy until a manager routes it again.
- Negative: a failed notification route leaves a log line only. No retry follows.
- Negative: a plugin outside this repository breaks at the next build, because `libs/notifierapi` ends.

### Work that follows

- `NOTIF` specification: the tables, the policies, the routes, the deck page, the capability.
- Remove the `notifier` section from `config.yaml`, the chart, and the compose files.
- Remove the notification keys from the configuration of each plugin.
- Rebuild every plugin, as `BLD-004` gives.
- Answer `GARDEN` question 2 again: the reminder now names a notification topic.
- Examine every specification that cites `NTF-002`, `NTF-004`, or `NTF-005`.
