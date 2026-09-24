---
id: TRC
title: Traceability
type: guideline
status: active
created: 2026-09-11
updated: 2026-09-24
scope: [specs/, code comments]
related: [LNG, PRC, GIT, RES]
---

# Traceability

This guideline defines the identifiers of Maroid.
Every item in a specification document has an identifier.
An identifier never changes and never repeats.

## TRC-001

The count of the segments tells you the level of an identifier.

| Level     | Form                 | Segments | Example        |
| --------- | -------------------- | -------- | -------------- |
| Guideline | `<GID>-<NNN>`        | 2        | `PLG-004`      |
| Feature   | `<KEY>-<TYPE>-<NNN>` | 3        | `NOTIF-FR-001` |

Three identifier kinds do not use this grammar:

| Identifier      | Form           | Example      |
| --------------- | -------------- | ------------ |
| Glossary term   | `GLO-<term>`   | `GLO-plugin` |
| Decision record | `ADR-####`     | `ADR-0003`   |
| External rule   | `Z-<number>`   | `Z-118`      |

`RES-001` names the document and the revision that a `Z` number indexes. A
prefix of one character is external, and it belongs to no file of `specs/`.

## TRC-002

| Prefix kind          | Characters | Owner                  |
| -------------------- | ---------- | ---------------------- |
| Guideline ID (`GID`) | 2 to 3     | One guideline file.    |
| Feature key (`KEY`)  | 4 to 8     | One feature directory. |

The length rule guarantees that the two sets cannot collide.
A prefix of two or three characters is always a guideline.
A prefix of four or more characters is always a feature.

A prefix uses uppercase letters and digits.
A prefix names the capability, not the technology.
A prefix becomes permanent when the owner approves the first document that uses it.
A retired prefix never repeats.

## TRC-003

The `TYPE` segment exists at the feature level only.
A guideline rule needs no type. The guideline file gives the subject.

| Type  | Name                       | Document          | Answers                               |
| ----- | -------------------------- | ----------------- | ------------------------------------- |
| `FR`  | Functional requirement     | `requirements.md` | What must the product do?             |
| `NFR` | Non-functional requirement | `requirements.md` | How well must the product do it?      |
| `INV` | Invariant                  | `requirements.md` | What is always true?                  |
| `DD`  | Design decision            | `spec.md`         | Which technical option did we select? |
| `SC`  | Scenario                   | `spec.md`         | How do we prove the behavior?         |

Examples:

```
ARC-005          A rule in the architecture guideline.
PLG-004          A rule in the plugin model guideline.
GLO-plugin       A glossary term.
ADR-0003         An architecture decision record.

NOTIF-FR-001     A functional requirement of the NOTIF feature.
NOTIF-NFR-002    A non-functional requirement of the NOTIF feature.
NOTIF-INV-001    An invariant of the NOTIF feature.
NOTIF-DD-004     A design decision of the NOTIF specification.
NOTIF-SC-007     A scenario of the NOTIF specification.
```

## TRC-004

A counter belongs to one guideline file, or to one pair of `KEY` and `TYPE`.
`NOTIF-FR-001` and `NOTIF-DD-001` are two different items.

Allocate the next free number. Do not fill a gap.
Find the next free number in the document and in its `Retired identifiers` section.
Write the identifier one time. The identifier becomes permanent when the owner
approves the document. A draft renumbers freely.

## TRC-005

An approved identifier never repeats. This rule has no exception.

A document with the status `draft` or `in-review` holds no approved identifier.
It renumbers freely, and it writes no retired row.

To remove an item:

1. Delete the item from the body of the document.
2. Add one row to the `Retired identifiers` section at the end of the same document.
3. Give the date and the reason.

```markdown
## Retired identifiers

| ID           | Retired    | Reason                             |
| ------------ | ---------- | ---------------------------------- |
| NOTIF-FR-004 | 2026-09-11 | The behavior moved to CHAN-FR-002. |
```

A document with no retired identifier holds the sentence
"This file has no retired identifier." in place of the table.

Each guideline file and each feature document holds its own section.
A superseded item keeps the identifier. Write the successor in the reason.

## TRC-006

- A specification section cites the requirement identifiers that it realizes.
- A requirement statement cites a guideline rule only when the rule limits the statement.
- A guideline rule can cite another guideline rule.
- Write a reference as the bare identifier: `PLG-004`, `NOTIF-FR-001`.
- A broken reference is a defect. An identifier that no document defines is a defect.

The direction of the references:

```
<GID>-###  <---  <KEY>-FR-###  <---  <KEY>-DD-###  <---  code
                      ^                     ^
                      |                     |
                <KEY>-INV-###         <KEY>-SC-###
```

A document refers upward. A document does not refer downward.
A guideline does not name a feature. A requirements document does not name a design decision.

## TRC-007

A test cites the identifier that it verifies, in a comment above the test.
`TST-001` gives the form.

```go
// NOTIF-SC-003: A second enqueue with the same business key adds no row.
func TestEnqueueIsIdempotent(t *testing.T) {
```

A migration cites the invariant that its constraint enforces:

```sql
-- NOTIF-INV-001: One row for each business key. The constraint enforces it.
CREATE UNIQUE INDEX ...
```

**Production code carries no identifier.** A comment there gives the reason that
the code cannot give, in words that a reader checks against the code in front of
them.

**Why:** An identifier in production code goes stale when a document moves, and
nothing fails when it does. A reader of the code cannot check it without opening
`specs/`. The test is the one place where the citation earns its keep, because the
test and the statement say the same thing, and the test fails when they disagree.

The trace runs from a statement to the test that verifies it, and from the test to
the code it exercises.

## TRC-008

Every markdown document in `specs/` starts with a YAML frontmatter block.
The file name holds the slug only. The frontmatter holds the identifier.
An artifact that is not markdown carries no frontmatter, because its own schema
owns the file. `SPC-001` lists each one.
A file in `specs/templates/` carries the frontmatter of its target type,
with a placeholder in each value.

Common fields:

| Field     | Rule                                                                     |
| --------- | ------------------------------------------------------------------------ |
| `id`      | The prefix that this file owns. An index file has no `id`.               |
| `title`   | The name of the document. The first heading repeats it.                  |
| `type`    | One of: `guideline`, `glossary`, `adr`, `requirements`, `spec`, `index`. |
| `status`  | See the table below.                                                     |
| `created` | The date of the first version, as `YYYY-MM-DD`.                          |
| `updated` | The date of the last change, as `YYYY-MM-DD`.                            |

Fields for each type:

| Type           | Additional fields                                                              |
| -------------- | ------------------------------------------------------------------------------ |
| `guideline`    | `scope` (list of paths), `related` (list of guideline identifiers)             |
| `requirements` | `approved_by`, `approved_on`, `constrained_by` (list of guideline identifiers) |
| `spec`         | `approved_by`, `approved_on`, `constrained_by`, `requirements` (path)          |
| `adr`          | `changes` (list of rule identifiers), `supersedes`, `superseded_by`, `decided` |

Values for `status`:

| Type                    | Values                                           |
| ----------------------- | ------------------------------------------------ |
| `guideline`, `glossary` | `active`, `retired`                              |
| `requirements`, `spec`  | `draft`, `in-review`, `approved`                 |
| `adr`                   | `proposed`, `accepted`, `rejected`, `superseded` |
| `index`                 | `active`                                         |

Example:

```yaml
---
id: PLG
title: The plugin model
type: guideline
status: active
created: 2026-09-11
updated: 2026-09-21
scope: [plugins/, apps/hub/internal/plugin/, libs/pluginapi/]
related: [ARC, DAT, UI, BLD]
---
```

## The prefix register

A prefix becomes permanent when the owner approves the first document that uses it.
A retired prefix never repeats.
Add a row before you create the file or the directory.

### Guideline identifiers, two to three characters

| ID    | File                              | Status |
| ----- | --------------------------------- | ------ |
| `ARC` | `guidelines/architecture.md`      | Active |
| `PLG` | `guidelines/plugin-model.md`      | Active |
| `DAT` | `guidelines/data.md`              | Active |
| `UI`  | `guidelines/web-ui.md`            | Active |
| `CFG` | `guidelines/configuration.md`     | Active |
| `GO`  | `guidelines/go-style.md`          | Active |
| `TS`  | `guidelines/frontend-style.md`    | Active |
| `TST` | `guidelines/testing.md`           | Active |
| `BLD` | `guidelines/build-release.md`     | Active |
| `PRC` | `guidelines/process.md`           | Active |
| `TRC` | `guidelines/traceability.md`      | Active |
| `LNG` | `guidelines/language.md`          | Active |
| `GLO` | `guidelines/glossary.md`          | Active |
| `GIT` | `guidelines/git.md`               | Active |
| `API` | `guidelines/http-api.md`          | Active |
| `SEC` | `guidelines/security.md`          | Active |
| `TG`  | `guidelines/telegram.md`          | Active |
| `MQT` | `guidelines/mqtt.md`              | Active |
| `JOB` | `guidelines/jobs.md`              | Active |
| `NTF` | `guidelines/notifications.md`     | Active |
| `PKG` | `guidelines/package-layout.md`    | Active |
| `DEP` | `guidelines/dependencies.md`      | Active |
| `LIF` | `guidelines/lifecycle.md`         | Active |
| `CLI` | `guidelines/cli.md`               | Active |
| `REP` | `guidelines/repository.md`        | Active |
| `EXT` | `guidelines/external-services.md` | Active |
| `LOG` | `guidelines/logging.md`           | Active |
| `OWN` | `guidelines/ownership.md`         | Active |
| `SPC` | `guidelines/specification.md`     | Active |
| `ERR` | `guidelines/errors.md`            | Active |
| `RES` | `guidelines/rest.md`              | Active |

### Decision records

| ID    | Directory | Status |
| ----- | --------- | ------ |
| `ADR` | `adr/`    | Active |

### Feature keys, four to eight characters

| Key     | Directory         | Feature                                        | Status   |
| ------- | ----------------- | ---------------------------------------------- | -------- |
| `IDENT` | `features/ident/` | The user record and the ownership of a row.    | Approved |
| `PSET`  | `features/pset/`  | The settings of a user for a plugin.           | Approved |
| `EXTID` | `features/extid/` | External identities and the delegated sign in. | Approved |
| `THEMING` | `features/theming/` | The shared visual theme, and the identity provider's pages. | Approved |
| `MCPHUB` | `features/mcphub/` | The hub as a Model Context Protocol server. | Approved |
| `PCAP`  | `features/pcap/`  | The capabilities that a plugin declares.       | Approved |
| `WEBSESS` | `features/websess/` | The session of a person at the web shell. | Approved |
| `APIFMT` | `features/apifmt/` | The shape of a successful answer, and the routes that carry it. | Draft |

## Retired identifiers

This file has no retired identifier.
