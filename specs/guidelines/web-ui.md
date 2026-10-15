---
id: UI
title: The web user interface
type: guideline
status: active
created: 2026-09-11
updated: 2026-10-16
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
The object maps a page route to a `RouteMounter` function.
A segment of the form `:name` is a parameter segment. It matches one non-empty
segment of the address, and the page receives the value under `name`.
`UIManifest.Routes` lists the navigation entries. An entry holds no parameter
segment, and the hub refuses a plugin whose entry holds one. `ADR-0012`.
An entry is not at the root of the address of the plugin, where the deck serves the
plugin detail page. The hub refuses a plugin whose entry is `/` or empty. `ADR-0020`.

## UI-006

A plugin user interface imports `@maroid/plugin-sdk` for a shared type and a shared helper.
A plugin user interface does not import a module from `apps/deck`.

**Why:** The shell can change. The plugins do not break.

## UI-007

A plugin user interface reaches the hub through `@maroid/api-client`.
It does not build a request URL itself.

## UI-008

`@maroid/plugin-sdk` gives a plugin user interface a `PluginHost` with:

| Member     | Gives                                                                                    |
| ---------- | ---------------------------------------------------------------------------------------- |
| `user`     | The current user, or null.                                                               |
| `api`      | A client already scoped to the mount prefix of `API-004`, for the acting workspace.      |
| `can`      | Whether the role of the person holds a permission of the plugin. `SEC-013`.              |
| `href`     | A deck URL for a plugin-relative path.                                                   |
| `navigate` | Client-side navigation to a plugin-relative path.                                        |
| `setTitle` | Names the page in the last link of the trail. The deck clears it when the page unmounts. |

The deck mounts a page with the props `host` and `params`. `params` maps the name of
each parameter segment of the matched route to its decoded value. `ADR-0012`.

`PluginHost` is to a plugin user interface what `pluginapi.Host` is to a plugin.

## UI-009

A plugin user interface declares the Module Federation name as the plugin
identifier, exposes `./entry`, and writes a manifest.
`./entry` exports the `routes` object that `UI-005` gives.

## Retired identifiers

This file has no retired identifier.
