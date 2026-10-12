---
id: ADR-0016
title: A widget declares a permission and follows the enablement
type: adr
status: accepted
created: 2026-10-12
updated: 2026-10-12
decided: 2026-10-12
changes: [SEC-012, SEC-013]
supersedes:
superseded_by:
---

# A widget declares a permission and follows the enablement

## Context

`WIDGET` adds a fourth kind of entry that acts in a workspace: a widget, whose data
the plugin reads for the acting user. `SEC-013` names three kinds of entries that
declare a permission: a route, an MCP tool, and a Telegram command. `SEC-012` names
how each surface of a disabled plugin is absent. Neither names a widget.
`WIDGET-FR-008` and `WIDGET-FR-009` need both rules for a widget.

## Decision

A widget declares one permission, as a route does, and the hub checks it before it
asks the plugin for the data. A widget of a plugin that is disabled in the acting
workspace is absent from the list of the home page, and a read of it answers `not-found`.

## Rationale

A widget reads the same records as a route of its plugin. A check that differs
between the two would let a viewer read on the home page what the route denies.

## Alternatives

| Alternative | Why we did not select it |
| ----------- | ------------------------ |
| The widget reads through a route of the plugin, and the route checks. | The hub then calls its own HTTP routes, and a plugin with no route has no widget. |
| No permission. Every member sees every widget. | A role that cannot read the plants would read them on the home page. |

## Consequences

- Positive: one rule covers every entry that acts in a workspace.
- Negative: none.
- Work that follows: `SEC-012` adds the sentence of the widget. `SEC-013` adds the
  widget to the list of the entries that declare one permission.
