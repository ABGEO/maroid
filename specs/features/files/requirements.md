---
id: FILES
title: The files of the hub and of a plugin
type: requirements
status: approved
created: 2026-11-18
updated: 2026-11-18
approved_by: Temuri
approved_on: 2026-11-18
constrained_by: [ARC, PLG, DAT, OWN, SEC, CFG, LOG, UI]
---

# Requirements: The files of the hub and of a plugin

## 1. Problem

A plugin that needs a photo or a document has no place for the bytes. A table of
the plugin is the wrong place: a phone photo holds 5 to 12 MB, and every read of the
row then carries it. Each plugin that solves it alone writes its own storage, its own
access check, and its own cleanup.

A photo from a phone also brings two problems with it. Many phones save HEIC, which
most browsers do not display. The photo carries the place where a person took it, and
every member of a shared workspace reads that place.

`GARDEN` lists the photos of a plant as out of scope for this reason.

## 2. Users

| Person          | Need                                                                                   |
| --------------- | -------------------------------------------------------------------------------------- |
| A plugin author | Declares what a person uploads, keeps the identifier of a file, and writes no storage. |
| An editor       | Uploads a photo from a phone or a computer, and works on while Maroid prepares it.     |
| A viewer        | Opens a page of photos without the download of each full photo.                        |
| The owner       | Runs one file store, which no abandoned upload fills, and which no stranger reads.     |

## 3. Out of scope

- An upload through the Telegram bot or through an MCP tool.
- A limit on the count of the files, or on their total size, for a workspace or a user.
- A change to an image by a person: a crop, a rotation, a filter.
- The processing of a video, or of the frames of an animated image.
- A check of a file that is not an image for malware.
- A download of an image in the quality of the upload.
- A list of the files of every workspace for an administrator.
- A move of a file to another workspace, another user, or another consumer.
- A deletion of the files of a plugin when a workspace disables the plugin.
- An end of a read address before its expiry.
- An upload that continues after the person closes the browser.
- A public address that a person shares outside Maroid.
- A file kind of the hub. No part of the hub consumes a file yet.

## 4. Functional requirements

### `FILES-FR-001`

A consumer must declare each file kind that it accepts.

**Why:** Maroid checks an upload against the declaration. A plugin then writes no check
of its own.

**Example:** the garden plugin declares `plant-photo`: images, 25 MB, the workspace
scope, and its permission to edit a plant.

### `FILES-FR-002`

The hub must refuse to load a plugin that declares a file kind with no media type, a
size limit above 25 MB, or a permission that the plugin does not declare.

**Why:** A broken declaration fails at the start, and no person meets it.

### `FILES-FR-003`

A member must start an upload of a file kind only when their role holds the
permission of that file kind.

**Example:** a viewer starts an upload of `plant-photo`. Maroid answers `permission-denied`.

### `FILES-FR-004`

Maroid must refuse to start an upload whose media type the file kind does not accept.

### `FILES-FR-005`

Maroid must refuse to start an upload whose size passes the size limit of the file kind.

### `FILES-FR-006`

The file store must refuse bytes whose size differs from the size that the start of
the upload declared.

**Why:** Without it, a person who holds an upload address writes any size, and
`FILES-FR-005` limits nothing.

**Example:** the start declares 8 MB. A client sends 2 GB. The file store keeps nothing.

### `FILES-FR-007`

The browser must send the bytes of a file to the file store, and not to the hub.

**Why:** A 25 MB upload over a phone network takes minutes. It must not hold a request
of the hub for that time.

### `FILES-FR-008`

Maroid must give the identifier of a file to the consumer before it processes the bytes.

**Why:** A person saves the plant at once and does not wait for the photo.

### `FILES-FR-009`

Maroid must process the bytes of a file after the upload ends, while the person works on.

**Examples:**

- Normal case: an editor uploads a photo and leaves the page. The file becomes `ready`.
- Limit case: the browser closes before the upload ends. The file stays `pending`
  until `FILES-FR-019` deletes it.

### `FILES-FR-010`

Maroid must set the file state to `failed` when the content of the bytes is not of a
media type that the file kind accepts.

**Why:** A name or a header comes from the client. The content is the only fact.

**Example:** a client uploads a program under the name `photo.jpg`. The file becomes `failed`.

### `FILES-FR-011`

Maroid must record the reason of a `failed` file: a media type that the file kind does
not accept, a damaged image, or bytes that never arrived.

### `FILES-FR-012`

Maroid must derive two variants from an image: a thumbnail whose long edge is at most
480 pixels, and a display image whose long edge is at most 2048 pixels.

**Why:** A page of 24 photos must not download 24 photos of 12 MB.

**Examples:**

- Normal case: a photo of 8000 by 6000 pixels gives a thumbnail of 480 by 360 pixels.
- Limit case: a photo of 300 by 200 pixels gives two variants of 300 by 200 pixels.
  Maroid enlarges no image.

### `FILES-FR-013`

Maroid must turn each variant to the orientation that the camera recorded.

**Why:** A phone records the rotation beside the pixels. A variant without it lies on its side.

### `FILES-FR-014`

Maroid must accept a HEIC or a HEIF image.

**Why:** Many phones save this format by default.

### `FILES-FR-015`

Maroid must give each variant a format that the current versions of Chrome, Firefox,
and Safari display.

### `FILES-FR-016`

Maroid must delete the uploaded bytes of an image after it derives the variants.

**Why:** The uploaded bytes hold the place where the photo was taken. No reader needs them.

### `FILES-FR-017`

Maroid must keep a file that is not an image as the person uploaded it.

### `FILES-FR-018`

A consumer must read the facts of a file: the file state, the name at the upload, the
media type, the size, the width and the height of an image, the time of the upload,
and the user who uploaded it.

### `FILES-FR-019`

Maroid must delete an unclaimed file 24 hours after the start of its upload.

**Why:** A person who closes a form after the upload leaves a file that nothing references.

**Example:** an editor uploads a photo and cancels the form. Next day the file is absent.

### `FILES-FR-020`

A consumer must claim a file in any file state.

**Why:** A person saves a plant while Maroid still processes its photo.

### `FILES-FR-021`

A consumer must delete a file that it claimed.

### `FILES-FR-022`

Maroid must refuse to delete a file while a row of the consumer references it.

**Why:** A row that names a deleted file shows a broken image.

### `FILES-FR-023`

A consumer must get a read address for a `ready` file of the acting workspace or of the
acting user.

**Examples:**

- Normal case: the gallery asks for the thumbnails of a plant. Each one comes with a read address.
- Unwanted case: the gallery asks for a file of another workspace. Maroid answers `not-found`.
- Limit case: the file is `processing`. Maroid gives no read address, and reports the file state.

### `FILES-FR-024`

Maroid must answer the start of an upload with a failure that names the file store
when the file store does not answer.

### `FILES-FR-025`

The hub must report that it is not ready when the file store does not answer.

**Why:** A hub that cannot reach the file store accepts a form whose photo it cannot keep.

### `FILES-FR-026`

A plugin page must upload a file through a control of the component library.

**Why:** Each plugin then shows the same upload, and writes no upload code.

### `FILES-FR-027`

The upload control must show the progress of each file during the upload.

### `FILES-FR-028`

The upload control must show the file state of each file after the upload.

## 5. Non-functional requirements

### `FILES-NFR-001`

Maroid must make a 12 MB HEIC photo of 48 megapixels `ready` in less than 30 seconds,
at the 95th percentile. Measure from the end of the upload to the file store to the
file state `ready`.

**Why:** A person who uploads a photo looks for it in the gallery a moment later.

### `FILES-NFR-002`

A thumbnail must hold at most 100 KB.

**Why:** A page of 24 thumbnails then downloads at most 2.4 MB.

### `FILES-NFR-003`

An upload address must stop working at most 15 minutes after Maroid issues it.

### `FILES-NFR-004`

A read address must stop working at most 1 hour after Maroid issues it.

**Why:** A read address is a credential. A copy of it in a log or a history works until then.

### `FILES-NFR-005`

The file store must hold no bytes of a deleted file 1 hour after the deletion.

## 6. Invariants

### `FILES-INV-001`

The file store returns or accepts bytes only for a request that carries a signature
that Maroid issued.

### `FILES-INV-002`

A consumer reaches no file that another consumer started.

### `FILES-INV-003`

A file of one workspace reaches no other workspace. A file of one user reaches no other user.

### `FILES-INV-004`

No byte that a read address returns holds the place where a photo was taken, or the
device that took it.

### `FILES-INV-005`

A file keeps its consumer, its file kind, and its scope for its life.

## 7. Constraints from the guidelines

| Rule             | Guideline        | Effect on this feature                                                                                                                                                     |
| ---------------- | ---------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `ARC-008`        | Architecture     | The hub holds no rule that names one plugin or one file kind.                                                                                                              |
| `ARC-010`        | Architecture     | The worker runs the cron jobs and the MQTT subscribers. A process that runs `FILES-FR-009` outside a request is a new duty. If the worker runs it, `ARC-010` needs an ADR. |
| `PLG-006`        | Plugin model     | A file kind is a new capability: an interface, a registry, and a registrar.                                                                                                |
| `PLG-007`        | Plugin model     | A plugin reaches a file through the host.                                                                                                                                  |
| `DAT-003`        | Data             | A row of a plugin references a file of the hub. `OWN-005` makes the same exception for a user. If a plugin schema holds the reference, `DAT-003` needs an ADR.             |
| `DAT-009`        | Data             | The identifier of a file is a UUID version 7.                                                                                                                              |
| `OWN-003`        | Record ownership | The start of an upload reads the acting user or the acting workspace of its unit of work.                                                                                  |
| `OWN-004`        | Record ownership | A file belongs to one workspace or to one user.                                                                                                                            |
| `OWN-006`        | Record ownership | The isolation of a file runs in the database, and fails closed.                                                                                                            |
| `SEC-012`        | Security         | A plugin that a workspace disables gets no read address and starts no upload in that workspace.                                                                            |
| `SEC-013`        | Security         | Each file kind names a permission. The hub checks it before it starts an upload.                                                                                           |
| `CFG-007`        | Configuration    | The address and the credential of the file store are values of the installation.                                                                                           |
| `LOG-008`        | Logging          | No log holds an upload address or a read address.                                                                                                                          |
| `UI-006`         | Web UI           | A plugin page reaches the upload control through the plugin SDK.                                                                                                           |
| `GLO-dependency` | Glossary         | The term lists the database, the IdP, and the secret store. `FILES-FR-025` adds the file store, and the term changes at the approval.                                      |

## 8. Open questions

| #   | Question                                                                        | Owner  | Answer                                                                                                                     |
| --- | ------------------------------------------------------------------------------- | ------ | -------------------------------------------------------------------------------------------------------------------------- |
| 1   | Does a file belong to a workspace, to a user, or to the row that references it? | Temuri | To a workspace or to a user. The file kind names which. `OWN-004`.                                                         |
| 2   | What is asynchronous in an upload?                                              | Temuri | The processing. The person gets the identifier at once. `FILES-FR-008`, `FILES-FR-009`.                                    |
| 3   | Which processing does an image get?                                             | Temuri | Two variants, the removal of the place and the device, and HEIC. Maroid keeps no uploaded image. `FILES-FR-012` to `-016`. |
| 4   | When does a file go away?                                                       | Temuri | An unclaimed file after 24 hours. A claimed file when its consumer deletes it. `FILES-FR-019`, `FILES-FR-021`.             |
| 5   | Does Maroid accept a file that is not an image?                                 | Temuri | Yes, any media type that a file kind declares, kept as uploaded. `FILES-FR-017`.                                           |
| 6   | Where does a person upload?                                                     | Temuri | In the deck only.                                                                                                          |
| 7   | Which limit holds?                                                              | Temuri | 25 MB for a file. No quota. `FILES-FR-002`.                                                                                |
| 8   | Do 480 and 2048 pixels fit the gallery and the viewer?                          | Temuri | Yes. `FILES-FR-012`.                                                                                                       |
| 9   | Does the hub consume a file in the first iteration?                             | Temuri | No. A file kind of the hub waits for a later change.                                                                       |
| 10  | Is the file store a dependency of the readiness?                                | Temuri | Yes. `FILES-FR-025`. `GLO-dependency` names it at the approval.                                                            |

## Retired identifiers

This file has no retired identifier.
