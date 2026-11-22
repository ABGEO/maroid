---
id: TG
title: Telegram
type: guideline
status: active
created: 2026-09-11
updated: 2026-11-23
scope: [apps/hub/internal/telegram/, libs/pluginapi/telegram/]
related: [PLG, SEC, API, OWN]
---

# Telegram

The library is telego.
The hub resolves the acting user and the acting workspace before it dispatches an update. See `OWN-003`.

## TG-001

The bot receives an update through a webhook. The bot does not poll.
The default path is `/telegram/webhook`. `SEC-007` and `SEC-010` protect it, and
`X-Telegram-Bot-Api-Secret-Token` is the header that `SEC-010` reads.

## TG-002

The hub registers the webhook with Telegram at the start when `telegram.setup` is true.

## TG-003

A plugin declares a command as `pluginapi.TelegramCommand`.
The interface has `Meta`, `Validate`, and `Handle`.
`Validate` decides whether the update belongs to the command. `Handle` runs it.

## TG-004

Command metadata carries a scope. The hub groups the commands by scope and
registers each group with Telegram.

## TG-005

A conversation is a state machine.
`Conversation` gives an identifier, an entry step, and the steps.
`Step` gives `OnEnter` and `OnMessage`. `OnMessage` returns the identifier of the next step.

## TG-006

Conversation state lives in a `Store` and the key is the Telegram user identifier.

The store holds the state in memory today. A restart of the hub loses every open
conversation. A conversation must not carry a state that the user cannot repeat.

A reply claim (`GLO-reply-claim`) lives in the database, so a restart of the hub
keeps it. The store of a conversation stays in memory. `ADR-0025`.

## TG-007

A plugin sends a message through the host. `pluginapi.TelegramBot` answers within
an update: a command or a step of a conversation, with the chat of that update.
`message.Messenger` sends a plugin message (`GLO-plugin-message`) to the
acting user, and the plugin never names a chat. A plugin does not use the telego
client directly. `ADR-0025`.

**Why:** The narrow interface keeps the plugin free of the bot lifecycle. The hub
owns the chat of a person.

## TG-008

The hub routes one update to one receiver, in this order:

1. A command goes to its command.
2. A press on a button of a plugin message goes to the plugin that sent it.
3. A reply to a plugin message goes to the plugin that holds an open claim on it.
4. A plain message goes to the open conversation of the sender.
5. A plain message with no conversation goes to the oldest open claim of the sender.
6. Any other update goes to the handlers of the hub, as before.

A press and a reply reach the plugin with the acting user and no acting workspace.
`ADR-0025`.

## Retired identifiers

This file has no retired identifier.
