---
id: ADR-0012
title: A plugin page route takes parameter segments
type: adr
status: accepted
created: 2026-10-11
updated: 2026-10-11
decided: 2026-10-11
changes: [UI-005, UI-008]
supersedes:
superseded_by:
---

# A plugin page route takes parameter segments

## Context

`UI-005` gives a remote a `routes` object whose keys are fixed paths, and asks the
plugin to declare the same paths in `UIManifest.Routes` for the navigation. The deck
opens a page by an exact match of the path. A page of one record therefore has no
address, and `GARDEN-DD-010` put the plant view inside the plant list.
`PLROUTE-FR-001` to `PLROUTE-FR-012` ask for an address with a value, and for a
trail that names the record.

## Decision

A key of `routes` holds parameter segments of the form `:name`, the deck gives the
page the values as the prop `params`, and `PluginHost` gains `setTitle` for the last
link of the trail. `UIManifest.Routes` lists the navigation entries only, and the hub
refuses an entry that holds a parameter segment.

## Rationale

The change adds to the contract: a remote with fixed keys alone keeps working, and
no plugin changes until it needs a parameter. A whole-segment parameter covers every
record page that the plugins show today, and keeps the match to one comparison for
each segment.

## Alternatives

| Alternative                                      | Why we did not select it                                                                                                                  |
| ------------------------------------------------ | ----------------------------------------------------------------------------------------------------------------------------------------- |
| A query parameter, `/plants?plant=<id>`          | The deck matches the path alone, so it needs the same change in the loader, and the trail still cannot tell the page apart from the list. |
| A catch-all route that parses the rest itself    | Every plugin repeats the parsing, and the deck cannot build a trail or mark the navigation for a path it does not understand.             |
| A page title that the deck reads from the record | The deck knows no resource of a plugin, and `ARC-008` forbids a rule that names one.                                                      |

## Consequences

- Positive: every plugin gets an address for each record page, which a reload, a
  link, and the back button of the browser keep.
- Negative: `UIManifest.Routes` and the keys of `routes` stop being the same set,
  so a reader of a remote checks two lists.
- Work that follows: `UI-005` allows a parameter segment in a key of `routes` and
  limits `UIManifest.Routes` to the navigation entries. `UI-008` adds `setTitle` to
  `PluginHost` and the prop `params` to a mounted page. `PLROUTE` builds the change,
  and `GARDEN-DD-010` retires.
