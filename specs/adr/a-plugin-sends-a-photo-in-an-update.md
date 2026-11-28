---
id: ADR-0029
title: A plugin sends a photo in an update
type: adr
status: accepted
created: 2026-11-28
updated: 2026-11-28
decided: 2026-11-28
changes: []
supersedes:
superseded_by:
---

# A plugin sends a photo in an update

## Context

`TG-007` gives a plugin `pluginapi.TelegramBot` to answer within an update. The
interface holds `SendMessage`, `AnswerCallbackQuery`, and `EditMessageText`. It sends
text only. `PARKMAP-FR-018` demands an image in a step of a conversation: the map of
the parking lots near the member.

A change to `libs/pluginapi` changes the build identifier of the package. Every plugin
then builds again (`BLD-002`, `BLD-004`). A test fake that asserts the interface fails
to compile until it gains the new method.

## Decision

`pluginapi.TelegramBot` gains `SendPhoto(ctx context.Context, params *telego.SendPhotoParams) (*telego.Message, error)`.
`pluginapi.APIVersion` stays `v1`.

## Rationale

The method mirrors `SendMessage`. The plugin sets the chat, the thread, and the topic
of the update as it does for a text, so `TG-007` holds as it stands. The hub wraps
the telego call and adds no rule, so `ARC-008` holds.

`APIVersion` stayed `v1` through each earlier change of the host interface. The Go
loader refuses a plugin that another build of `libs/pluginapi` built, and names the
package. That refusal comes before the check of `PLG-004`.

## Alternatives

| Alternative                                                   | Why we did not select it                                                                                        |
| ------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------- |
| A second interface, `TelegramPhotoSender`, that a host offers | A second getter on `Host` for one method. Every plugin still builds again, because `Host` changes.              |
| A photo in a plugin message (`message.Messenger`)             | `TGMSG` keeps a photo out of scope. A plugin message has no step of a conversation, and the map needs the step. |
| A raise of `APIVersion` to `v2`                               | Every plugin changes its metadata for one method. The Go loader already refuses the old build.                  |

## Consequences

- Positive: a plugin sends an image in any command or step, with the thread of the update.
- Negative: every plugin builds again. Each test fake of `pluginapi.TelegramBot` gains `SendPhoto`.
- Work that follows: `PARKMAP` build plan, step 1.
