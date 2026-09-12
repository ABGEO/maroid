---
id: UI
title: The web user interface
type: guideline
status: active
created: 2026-09-11
updated: 2026-09-11
scope: [apps/deck/, plugins/*/ui/, libs/plugin-sdk/, libs/api-client/]
related: [ARC, PLG, TS, BLD, API]
---

# The web user interface

This guideline holds the contract between the web shell and a plugin user interface.
`TS` holds the code style of the same files.

## UI-001

`apps/deck` is the only web shell. A plugin does not serve a full page.

## UI-002

A plugin user interface is a Module Federation remote.
It builds with Vite. The build output goes to `plugins/<name>/ui/dist`.

## UI-003

The plugin embeds the build output with `go:embed all:ui/dist`.
The plugin returns the assets in `UIManifest`.

**Why:** One artifact holds the backend and the frontend of the plugin.
The versions cannot separate.

## UI-004

The hub serves the assets of a plugin at `/plugins/{id}/ui/`.
The manifest of the remote is `/plugins/{id}/ui/mf-manifest.json`.

## UI-005

A plugin user interface exports an object with the name `routes`.
The object maps a path to a `RouteMounter` function.
The plugin declares the same paths in `UIManifest.Routes` for the navigation.

## UI-006

A plugin user interface imports `@maroid/plugin-sdk` for a shared type and a shared helper.
A plugin user interface does not import a module from `apps/deck`.

**Why:** The shell can change. The plugins do not break.

## UI-007

A plugin user interface reaches the hub through `@maroid/api-client`.
It does not build a request URL itself.

## UI-008

`@maroid/plugin-sdk` gives a plugin user interface a `PluginHost` with:

| Member     | Gives                                             |
| ---------- | ------------------------------------------------- |
| `user`     | The current user, or null.                        |
| `api`      | A client already scoped to `/plugins/{id}/api`.   |
| `href`     | A deck URL for a plugin-relative path.            |
| `navigate` | Client-side navigation to a plugin-relative path. |

`PluginHost` is to a plugin user interface what `pluginapi.Host` is to a plugin.

## UI-009

A plugin user interface declares the Module Federation name as the plugin
identifier, exposes `./entry`, and writes a manifest.
`./entry` exports the `routes` object that `UI-005` gives.

## Retired identifiers

This file has no retired identifier.
