---
id: NOTIF
title: The notification providers of a workspace and the routes of a notification
type: requirements
status: approved
created: 2026-10-11
updated: 2026-10-11
approved_by: Temuri
approved_on: 2026-10-11
constrained_by: [NTF, PLG, CFG, OWN, SEC, ARC, EXT, LOG, JOB, PRC]
---

# Requirements: The notification providers of a workspace and the routes of a notification

## 1. Problem

The configuration file of the installation holds every place that a notification goes.
One Telegram bot token, one chat, and one forum thread serve every workspace. A second
household that enables the garden plugin gets its reminder in the chat of the first
household. `GARDEN` question 2 accepted this until a rework, and `CFG-007` puts a value
of one workspace in a scoped record.

A plugin names a channel of the installation. A workspace cannot change where a
notification goes, cannot stop one notification, and cannot hold its own bot.

The bot token sits in the configuration file in plain text. `PSET` gave every other
credential of a workspace a protection. The bot token has none.

## 2. Users

- A manager: adds the bot of the household one time, and chooses the chat and the
  thread of each notification topic.
- A member: sees which notification topics reach the household, and where.
- A plugin author: declares the notification topics of the plugin, and writes no
  routing and no delivery code.

## 3. Out of scope

- A notification provider or a notification route that a user owns. A run with no
  acting workspace delivers nothing.
- A notification provider kind other than Telegram.
- A notification provider kind that a plugin adds.
- A retry after a failed delivery, and a queue that survives a restart.
- A history of the deliveries, and the result of the last delivery.
- An import of the configuration file of today.
- The use of the bot of the hub as a notification provider.
- A limit on the rate of the deliveries.
- A choice of the format of a message for each notification provider kind.

## 4. Definitions

| Term                            | Meaning                                                                                                  |
| ------------------------------- | -------------------------------------------------------------------------------------------------------- |
| notification topic              | One kind of notification that a plugin declares and sends, with an identifier, a name, and a description. |
| notification provider           | An account at an external messaging service that a workspace stores. Maroid delivers a notification through it. A Telegram bot is one. Never write "provider" alone. |
| notification provider kind      | The external messaging service of a notification provider. It fixes the fields of the notification provider and of a destination. Telegram is the only one. |
| destination                     | The place at the external messaging service that receives a notification. For Telegram: a chat, and a thread of a forum when the chat has threads. |
| notification route              | The rule of a workspace that sends one notification topic through one notification provider to one destination. |
| unrouted                        | The state of a notification topic in a workspace with no enabled notification route.                      |

## 5. Functional requirements

### Notification topics

### `NOTIF-FR-001`

A plugin must declare each notification topic that it sends, with an identifier, a name,
and an optional description.

**Why:** The manager routes a notification topic before the plugin sends the first one.

### `NOTIF-FR-002`

Maroid must refuse to load a plugin that declares two notification topics with one identifier.

### `NOTIF-FR-003`

A plugin must name a notification topic when it sends a notification. It names no
notification provider and no destination.

**Why:** The manager changes where a notification goes. The plugin does not change.

### Notification providers

### `NOTIF-FR-004`

A member must list the notification providers of the acting workspace, with the name, the
notification provider kind, the enabled state, and each value that is not a secret.

### `NOTIF-FR-005`

Maroid must tell a member whether each secret field of a notification provider holds a value.

### `NOTIF-FR-006`

A manager must add a notification provider with a name, a notification provider kind, and
a value for each field that the kind declares.

**Examples:**

- Normal case: the manager adds "Family bot" of the kind Telegram with its bot token.
- Limit case: the name holds 64 characters after Maroid trims the white space at each end.
  Maroid stores it.
- Unwanted case: the name holds 65 characters. Maroid stores nothing.

### `NOTIF-FR-007`

A manager must change the name and the field values of a notification provider.

### `NOTIF-FR-008`

Maroid must keep a stored secret of a notification provider when the save names no value
for that field.

**Why:** The manager changes the name, and the deck holds no secret to send back.

### `NOTIF-FR-009`

A manager must remove a notification provider. Maroid removes each notification route of it.

**Examples:**

- Normal case: "Family bot" carries two notification routes. The manager removes it. The
  two notification topics become unrouted.
- Unwanted case: the notification provider belongs to another workspace. Maroid answers
  that it does not exist.

### `NOTIF-FR-010`

A manager must disable a notification provider, and enable it again. Maroid keeps every
value and every notification route of it.

### `NOTIF-FR-011`

Maroid must reject a save of a notification provider or of a notification route when a
value does not match the fields that the notification provider kind declares.

**Examples:**

- Unwanted case: the save leaves the bot token empty. Maroid stores nothing.
- Unwanted case: the save names a field that the kind does not declare. Maroid stores nothing.

### `NOTIF-FR-012`

Maroid must name each field that caused it to reject a save.

### `NOTIF-FR-013`

Maroid must reject a Telegram notification provider whose bot token Telegram refuses.

**Why:** A wrong token becomes a reminder that never arrives, and nobody sees the cause.

**Examples:**

- Normal case: Telegram accepts the token. Maroid stores the notification provider.
- Unwanted case: Telegram refuses the token. Maroid stores nothing, and says that Telegram
  refused it.
- Unwanted case: Telegram does not answer. Maroid stores nothing, and says that it could
  not reach Telegram.

### Notification routes

### `NOTIF-FR-014`

A member must list each notification topic of each plugin that the acting workspace
enables, with the notification provider and the destination of each notification route.

**Why:** A member checks where a reminder goes. A destination holds no secret.

### `NOTIF-FR-015`

Maroid must mark each unrouted notification topic in that list.

### `NOTIF-FR-016`

A manager must add a notification route with a notification topic, a notification provider
of the acting workspace, and a value for each destination field that the kind declares.

**Examples:**

- Normal case: the care reminder goes to "Family bot", the chat of the household, thread 2749.
- Normal case: a second notification route sends the care reminder to a second chat.
- Unwanted case: the notification topic belongs to a plugin that the workspace does not
  enable. Maroid stores nothing.

### `NOTIF-FR-017`

Maroid must reject a notification route that repeats the notification topic, the
notification provider, and the destination of another notification route.

### `NOTIF-FR-018`

A manager must change the destination of a notification route.

### `NOTIF-FR-019`

A manager must remove a notification route.

### `NOTIF-FR-020`

A manager must disable a notification route, and enable it again. Maroid keeps its values.

### `NOTIF-FR-021`

A manager must send a test message through a notification route, and see whether the
external messaging service accepted it.

**Examples:**

- Normal case: Telegram accepts the message. The deck says that it was sent.
- Unwanted case: the bot is not a member of the chat. The deck shows the refusal of Telegram.

### `NOTIF-FR-022`

Maroid must keep the notification routes of a plugin that the workspace disables, and use
them again when the workspace enables the plugin.

### `NOTIF-FR-023`

Maroid must not list or use a notification route whose notification topic the plugin no
longer declares.

### Delivery

### `NOTIF-FR-024`

Maroid must deliver a notification through each enabled notification route of its
notification topic in the acting workspace, when the notification provider of that route
is enabled.

### `NOTIF-FR-025`

Maroid must try each remaining notification route when one notification route fails.

### `NOTIF-FR-026`

Maroid must report a failure to the plugin only when every notification route that it tried failed.

**Examples:**

- Normal case: one notification route of two fails. The plugin gets no failure.
- Unwanted case: both fail. The plugin gets a failure.

### `NOTIF-FR-027`

Maroid must report no failure to the plugin, and deliver nothing, when the notification
topic is unrouted in the acting workspace.

**Why:** Each workspace that enables a plugin runs its job, and few of them route every
notification topic. `PSET-FR-014` treats an absent settings record the same way.

### `NOTIF-FR-028`

Maroid must report no failure to the plugin, and deliver nothing, when the unit of work has
no acting workspace.

### `NOTIF-FR-029`

Maroid must report a failure to the plugin when the plugin sends a notification topic that
it does not declare.

**Why:** That is a fault in the plugin. An unrouted notification topic is a choice of the manager.

### `NOTIF-FR-030`

Maroid must log each notification that it delivers to nobody, with the workspace, the
plugin, the notification topic, and the reason.

### `NOTIF-FR-031`

Maroid must log each failed notification route, with the workspace, the notification
topic, the notification route, and the answer of the external messaging service.

### `NOTIF-FR-032`

Maroid must deliver the title, the body, and every attachment of a notification through a
Telegram notification provider.

**Examples:**

- Normal case: one image. Telegram shows a photo with the title and the body as its caption.
- Normal case: no attachment. Telegram shows a text message.
- Limit case: two images and one PDF. Telegram shows the text, the two photos, and the document.

### Secrets

### `NOTIF-FR-033`

Maroid must store each secret of a notification provider with the protection of its
workspace, as `PSET-FR-015` and `PSET-FR-016` give for a setting.

### `NOTIF-FR-034`

Maroid must return no secret of a notification provider to any reader.

### `NOTIF-FR-035`

Maroid must write no secret of a notification provider to a log.

## 6. Non-functional requirements

### `NOTIF-NFR-001`

Maroid must apply a change to a notification provider or a notification route to each
notification that a plugin sends more than 1 second after the response of the save.

**Why:** A manager who corrects a wrong chat must see the next reminder arrive there.

### `NOTIF-NFR-002`

Maroid must find the notification routes of one notification in no more than 50
milliseconds at the 95th percentile. Measure from the call of the plugin to the first call
to the external messaging service.

## 7. Invariants

### `NOTIF-INV-001`

A notification route names a notification provider of its own workspace.

### `NOTIF-INV-002`

The notification providers and the notification routes of one workspace reach no member of
another workspace, and no notification of another workspace.

### `NOTIF-INV-003`

The database holds a secret of a notification provider in a protected form only.

### `NOTIF-INV-004`

The configuration of the installation holds no notification provider and no notification route.

## 8. Constraints from the guidelines

| Rule      | Guideline         | Effect on this feature                                                              |
| --------- | ----------------- | ----------------------------------------------------------------------------------- |
| `NTF-001` to `NTF-006` | Notifications | `ADR-0013` changes them. The channel and the transport of the configuration end. |
| `PLG-006` | Plugin model      | The notification topics of a plugin are a capability: an interface, a registry, a registrar. |
| `PLG-007` | Plugin model      | A plugin sends a notification through the host.                                     |
| `PLG-011` | Plugin model      | `NOTIF-FR-002`. A registry rejects a second notification topic with one identifier. |
| `ARC-008` | Architecture      | No rule names a plugin or a notification topic of a plugin.                         |
| `CFG-007` | Configuration     | `NOTIF-INV-004`. The routing of a workspace leaves the configuration file.          |
| `OWN-004` | Record ownership  | A notification provider and a notification route belong to one workspace.          |
| `OWN-006` | Record ownership  | The isolation runs in the database and fails closed.                                |
| `SEC-012` | Security          | `NOTIF-FR-014` and `NOTIF-FR-016` read the plugins that the workspace enables.      |
| `SEC-013` | Security          | A manager holds each change. A viewer holds each list.                              |
| `EXT-005` | External services | `NOTIF-FR-013`. A status that is not a success from Telegram is a refusal.          |
| `LOG-005` | Logging           | `NOTIF-FR-035`. An error attribute carries no secret.                               |
| `JOB-007` | Jobs              | `NOTIF-FR-027`. An unrouted notification topic does not fail the run.               |

## 9. Open questions

| #   | Question                                                                                                                   | Owner  | Answer                                                         |
| --- | -------------------------------------------------------------------------------------------------------------------------- | ------ | -------------------------------------------------------------- |
| 1   | `NOTIF-FR-013` refuses the save when Telegram does not answer. Do you accept that a Telegram outage blocks the save?       | Temuri | Yes.                                                           |
| 2   | `NOTIF-FR-026` lets a job pass when one notification route of two fails. Is a log line enough for the failed one?          | Temuri | Yes.                                                           |
| 3   | Does a viewer see the destination of a notification route (the chat identifier), or the notification provider name only? | Temuri | A viewer sees the destination. `NOTIF-FR-014`.                 |

## Retired identifiers

This file has no retired identifier.
