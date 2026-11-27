---
id: ADR-0030
title: A client accepts the media type that it reads
type: adr
status: accepted
created: 2026-11-28
updated: 2026-11-28
decided: 2026-11-28
changes: [EXT-002]
supersedes:
superseded_by:
---

# A client accepts the media type that it reads

## Context

`EXT-002` says: "The client sets `Accept: application/json` once, at construction."
Each external service so far answered with JSON. `PARKMAP-FR-016` adds a map source
that answers with PNG images. A client that asks a map source for JSON asks for a
media type that the source never sends.

## Decision

`EXT-002` reads: "The client sets the `Accept` header to the media type that it reads,
once, at construction."

## Rationale

The rule exists so that each request states the answer it expects, in one place. The
new text keeps that aim. Every JSON client already obeys it.

## Alternatives

| Alternative                                       | Why we did not select it                                                                     |
| ------------------------------------------------- | -------------------------------------------------------------------------------------------- |
| `Accept: application/json` for the map source too | The header then names a media type that the source never sends. A strict source answers 406. |
| No `Accept` header on the map source              | A request that states no expected answer breaks the aim of `EXT-002`.                        |

## Consequences

- Positive: a client of an image service obeys `EXT-002`.
- Negative: none. Every JSON client stays as it is.
- Work that follows: correct `EXT-002` after the acceptance.
