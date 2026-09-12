---
id: LNG
title: Language
type: guideline
status: active
created: 2026-09-11
updated: 2026-09-12
scope: [specs/, CLAUDE.md, code comments, commit messages]
related: [TRC, GLO, PRC, GO, TS]
---

# Language

This guideline holds the rules for the text of Maroid.
The aim is one reading. A statement must not permit two interpretations.

A table in this guideline can name a forbidden word. That is a citation, not a use.

## LNG-001

These artifacts obey this guideline:

| Artifact                                               | Obeys |
| ------------------------------------------------------ | ----- |
| Every document in `specs/`                             | Yes   |
| `CLAUDE.md`                                            | Yes   |
| Every code comment, in Go, TypeScript, Svelte, and SQL | Yes   |
| Every commit message, pull request, and issue          | Yes   |
| A reply in a chat session                              | No    |

The base standard is ASD-STE100 (Simplified Technical English).
The rules below are the part of the standard that applies most often, and two
additional rules for this repository: `LNG-006` and `LNG-007`.

## LNG-002

One word has one meaning. One thing has one name. Do not use a synonym.
A domain noun is a term in `GLO`.

| Wrong                                              | Correct              |
| -------------------------------------------------- | -------------------- |
| "the record", "the entry", "the row" for one thing | "the row" everywhere |
| "the plugin", "the module", "the extension"        | "the plugin"         |

## LNG-003

| Rule                                                | Wrong                              | Correct                                   |
| --------------------------------------------------- | ---------------------------------- | ----------------------------------------- |
| Use a simple verb.                                  | "utilize", "facilitate"            | "use", "help"                             |
| Do not use a noun cluster of more than three words. | "plugin notification retry policy" | "the retry policy for the notification"   |
| Do not omit an article.                             | "Hub loads plugin."                | "The hub loads the plugin."               |
| Do not use a contraction.                           | "does not" as "doesn't"            | "does not"                                |
| Do not use jargon, slang, or a metaphor.            | "fire and forget"                  | "the caller does not wait for the result" |

## LNG-004

| Item                         | Limit               |
| ---------------------------- | ------------------- |
| An instruction sentence      | 20 words maximum    |
| A descriptive sentence       | 25 words maximum    |
| A paragraph                  | 6 sentences maximum |
| Instructions in one sentence | 1                   |

## LNG-005

- Use the active voice. Name the actor.
  - Wrong: "The plugin is loaded at the start."
  - Correct: "The hub loads the plugin at the start."
- Use the simple present, the simple past, or the simple future.
  Do not use a perfect tense.
- Do not use a verb that ends in `-ing` as a noun.
  - Wrong: "Loading the plugin needs the symbol."
  - Correct: "The hub needs the symbol when it loads the plugin."
- Write one instruction in one sentence.
- Start an instruction with the verb.
- Put the condition before the instruction.
  - Correct: "If the API version is different, the hub refuses the plugin."

## LNG-006

Do not use an em dash. The character is forbidden in every artifact that `LNG-001` lists.

Replace it with a period, a comma, a colon, a semicolon, or a pair of parentheses.

| Wrong                                                | Correct                                            |
| ---------------------------------------------------- | -------------------------------------------------- |
| `The hub loads the plugin — one symbol only.`        | `The hub loads the plugin. It reads one symbol.`   |
| `Three plugins — telasi, gwp, and parking — failed.` | `Three plugins failed: telasi, gwp, and parking.`  |
| `The schema name — see DAT-001 — comes from the ID.` | `The schema name comes from the ID (see DAT-001).` |

An en dash is correct in a numeric range only: `2026-09-11`, `pages 10 to 14`.
Prefer the word "to".

**Why:** The em dash joins two ideas without a grammatical relation between them.
`LNG-004` demands short sentences with one idea. The two rules cannot both hold.

## LNG-007

Do not use a filler word. A filler word adds length and adds no fact.

**Empty verbs and adjectives:**

| Forbidden                                                   | Use instead                                     |
| ----------------------------------------------------------- | ----------------------------------------------- |
| delve, dive into, deep dive                                 | examine, study                                  |
| leverage, utilize, harness                                  | use                                             |
| streamline, elevate, unlock, empower, supercharge           | Name the exact change.                          |
| seamless, effortless, intuitive                             | Name the number of steps.                       |
| robust, resilient, reliable                                 | Give the failure rate or the recovery behavior. |
| comprehensive, extensive, holistic                          | Give the full list.                             |
| crucial, vital, critical, essential, key                    | Delete the word, or give the consequence.       |
| cutting-edge, state-of-the-art, best-in-class, game-changer | Delete the word.                                |
| meticulous, intricate, nuanced, sophisticated               | Delete the word.                                |
| boasts, showcases, underscores, highlights                  | has, shows                                      |
| powerful, rich, modern, clean                               | Name the property.                              |

**Empty phrases:**

| Forbidden                                                          | Use instead                  |
| ------------------------------------------------------------------ | ---------------------------- |
| it is worth noting that, it is important to note, please note that | Delete. State the fact.      |
| keep in mind, bear in mind, remember that                          | Delete.                      |
| essentially, basically, simply put, in essence, at its core        | Delete.                      |
| furthermore, moreover, additionally                                | Delete, or "also".           |
| in conclusion, overall, all in all, at the end of the day          | Delete.                      |
| when it comes to, in terms of, with regard to                      | for, about                   |
| a wide range of, a variety of, a plethora of                       | Give the number or the list. |
| plays a key role, serves as, acts as                               | Name the function.           |
| not only X but also Y                                              | X and Y.                     |
| in today's fast-paced world, in the modern landscape               | Delete.                      |
| let us dive in, let us explore, I hope this helps, feel free to    | Delete.                      |

**Empty nouns:**

| Forbidden                                      | Use instead          |
| ---------------------------------------------- | -------------------- |
| tapestry, landscape, realm, journey, testament | Delete.              |
| ecosystem, synergy, paradigm                   | Name the components. |

Do not use an emoji in a document or in a code comment.

**Why:** A reader must find the fact. A filler word hides it.

## LNG-008

Do not use a vague word in a requirement. Each word below makes a statement untestable.

| Forbidden                       | Replace with                          |
| ------------------------------- | ------------------------------------- |
| fast, quick, responsive         | A time limit and a measurement point. |
| easy, simple                    | A count of the steps or the clicks.   |
| flexible, scalable, extensible  | The exact dimension and the limit.    |
| appropriate, sufficient, proper | The exact condition.                  |
| should, may, could              | "must", or delete the statement.      |
| etc., and so on, such as        | The full list.                        |
| support, handle, manage         | The exact behavior.                   |

## LNG-009

- Use a vertical list for three or more items.
- Use a table for a set of items with the same shape.
- Give each statement a heading or a table row of its own.
- Write a warning before the step that needs it.

## LNG-010

Use these patterns for a statement.

**A functional requirement:**

> The `<actor>` must `<verb>` `<object>` when `<condition>`.
>
> `NOTIF-FR-001`: The hub must reject a second notification with the same
> business key, the same event type, and the same plugin identifier.

**A non-functional requirement:** give the quantity, the limit, and the measurement point.

> `NOTIF-NFR-001`: The hub must send a queued notification in less than 30 seconds.
> Measure from the write to the outbox to the call to the Telegram API.

**An invariant:**

> `NOTIF-INV-001`: One business key has one row in the outbox at all times.

**A design decision:**

> `NOTIF-DD-001`
> **Decision:** A unique index on `(plugin_id, event_type, business_key)` enforces `NOTIF-INV-001`.
> **Rationale:** The database enforces the constraint. Two workers cannot break it.
> **Alternatives:** An application-level check. It fails when two workers run.

**A scenario:**

> `NOTIF-SC-001` (verifies `NOTIF-FR-001`)
> **Given** the outbox holds a row for the key `("dev.maroid.telasi", "bill.due", "2026-09")`.
> **When** the plugin enqueues the same key a second time.
> **Then** the outbox holds one row. The call returns no error.

## LNG-011

Write a comment only when the code cannot explain itself.

Before you write a comment, try these three steps in order:

1. Give the variable, the function, or the type a name that states the intent.
2. Extract the block into a function whose name states the intent.
3. Replace a literal value with a named constant.

Write the comment only when all three steps fail.

Do not write a comment that repeats the code.

| Wrong                                                              | Correct                        |
| ------------------------------------------------------------------ | ------------------------------ |
| `// Add the UI capability if present.` above the code that adds it | Delete it. The code says this. |
| `// Loop over the plugins.` above the loop                         | Delete it.                     |
| `// Increment the counter.`                                        | Delete it.                     |

A comment gives the reason. The code shows the action.

**Why:** A compiler does not check a comment. A test does not check a comment.
When the code changes, the comment stays as it was, and a reader trusts it.
A wrong comment costs more than no comment.

## LNG-012

These comments are correct:

| Kind                                                   | Rule                                                                      |
| ------------------------------------------------------ | ------------------------------------------------------------------------- |
| A documentation comment on an exported identifier      | Required. See `GO-003`.                                                   |
| A statement identifier at the code that realizes it    | Required. See `TRC-007`.                                                  |
| A reason that the code cannot show                     | Correct. Give the constraint, the standard, or the defect that forced it. |
| A warning about a result that a reader does not expect | Correct.                                                                  |
| `// @todo:` with the work named                        | Correct.                                                                  |

A comment that is not in this table is wrong. Delete it.

Every comment obeys `LNG-002` to `LNG-007`.
Write short sentences. Use the active voice. Use no em dash and no filler word.

A reason that the code cannot show:

```go
// A cookie holds the redirect URL, so a client cannot change it.
// The allowed list still applies, because an attacker can steal a cookie.
```

A warning:

```go
// The caller must hold the lock. This function does not take it.
```

## LNG-013

Write the minimum text that answers the question.
An agent loads a document into its context. A token that the document spends is a
token that the work cannot spend.

| Document                | Maximum lines                                            |
| ----------------------- | -------------------------------------------------------- |
| A guideline             | 100                                                      |
| An index                | 150. It grows by one row for each guideline.             |
| The process guide       | 200                                                      |
| A requirements document | 150                                                      |
| A specification file    | 500. Divide a larger one into `spec-<subject>.md` files. |
| A template              | 150. It carries the shape of every section.              |

Apply these practices:

- Write no preamble. Do not describe what the document will say.
- Write no closing summary.
- Do not repeat a rule from another document. Cite the identifier.
- Give one example. Give a second example only when the case differs.
- Delete a template section that the feature does not need. Keep no empty heading.
- Use a table for items that share a shape. See `LNG-009`.

**Why:** A long document hides the rule inside it, and it costs the context that
the work needs.

## Retired identifiers

This file has no retired identifier.
