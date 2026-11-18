---
id: FILES
title: The scenarios of the files of the hub and of a plugin
type: spec
status: approved
created: 2026-11-18
updated: 2026-11-18
approved_by: Temuri
approved_on: 2026-11-18
constrained_by: [TST, OWN, TRC]
requirements: features/files/requirements.md
---

# Specification: The scenarios of the files of the hub and of a plugin

An integration scenario runs against PostgreSQL and a RustFS that `teststore.Start`
owns, with the hub key of `FILES-DD-003`. "The probe" is a test plugin that declares
the kind `photo` (workspace, images, 25 MB, permission `photo.upload` for an editor)
and the kind `paper` (workspace, `application/pdf`, 5 MB, the same permission). Editor
A and viewer V are members of workspace A. Editor B is a member of workspace B.
`testdata/` holds `portrait.jpg` (EXIF orientation 6), `phone.heic` (12 MB, 48
megapixels), `small.png` (300 by 200), `gps.jpg` (GPS and device tags), and `paper.pdf`.

## `FILES-SC-001`

**Verifies:** `FILES-FR-001`
**Layer:** unit

**Given** the probe declares `photo` and `paper`.
**When** the registrar loads the probe.
**Then** `FileKindRegistry.Get("probe", "photo")` answers the kind, and the capability
report of the probe holds `files`.

## `FILES-SC-002`

**Verifies:** `FILES-FR-002`
**Layer:** unit

**Given** the probe declares one kind with `MaxSize` of 26 MB.
**When** the registrar loads the probe.
**Then** the load fails with `ErrInvalidFileKind`, and the error names the probe, the
kind, and `max_size`.

## `FILES-SC-003`

**Verifies:** `FILES-FR-002`
**Layer:** unit

**Given** four probes: one declares no media type, one names an undeclared permission,
one declares an image kind with `application/pdf`, and one declares a kind that is not
an image with `image/jpeg`.
**When** the registrar loads each one.
**Then** each load fails with `ErrInvalidFileKind`, and no kind of any of them reaches
the registry.

## `FILES-SC-004`

**Verifies:** `FILES-FR-003`, `FILES-FR-008`
**Layer:** integration

**Given** editor A acts in workspace A, which enables the probe.
**When** the editor sends `POST .../files` with `photo`, `cat.jpg`, `image/jpeg`, and
3 MB.
**Then** the hub answers 201 with a file in `pending` and an upload address, and
`Location` names the file. The row and the tasks `check` and `expire` exist.

## `FILES-SC-005`

**Verifies:** `FILES-FR-003`
**Layer:** integration

**Given** viewer V acts in workspace A.
**When** the viewer sends the start of `FILES-SC-004`.
**Then** the hub answers 403, `permission-denied`, with `probe.photo.upload` and
`editor`. No row exists.

## `FILES-SC-006`

**Verifies:** `FILES-FR-004`
**Layer:** integration

**Given** editor A acts in workspace A.
**When** the editor starts `photo` with `application/pdf`.
**Then** the hub answers 422 with a failure on the field `media_type`.

## `FILES-SC-007`

**Verifies:** `FILES-FR-005`
**Layer:** integration

**Given** editor A acts in workspace A.
**When** the editor starts `paper` with 5 MB plus 1 byte, then with 5 MB.
**Then** the first answers 422 with a failure on the field `size`. The second answers 201.

## `FILES-SC-008`

**Verifies:** `FILES-FR-006`
**Layer:** integration

**Given** a start that declared 1 MB answered an upload address.
**When** a client sends 2 MB to the address, then sends 1 MB with `text/plain`.
**Then** the file store refuses both with 403. `Head` finds no upload.

## `FILES-SC-009`

**Verifies:** `FILES-FR-007`
**Layer:** manual

**Given** the deck runs against the hub and RustFS.
**When** editor A uploads a photo of 20 MB, and the network panel of the browser records.
**Then** the `PUT` goes to `files.public_endpoint`. No request to the hub carries more
than 2 KB.

## `FILES-SC-010`

**Verifies:** `FILES-FR-009`
**Layer:** integration

**Given** editor A uploaded `cat.jpg` and sent the completion.
**When** the worker runs one cycle.
**Then** the file is `ready`, and no request of the editor waited for the worker.

## `FILES-SC-011`

**Verifies:** `FILES-FR-009`
**Layer:** integration

**Given** editor A uploaded `cat.jpg` and sent no completion.
**When** the clock passes `upload_ttl` plus 1 minute, and the worker runs two cycles.
**Then** the task `check` wrote `process`, and the file is `ready`.

## `FILES-SC-012`

**Verifies:** `FILES-FR-010`, `FILES-FR-011`
**Layer:** integration

**Given** editor A started `photo` with `image/jpeg`, and uploaded the bytes of
`paper.pdf` under that type.
**When** the worker processes the file.
**Then** the file is `failed` with `media_type`, and the file store holds no variant.

## `FILES-SC-013`

**Verifies:** `FILES-FR-011`
**Layer:** integration

**Given** two files of `photo`: one holds the first 4 KB of `portrait.jpg` alone, and
one never received bytes.
**When** the worker processes the first, and the task `check` of the second runs.
**Then** the first is `failed` with `damaged`. The second is `failed` with `bytes_absent`.

## `FILES-SC-014`

**Verifies:** `FILES-FR-012`
**Layer:** unit

**Given** an image of 8000 by 6000 pixels.
**When** `variantsOf` runs.
**Then** the thumbnail is 480 by 360 pixels and the display image is 2048 by 1536 pixels.

## `FILES-SC-015`

**Verifies:** `FILES-FR-012`
**Layer:** unit

**Given** `small.png`, 300 by 200 pixels.
**When** `variantsOf` runs.
**Then** both variants are 300 by 200 pixels.

## `FILES-SC-016`

**Verifies:** `FILES-FR-013`
**Layer:** unit

**Given** `portrait.jpg`, stored 4000 by 3000 with EXIF orientation 6.
**When** `variantsOf` runs.
**Then** the display image is 1536 by 2048 pixels, and its top row matches the left
column of the stored pixels.

## `FILES-SC-017`

**Verifies:** `FILES-FR-014`
**Layer:** integration

**Given** editor A uploaded `phone.heic` as `image/heic`.
**When** the worker processes the file.
**Then** the file is `ready` with `image/heic` and the width and the height of the photo.

## `FILES-SC-018`

**Verifies:** `FILES-FR-015`
**Layer:** unit

**Given** the variants of `phone.heic`.
**When** `mimetype` reads each one.
**Then** each one is `image/jpeg`.

## `FILES-SC-019`

**Verifies:** `FILES-FR-016`
**Layer:** integration

**Given** the file of `FILES-SC-017` is `ready`.
**When** the worker runs the task `discard_upload`, then a client repeats the `PUT`
within `upload_ttl`, and the task `check` runs.
**Then** after each task the file store holds `thumbnail.jpg` and `display.jpg`, and
no `upload`.

## `FILES-SC-020`

**Verifies:** `FILES-FR-017`
**Layer:** integration

**Given** editor A uploaded `paper.pdf` as `paper`.
**When** the worker processes the file.
**Then** the file is `ready`, the file store holds `upload` with the bytes of the file,
and `Get` answers the variant `original` alone.

## `FILES-SC-021`

**Verifies:** `FILES-FR-018`
**Layer:** integration

**Given** the file of `FILES-SC-017` is `ready`.
**When** the probe calls `Get` with its identifier.
**Then** the file holds the state, the name `phone.heic`, `image/heic`, the size, the
width, the height, the time of the start, and editor A as the uploader.

## `FILES-SC-022`

**Verifies:** `FILES-FR-019`
**Layer:** integration

**Given** editor A uploaded a photo and the probe never claimed it.
**When** the clock passes 24 hours and the worker runs two cycles.
**Then** the row is absent, and the file store holds no object under its key.

## `FILES-SC-023`

**Verifies:** `FILES-FR-019`
**Layer:** integration

**Given** a row of the probe references a file, and the probe never claimed it.
**When** the task `expire` runs.
**Then** the row of the file stays, and `claimed_at` holds the time of the task.

## `FILES-SC-024`

**Verifies:** `FILES-FR-020`
**Layer:** integration

**Given** a file in `pending`, a file in `processing`, and a file in `failed`.
**When** the probe claims each one.
**Then** each claim succeeds, and the task `expire` of each one leaves its row.

## `FILES-SC-025`

**Verifies:** `FILES-FR-021`
**Layer:** integration

**Given** a claimed file that no row references.
**When** the probe calls `Delete` in its transaction, and commits.
**Then** the row is absent, and a task `purge` is due.

## `FILES-SC-026`

**Verifies:** `FILES-FR-022`
**Layer:** integration

**Given** a row of the probe references a file.
**When** the probe calls `Delete` with that file in its transaction, before it deletes the row.
**Then** the call answers `ErrFileReferenced`, and the row of the file stays.

## `FILES-SC-027`

**Verifies:** `FILES-FR-023`
**Layer:** integration

**Given** the file of `FILES-SC-017` is `ready`.
**When** the probe calls `Get`, and a client reads each address.
**Then** the file holds an address for `thumbnail` and for `display`, and each read
answers 200 with `image/jpeg`.

## `FILES-SC-028`

**Verifies:** `FILES-FR-023`, `FILES-INV-003`
**Layer:** integration

**Given** a file of workspace B.
**When** editor A, in workspace A, sends `GET .../files/{fileId}`, and the probe calls
`Get` with it.
**Then** the route answers 404, and `Get` answers no file.

## `FILES-SC-029`

**Verifies:** `FILES-FR-023`
**Layer:** integration

**Given** a file in `processing`.
**When** the probe calls `Get`.
**Then** the file holds `processing` and no address.

## `FILES-SC-030`

**Verifies:** `FILES-FR-024`
**Layer:** integration

**Given** the RustFS of the test stops.
**When** editor A starts an upload.
**Then** the hub answers 503, `/problems/hub/not-ready`, with `file-store`, within 2
seconds, and writes no row.

## `FILES-SC-031`

**Verifies:** `FILES-FR-025`
**Layer:** integration

**Given** the RustFS of the test stops.
**When** a client reads `/readyz`.
**Then** the answer is 503, and the check `file-store` fails.

## `FILES-SC-032`

**Verifies:** `FILES-FR-026`
**Layer:** manual

**Given** the page of a probe plugin that mounts `FileUpload` with `host.files`.
**When** a person drops one file on it.
**Then** the control calls `start` with the kind, the name, the type, and the size, and
the page holds no code of its own for the upload.

## `FILES-SC-033`

**Verifies:** `FILES-FR-027`
**Layer:** manual

**Given** the deck runs, with the network slowed to 1 Mbit/s.
**When** editor A uploads two photos of 10 MB.
**Then** the control shows a progress for each file, and each one grows to 100 percent.

## `FILES-SC-034`

**Verifies:** `FILES-FR-028`
**Layer:** manual

**Given** the deck runs.
**When** editor A uploads `phone.heic` and `paper.pdf` renamed as `paper.jpg`.
**Then** the control shows `processing`, then `ready` for the first, and `failed` with
its reason for the second.

## `FILES-SC-035`

**Verifies:** `FILES-NFR-001`
**Layer:** integration

**Given** the worker runs with `concurrency` 2 and `poll_interval` 2 s.
**When** editor A uploads `phone.heic` 20 times, one after the other, and the test
measures from each completion to the state `ready`.
**Then** the 95th percentile is less than 30 seconds.

## `FILES-SC-036`

**Verifies:** `FILES-NFR-002`
**Layer:** unit

**Given** `phone.heic`, `portrait.jpg`, and `small.png`.
**When** `variantsOf` runs on each one.
**Then** each thumbnail holds at most 100 KB.

## `FILES-SC-037`

**Verifies:** `FILES-NFR-003`
**Layer:** integration

**Given** the hub signs an upload address with a clock 15 minutes and 1 second behind
the clock of the file store.
**When** a client sends the bytes to the address.
**Then** the file store refuses the request, and `X-Amz-Expires` of the address is at
most 900.

## `FILES-SC-038`

**Verifies:** `FILES-NFR-004`
**Layer:** unit

**Given** `files.read_ttl` is 1 hour.
**When** `Get` signs the addresses of a ready file.
**Then** `X-Amz-Expires` of each one is at most 3600, and `ExpiresAt` is at most one
hour after the call.

## `FILES-SC-039`

**Verifies:** `FILES-NFR-005`
**Layer:** integration

**Given** a claimed file that no row references.
**When** the probe calls `Delete`, and the worker runs.
**Then** the file store holds no object under the key of the file within 10 seconds.

## `FILES-SC-040`

**Verifies:** `FILES-INV-001`
**Layer:** manual

**Given** the bucket of the hub on the shared RustFS.
**When** a client reads and writes an object of a ready file with no signature, and with
the key of another project.
**Then** the file store refuses each request with 403.

## `FILES-SC-041`

**Verifies:** `FILES-INV-002`
**Layer:** integration

**Given** the probe and a second probe both enabled in workspace A. The second probe
holds a file.
**When** the probe calls `Get`, `Claim`, and `Delete` with that identifier.
**Then** `Get` answers no file, and `Claim` and `Delete` answer `ErrFileAbsent`.

## `FILES-SC-042`

**Verifies:** `FILES-INV-003`
**Layer:** integration

**Given** a file of a kind of the user scope, of editor A.
**When** editor B calls `Get` with it through the probe, in workspace A.
**Then** `Get` answers no file.

## `FILES-SC-043`

**Verifies:** `FILES-INV-004`
**Layer:** integration

**Given** editor A uploaded `gps.jpg`, and the file is `ready`.
**When** a client reads each address, and a reader of EXIF reads each answer.
**Then** no answer holds a GPS tag, a make, or a model.

## `FILES-SC-044`

**Verifies:** `FILES-INV-005`
**Layer:** integration

**Given** a file of the probe in workspace A.
**When** a statement updates its `plugin_id`, its `kind`, or its `workspace_id`.
**Then** the trigger `files_keep_identity` refuses each one.

## `FILES-SC-045`

**Verifies:** `FILES-FR-021`
**Layer:** integration

**Given** a row of the probe references a claimed file.
**When** the probe deletes the row, calls `Delete` with the file in the same transaction, and commits.
**Then** the row of the file is absent, `Get` omits it, and a task `purge` is due.

## `FILES-SC-046`

**Verifies:** `FILES-FR-021`
**Layer:** integration

**Given** a row of the probe references a claimed file.
**When** the probe deletes the row, calls `Delete` with the file in the same transaction, then fails, so the transaction rolls back.
**Then** the row of the probe and the row of the file stay, and no task `purge` exists.

## Retired identifiers

This file has no retired identifier.
