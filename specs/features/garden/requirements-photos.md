---
id: GARDEN
title: The photos of a plant
type: requirements
status: approved
created: 2026-11-18
updated: 2026-11-18
approved_by: Temuri
approved_on: 2026-11-18
constrained_by: [OWN, SEC, RES, DAT, PLG]
---

# Requirements: The photos of a plant

`requirements.md` holds the plants and their care. This file holds the photos of a plant:
the moments of its gallery and its profile picture. `FILES` holds the upload, the
variants, and the storage of each photo.

## Problem

A person who keeps a plant wants to see how it grew: the day of a repotting, the first
new leaf, the first flower. The care history holds a time and a note, and no picture.
A person who reads the list of plants knows a plant by its name alone.

## Out of scope

- A link between a moment and a maintenance entry, and a tag of a moment.
- A day that Maroid reads from the metadata of a photo.
- A limit of the photos of one moment.
- A page of moments. `GARDEN-FR-057` shows every moment.
- A photo in the bot, the widget, or the reminder.
- The download of the uploaded bytes. `FILES` keeps the variants of an image only.
- The order of the photos of one moment, a move of a photo to another moment, and a
  note for one photo.
- A timeline of every plant of a workspace, and a view that compares two photos.

## Definitions

| Term            | Meaning                                                                                  |
| --------------- | ---------------------------------------------------------------------------------------- |
| moment          | One entry of the gallery of a plant: a day, an optional note, and one or more photos.    |
| photo           | A file of the plant photo kind that one moment holds.                                    |
| gallery         | Every moment of one plant.                                                               |
| profile picture | The one photo of the gallery that names the plant in the shell, when the editor set one. |

## Functional requirements

### Moments

### `GARDEN-FR-045`

An editor must add a moment to a plant with these values:

| Value  | Rule                                        |
| ------ | ------------------------------------------- |
| Day    | Required. A date that is not in the future. |
| Note   | Optional. 1 to 1000 characters.             |
| Photos | Required. One or more.                      |

**Examples:**

- Normal case: the day of today, the note "Repotted into a 14 cm pot", and 2 photos. Maroid adds the moment.
- Limit case: the day of yesterday, no note, and 1 photo. Maroid adds the moment.
- Unwanted case: the day of tomorrow. Maroid refuses the moment.
- Unwanted case: no photo. Maroid refuses the moment.

### `GARDEN-FR-046`

The form of a new moment must offer the current date of the browser as the day.

**Why:** A person takes most photos on the day of the upload. Maroid keeps no metadata of a photo (`FILES-INV-004`).

**Examples:**

- Normal case: an editor uploads a photo that the phone took 3 days ago. The form offers today, and the editor sets the day to 3 days ago.

### `GARDEN-FR-047`

A photo must be a JPEG, PNG, WebP, HEIC, or HEIF image of at most 25 MB.

**Why:** These are the types that a phone or a camera writes, and the types from which `FILES` derives the variants.

### `GARDEN-FR-048`

Maroid must refuse a moment while one of its photos is not ready.

**Why:** The gallery never shows a photo that is pending, processing, or failed.

**Examples:**

- Normal case: both photos are ready. Maroid adds the moment.
- Unwanted case: one photo is still processing. Maroid refuses the moment, and the editor saves it about 2 seconds later.
- Unwanted case: one photo failed, because its content is a PDF. The form shows the reason, and the editor saves the moment with the others, without the failed photo.

### `GARDEN-FR-049`

An editor must change the day and the note of a moment, with the rules of `GARDEN-FR-045`.

### `GARDEN-FR-050`

An editor must add photos to a moment, with the rules of `GARDEN-FR-047` and `GARDEN-FR-048`.

### `GARDEN-FR-051`

An editor must delete one photo of a moment. When the photo is the last one, Maroid must delete the moment.

**Examples:**

- Normal case: a moment holds 3 photos. The editor deletes 1. The moment holds 2.
- Limit case: a moment holds 1 photo. The editor deletes it. The gallery holds no such moment.

### `GARDEN-FR-052`

An editor must delete a moment with each of its photos.

### `GARDEN-FR-053`

Maroid must delete the file of each photo that a delete of a photo, of a moment, or of a plant removes.

**Why:** No deleted photo stays readable or fills the file store.

### `GARDEN-FR-054`

The delete of a plant must delete its moments and their photos.

**Why:** `GARDEN-FR-006` removes every trace of a plant entered by mistake.

### `GARDEN-FR-055`

An editor must change the gallery and the profile picture of an inactive plant as of an active one.

**Why:** A plant that dies or leaves keeps its story, and an editor adds a last photo.

### Gallery

### `GARDEN-FR-056`

The plant page must show the moments of the plant, the latest day first. Two moments of one day show the latest added first. Each moment shows its day, the time since that day, its note, and the thumbnail of each photo.

**Examples:**

- Normal case: moments of 12 March and 2 June. 2 June comes first, with "5 months ago".
- Limit case: two moments of one day. The one added last comes first.

### `GARDEN-FR-057`

The plant page must show every moment of the plant, with no action that loads more.

**Why:** A plant gets a few moments each year.

### `GARDEN-FR-058`

When the plant has no moment, the plant page must show "No photo yet." and, for an editor, the action that adds the first moment.

### `GARDEN-FR-059`

A person must open a photo in a full view. The full view shows the display image, the day and the note of its moment, and its position among the photos of the plant, as "3 of 12".

### `GARDEN-FR-060`

The full view must move to the previous and to the next photo of the plant, in the order of `GARDEN-FR-056`, with a control and with the arrow keys. After the last photo, it moves to the first.

**Examples:**

- Normal case: the full view shows photo 3 of 12. The right arrow shows photo 4.
- Limit case: the full view shows photo 12 of 12. The right arrow shows photo 1.
- Limit case: the plant holds 1 photo. The full view offers no move.

### `GARDEN-FR-061`

A viewer must read the gallery and the full view. Maroid must refuse an add, a change, or a delete of a moment, a photo, or the profile picture from a viewer.

### Profile picture

### `GARDEN-FR-062`

An editor must set a photo of the gallery as the profile picture of its plant. The photo stays in its moment.

### `GARDEN-FR-063`

When an editor adds a moment to a plant that has no profile picture, Maroid must make the first photo of that moment the profile picture.

**Examples:**

- Normal case: a new plant gets its first moment with 3 photos. The first photo becomes the profile picture.
- Limit case: the profile photo was deleted. The next new moment sets its first photo as the profile picture.
- Unwanted case: the plant has a profile picture. A new moment leaves it as it is.

### `GARDEN-FR-064`

The delete of the photo that is the profile picture must leave the plant with no profile picture.

**Why:** The editor chooses the profile picture. Maroid never puts another photo in its place.

### `GARDEN-FR-065`

The plant page and each line of the list of plants must show the profile picture as a round thumbnail. A plant with no profile picture shows the first letter of its common name in its place.

## Non-functional requirements

### `GARDEN-NFR-005`

The gallery must answer in less than 1 second, for a plant with 100 moments and 300 photos. Measure at the hub, from the request to the last byte of the answer.

### `GARDEN-NFR-006`

A page of the list of plants must answer in less than 1 second with the profile pictures, for a workspace with 500 plants that each have one. Measure at the hub, from the request to the last byte of the answer.

## Invariants

### `GARDEN-INV-004`

A moment holds at least one photo.

### `GARDEN-INV-005`

A file is the photo of at most one moment.

### `GARDEN-INV-006`

The profile picture of a plant is a photo of a moment of that plant.

### `GARDEN-INV-007`

A moment, its photos, and its plant are in one workspace.

## Constraints from the guidelines

| Rule      | Guideline        | Effect on this feature                                                                         |
| --------- | ---------------- | ---------------------------------------------------------------------------------------------- |
| `OWN-004` | Record ownership | Every moment and photo is scoped to a workspace.                                               |
| `SEC-013` | Security         | A read checks the permission `plants.read` (viewer). A change checks `plants.write` (editor).  |
| `RES-005` | REST             | The list of moments answers a page. The plant page reads every page to meet `GARDEN-FR-057`.   |
| `DAT-003` | Data             | A row of the plugin references a file of the hub, as the accepted ADR of the reference allows. |
| `PLG-007` | Plugin model     | The plugin reaches a file through the host, and declares the plant photo kind as a capability. |
