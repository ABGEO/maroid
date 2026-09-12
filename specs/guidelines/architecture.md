---
id: ARC
title: Architecture
type: guideline
status: active
created: 2026-09-11
updated: 2026-09-11
scope: [apps/, libs/, plugins/, chart/]
related: [PLG, DAT, UI, BLD, JOB, API]
---

# Architecture

This guideline holds the shape of the system and the technology of each layer.
A change to a rule in this file needs an ADR.

## ARC-001

Maroid is a hub-and-spoke system. `apps/hub` is the only host process.
A plugin does not run alone.

**Why:** One process owns the configuration, the database connection, and the
lifecycle. A plugin adds a capability to that process.

## ARC-002

The backend language is Go.

## ARC-003

The frontend language is TypeScript. The frontend framework is Svelte 5.
The web shell uses SvelteKit.

## ARC-004

The database is PostgreSQL.

## ARC-005

The repository holds four kinds of module:

| Path        | Kind   | Rule                                     |
| ----------- | ------ | ---------------------------------------- |
| `apps/hub`  | Host   | The only process that loads a plugin.    |
| `libs/*`    | Shared | A plugin and the hub can both import it. |
| `plugins/*` | Plugin | Builds to a shared object. See `PLG`.    |
| `apps/deck` | Shell  | The web application. Not a Go module.    |

**Why:** The direction of the dependencies stays clear.

## ARC-006

`go.work` lists every Go module.
Each Go module declares the same Go version as `go.work`.

**Why:** The hub and the plugins must compile with the same toolchain.
See `BLD-002`.

## ARC-007

`pnpm-workspace.yaml` lists every TypeScript package.

## ARC-008

The hub does not know the name of a plugin.
The hub does not hold a rule that applies to one plugin only.

**Why:** The hub is a platform. A special case in the hub breaks the platform claim.

## ARC-009

The deployment unit is a container image. The Helm chart in `chart/` deploys it.

## ARC-010

The hub runs as two processes from one binary:

| Command             | Runs                                    |
| ------------------- | --------------------------------------- |
| `maroid serve http` | The HTTP API and the Telegram webhook.  |
| `maroid worker`     | The cron jobs and the MQTT subscribers. |

Both processes load the same plugins. The chart deploys them separately.

**Why:** A request and a background job scale differently and fail independently.

## Retired identifiers

This file has no retired identifier.
