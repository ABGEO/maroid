---
title: Spec driven development in Maroid
type: index
status: active
created: 2026-09-11
updated: 2026-09-12
---

# Spec driven development in Maroid

This directory holds the specifications of Maroid.
Code comes from these documents. The documents do not come from the code.

Read this file one time in full. It is the narrative guide to the process.
The binding rules live in the guidelines:

| Guideline | File                                                       | Holds                                |
| --------- | ---------------------------------------------------------- | ------------------------------------ |
| `PRC`     | [guidelines/process.md](./guidelines/process.md)           | The rules of the process.            |
| `TRC`     | [guidelines/traceability.md](./guidelines/traceability.md) | The identifiers and the frontmatter. |
| `LNG`     | [guidelines/language.md](./guidelines/language.md)         | How to write every document.         |

[guidelines/README.md](./guidelines/README.md) indexes every guideline.

## 1. Roles

Maroid has one maintainer. Two roles apply:

| Role    | Who    | Duty                                                       |
| ------- | ------ | ---------------------------------------------------------- |
| Owner   | Temuri | Decides. Reviews. Approves each stage.                     |
| Drafter | Claude | Writes the drafts. Asks the questions. Generates the code. |

The owner approves a stage before the next stage starts.
The drafter does not start the next stage without the approval.

## 2. The four stages

| Stage           | Input                             | Output                           | Approval           |
| --------------- | --------------------------------- | -------------------------------- | ------------------ |
| 0 Guidelines    | The system as it is               | `guidelines/<slug>.md`           | Owner, per rule    |
| 1 Requirements  | An idea, a problem, a need        | `features/<key>/requirements.md` | Owner, per feature |
| 2 Specification | The requirements, the guidelines  | `features/<key>/spec.md`         | Owner, per feature |
| 3 Code          | The specification, the guidelines | Source, tests, migrations        | Owner, per change  |

### Stage 0: Guidelines

The guidelines are the files in `guidelines/`. They hold the rules of the whole system.
One file holds one subject. The file identifier is the prefix of each rule in it.
`PLG-004` is the fourth rule in `guidelines/plugin-model.md`.

[guidelines/README.md](./guidelines/README.md) is the index. It tells you which
file owns a new rule.

Rules:

- The guidelines exist before the first requirements document.
- A guideline rule is not about one feature.
- A change to a rule needs an ADR in `adr/`.
- After a change to a rule, examine each specification that refers to the rule.

### Stage 1: Requirements

The requirements document tells what the product must do.
It does not tell how the product does it.

Rules:

- One statement has one identifier and one behavior.
  Divide a statement that joins two behaviors with the word "and".
- A statement stays true after a full re-implementation.
  If the product moves to a different language, the statement does not change.
- A statement is testable from the outside.
  A non-functional statement gives the quantity, the limit, and the measurement point.
- A statement gives the reason in a `Why` line.
  Omit the `Why` line only when the reason is obvious.
- A statement names no technology.
  No algorithm, no data structure, no library, no table, no screen layout.
- A domain noun is a term in the [glossary](./guidelines/glossary.md).
- Add two to four examples to a statement that is not trivial:
  a normal case, a limit case, and an unwanted case.
  These examples become scenarios in stage 2.

The drafter does more than write down the words of the owner.
The drafter must:

1. Find the contradictions between the statements. Remove them.
2. Estimate the cost. Tell the owner when a statement is expensive.
3. Ask for the aim behind an expensive statement. Propose a cheaper alternative.
4. Close the gaps. Ask for each state, each transition, and each error case.
5. Ask a question when two readings of a statement are possible.

### Stage 2: Specification

The specification tells how the product does what the requirements demand.
It holds every technical decision that the code generation needs.

Rules:

- The specification satisfies the guidelines. It does not contradict a rule.
- Each specification section cites the requirement identifiers that it realizes.
- A section that cites no requirement and no rule gives the reason.
- Each design decision has an identifier, a `Decision` line, and a `Rationale` line.
- Each non-functional requirement has a scenario that measures it.
- Keep the document within the size that `LNG-013` gives.
  Divide a specification larger than 500 lines into subject files: `spec-<subject>.md`.
- The specification declares each contract surface that the feature adds, and carries
  `api.yaml` when it adds an HTTP route. See `SPC`.
- A design decision that changes more than one plugin becomes an ADR.

### Stage 3: Code

Rules:

- Write the tests first. Run them. Confirm that they fail.
- Write the code. Run the tests until they pass.
- Do not change a scenario or a test to make a test pass.
  A wrong test is a signal. Go to section 5.
- Put the statement identifier in a comment at the place that realizes it.
- Run the linter and the build of each artifact that the change touches.
  `BLD-001` gives the command for each artifact.
- A pass is an exit code. A visual check is not a pass.

## 3. The two lanes

Not every change needs a full chain.

### The full lane

Use the full lane for:

- A new plugin.
- A new feature in `apps/hub` or `apps/deck`.
- A change to the behavior that a user can see.
- A change to a contract between the hub and a plugin.

The full lane runs stage 1, then stage 2, then stage 3.

### The fast lane

Use the fast lane for:

- A defect correction.
- A refactor that keeps the behavior.
- A dependency update.
- A build or tooling change.

The fast lane makes no new document. The change cites the identifier that it restores.

**The gate:** if no statement covers the work, the fast lane is not correct.
Stop. Open the full lane.

## 4. Directory layout

```
specs/
  README.md                  This guide. Type: index.
  guidelines/                The locked rules. One file for each subject.
    README.md                The index and the change procedure. Holds no rule.
    architecture.md          ARC
    plugin-model.md          PLG
    data.md                  DAT
    web-ui.md                UI
    configuration.md         CFG
    go-style.md              GO
    frontend-style.md        TS
    testing.md               TST
    build-release.md         BLD
    process.md               PRC
    traceability.md          TRC
    language.md              LNG
    glossary.md              GLO
  adr/                       The decision records. One file for each decision.
    README.md                The ADR index.
    template.md              The ADR template.
  templates/
    requirements.md          The template for stage 1.
    spec.md                  The template for stage 2.
    api.yaml                 The template for an OpenAPI artifact.
  features/
    <key>/                   One directory for each feature. The key is lowercase.
      requirements.md        Stage 1.
      spec.md                Stage 2.
      spec-<subject>.md      Optional. Use when spec.md is larger than 500 lines.
      api.yaml               Optional. OpenAPI, when the feature adds a route. SPC-001.
```

A file name holds the slug only. The frontmatter holds the identifier. See `TRC-008`.

The feature key directory name is the lowercase form of the feature key.
The feature key `NOTIF` uses the directory `features/notif/`.

## 5. The change process

The change process is the same process. It starts at the correct stage.

Find the earliest document that is wrong. Check in this order:

| Symptom                                        | Classification       | Action                                                            |
| ---------------------------------------------- | -------------------- | ----------------------------------------------------------------- |
| The code contradicts the specification         | Defect               | Correct the code. Fast lane.                                      |
| The specification contradicts the requirements | Specification change | Correct `spec.md`. Generate the code again.                       |
| The requirements do not express the need       | Requirement change   | Correct `requirements.md`, then the specification, then the code. |
| A guideline rule is wrong                      | Guideline change     | Write an ADR. Then examine every specification.                   |

Rules:

- Correct the wrong document first. Do not correct the code first.
- After a change to an identifier, find each reference to the identifier.
  Examine each reference.
- The problem comes before the solution.
  A new capability starts with a requirement statement, not with a technology choice.
