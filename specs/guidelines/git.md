---
id: GIT
title: Git conventions
type: guideline
status: active
created: 2026-09-11
updated: 2026-09-11
scope: [.]
related: [PRC, LNG]
---

# Git conventions

This guideline holds every rule about a commit.
A commit message names no statement identifier. The specification holds the
traceability, not the history.

## GIT-001

The owner makes every commit.
An agent commits only when the owner asks for a commit in the same session.
An agent does not push without a direct instruction.

**Why:** The owner decides what enters the history.

## GIT-002

A commit message is one line. It has no body.

```
<type>(<scope>): <description>
```

**Why:** A one-line history stays readable in `git log --oneline`.
A body repeats what the specification already holds.

## GIT-003

The type comes from this table. Do not invent a type.

| Type       | Use                                                            |
| ---------- | -------------------------------------------------------------- |
| `feat`     | New behavior that a user can see.                              |
| `fix`      | A defect correction.                                           |
| `refactor` | A change that keeps the behavior.                              |
| `perf`     | A change that improves the speed or the memory use.            |
| `test`     | A change to a test only.                                       |
| `docs`     | A change to a document only, including a document in `specs/`. |
| `build`    | A change to the build, the container image, or the chart.      |
| `chore`    | A change to the tooling or to a dependency.                    |

## GIT-004

The scope is the directory name of the module that the change touches.

| Scope        | For                                                   |
| ------------ | ----------------------------------------------------- |
| `hub`        | `apps/hub`                                            |
| `deck`       | `apps/deck`                                           |
| `<plugin>`   | `plugins/<plugin>`, for example `jasmine` or `telasi` |
| `<lib>`      | `libs/<lib>`, for example `pluginapi` or `notifier`   |
| `<firmware>` | `firmware/<name>`, for example `jasmine-node`         |
| `specs`      | `specs/`                                              |
| `deps`       | A dependency update                                   |

Write no scope when the change touches no single module.

## GIT-005

The description uses the imperative mood: "add", not "added" and not "adds".
The description starts with a lowercase letter.
The description has no full stop at the end.
The full line has 72 characters maximum.

The description obeys `LNG`. Use no em dash. Use no filler word.

```
feat(jasmine): add the watering schedule for a plant
fix(hub): log a conversation step error instead of a silent discard
refactor(deck): replace the manual fetch with the api client
chore(deps): update the internal packages
docs(specs): add the git conventions guideline
```

## GIT-006

The git user of the system is the sole author of a commit.

Do not add a `Co-Authored-By` trailer.
Do not add a trailer, a line, or a footer that names an agent, a tool, or a model.

**Why:** A commit records who is responsible for the change. The owner is responsible.

## Retired identifiers

This file has no retired identifier.
