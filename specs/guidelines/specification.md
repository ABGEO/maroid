---
id: SPC
title: The specification artifacts
type: guideline
status: active
created: 2026-09-12
updated: 2026-09-13
scope: [specs/features/]
related: [PRC, TRC, LNG, API, MQT, JOB, TG, CLI, NTF, CFG]
---

# The specification artifacts

A specification describes a contract that the code must match.
This guideline gives the files that carry the contract and the shape of each declaration.

## SPC-001

A feature directory holds these files and no other:

| File                        | Holds                                         | Present when                                      |
| --------------------------- | --------------------------------------------- | ------------------------------------------------- |
| `requirements.md`           | Stage 1                                       | Always                                            |
| `requirements-<subject>.md` | One subject of the requirements               | `requirements.md` passes the target of `LNG-013`  |
| `spec.md`                   | Stage 2                                       | Always                                            |
| `spec-<subject>.md`         | One subject of the specification              | `spec.md` passes the target of `LNG-013`          |
| `api.yaml`                  | The OpenAPI description of the routes it adds | The feature adds or changes an HTTP route         |

A file that is not markdown carries no frontmatter. See `TRC-008`.

## SPC-002

`api.yaml` uses OpenAPI 3.1.
It describes every route that the feature adds or changes, and no other route.
A path holds the effective prefix that `API-004` gives, not the pattern of the plugin.

**Why:** A client and a generator read the file. A route that only prose describes
drifts from the code, and no tool catches the drift.

## SPC-003

A specification declares each contract surface that the feature adds.
Section 4.2 holds the data model. Section 4.3 holds every other surface.
One table for each surface. These columns are the minimum:

| Surface               | Columns                                            | Governed by                     |
| --------------------- | -------------------------------------------------- | ------------------------------- |
| Data model            | Table, Schema, Scope, Migration, Realizes          | `DAT-001`, `DAT-005`, `OWN-004` |
| HTTP route            | Method, Path, Access, Realizes                     | `API-003`, `API-004`            |
| MQTT subscriber       | Identifier, Relative topic, QoS, Handles, Realizes | `MQT-001`, `MQT-003`            |
| Cron job              | Identifier, Schedule, Acts for, Behavior, Realizes | `JOB-005`, `OWN-009`            |
| Background worker     | Name, Prepare verifies, Start runs, Realizes       | `JOB-001`                       |
| Telegram command      | Command, Scope, Validates, Handles, Realizes       | `TG-003`, `TG-004`              |
| Telegram conversation | Identifier, Entry step, Steps, Realizes            | `TG-005`                        |
| CLI command           | Command, Flags, Does, Realizes                     | `CLI-001`                       |
| Notification channel  | Channel, Sent when, Realizes                       | `NTF-002`                       |
| Configuration scheme  | Key, Type, Default, Secret, Realizes               | `CFG-003`, `CFG-006`            |
| MCP tool              | Tool, Declared by, Annotations, Realizes           | `PLG-006`, `PLG-011`            |

Delete a table that the feature does not need. Keep no empty heading. See `LNG-013`.

A topic column holds the relative topic. `MQT-002` derives the effective topic.

**Why:** A derived value that a second document repeats becomes wrong when the
source changes.

## SPC-004

Every row of a declaration table names a requirement identifier in its `Realizes`
column. The coverage table in section 2 names the section that holds the table.

**Why:** A surface that no requirement demands is scope that nobody approved.

## SPC-005

A diagram is a mermaid block inside the specification. A diagram is not a separate file.

Draw a diagram when the design holds an order that a table cannot carry:

- A sequence that crosses two or more components.
- A state machine, such as a conversation. See `TG-005`.
- A data model with three or more related tables.

**Why:** A reader follows one document. A second file adds a reference that breaks.

## SPC-006

An artifact belongs to the specification that the owner approves.
The owner approves a change to an artifact as they approve a change to `spec.md`.

## Retired identifiers

This file has no retired identifier.
