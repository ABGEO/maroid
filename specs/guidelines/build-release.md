---
id: BLD
title: Build and release
type: guideline
status: active
created: 2026-09-11
updated: 2026-09-17
scope: [apps/, libs/, plugins/, .docker/, chart/]
related: [ARC, PLG, UI, GO, TS]
---

# Build and release

## BLD-001

Run the linter and the build of each artifact that the change touches.
Run them for each artifact that depends on a changed artifact.
Every command must pass before the work is complete.

Do not run a repository-wide script. Run the command for the artifact.

| Artifact you changed  | Lint or check                                                            | Build                                                                                  |
| --------------------- | ------------------------------------------------------------------------ | -------------------------------------------------------------------------------------- |
| `plugins/<name>` (Go) | `cd plugins/<name> && golangci-lint run ./...`                           | `CGO_ENABLED=1 go build -buildmode=plugin -o build/plugins/<name>.so ./plugins/<name>` |
| `plugins/<name>/ui`   | None today. The build reports a type error.                              | `pnpm --filter ./plugins/<name>/ui build`                                              |
| `apps/hub`            | `cd apps/hub && golangci-lint run ./...`                                 | `go build ./apps/hub/...`                                                              |
| `apps/deck`           | `pnpm --filter @maroid/deck lint` and `pnpm --filter @maroid/deck check` | `pnpm --filter @maroid/deck build`                                                     |
| `libs/<name>` (Go)    | `cd libs/<name> && golangci-lint run ./...`                              | `cd libs/<name> && go build ./...`                                                     |
| `libs/plugin-sdk`     | `pnpm --filter @maroid/plugin-sdk check`                                 | `pnpm --filter @maroid/plugin-sdk build`                                               |
| `libs/api-client`     | `pnpm --filter @maroid/api-client check`                                 | None today.                                                                            |
| `libs/theme`          | `pnpm --filter @maroid/theme lint`                                       | None today.                                                                            |
| `apps/gate`           | None today. The build reports a type error.                              | `pnpm --filter @maroid/gate build`                                                     |

**Why:** A repository-wide script hides which artifact failed. It also reports a
failure in an artifact that the change did not touch.

## BLD-002

The hub and the plugins build with the same Go toolchain and with the same versions
of the shared modules.

**Why:** The Go plugin mechanism compares the build identifiers of the packages.
A different version stops the load of the shared object.

## BLD-003

For a plugin that has a user interface, build the user interface before the
shared object.

```
pnpm --filter ./plugins/<name>/ui build
CGO_ENABLED=1 go build -buildmode=plugin -o build/plugins/<name>.so ./plugins/<name>
```

**Why:** `go:embed all:ui/dist` needs the directory `ui/dist` before the Go build starts.

## BLD-004

A change in `libs/pluginapi` touches every plugin.
Build every plugin after such a change, and confirm that the hub loads each one.

**Why:** `BLD-002` fails at runtime, not at build time. Only a load proves the change.

## Retired identifiers

This file has no retired identifier.
