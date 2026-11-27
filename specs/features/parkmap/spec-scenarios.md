---
id: PARKMAP
title: The scenarios of the map of the nearby parking lots
type: spec
status: approved
created: 2026-11-28
updated: 2026-11-28
approved_by: Temuri
approved_on: 2026-11-28
constrained_by: [TST, TRC]
requirements: features/parkmap/requirements.md
---

# Specification: The scenarios of the map of the nearby parking lots

A unit scenario of the search runs `NearbyLots` against a fake `APIClientService`. A unit
scenario of the map runs `LotMap` against an `httptest` server that plays the map source
and answers a plain white PNG tile of 256 pixels. A unit scenario of the step runs
`SelectLot` with `fakeBot`, a fake `LotFinder`, and a fake `LotMapRenderer`, unless it
names the real one. "The location" is `41.708870, 44.764440`. "The fixture" is the
answer of the parking service for that location in `service/testdata/paliashvili.json`:
`A306` and `A302`.

## The search

### `PARKMAP-SC-001`

**Verifies:** `PARKMAP-FR-001`
**Layer:** unit

**Given** the fake service answers two lots: one at 8 m from the location, one at 25 m.
**When** `FindNearby` runs for the location.
**Then** it returns both lots.

### `PARKMAP-SC-002`

**Verifies:** `PARKMAP-FR-001`, `PARKMAP-FR-002`
**Layer:** unit

**Given** the fake service answers one lot whose outline contains the location.
**When** `FindNearby` runs.
**Then** it returns the lot with the distance 0.

### `PARKMAP-SC-003`

**Verifies:** `PARKMAP-FR-001`
**Layer:** unit

**Given** the fake service answers one lot whose nearest point lies 35 m from the location, in a corner of the box.
**When** `FindNearby` runs.
**Then** it returns no lot and no error.

### `PARKMAP-SC-004`

**Verifies:** `PARKMAP-FR-001`
**Layer:** unit

**Given** the fake service records its arguments.
**When** `FindNearby` runs for the location.
**Then** `top - lat` and `lat - bottom` equal `0.00026980` within `1e-7`, and
`right - lng` and `lng - left` equal `0.00036140` within `1e-7`.

### `PARKMAP-SC-005`

**Verifies:** `PARKMAP-FR-002`
**Layer:** unit

**Given** a lot of 60 m whose end lies 5 m from the location, and whose centroid lies 35 m from it.
**When** `FindNearby` runs.
**Then** it returns the lot with a distance of 5 m within 0.1 m.

### `PARKMAP-SC-006`

**Verifies:** `PARKMAP-FR-003`
**Layer:** unit

**Given** the fake service answers 9 lots within 30 m, at 3, 6, 9, and up to 27 m.
**When** `FindNearby` runs.
**Then** it returns 6 lots: the lots at 3 m to 18 m.

### `PARKMAP-SC-007`

**Verifies:** `PARKMAP-FR-004`
**Layer:** unit

**Given** the lots `A306` at 4 m, `A302` at 9 m, and `B12`, `A102`, and `A12`, each at 12 m.
**When** `FindNearby` runs.
**Then** the order is `A306`, `A302`, `A12`, `A102`, `B12`.

### `PARKMAP-SC-008`

**Verifies:** `PARKMAP-FR-005`, `PARKMAP-INV-001`
**Layer:** unit

**Given** the fake finder returns `A306` and `A302`.
**When** `OnEnter` runs with the location.
**Then** `fakeBot` records `SendPhoto`, then the `SendMessage` of the lot list. The renderer
received `A306` and `A302` in that order. The keyboard holds `A306`, then `A302`, then
"Enter manually" and "Cancel".

### `PARKMAP-SC-009`

**Verifies:** `PARKMAP-FR-005`
**Layer:** unit

**Given** the fake finder returns one lot.
**When** `OnEnter` runs.
**Then** `fakeBot` records one `SendPhoto`, then the lot list with one lot button.

### `PARKMAP-SC-010`

**Verifies:** `PARKMAP-FR-013`
**Layer:** unit

**Given** the fake finder returns no lot.
**When** `OnEnter` runs.
**Then** the renderer records no call. `fakeBot` records no `SendPhoto`. The lot list
holds "No parking lots found near your location." and the two buttons of today.

## When the map fails

### `PARKMAP-SC-011`

**Verifies:** `PARKMAP-FR-014`
**Layer:** unit

**Given** the real `LotMap`, an empty cache, and a map source that answers 503.
**When** `OnEnter` runs with the fixture.
**Then** `fakeBot` records no `SendPhoto`. The lot list holds the same text as in `PARKMAP-SC-008`.

### `PARKMAP-SC-012`

**Verifies:** `PARKMAP-FR-014`
**Layer:** unit

**Given** the real `LotMap`, `map.timeout` of 300 ms, and a map source that answers after 2 s.
**When** `OnEnter` runs with the fixture.
**Then** `fakeBot` records no `SendPhoto`. It records the lot list less than 500 ms after the start of `OnEnter`.

### `PARKMAP-SC-013`

**Verifies:** `PARKMAP-FR-014`
**Layer:** unit

**Given** the fake finder returns `ErrRequestFailed`.
**When** `OnEnter` runs.
**Then** `OnEnter` returns an error that matches `ErrRequestFailed`. The renderer records no call.

### `PARKMAP-SC-014`

**Verifies:** `PARKMAP-FR-014`
**Layer:** unit

**Given** the fake finder returns one lot, and `fakeBot.SendPhoto` returns an error.
**When** `OnEnter` runs.
**Then** `OnEnter` returns no error. `fakeBot` records the lot list.

### `PARKMAP-SC-015`

**Verifies:** `PARKMAP-FR-015`
**Layer:** unit

**Given** the case of `PARKMAP-SC-011`, with a logger that writes JSON to a buffer.
**When** `OnEnter` runs.
**Then** the buffer holds one record at `WARN` with the message `parking map not sent`, an
`error` attribute that names the status 503, and `lots` of 2. The buffer holds neither
`41.70887` nor `44.76444`.

## The map

### `PARKMAP-SC-016`

**Verifies:** `PARKMAP-FR-006`, `PARKMAP-FR-008`
**Layer:** unit

**Given** the real `LotMap` and the fixture.
**When** `Render` runs.
**Then** the pixel on the midpoint of the longest edge of `A306` carries `#2563EB`, and the
one of `A302` carries `#16A34A`. A pixel inside each outline, clear of its label, differs from white.

### `PARKMAP-SC-017`

**Verifies:** `PARKMAP-FR-007`
**Layer:** unit

**Given** an outline of 40 m on an east and west axis, and a pin rectangle over its centroid.
**When** the label placement runs.
**Then** the label sits at 25 % of the axis, its text is the lot number, and its rectangle meets
no pin. With no pin over the centroid, the label sits at the centroid. With a pin over all
three points, the label sits at the centroid.

### `PARKMAP-SC-018`

**Verifies:** `PARKMAP-FR-009`
**Layer:** unit

**Given** the real `LotMap` and the fixture.
**When** `Render` runs.
**Then** the pixel 18 pixels above the location carries `#DC2626`.

### `PARKMAP-SC-019`

**Verifies:** `PARKMAP-FR-010`
**Layer:** unit

**Given** the fixture, and a lot of 90 m whose end lies 10 m from the location.
**When** the zoom and the center are computed.
**Then** each outline point and the pin lie at least 48 pixels inside each edge of the image of 800 by 600.

### `PARKMAP-SC-020`

**Verifies:** `PARKMAP-FR-011`
**Layer:** unit

**Given** the fixture, `map.max_zoom` of 19.
**When** the zoom is computed.
**Then** the zoom is 19. For two points 400 m apart, the zoom is the highest value at which
both fit, and the next value does not fit.

### `PARKMAP-SC-021`

**Verifies:** `PARKMAP-FR-012`
**Layer:** unit

**Given** the real `LotMap`, `map.attribution` of `© OpenStreetMap contributors`, and the fixture.
**When** `Render` runs.
**Then** the draw of the attribution received that text. The pixel 2 pixels above the bottom
left corner differs from the pixel of the tile under it.

## The map source

### `PARKMAP-SC-022`

**Verifies:** `PARKMAP-FR-016`
**Layer:** unit

**Given** a configuration map with no `map` key.
**When** `DecodeAndValidateConfig` runs.
**Then** `Map.TileURL` is `https://tile.openstreetmap.org/{z}/{x}/{y}.png`, `Map.MaxZoom` is 19,
and `Map.Timeout` is 3 s. With `map.tile_url` set to the test server, `Render` requests each
tile from the test server. With `map.timeout` of 5 s, the validation fails.

### `PARKMAP-SC-023`

**Verifies:** `PARKMAP-FR-017`, `PARKMAP-INV-002`
**Layer:** unit

**Given** a map source that records each request.
**When** `Render` runs with the fixture, in a context that carries a session token for `APIClient`.
**Then** each request has `User-Agent: Maroid/1.0 (parking map)` and `Accept: image/png`. No
request has `Authorization` or `Cookie`. Each path has the form `/{z}/{x}/{y}.png`, with no query.

## The image in a conversation

### `PARKMAP-SC-024`

**Verifies:** `PARKMAP-FR-018`
**Layer:** unit

**Given** the host wrapper on a telego bot whose API server is an `httptest` server.
**When** a plugin calls `SendPhoto` with a PNG of 3 bytes.
**Then** the server receives one `sendPhoto` request with a multipart part `photo` that holds the 3 bytes.

### `PARKMAP-SC-025`

**Verifies:** `PARKMAP-FR-019`
**Layer:** unit

**Given** an update in chat 7, thread 3, and direct messages topic 11. The fake finder returns one lot.
**When** `OnEnter` runs.
**Then** the `SendPhoto` parameters carry the chat 7, `MessageThreadID` 3, and `DirectMessagesTopicID` 11.

## The limits

### `PARKMAP-SC-026`

**Verifies:** `PARKMAP-NFR-001`
**Layer:** unit

**Given** a fake service that answers after 900 ms, the real `LotMap` with `map.timeout` of 3 s,
and a map source that never answers.
**When** `OnEnter` runs.
**Then** `fakeBot` records the lot list less than 5 s after the start of `OnEnter`.

### `PARKMAP-SC-027`

**Verifies:** `PARKMAP-NFR-002`
**Layer:** unit

**Given** the real `LotMap` with a cache that holds every tile of the fixture map.
**When** `Render` and the `SendPhoto` of `fakeBot` run.
**Then** both end less than 1 s after the start of `Render`. The map source records no request.

### `PARKMAP-SC-028`

**Verifies:** `PARKMAP-NFR-003`
**Layer:** unit

**Given** the real `LotMap`, a map source that counts requests by path, and the cache clock at day 0.
**When** `Render` runs with the fixture at day 0, at day 6, and at day 7 and 1 second.
**Then** after day 6 each path has 1 request. After day 7 and 1 second each path has 2.

### `PARKMAP-SC-029`

**Verifies:** `PARKMAP-NFR-004`
**Layer:** unit

**Given** a `TileCache` with the limit of 50 MB.
**When** the test puts 60 tiles of 1 MB, and reads the first tile after the tenth put.
**Then** the sum of the held bytes is at most 50 MB. The first tile stays. The second tile is gone.

## The live check

### `PARKMAP-SC-030`

**Verifies:** `PARKMAP-FR-005`, `PARKMAP-FR-007`, `PARKMAP-INV-001`
**Layer:** manual

**Given** the hub with the parking plugin, and a member with a connection.
**When** the member runs `/start` on a phone next to a parking lot in Tbilisi, and shares the location.
**Then** the chat shows the map, then the lot list. Each number on the map has a button. The
pin stands on the street where the phone is.

## Retired identifiers

This file has no retired identifier.
