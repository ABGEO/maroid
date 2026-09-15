---
title: Architecture decision records
type: index
status: active
created: 2026-09-11
updated: 2026-09-15
---

# Architecture decision records

An ADR records a decision that outlives one feature.
An accepted ADR is the only thing that changes a guideline.

Write an ADR when:

- A guideline rule is wrong or incomplete.
- A design decision in a specification changes more than one plugin.
- The system needs a new technology.
- The guidelines need a new file.

Do not write an ADR for a decision inside one feature.
That decision is a `DD` item in the specification of the feature.

## The identifier and the file name

The identifier has the form `ADR-####`. Four digits. The number starts at `0001`.
The number is permanent. A rejected ADR keeps its number.

The file name holds the slug only, as `TRC-008` demands.
The frontmatter holds the identifier.
`ADR-0003` can live in `outbox-relay.md`. This index gives the order.

## The status

| Status     | Meaning                                                |
| ---------- | ------------------------------------------------------ |
| proposed   | The drafter wrote it. The owner did not decide.        |
| accepted   | The owner accepted it. The guidelines change.          |
| rejected   | The owner rejected it. The file stays for the history. |
| superseded | A later ADR replaced it. Name the later ADR.           |

## The index

| ID         | Title                               | File                                   | Status   | Date       | Rules that change                                                  |
| ---------- | ----------------------------------- | -------------------------------------- | -------- | ---------- | ------------------------------------------------------------------ |
| `ADR-0001` | PostgreSQL 18 and the native uuidv7 | `postgresql-18-and-uuidv7.md`          | accepted | 2026-09-12 | `DAT-008`, `DAT-009`                                               |
| `ADR-0002` | Dex as the authorization server     | `dex-as-the-authorization-server.md`   | accepted | 2026-09-15 | `SEC-001`, `SEC-002`, `SEC-003`, `OWN-001`, `OWN-003`, `API-003`, `GLO-user` |
