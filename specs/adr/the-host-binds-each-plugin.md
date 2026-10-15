---
id: ADR-0019
title: The host binds each plugin to its declared metadata
type: adr
status: accepted
created: 2026-10-15
updated: 2026-10-15
decided: 2026-10-15
changes: [PLG-002, PLG-013, DAT-004, LOG-004, GLO-host, GLO-declared-metadata]
supersedes:
superseded_by:
---

# The host binds each plugin to its declared metadata

## Context

One host serves every plugin. The database, the plugin secrets, the settings, the
notifications, and the page addresses each take a plugin identifier from the caller.
Each plugin builds the bound form in its constructor, from an identifier that it names
itself. A wrong identifier reaches the data of a different plugin, and the hub does not
detect it. The host also hands every plugin the pool of the hub, and `DAT-004` forbids
its use with no check. `PLGBIND` gives the requirements.

The hub learns the metadata of a plugin from a method, after the constructor ran. The
constructor needs the identifier earlier, so each plugin keeps it in a package variable
as well. The method computes the value on each call, and it can change after the load.

A Go shared object runs in the memory of the hub. With the package `unsafe` it reads and
writes any value of the process. No binding inside the process stops a plugin that
attacks on purpose.

## Decision

A plugin exports its metadata as a symbol, and declares no metadata method. The hub reads
the symbol before it calls `New`, checks it, keeps one copy for the life of the process,
and builds one bound host for the plugin from that copy. The bound host offers no
service that takes a plugin identifier, and no database pool.

A loaded plugin is trusted code. The binding prevents a mistake. It is no defense against
an attack.

## Rationale

The metadata comes from one place, so no check compares two copies. The hub refuses a
plugin with a wrong name, a wrong identifier, or a wrong API version before any code of
the plugin runs. The hub knows the version of the plugin before it builds the logger, so
it adds every attribute of `LOG-004` itself. The hub builds each bound service, so a
plugin author writes no binding and cannot write a wrong one.

The trust statement keeps a reader from taking the binding for a sandbox. An isolated
plugin needs a separate process or a WebAssembly module. The services of the bound host
take and return plain values, and the metadata is a plain value, so a later move out of
the process keeps their shape.

## Alternatives

| Alternative                                            | Why we did not select it                                                                                   |
| ------------------------------------------------------ | ---------------------------------------------------------------------------------------------------------- |
| A symbol for the identifier, and the metadata method   | Two copies of the identifier, and a check that compares them. The method still runs after the constructor. |
| The identifier in the configuration entry              | A second copy of the name, and the operator can set a wrong one.                                           |
| The identifier in the context of each call             | The plugin builds the context that it passes. A service bound at construction needs no check on each call. |
| A database account for each plugin                     | A real limit on the database. It needs a pool and a secret for each plugin. A later decision.              |
| A plugin in a separate process or a WebAssembly module | The real isolation. It changes every capability. A later decision, after a trial port of one plugin.       |

## Consequences

`PLG-002` reads: "A plugin exports two symbols. `Meta` has the type `pluginapi.Metadata`.
`New` has the type `pluginapi.Constructor`. The hub reads `Meta` first, and keeps that
copy for the life of the process."

`PLG-013` is new: "A loaded plugin is trusted code. It shares the memory of the hub.
The bound host prevents a mistake, not an attack."

`DAT-004` reads: "A plugin reaches the database through the `pluginapi.PluginDB` that its
host gives. The host gives no `*sqlx.DB`. `PluginDB` sets the `search_path` to the schema
of the plugin and to `public`."

`LOG-004` reads: "The hub adds `plugin`, `plugin_version`, and `plugin_api_version` to the
logger that it gives a plugin."

`GLO-host` reads: "The `pluginapi.Host` interface. The hub builds one for each plugin, and
each of its services acts for the plugin of the declared metadata. It is the only path
from a plugin to an external resource."

`GLO-declared-metadata` is new: "The metadata that a plugin exports as a symbol. The hub
reads it before it builds the plugin, and keeps that copy."

- Positive: a plugin passes no identifier, so a copied line cannot reach a different
  plugin. `DAT-004` becomes a fact of the contract. The hub checks the metadata before
  any code of the plugin runs.
- Negative: every plugin moves its metadata to a symbol and changes its constructor.
  The interface `pluginapi.Plugin` loses its one method. Every plugin rebuilds (`BLD-004`).
- Work that follows: `PLGBIND` stage 2 and stage 3. A trial port of one plugin to a
  WebAssembly module, before a decision on isolation.
