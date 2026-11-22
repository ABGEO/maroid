---
id: ADR-0025
title: A plugin messages the acting user on Telegram
type: adr
status: accepted
created: 2026-11-23
updated: 2026-11-23
decided: 2026-11-23
changes: [TG-006, TG-007, TG-008, OWN-003, GLO-plugin-message, GLO-reply-claim]
supersedes:
superseded_by:
---

# A plugin messages the acting user on Telegram

## Context

A plugin reaches a person on Telegram in three ways today:

- `pluginapi.TelegramBot` (`TG-007`) sends to a chat identifier. A command and a
  step of a conversation take the identifier from the update that they answer.
- A conversation (`TG-005`) keeps its state in memory (`TG-006`). A restart loses it,
  and one person holds one conversation at a time.
- A notification (`NTF-002`) goes to the routes of a workspace. A manager chooses the
  destination, and nothing comes back.

A plugin that asks a person a question at 09:00 of their day, and waits hours for the
answer, fits none of them:

- A job holds no update, so it holds no chat identifier. The hub keeps the chat of a
  person, and no plugin can read it.
- A press on a button of a message that a job sent reaches no plugin. The hub routes
  a callback query to a conversation or to the workspace selection alone.
- A typed answer that arrives hours later reaches the conversation engine, which
  holds no state for it, or a restart lost the state.

`TGMSG-FR-001` to `TGMSG-FR-021` state the need. Pulse is the first plugin that
needs it.

## Decision

The host gives each plugin a `message.Messenger` that sends a message to the acting
user of the context and to nobody else. The hub resolves the chat from the Telegram
identity of that user. The hub records each message that a plugin sends, and routes
a press on its buttons, and a reply that a plugin claimed, back to that plugin. The
plugin receives them with the acting user, and with no acting workspace.

A reply claim lives in the database, so a restart keeps it. A conversation keeps its
state in memory, as today.

## Rationale

The hub owns the chat of a person, so the hub can confine a plugin to the acting
user. A plugin that never holds a chat identifier cannot write to the wrong person.

The record of a sent message names its owner. A press then reaches its owner with no
prefix in the callback data, so a plugin keeps the 64 bytes that Telegram allows,
and the callback data of a plugin message never collides with the data of the hub.
Telegram does not prove that the callback data came from the button, so the owner
comes from the record, not from the data.

No workspace keeps the change inside the rules that exist. A press in a workspace
would need the enablement of `SEC-012` and the permission of `SEC-013` at the press.
The first plugin acts for a person alone. A later decision can add a workspace.

A claim names one message. The routing order keeps every path of today: a command
first, then a conversation, then a claim. A plugin that holds a claim takes no
message that another flow would take.

## Alternatives

| Alternative                                            | Why we did not select it                                                                                                      |
| ------------------------------------------------------ | ----------------------------------------------------------------------------------------------------------------------------- |
| `TelegramBot` reads the chat of a user identifier      | A plugin then writes to any person whose identifier it holds. `OWN-005` keeps a plugin inside the acting user.                |
| A prefix of the plugin in the callback data            | The prefix spends bytes of the 64, and a forged press names any plugin. The record names the owner, and the policy hides it.  |
| A durable conversation instead of a claim              | A conversation owns the chat of a person until it ends. A question that waits for hours would block every other conversation. |
| Each press and reply runs in the workspace of the send | It needs `SEC-012` and `SEC-013` at the press. No plugin needs it yet.                                                        |
| The plugin receives every update of the person         | A plugin would read the messages of another plugin and of the hub.                                                            |

## Consequences

`TG-006` gains a paragraph:

> A reply claim (`GLO-reply-claim`) lives in the database, so a restart of the hub
> keeps it. The store of a conversation stays in memory. `ADR-0025`.

`TG-007` becomes:

> A plugin sends a message through the host. `pluginapi.TelegramBot` answers within
> an update: a command or a step of a conversation, with the chat of that update.
> `message.Messenger` sends a plugin message (`GLO-plugin-message`) to the
> acting user, and the plugin never names a chat. A plugin does not use the telego
> client directly. `ADR-0025`.
>
> **Why:** The narrow interface keeps the plugin free of the bot lifecycle. The hub
> owns the chat of a person.

`TG-008` is new:

> The hub routes one update to one receiver, in this order:
>
> 1. A command goes to its command.
> 2. A press on a button of a plugin message goes to the plugin that sent it.
> 3. A reply to a plugin message goes to the plugin that holds an open claim on it.
> 4. A plain message goes to the open conversation of the sender.
> 5. A plain message with no conversation goes to the oldest open claim of the sender.
> 6. Any other update goes to the handlers of the hub, as before.
>
> A press and a reply reach the plugin with the acting user and no acting workspace.
> `ADR-0025`.

`OWN-003` changes the acting workspace of the row "A Telegram update" to:

> The workspace that the chat selected. A press on a plugin message and a reply to a
> claim carry none. See `TG-008`.

The glossary gains:

| ID                   | Term           | Definition                                                                                                |
| -------------------- | -------------- | --------------------------------------------------------------------------------------------------------- |
| `GLO-plugin-message` | plugin message | A Telegram message that a plugin sends to its acting user through the host. A press on it works 30 days.  |
| `GLO-reply-claim`    | reply claim    | The mark of a plugin on one of its plugin messages that routes the next typed answer of the person to it. |

- Positive: a job of a plugin starts an exchange with one person, and the answer comes
  back after a restart.
- Positive: no plugin holds a chat identifier for a message that it starts.
- Negative: the hub writes one row for each plugin message, and reads one row for each
  press and each plain message of a person with an open claim.
- Negative: the contract between the hub and a plugin grows, so every plugin builds
  again (`BLD-004`).
- Negative: a person with no Telegram identity, or who never opened the bot, receives
  no plugin message. The plugin learns it and decides what to do.
- Work that follows:
  - `TGMSG` requirements and specification.
  - A core migration adds the table of the plugin messages.
  - `PCAP` reports the new capability, and the plugin page names it.
  - `pulse` builds on it.
