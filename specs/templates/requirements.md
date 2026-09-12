---
id: KEY
title: The name of the feature
type: requirements
status: draft
created: YYYY-MM-DD
updated: YYYY-MM-DD
approved_by:
approved_on:
constrained_by: []
---

# Requirements: The name of the feature

Copy this file to `specs/features/<key>/requirements.md`.
Set every field in the frontmatter. See `TRC-008`.

Add the key to the feature key register in `specs/guidelines/traceability.md` first.
A feature key has four to eight characters.
A shorter prefix belongs to a guideline. See `TRC-002`.

Write every statement in the language that `LNG` demands.
A statement tells what the product must do. A statement does not tell how.

## 1. Problem

Describe the problem in the words of the person who has it.
Give the evidence. Do not describe the solution.

## 2. Users

Who uses this feature? What does each person need?

## 3. Out of scope

What this feature does not do. One line for each item.

## 4. Definitions

A new domain term. Move each term to `specs/guidelines/glossary.md` at the approval.

| Term | Meaning |
| ---- | ------- |
|      |         |

## 5. Functional requirements

### `<KEY>-FR-001`

The `<actor>` must `<verb>` `<object>` when `<condition>`.

**Why:** The reason. Omit this line only when the reason is obvious.

**Examples:**

- Normal case:
- Limit case:
- Unwanted case:

### `<KEY>-FR-002`

## 6. Non-functional requirements

### `<KEY>-NFR-001`

Give the quantity, the limit, the condition, and the measurement point.

**Why:**

## 7. Invariants

### `<KEY>-INV-001`

A condition that is always true.

**Why:**

## 8. Constraints from the guidelines

Name each rule that limits this feature. A rule identifier has two segments.
List the guideline identifiers in the `constrained_by` field of the frontmatter.

| Rule      | Guideline    | Effect on this feature |
| --------- | ------------ | ---------------------- |
| `PLG-###` | Plugin model |                        |
| `DAT-###` | Data         |                        |

## 9. Open questions

| #   | Question | Owner | Answer |
| --- | -------- | ----- | ------ |
| 1   |          |       |        |

Answer every question before the approval. An open question blocks stage 2.

## Retired identifiers

This file has no retired identifier.
