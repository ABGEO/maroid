---
id: FILES
title: The design decisions of the files of the hub and of a plugin
type: spec
status: approved
created: 2026-11-18
updated: 2026-11-18
approved_by: Temuri
approved_on: 2026-11-18
constrained_by: [ARC, DAT, OWN, SEC, CFG, UI]
requirements: features/files/requirements.md
---

# Specification: The design decisions of the files of the hub and of a plugin

### `FILES-DD-001`

**Realizes:** `FILES-FR-001`, `FILES-FR-002`
**Decision:** `FilePlugin.FileKinds` declares the kinds. `FileKindRegistrar` refuses
the plugin when a key fails `^[a-z0-9-]{1,64}$`, the scope is unknown, the list of media
types is empty or holds a type that `mime.ParseMediaType` refuses, `MaxSize` is not in
1 to `MaxFileSize`, an image kind holds a type outside the five image types, a kind that
is not an image holds one of the five image types, or the permission is absent from the
declared permissions (`permissionOf`, as for a widget).
**Rationale:** The registrar of the widgets already checks a permission at the load.
One broken kind fails the load of the plugin, so no person meets it. A photo in a kind
that is not an image stays as uploaded, with its EXIF, and breaks `FILES-INV-004`.
**Alternatives:** A kind in the call of each upload. The hub then checks nothing before
the first person uploads.

### `FILES-DD-002`

**Realizes:** `FILES-INV-003`, `FILES-INV-005`
**Decision:** One table for each scope, as `PSECRET-DD-001` gives the plugin secrets,
and one repository type over both. A trigger keeps the plugin, the kind, and the owner.
**Rationale:** `OWN-005` gives a scope column that is `NOT NULL`, so one table cannot
hold both scopes under one policy.
**Alternatives:** One table with two nullable scope columns. It breaks `OWN-005`.

### `FILES-DD-003`

**Realizes:** `FILES-INV-001`, `FILES-INV-002`
**Decision:** One bucket of the shared RustFS server holds every file. The key of each
object is `plugins/<pluginId>/<workspaces|users>/<ownerId>/<fileId>/<object>`, and
`<object>` is `upload`, `thumbnail.jpg`, or `display.jpg`. The key of the hub reaches
that bucket alone through a policy of the server. The bucket holds no anonymous policy.
`repository.File` filters every read by `plugin_id`.
**Rationale:** The owner runs one S3 server for several projects, with IAM. A prefix
for each consumer keeps a later file of the hub at `hub/`. The bucket refuses an unsigned
request, so only an address that the hub signs reaches a byte.
**Alternatives:** A bucket for each plugin. It needs a new policy at each new plugin.

### `FILES-DD-004`

**Realizes:** `FILES-FR-006`, `FILES-FR-007`, `FILES-NFR-003`
**Decision:** The upload address is a presigned `PUT` of AWS signature version 4 that
signs `content-length` and `content-type`, and expires after `files.upload_ttl`.
`filestore` signs with `aws-sdk-go-v2` against `files.public_endpoint`, with path style.
**Rationale:** The file store refuses a body whose length or type differs from the
signed headers, so the size limit holds without the hub on the path. One PUT carries
25 MB, so no multipart upload is needed.
**Alternatives:** A presigned `POST` with a policy of `content-length-range`. Not every
S3 server accepts it. A check of the size after the upload alone. The file store then
holds 2 GB until the worker runs.

### `FILES-DD-005`

**Realizes:** `FILES-FR-009`, `FILES-NFR-001`, `FILES-NFR-005`
**Decision:** An outbox. Each change of a file and the next task of that file commit
in one transaction. The `files` worker of `maroid worker` leases due tasks every
`files.poll_interval` with `FOR UPDATE SKIP LOCKED`, and leases again at once when a
lease filled every slot of `files.concurrency`. It sets `locked_until` to 5 minutes
ahead, and gives each task 4 minutes. Each step of a task writes in its own transaction
that `database.WithScopeTx` scopes to the owner of the task, and the task goes away in
a later transaction. A task therefore runs at least once, and every task is idempotent.
**Rationale:** A task cannot exist without its change, and a change cannot lose its
task. Two workers never run one task while the lease holds. No transaction stays open
across a read of the file store or the decode of an image. A poll of 2 seconds leaves
28 seconds of the 30 seconds of `FILES-NFR-001` to the image work. A purge runs at
once, inside the hour of `FILES-NFR-005`.
**Alternatives:** A cron job each minute. It waits up to 60 seconds. An event of the
bucket. It binds the hub to the event setup of a server that other projects share.

### `FILES-DD-006`

**Realizes:** `FILES-FR-009`, `FILES-FR-016`
**Decision:** `Start` writes a `check` task at `upload_ttl` plus 1 minute and an
`expire` task at 24 hours. `Complete` writes a `process` task at once. `check` writes a
`process` task when the upload exists and the file is `pending`, sets `bytes_absent`
when it does not, and writes `discard_upload` when an image file is `ready` and an upload
exists. `Complete` and `check` write no `process` task while one waits for the file.
`process` ends with `ready` and a `discard_upload` task in one transaction. A `process`
task whose kind the plugin declares no more waits for a load of the plugin, and fails
the file with `media_type` once the file is 24 hours old.
**Rationale:** A browser that closes after the PUT and before the completion leaves no
file in `pending` forever. A PUT that the browser repeats after the worker ran leaves
no upload behind.
**Alternatives:** The completion alone. A lost completion leaves the file `pending`.

### `FILES-DD-007`

**Realizes:** `FILES-FR-012` to `-015`, `FILES-FR-017`, `FILES-NFR-001`, `FILES-INV-004`
**Decision:** A pipeline in Go with no cgo. `image/jpeg`, `image/png`, and
`golang.org/x/image/webp` decode the common types. `github.com/gen2brain/h265/heic`
decodes HEIC and HEIF, and applies their `irot` and `imir` properties. `orient` applies
the EXIF orientation of a JPEG, of the `eXIf` chunk of a PNG, and of the `EXIF` chunk of
a WebP. `golang.org/x/image/draw` resizes with `CatmullRom` to a long edge of at most
480 and 2048 pixels, and never enlarges. A transparent pixel lands on white. `image/jpeg`
encodes each variant from the pixels, so no variant carries a metadata block. A file whose
`image.DecodeConfig` passes 100 megapixels is `damaged`. A file of a kind that is not
an image stays as `upload` and gets the variant `original`.
**Rationale:** The runtime image stays `distroless/base`, and the variants need no
encoder outside the standard library. A variant written from
decoded pixels holds no EXIF, so `FILES-INV-004` needs no strip step. Step 1 of the
build plan times the pipeline against `FILES-NFR-001` before any other step.
**Alternatives:** libvips through cgo. The runtime image moves to Debian with libvips.
imgproxy as a service. One more deployment. Both stay the fallback when step 1 fails.

### `FILES-DD-008`

**Realizes:** `FILES-FR-010`, `FILES-FR-011`
**Decision:** `sniff` reads the first 3072 bytes with a ranged read and detects the
type with `github.com/gabriel-vasile/mimetype`. `image/heif` and `image/heic` match each
other. A type outside the kind sets `media_type`. A decode error sets `damaged`. `check`
sets `bytes_absent`. The file keeps the detected type.
**Rationale:** The name and the header come from the client. The content is the fact.
**Alternatives:** `http.DetectContentType`. It knows no HEIC.

### `FILES-DD-009`

**Realizes:** `FILES-FR-019` to `FILES-FR-022`
**Decision:** A plugin table references the file with
`REFERENCES public.workspace_files (id) ON DELETE RESTRICT`, or the user table.
`Claim` sets `claimed_at` on a file of the plugin in the acting scope, and answers
`ErrFileAbsent` for any other. `Delete` runs in the transaction of the plugin
(`ADR-0023`): it deletes each row and writes a `purge` task, and maps SQLSTATE `23503`
to `ErrFileReferenced`. The plugin deletes its own rows first. `expire` deletes an
unclaimed row with a `purge` task. When the foreign key refuses it, `expire` sets
`claimed_at`, because a row of the consumer references the file.
**Rationale:** The database refuses a delete while a row references the file, in every
schema, and a lost claim heals at the expiry. PostgreSQL runs a check of a foreign key
past row level security, so the reference needs no scope. `Claim` checks the scope and
the plugin, so a plugin learns at once of an identifier that it does not own.
**Alternatives:** A list of references in the hub. The consumer then deletes its row
and the reference in two transactions, and the hub cannot see a row that points at it.
`Delete` in a transaction of the hub. The plugin then calls it after its commit, and a
failure between the two leaves a claimed file that nothing deletes.

### `FILES-DD-010`

**Realizes:** `FILES-FR-023`, `FILES-NFR-004`
**Decision:** `Store.Get` signs a `GET` against `files.public_endpoint` for each variant
of a ready file, with `files.read_ttl`. The address of an original carries
`response-content-disposition=attachment` with the name of the file. `Get` takes at
most `MaxFilesPerGet` (100) identifiers, which bounds the query and the signatures.
**Rationale:** A signature costs one HMAC and no call to the file store, so a page of 24
photos signs 48 addresses in less than 1 millisecond. A file that is not an image
downloads under its own name and never renders in the page of the deck.
**Alternatives:** A route of the hub that redirects to the file store. It adds a
request of the hub for each image.

### `FILES-DD-011`

**Realizes:** `FILES-FR-007`, `FILES-FR-023`
**Decision:** `filestore.Client` holds two clients of `aws-sdk-go-v2`: one against
`files.endpoint` for the worker, the hub, and the readiness, and one against
`files.public_endpoint` that only signs.
**Rationale:** A signature binds the host. The hub reaches the file store inside the
cluster, and the browser reaches it at a public name.
**Alternatives:** `minio-go`. Its company archived the server of the same name, and
the AWS SDK comes from the author of the S3 API.

### `FILES-DD-012`

**Realizes:** `FILES-FR-024`, `FILES-FR-025`
**Decision:** `Ping` is a `HeadBucket` with a limit of 1 second. The readiness adds the
check `file-store`, and `Start` calls `Ping` before it writes a row.
**Rationale:** A signature never reaches the file store, so without `Ping` a start
succeeds while the file store is down.
**Alternatives:** The answer of the readiness, read from a cache. The readiness of
health-go keeps no answer that a handler reads.

### `FILES-DD-013`

**Realizes:** `FILES-FR-026` to `FILES-FR-028`
**Decision:** `@maroid/ui` gains `FileUpload`. It runs `@uppy/core` with `@uppy/aws-s3`,
whose `generateObjectKey` answers the Uppy file, and whose `signRequest` calls
`client.start` for that file and answers the `PUT` with its headers. It sets
`shouldUseMultipart: false`. It draws its own daisyUI markup over the
`useDropzone` hook of `@uppy/svelte`, and loads no Uppy style sheet and no Dashboard.
It shows the `upload-progress` events for each file. After `upload-success` it calls
`client.complete`, then reads `client.get` every 2 seconds for at most 60 seconds, and
shows the file state. `PluginHost.files` gives the client, scoped to
`/workspaces/{workspaceId}/plugins/{id}/files`.
**Rationale:** Uppy brings the retries, the limit of the parallel uploads, and the
checks of the type and the size in the browser. The hooks leave the markup to the
component library, so no foreign style and no mark of Uppy reaches a page.
**Alternatives:** One `XMLHttpRequest` for each file. It gives the progress and nothing else.

### `FILES-DD-014`

**Realizes:** `FILES-FR-015`, `FILES-NFR-002`
**Decision:** Each variant is a baseline JPEG from `image/jpeg`. The thumbnail takes
quality 75 and the display image takes quality 80.
**Rationale:** Every browser displays JPEG, and the standard library encodes it with no
dependency. On a phone photo of 12 megapixels the thumbnail holds 27 KB, about a quarter
of the limit, against 24 KB as WebP. The display image holds about 420 KB.
**Alternatives:** WebP through `github.com/gen2brain/webp`. It saves about 10 percent and
runs libwebp in a WebAssembly runtime, a dependency that few projects use. AVIF. It
encodes several times slower and needs the same kind of dependency.

## Retired identifiers

This file has no retired identifier.
