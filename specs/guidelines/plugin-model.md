---
id: PLG
title: The plugin model
type: guideline
status: active
created: 2026-09-11
updated: 2026-10-05
scope: [plugins/, apps/hub/internal/plugin/, libs/pluginapi/]
related: [ARC, DAT, UI, BLD]
---

# The plugin model

This guideline holds the contract between the hub and a plugin.

## PLG-001

A plugin builds to a Go shared object with `CGO_ENABLED=1` and `-buildmode=plugin`.
The output goes to `build/plugins/<name>.so`.
The hub opens the file at runtime.

**Why:** A new plugin needs no change in the hub and no new build of the hub.

## PLG-002

A plugin exports one symbol with the name `New`.
The type of the symbol is `pluginapi.Constructor`.

**Why:** The loader looks up one symbol. The contract stays small.

## PLG-003

A plugin has an identifier with the form `dev.maroid.<name>`.
The identifier uses lowercase letters, digits, and the hyphen. A dot separates each segment.

## PLG-004

A plugin declares `pluginapi.APIVersion` in the metadata.
The hub refuses a plugin that declares a different version, and reports the reason.

**Why:** A plugin that a different API version built can fail in an unclear way.
The hub must fail early and with a clear message.

## PLG-005

A plugin gets a capability when it implements the related interface in `libs/pluginapi`.
The hub finds the capability with a type assertion in a registrar.
A plugin does not declare a capability in a manifest file.

**Why:** The Go compiler checks the contract. A manifest cannot.

## PLG-006

A new capability needs three parts:

1. An interface in `libs/pluginapi`.
2. A registry in `apps/hub/internal/registry`.
3. A registrar in `apps/hub/internal/plugin/registrar`.

The registrar goes into the list in `apps/hub/internal/plugin/loader`.

## PLG-007

A plugin reaches every external resource through the `pluginapi.Host` interface.
A plugin does not import a package from `apps/hub`.

**Why:** The hub can change its internal structure. The plugins do not break.

## PLG-008

A plugin does not import a package from another plugin.
Two plugins share code only through a package in `libs/`.

**Why:** A plugin is a separate module and a separate binary.
A direct import makes the release order fixed.

## PLG-009

A plugin does not hold a state between the process starts.
Durable state goes into the schema of the plugin. See `DAT-001`.

## PLG-010

The configuration lists every plugin to load. An entry gives a path, a flag that
loads it, and a configuration map. The hub skips an entry whose flag is off.

Loading is not enablement. A loaded plugin serves only the workspaces that enable
it. See `SEC-012`.

The hub loads the plugins before it builds the command tree.

**Why:** A plugin adds a CLI command, so the commands cannot exist first.

## PLG-011

A registry rejects a second registration under an identifier that it already holds,
and returns a sentinel error.

**Why:** Two plugins with one identifier would replace each other in silence.

## PLG-012

A plugin declares a name in `pluginapi.Metadata.Name`. The name holds 1 to 64
characters after the hub trims the white space at each end. A plugin may declare a
description in `pluginapi.Metadata.Description`, of at most 280 characters.

The hub refuses to load a plugin whose name or description breaks a limit, and the
error names the plugin and the field.

**Why:** Every client shows a plugin by its name. A name that only the user interface
declares leaves the other plugins nameless.

## Retired identifiers

This file has no retired identifier.
