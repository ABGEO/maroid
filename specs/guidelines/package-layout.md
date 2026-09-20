---
id: PKG
title: Package layout
type: guideline
status: active
created: 2026-09-11
updated: 2026-09-20
scope: [apps/hub/internal/, libs/, plugins/]
related: [PLG, GO, REP, EXT]
---

# Package layout

## PKG-001

A plugin uses this layout. Create a directory only when the plugin needs it.

| Directory                     | Holds                                                 |
| ----------------------------- | ----------------------------------------------------- |
| `config/`                     | The typed configuration of the plugin. See `CFG-006`. |
| `dto/`                        | The shapes of an external API. See `EXT-006`.         |
| `model/`                      | The domain entities. See `REP-005`.                   |
| `repository/`                 | The data access. See `REP-001`.                       |
| `service/`                    | The clients of an external system. See `EXT-001`.     |
| `job/`                        | The cron jobs. See `JOB-005`.                         |
| `handler/`                    | The HTTP handlers. See `API-004`.                     |
| `telegram/command/`           | The bot commands. See `TG-003`.                       |
| `telegram/conversation/step/` | The conversation steps. See `TG-005`.                 |
| `mcp/tool/`                   | The Model Context Protocol tools. See `PLG-006`.      |
| `mqtt/subscriber/`            | The MQTT subscribers. See `MQT-001`.                  |
| `db/migrations/`              | The migrations. See `DAT-006`.                        |
| `errs/`                       | The sentinel errors of the plugin. See `GO-005`.      |
| `ui/`                         | The user interface. See `UI-002`.                     |

## PKG-002

`main.go` holds the plugin type, the `New` constructor, and the capability methods.
It holds no business logic.

**Why:** The file is the contract with the hub. A reader sees every capability in one screen.

## PKG-003

A package with more than one file keeps its package comment in `doc.go`.
`doc.go` holds the comment and nothing else.

## PKG-004

Every package of the hub lives under `apps/hub/internal`.
The hub exports nothing.

**Why:** `PLG-007` forbids a plugin to import the hub. `internal` makes the compiler enforce it.

## PKG-005

A package name is one lowercase noun. It uses no underscore and no abbreviation
that the glossary does not hold.

## Retired identifiers

This file has no retired identifier.
