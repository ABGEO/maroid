---
id: TG
title: Telegram
type: guideline
status: active
created: 2026-09-11
updated: 2026-09-24
scope: [apps/hub/internal/telegram/, libs/pluginapi/telegram/]
related: [PLG, SEC, API, OWN]
---

# Telegram

The library is telego.
The hub resolves the acting user before it dispatches an update. See `OWN-003`.

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

## TG-007

A plugin sends a message through `pluginapi.TelegramBot`.
The interface gives `SendMessage`, `AnswerCallbackQuery`, and `EditMessageText`.
A plugin does not use the telego client directly.

**Why:** The narrow interface keeps the plugin free of the bot lifecycle.

## Retired identifiers

This file has no retired identifier.
