---
id: ADR-0020
title: The root of the address of a plugin is the plugin detail page
type: adr
status: accepted
created: 2026-10-16
updated: 2026-10-16
decided: 2026-10-16
changes: [UI-005]
supersedes:
superseded_by:
---

# The root of the address of a plugin is the plugin detail page

## Context

`/w/<workspace>/plugins/<plugin>` opens the pages of the plugin today. No plugin
declares a page there, so the address answers the not-found page of
`PLROUTE-FR-007`. `PLGINFO-FR-012` gives the address to the plugin detail page, and
asks Maroid to refuse a plugin that declares a page route at the root.

## Decision

`UIManifest.Routes` holds no entry at the root of the address of the plugin, and the
hub refuses a plugin whose entry is `/` or empty. The deck serves the plugin detail
page at the root. A key of `routes` at the root is not checked, and never opens.

## Rationale

The root answers a dead end today, so the change breaks no plugin. The hub reads the
navigation entries at the load and already refuses an entry with a parameter
segment. The keys of `routes` live in the remote, which the hub never reads, and the
owner keeps the check to the hub.

## Alternatives

| Alternative                                        | Why we did not select it                                                                                            |
| -------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------- |
| An address of its own, such as `<plugin>/overview` | A plugin can declare `/overview` too, so the word needs the same reservation, and the address grows by one segment. |
| The deck also refuses a remote with a key at `/`   | The owner chose the hub check alone. A key at `/` is a defect that the author sees at once: the page never opens.   |

## Consequences

- Positive: each plugin has one address that describes it, and the trail links the
  name of the plugin to it.
- Negative: a key of `routes` at `/` passes the load and never opens.
- Work that follows: `UI-005` forbids an entry at the root. `PLGINFO` builds the
  check and the page.
