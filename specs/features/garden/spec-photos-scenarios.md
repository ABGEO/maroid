---
id: GARDEN
title: The scenarios of the photos of a plant
type: spec
status: approved
created: 2026-11-18
updated: 2026-11-18
approved_by: Temuri
approved_on: 2026-11-18
constrained_by: [TST, OWN, TRC]
requirements: features/garden/requirements-photos.md
---

# Specification: The scenarios of the photos of a plant

`spec-photos.md` holds the design. The setup of `spec-scenarios.md` holds: the
container of `testdb.Start`, the stand-ins of the hub, the migrations of jasmine, and
workspaces H and G. A plugin test cannot import the hub (`PLG-007`), so a stand-in of
`public.workspace_files` holds the files, and a fake `PluginFiles` reads, claims, and
deletes them: `Delete` removes the row through the transaction of the plugin and
records a purge, as the hub does. `FILES-SC-045` and `FILES-SC-046` verify the hub. A
ready photo is a row of the stand-in of jasmine, kind `plant-photo`, state `ready`, that
the setup inserts. Today is the UTC date of the run. Plant `P001` of H is active. A
viewer holds `plants.read` alone: for `GARDEN-SC-063` the router stands in for the
permission check of the hub and answers 403 for a route whose permission the viewer
lacks.

### `GARDEN-SC-050`

**Verifies:** `GARDEN-FR-045`, `GARDEN-FR-063`
**Layer:** integration

**Given** an editor and the ready photos A and B, unclaimed.
**When** the editor posts a moment of `P001` with today, the note "Repotted into a 14 cm pot", and `file_ids` `[A, B]`.
**Then** the answer is 201 with a `Location`, the note, and two photos in the order A, B, each with `thumbnail` and `display`. Both files are claimed. The profile picture of `P001` is the photo of A.

### `GARDEN-SC-051`

**Verifies:** `GARDEN-FR-045`
**Layer:** integration

**Given** an editor and the ready photo A.
**When** the editor posts a moment of `P001` with yesterday, no note, and `[A]`.
**Then** the answer is 201, and the moment holds no member `note`.

### `GARDEN-SC-052`

**Verifies:** `GARDEN-FR-045`
**Layer:** integration

**Given** an editor and the ready photo A.
**When** the editor posts a moment with the day after tomorrow, then one with `file_ids` `[]`, then one with a note of 1001 characters.
**Then** each answer is 422 with the field `taken_on`, `file_ids`, then `note`. `P001` holds no moment.

### `GARDEN-SC-053`

**Verifies:** `GARDEN-FR-048`
**Layer:** integration

**Given** an editor, the ready photo A, the processing file B, the failed file C, a ready file D of a user kind of another plugin, and a ready photo E of G.
**When** the editor posts a moment with `[A, B]`, then `[A, C]`, then `[A, D]`, then `[A, E]`.
**Then** the first answer is 409 `photo-not-ready` with "1 photo is not ready yet." The others are 422 with the field `file_ids`. `P001` holds no moment, and A stays unclaimed.

### `GARDEN-SC-054`

**Verifies:** `GARDEN-FR-047`
**Layer:** unit

**Given** the plugin.
**When** the test calls `FileKinds`.
**Then** it answers one kind: `plant-photo`, `workspace`, the five image types, `pluginapi.MaxFileSize`, `plants.write`, and `Image`.

### `GARDEN-SC-055`

**Verifies:** `GARDEN-INV-005`
**Layer:** integration

**Given** a moment of `P001` that holds the photo of A, and the ready photo B.
**When** the editor posts a moment of `P002` with `[A]`, then one with `[B, B]`.
**Then** the first answer is 422, "The file is already a photo of a moment." The second is 422 with the field `file_ids`. `P002` holds no moment.

### `GARDEN-SC-056`

**Verifies:** `GARDEN-FR-049`
**Layer:** integration

**Given** a moment of `P001` with the day 2026-06-02 and a note.
**When** the editor puts the day 2026-06-01 and no note, then puts the day after tomorrow.
**Then** the first answer is 200 with 2026-06-01, no note, a new `ETag`, and the same photos. The second is 422 with the field `taken_on`.

### `GARDEN-SC-057`

**Verifies:** `GARDEN-FR-050`
**Layer:** integration

**Given** a moment that holds the photo of A, the ready photos B and C, and the processing file D.
**When** the editor posts `[B, C]` to its photos, then `[D]`.
**Then** the first answer is 201 with the photos of A, B, C in that order. The second is 409, and the moment still holds three photos.

### `GARDEN-SC-058`

**Verifies:** `GARDEN-FR-051`, `GARDEN-FR-053`, `GARDEN-INV-004`
**Layer:** integration

**Given** a moment M1 with three photos and a moment M2 with one photo.
**When** the editor deletes one photo of M1, then the photo of M2.
**Then** each answer is 204. M1 holds two photos. A read of M2 answers 404, and no moment of the plant holds no photo. `Get` answers neither file of the deleted photos, and the fake records a purge for each.

### `GARDEN-SC-059`

**Verifies:** `GARDEN-FR-052`, `GARDEN-FR-053`
**Layer:** integration

**Given** a moment with the photos of A and B.
**When** the editor deletes the moment.
**Then** the answer is 204. No row of `photos` names A or B. `Get(A, B)` answers no file, and the fake records a purge for each.

### `GARDEN-SC-060`

**Verifies:** `GARDEN-FR-053`, `GARDEN-FR-054`
**Layer:** integration

**Given** `P003` with two moments, three photos, and a profile picture.
**When** the editor deletes `P003` with an `If-Match` of an older version, then with none.
**Then** the first answer is 412, and the moments, the photos, and the three files stay, with no purge. The second is 204, and the moments, the photos, and the three files are gone, with three purges.

### `GARDEN-SC-061`

**Verifies:** `GARDEN-FR-055`
**Layer:** integration

**Given** the inactive plant `P004`, and the ready photos A and B.
**When** the editor posts a moment with `[A]`, adds `[B]`, and puts the photo of B as the profile picture.
**Then** the answers are 201, 201, and 200.

### `GARDEN-SC-062`

**Verifies:** `GARDEN-FR-056`, `GARDEN-FR-057`
**Layer:** integration

**Given** moments of `P001` added in this order: 2026-03-12, 2026-06-01 (M1), 2026-06-02, 2026-06-01 (M2).
**When** a viewer reads the moments with `limit=2`, then follows `next`.
**Then** the first page holds 2026-06-02, then M2. The second page holds M1, then 2026-03-12, and no `next`.

### `GARDEN-SC-063`

**Verifies:** `GARDEN-FR-061`
**Layer:** integration

**Given** a viewer, a moment with one photo, and the ready photo B.
**When** the viewer reads the moments and the moment, then posts a moment, puts the moment, adds `[B]`, deletes the photo, deletes the moment, and puts the profile picture.
**Then** both reads answer 200. Each write answers 403, and nothing changes.

### `GARDEN-SC-064`

**Verifies:** `GARDEN-FR-062`
**Layer:** integration

**Given** a moment of `P001` with the photos of A and B, the profile picture A, and a photo C of `P002`.
**When** the editor puts B as the profile picture of `P001`, then C, then an unknown UUID.
**Then** the first answer is 200 with `profile_photo.id` of B and a new `ETag`. B stays in its moment. The second and the third answer 422 with the field `photo_id`.

### `GARDEN-SC-065`

**Verifies:** `GARDEN-FR-063`
**Layer:** integration

**Given** `P005` with no moment, and the ready photos A, B, C, D.
**When** the editor posts a moment with `[A, B]`, then one with `[C]`, deletes the photo of A, then posts a moment with `[D]`.
**Then** the profile picture is A after the first, A after the second, none after the delete, and D after the last.

### `GARDEN-SC-066`

**Verifies:** `GARDEN-FR-064`
**Layer:** integration

**Given** a moment of `P001` with the photos of A and B, and the profile picture A.
**When** the editor deletes the photo of A.
**Then** `P001` answers no `profile_photo`. B stays a photo and is not the profile picture.

### `GARDEN-SC-067`

**Verifies:** `GARDEN-FR-065`
**Layer:** integration

**Given** `P001` with a profile picture, and `P002` with none.
**When** a viewer reads `/plants` and `/plants/{id}` of each.
**Then** `P001` answers `profile_photo` with `id` and `thumbnail.url`. `P002` answers no `profile_photo`.

### `GARDEN-SC-068`

**Verifies:** `GARDEN-INV-006`, `GARDEN-INV-007`
**Layer:** integration

**Given** a moment M of `P001` in H, a photo X of `P002`, and the migrations applied.
**When** the test inserts, as H, a photo of `P002` in M, then sets the profile picture of `P001` to X, then inserts, as G, a moment of `P001`.
**Then** each statement fails with SQLSTATE `23503`.

### `GARDEN-SC-076`

**Verifies:** `GARDEN-FR-053`
**Layer:** integration

**Given** a moment M of `P001` with one photo, and a new file F.
**When** one transaction locks M with `Moment.LockForUpdate`, a second transaction inserts a photo of F in M, and then the first deletes M and commits.
**Then** the insert waits for the lock, and fails with SQLSTATE `23503` once the first transaction commits. The same holds for `Plant.LockForUpdate`, the delete of `P001`, and an insert of a moment of `P001`.

### `GARDEN-SC-069`

**Verifies:** `GARDEN-NFR-005`
**Layer:** integration

**Given** `P001` with 100 moments and 300 photos.
**When** a client reads `/plants/{P001}/moments?limit=100` 20 times, and the test keeps the slowest call.
**Then** the slowest call answers in less than 1 second, from the call of the handler to the last byte.

### `GARDEN-SC-070`

**Verifies:** `GARDEN-NFR-006`
**Layer:** integration

**Given** H with 500 plants, each with one moment, one photo, and a profile picture.
**When** a client reads `/plants?limit=100` and each `next` page, and the test keeps the slowest call.
**Then** the slowest call answers in less than 1 second, from the call of the handler to the last byte.

### `GARDEN-SC-071`

**Verifies:** `GARDEN-FR-046`, `GARDEN-FR-048`
**Layer:** manual

**Given** an editor on the plant page, a photo that the phone took 3 days ago, and a PDF renamed `leaf.jpg`.
**When** the editor opens "Add a moment", drops both files, waits, sets the day to 3 days ago, and saves.
**Then** the day starts at today, and no later day is offered. "Add moment" stays disabled while a file uploads or processes. The PDF shows "The content is not a type that this field accepts.", and the save leaves it out. The moment shows one photo on its day.

### `GARDEN-SC-072`

**Verifies:** `GARDEN-FR-056`, `GARDEN-FR-057`, `GARDEN-FR-058`
**Layer:** manual

**Given** an editor, `P005` with no moment, and `P001` with moments on 12 March and 2 June.
**When** the editor opens `P005`, then `P001`.
**Then** `P005` shows "No photo yet." and "Add the first moment". `P001` shows 2 June first with the time since it, its note, and its thumbnails, then 12 March, with no action that loads more.

### `GARDEN-SC-073`

**Verifies:** `GARDEN-FR-059`, `GARDEN-FR-060`
**Layer:** manual

**Given** `P001` with 12 photos, and `P002` with 1.
**When** the owner opens the third photo of `P001`, presses the right arrow, opens the twelfth, presses the right arrow, then opens the photo of `P002`.
**Then** the full view shows "3 of 12" with the day and the note of its moment, then "4 of 12", then "1 of 12" after the twelfth. The photo of `P002` shows no move control.

### `GARDEN-SC-074`

**Verifies:** `GARDEN-FR-061`
**Layer:** manual

**Given** a viewer and `P001` with moments.
**When** the viewer opens `P001` and a photo.
**Then** the gallery and the full view show, with no "Add a moment", no menu, and no "Set as profile picture".

### `GARDEN-SC-075`

**Verifies:** `GARDEN-FR-065`
**Layer:** manual

**Given** `P001` with a profile picture, and `P002` "Monstera" with none.
**When** the owner opens the list of plants, then each plant page.
**Then** `P001` shows the round thumbnail in its line and in its header. `P002` shows "M" in its place.

## Retired identifiers

This file has no retired identifier.
