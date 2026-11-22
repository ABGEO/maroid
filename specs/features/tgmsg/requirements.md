---
id: TGMSG
title: The Telegram messages of a plugin
type: requirements
status: approved
created: 2026-11-23
updated: 2026-11-23
approved_by: Temuri
approved_on: 2026-11-23
constrained_by: [TG, OWN, PLG, SEC, ARC, DAT, BLD]
---

# Requirements: The Telegram messages of a plugin

## 1. Problem

A plugin answers a person on Telegram only inside an update: a command, or a step of
a conversation. A plugin cannot start an exchange with one person. A job that wants
to ask "How did you sleep?" at 09:00 of the day of a person has no chat to write to.
A press on a button of such a message reaches no plugin. A typed answer that comes
hours later reaches no plugin either, and a restart loses any state that waited for
it. `ADR-0025` gives the decision.

## 2. Users

- A person reads a plugin message, presses its buttons, and types or shares an
  answer.
- A plugin author sends a plugin message from a job, a request, or an update, and
  receives the press or the answer later.

## 3. Out of scope

- A message to a group, or to a person other than the acting user.
- A press or a reply that acts in a workspace.
- A durable conversation. `TG-006` keeps it in memory.
- A photo, a file, or formatted text in a plugin message.
- The deletion of a plugin message.

## 4. Definitions

| Term           | Meaning                                                                                            |
| -------------- | -------------------------------------------------------------------------------------------------- |
| plugin message | `GLO-plugin-message`, from `ADR-0025`.                                                             |
| reply claim    | `GLO-reply-claim`, from `ADR-0025`. A claim is open from the send until its expiry or its release. |
| button         | An inline button under a plugin message. It carries a label and a value of the plugin.             |
| reply keyboard | Buttons that replace the keyboard of the person. One of them can share the location of the person. |
| plain message  | A text or a location that the person sends, which is no command and no reply to a message.         |

## 5. Functional requirements

### `TGMSG-FR-001`

A plugin must send a plugin message to the acting user of the unit of work.

**Why:** A job for each user runs for one person. That person is the one to ask.

**Examples:**

- Normal case: the run of a job for Nina sends "Good morning". Nina's chat shows it.
- Limit case: a request of Beka sends a message. Beka's chat shows it, not Nina's.

### `TGMSG-FR-002`

Maroid must refuse a send in a unit of work that has no acting user.

**Examples:**

- Unwanted case: a job for each workspace sends a message. The plugin learns that no
  person acts, and no chat shows a message.

### `TGMSG-FR-003`

Maroid must tell the plugin, apart from a failure, that the acting user holds no
Telegram account.

**Why:** A plugin skips that person and writes no error.

### `TGMSG-FR-004`

Maroid must tell the plugin, apart from a failure, that Telegram refuses to deliver to
the person.

**Why:** A person who never opened the bot, or who blocked it, receives nothing. The
plugin decides whether to try again later.

### `TGMSG-FR-005`

A plugin message must carry a text and, under it, rows of buttons.

**Examples:**

- Normal case: "Mood?" with five buttons, 1 to 5, and a sixth, Skip.
- Limit case: a text with no button.

### `TGMSG-FR-006`

A plugin message must carry, instead of buttons, a reply keyboard. One of its buttons
can ask for the location of the person.

**Why:** Telegram asks for a location only from a reply keyboard.

### `TGMSG-FR-007`

A plugin message must remove the reply keyboard that an earlier plugin message
showed.

### `TGMSG-FR-008`

Maroid must refuse a plugin message with no text, a button value longer than 64
bytes, or both buttons and a reply keyboard.

**Why:** Telegram refuses each of them. The plugin learns it before the send.

### `TGMSG-FR-009`

A plugin must change the text and the buttons of a plugin message that it sent, for
30 days after the send.

**Examples:**

- Normal case: after a press on 4, the plugin changes "Mood?" to "Mood: 4" with no
  button.
- Unwanted case: a change on day 31 is refused.

### `TGMSG-FR-010`

A press on a button of a plugin message must reach the plugin that sent it, with the
value of the button and the message.

### `TGMSG-FR-011`

Maroid must answer every press, so the client of the person stops waiting. The answer
shows the short notice that the plugin gives, when it gives one.

**Examples:**

- Normal case: the plugin returns "Saved". The person sees "Saved".
- Unwanted case: the plugin fails. The person sees "Something went wrong. Try again."

### `TGMSG-FR-012`

A press that reaches no receiver must show "This button no longer works."

**Examples:**

- Limit case: a press on a plugin message of day 31.
- Unwanted case: a press on a message of a plugin that the hub no longer loads.

### `TGMSG-FR-013`

A plugin must claim the next reply when it sends a plugin message, with an expiry no
later than 30 days after the send.

**Examples:**

- Normal case: "Weight in kg?" claims the reply until the next check-in.
- Unwanted case: an expiry 31 days away is refused.

### `TGMSG-FR-014`

A plugin must release a reply claim that it holds.

**Why:** A plugin closes the question after the answer, after a skip, or when it
expires the question itself.

### `TGMSG-FR-015`

A reply to a plugin message with an open claim must reach the plugin that holds the
claim, as a text or as a location.

### `TGMSG-FR-016`

A reply to a plugin message with no open claim must show "This message takes no
answer now." and reach no plugin.

### `TGMSG-FR-017`

A plain message must reach the open conversation of the person before any claim.

**Examples:**

- Normal case: Nina parks with the parking conversation while "Weight in kg?" waits.
  "AB-123-CD" goes to the conversation.

### `TGMSG-FR-018`

A plain message with no open conversation must reach the oldest open claim of the
person.

**Examples:**

- Normal case: "72.5" goes to "Weight in kg?", the one open claim.
- Limit case: two claims are open. "72.5" goes to the older one.
- Unwanted case: no claim is open. Maroid answers as it does today.

### `TGMSG-FR-019`

A command must reach its command while a claim is open.

### `TGMSG-FR-020`

A press and a reply must reach the plugin with the person as the acting user and with
no acting workspace.

### `TGMSG-FR-021`

The page of a plugin must show that the plugin receives plugin messages back.

## 6. Invariants

### `TGMSG-INV-001`

A plugin writes only to its acting user. No plugin reads or names the chat of a
person.

**Why:** A plugin that names no chat cannot write to the wrong person.

### `TGMSG-INV-002`

A press or a reply reaches only the plugin that sent the message, and only for the
person who received it.

### `TGMSG-INV-003`

A reply claim survives a restart of the hub.

## 7. Constraints from the guidelines

| Rule      | Guideline    | Effect on this feature                                                         |
| --------- | ------------ | ------------------------------------------------------------------------------ |
| `TG-006`  | Telegram     | Changes with `ADR-0025`: a claim lives in the database.                        |
| `TG-007`  | Telegram     | Changes with `ADR-0025`: a second service sends to the acting user.            |
| `TG-008`  | Telegram     | New with `ADR-0025`: the routing order of an update.                           |
| `OWN-003` | Ownership    | Changes with `ADR-0025`: a press and a reply carry no acting workspace.        |
| `OWN-004` | Ownership    | The record of a plugin message is personal, so a user scopes it.               |
| `OWN-005` | Ownership    | The policy limits each read to the acting user.                                |
| `PLG-006` | Plugin model | Receiving a press and a reply is a capability: interface, registry, registrar. |
| `PLG-007` | Plugin model | A plugin sends through the host.                                               |
| `SEC-012` | Security     | No workspace acts, so the hub reads no enablement at a press.                  |
| `ARC-008` | Architecture | The routing names no plugin.                                                   |
| `DAT-009` | Data         | The record keys on a UUID version 7.                                           |
| `BLD-004` | Build        | `libs/pluginapi` changes, so every plugin builds again.                        |

## 8. Open questions

| #   | Question                                        | Owner | Answer                        |
| --- | ----------------------------------------------- | ----- | ----------------------------- |
| 1   | Which workspace does a press or a reply act in? | Owner | None. 2026-11-23.             |
| 2   | How long does a button or a claim work?         | Owner | 30 days. 2026-11-23.          |
| 3   | Does a conversation become durable?             | Owner | No, a claim only. 2026-11-23. |

## Retired identifiers

This file has no retired identifier.
