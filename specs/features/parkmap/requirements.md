---
id: PARKMAP
title: The map of the nearby parking lots
type: requirements
status: approved
created: 2026-11-28
updated: 2026-11-28
approved_by: Temuri
approved_on: 2026-11-28
constrained_by: [TG, PLG, BLD, EXT, CFG, ARC]
---

# Requirements: The map of the nearby parking lots

## 1. Problem

A member who starts a parking session shares their location. The bot answers with
buttons that carry lot numbers only, for example "A302" and "A306". Two parking lots
often face each other on one street. The member cannot tell which number belongs to
the curb next to the car, and a wrong choice charges the wrong parking lot.

The bot searches about 10 m around the location. It often finds no parking lot,
although one lies within 30 m.

## 2. Users

- A member who parks: sees the nearby parking lots and their own position on one
  image, and chooses the lot from it.

## 3. Out of scope

- A map for a lot number that the member types.
- A map in a command other than the one that starts a parking session, or on a page.
- A choice of a parking lot by a press on the map.
- The removal of the map from the chat after the choice or the cancel.
- A satellite image.
- A copy of the map that stays after a restart of the hub (`PLG-009`).
- The price, the free places, or the hours of a parking lot.

## 4. Definitions

`GLO-parking-service` applies as it stands. "Parking lot" keeps the meaning of
`PARKHIST`, section 4.

| Term            | Meaning                                                                                     |
| --------------- | ------------------------------------------------------------------------------------------- |
| shared location | The location that the member shares in the conversation that starts a parking session.      |
| outline         | The shape of a parking lot on the ground, as the parking service gives it.                  |
| lot list        | The message of that conversation with one button for each offered parking lot.              |
| offered lot     | A parking lot that the lot list carries a button for.                                       |
| map             | An image of the streets around the shared location, with the offered lots and a pin on it.  |
| map source      | The external service that gives the images of the streets, in square parts at fixed scales. |
| map tile        | One square part of the streets at one scale, as the map source gives it.                    |

## 5. Functional requirements

### The search

### `PARKMAP-FR-001`

The plugin must offer each parking lot whose outline lies within 30 m of the shared location.

**Why:** A search of 10 m often finds no parking lot next to a street that has one.

**Examples:**

- Normal case: one parking lot lies 8 m away, one 25 m away. The lot list offers both.
- Limit case: the shared location lies inside an outline. That parking lot lies at 0 m.
- Unwanted case: the nearest parking lot lies 35 m away. The lot list offers none.

### `PARKMAP-FR-002`

The plugin must measure the distance from the shared location to the nearest point of the outline.

**Why:** A parking lot can be 60 m long. Its center can be far from a car that stands at its end.

### `PARKMAP-FR-003`

The lot list must offer at most 6 parking lots.

**Examples:**

- Limit case: 9 parking lots lie within 30 m. The lot list offers the 6 nearest.

### `PARKMAP-FR-004`

The lot list must order the offered lots by their distance from the shared location, the nearest first.

**Examples:**

- Normal case: A306 lies at 4 m, A302 at 9 m. A306 comes first.
- Limit case: two parking lots lie at the same distance. The lower number comes first.

### The map

### `PARKMAP-FR-005`

The bot must send the map before the lot list when the plugin offers one or more parking lots.

**Examples:**

- Normal case: the plugin offers A302 and A306. The chat shows the map, then the lot list.
- Limit case: the plugin offers one parking lot. The chat shows the map with one outline.

### `PARKMAP-FR-006`

The map must show the outline of each offered lot.

### `PARKMAP-FR-007`

The map must show the number of each offered lot on its outline, as the button of the lot list shows it.

**Why:** The member matches the number on the map to the button.

### `PARKMAP-FR-008`

The map must show each offered lot in a color that no other offered lot on the map has.

**Why:** Two outlines on the two sides of one street are close. A color tells them apart.

### `PARKMAP-FR-009`

The map must show the shared location as a pin.

### `PARKMAP-FR-010`

The map must show each offered lot and the pin in full.

**Examples:**

- Normal case: the offered lots lie 4 m and 25 m away. Both outlines and the pin are in the image.
- Limit case: an outline of 60 m runs out of the area that the search covers. The map shows the full outline.

### `PARKMAP-FR-011`

The map must use the most detailed scale of the map source at which `PARKMAP-FR-010` holds.

**Why:** A parking lot is about 3 m wide. A less detailed scale shows it as a line.

### `PARKMAP-FR-012`

The map must name the source of the street data.

**Why:** The licence of the street data demands the attribution on each image.

### `PARKMAP-FR-013`

The bot must send no map when the plugin offers no parking lot.

**Why:** The member enters the number by hand or cancels. A map with no outline gives no choice.

### When the map fails

### `PARKMAP-FR-014`

The bot must send the lot list without the map when the plugin cannot make the map.

**Why:** The map helps the choice. The lot list is the choice. An outage of the map source must not stop a parking session.

**Examples:**

- Normal case: the map source answers with status 503. The chat shows the lot list only, with no note about the map.
- Limit case: the map source answers after the limit of `PARKMAP-NFR-001`. The chat shows the lot list only.
- Unwanted case: the parking service fails. The plugin makes no map, and the conversation reports the failure as it does today.

### `PARKMAP-FR-015`

The plugin must log each map that it cannot make, with the reason.

**Why:** The member sees no note. The log is the only trace of an outage of the map source.

### The map source

### `PARKMAP-FR-016`

The plugin must read the address of the map source from its configuration.

**Why:** The default map source limits heavy use. A second map source with the same tiles replaces it with no change to the code.

**Examples:**

- Normal case: the configuration sets no address. The plugin uses the default map source.

### `PARKMAP-FR-017`

The plugin must name Maroid as the client in each request to the map source.

**Why:** The usage policy of the default map source refuses a request that names no client.

### The image in a conversation

### `PARKMAP-FR-018`

Maroid must let a plugin send an image to the chat of the update that a step of a conversation answers.

**Why:** A plugin sends to a chat through the host only (`TG-007`), and the host sends text only.

### `PARKMAP-FR-019`

Maroid must send the image to the thread of the chat that the update came from.

**Examples:**

- Normal case: the member writes in a direct messages topic. The map arrives in the same topic.

## 6. Non-functional requirements

### `PARKMAP-NFR-001`

The bot must send the lot list less than 5 seconds after the hub receives the shared location,
when the parking service answers in less than 1 second.
Measure from the receipt of the update to the send of the lot list.

**Why:** A slow map source must not hold the member at the curb. `PARKMAP-FR-014` then sends the list alone.

### `PARKMAP-NFR-002`

The plugin must send the map less than 1 second after the answer of the parking service,
when the plugin holds every map tile that the map needs.
Measure from the answer of the parking service to the send of the map.

### `PARKMAP-NFR-003`

The plugin must request one map tile from the map source at most once in 7 days while the hub runs.

**Why:** The usage policy of the default map source demands a local copy. A member parks on the same streets each week.

### `PARKMAP-NFR-004`

The plugin must hold at most 50 MB of map tiles.
Measure the size of the map tiles that the plugin holds at one time.

**Why:** The copy lives in the memory of the hub (`PLG-009`).

## 7. Invariants

### `PARKMAP-INV-001`

The map shows the same parking lots that the lot list offers, no more and no fewer.

### `PARKMAP-INV-002`

No request to the map source carries the identity of a person, a parking account, or a session token.

**Why:** The map source learns the area of each map from the map tiles. It must learn nothing else.

## 8. Constraints from the guidelines

| Rule      | Guideline         | Effect on this feature                                                      |
| --------- | ----------------- | --------------------------------------------------------------------------- |
| `TG-005`  | Telegram          | The map is part of the step that offers the parking lots.                   |
| `TG-007`  | Telegram          | The plugin sends the map through the host, so the host gains an image.      |
| `ARC-008` | Architecture      | The image of the host names no plugin.                                      |
| `BLD-004` | Build and release | The change to the host interface touches every plugin.                      |
| `PLG-009` | Plugin model      | The copy of the map tiles lives while the hub runs, and a restart drops it. |
| `EXT-005` | External services | A failed map tile is an error. The plugin sends no map with a missing part. |
| `CFG-005` | Configuration     | The address of the map source comes from the configuration of the plugin.   |
| `CFG-007` | Configuration     | The address is a value of the installation, not of a person.                |

## 9. Open questions

| #   | Question                                              | Owner  | Answer                                                |
| --- | ----------------------------------------------------- | ------ | ----------------------------------------------------- |
| 1   | Does the map go in the message of the lot list?       | Temuri | No. It is a message of its own, before the lot list.  |
| 2   | How far does the search reach?                        | Temuri | 30 m, the nearest first, at most 6. `PARKMAP-FR-001`. |
| 3   | Does the bot send a map when the search finds no lot? | Temuri | No. `PARKMAP-FR-013`.                                 |
| 4   | Does the member see a note when the map fails?        | Temuri | No. The log records it. `PARKMAP-FR-015`.             |

## Retired identifiers

This file has no retired identifier.
