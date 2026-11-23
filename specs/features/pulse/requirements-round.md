---
id: PULSE
title: The check-ins of a person, the round on Telegram
type: requirements
status: approved
created: 2026-11-23
updated: 2026-11-23
approved_by: Temuri
approved_on: 2026-11-23
constrained_by: [TG]
---

# Requirements: The round on Telegram

[`requirements.md`](requirements.md) holds the problem and the definitions.

## 5. Functional requirements

### `PULSE-FR-011`

A round must open with one start prompt. The start prompt must name the check-in and
the count of its questions, with the buttons Start and Decline.

**Examples:**

- Normal case: "Evening check-in, 7 questions." with Start and Decline.
- Limit case: a check-in whose every question is archived holds no question. Its
  occurrence fires no round.

### `PULSE-FR-012`

Decline must close the round. The start prompt then shows that the person declined,
and the bot asks nothing and records nothing.

### `PULSE-FR-013`

Start must send the first question. The start prompt then shows that the round
started. The bot sends one question at a time, in the order of the check-in.

### `PULSE-FR-014`

An answer must change its question message to show the answer, and the bot must then
send the next question.

**Example:** "How is your mood? 4" replaces the buttons 1 to 5 and Skip.

### `PULSE-FR-015`

After the last question, the start prompt must show that the round is done, and the
bot sends no further message. A round with every question skipped is done.

### `PULSE-FR-016`

Every question must offer Skip. A skip records no value, and the question message
then shows "skipped".

### `PULSE-FR-017`

A question of the type scale, yes or no, or choice must take its answer from a
button. A scale shows one button for each value, and the labels of its two ends in
the text.

**Example:** "How is your mood? (1 bad, 5 great)" with the buttons 1 to 5 and Skip.

### `PULSE-FR-018`

A question of the type number must take its answer from a quick value button or from
a typed reply. A typed number takes a point or a comma. A reply that is no number, or
that falls outside the minimum and the maximum, must get a message that gives the
range, and the question keeps waiting.

**Examples:**

- Normal case: "72,5" on "Weight (kg)" records 72.5.
- Limit case: "250" on a maximum of 250 records 250.
- Unwanted case: "900" gets "Send a number from 30 to 250, or press Skip." The
  question still waits.

### `PULSE-FR-019`

A question of the type text must take its answer from a typed reply of 1 to 1000
characters. A longer reply must get a message that gives the limit, and the question
keeps waiting.

### `PULSE-FR-020`

A question of the type location must take a location that the person shares, from
the button "Send location" or from the attachment menu. Its keyboard holds "Send
location" and "Skip", and the bot removes the keyboard after the answer.

**Why:** Telegram Desktop shares no location from a keyboard button.

**Examples:**

- Normal case: Nina presses "Send location". Pulse records the latitude and the
  longitude, and the keyboard goes away.
- Unwanted case: Nina types "home". The bot answers that it needs a shared location,
  and the question keeps waiting.

### `PULSE-FR-021`

A press on a button of a round that is done, declined, or expired, or on a question
that holds an answer, must record nothing. The bot must say that the question is
closed.

**Example:** Nina presses 3 on the mood question of yesterday. The notice reads
"This check-in is closed." No value is recorded.

### `PULSE-FR-022`

A typed reply or a shared location must reach pulse only while a question of the type
number, text, or location waits for it. Any other message of the person reaches the
rest of the bot as before.

**Why:** A person in the middle of a conversation of another plugin must not answer
a pulse question by accident. `TG-008` gives the conversation the first turn.

## Retired identifiers

This file has no retired identifier.
