---
id: ADR-0009
title: The name of a plugin lives in its metadata
type: adr
status: accepted
created: 2026-10-05
updated: 2026-10-05
decided: 2026-10-05
changes: [PLG-012, GLO-plugin-name, GLO-plugin-description]
supersedes:
superseded_by:
---

# The name of a plugin lives in its metadata

## Context

`pluginapi.Metadata` carries the identifier, the version, and the API version. A plugin
declares no name there.

A plugin with a user interface declares a name in `UIManifest.Name`.

`PluginEntry`, which `GET /plugins`, the enablements of a workspace, and the MCP tool
`list_plugins` answer, carries no name. The deck reads the name of the UI manifest, and
falls back to the last segment of the identifier. A person therefore reads "gwp",
"tbilisi-energy", and "telasi" on the page of the plugins of a workspace and on the page
`/admin/plugins`.

A name is a fact about the plugin, not about one capability. A plugin with no user
interface still appears in a list, in a menu, and in the answer of an agent.

## Decision

`pluginapi.Metadata` gains a required `Name` and an optional `Description`. The hub
refuses to load a plugin whose name is empty or too long. `UIManifest` loses `Name`.
`PluginEntry` carries both values, so every client reads the name from one place.

## Rationale

One source gives one answer. A name in two places can differ, and a client must then
choose. A name in the UI manifest alone leaves every other plugin nameless.

The Go compiler and the loader check the metadata of every plugin at the load. A missing
name then fails the start of the hub, and not a page of the deck a week later.

A name of 64 characters fits one row of the sidebar. A description of 280 characters
fits one or two sentences on a card. The limits count characters, not bytes, because a
name in Georgian takes three bytes for each letter.

The API version stays `v1`. `BLD-004` rebuilds every plugin after a change in
`libs/pluginapi`, and a Go plugin built against another version of the package fails
to open in any case. A plugin that a developer rebuilds with no name fails with an
error that names the plugin and the missing field, which says more than a version
mismatch.

The description stays optional. A plugin whose purpose its name already gives needs no
sentence, and a required field invites filler.

## Alternatives

| Alternative                                              | Why we did not select it                                                                     |
| -------------------------------------------------------- | -------------------------------------------------------------------------------------------- |
| Keep the name in the UI manifest, and add one to the metadata. | Two sources of one fact. The deck needs a rule for which one wins.                      |
| Derive the name from the identifier.                     | `tbilisi-energy` is no name a person reads. A plugin cannot choose capitals, spaces, or Georgian. |
| An optional name in the metadata.                        | Every client keeps the fallback to the identifier, and the problem stays for each new plugin. |
| Require a unique name across the loaded plugins.         | The identifier is already unique, as `PLG-003` and `PLG-011` give. Two plugins of one name confuse a person, and do not break the hub. |
| Bump the API version to `v2`.                            | Every plugin rebuilds anyway, and the missing name gives the clearer error.                  |

## Consequences

### Rules that change

`PLG` gains `PLG-012`. The text:

> A plugin declares a name in `pluginapi.Metadata.Name`. The name holds 1 to 64
> characters after the hub trims the white space at each end. A plugin may declare a
> description in `pluginapi.Metadata.Description`, of at most 280 characters.
>
> The hub refuses to load a plugin whose name or description breaks a limit, and the
> error names the plugin and the field.
>
> **Why:** Every client shows a plugin by its name. A name that only the user interface
> declares leaves the other plugins nameless.

The glossary gains two terms.

| ID                        | Term               | Definition                                                                    |
| ------------------------- | ------------------ | ----------------------------------------------------------------------------- |
| `GLO-plugin-name`         | plugin name        | The name that a person reads for a plugin. The plugin declares it in its metadata. |
| `GLO-plugin-description`  | plugin description | One or two sentences that tell a person what a plugin does. It is optional.   |

`PLG-004` does not change.

### Specifications to examine

| Document                        | Why                                                                                  |
| ------------------------------- | ------------------------------------------------------------------------------------ |
| `features/pcap/requirements.md` | The report of a plugin gains the name and the description.                           |
| `features/pcap/spec.md`         | `PluginEntry`, `api.yaml`, the `ui` capability without `name`, and the deck that reads the name from the entry. |
| `features/plugacc/spec-clients.md` | The pages of the plugins read the name and show the description.                 |

### Code that changes

- `libs/pluginapi`: `Metadata.Name`, `Metadata.Description`, and `UIManifest` without `Name`.
- The loader of the hub: the check of `PLG-012`.
- `PluginEntry`, `api.yaml`, and the client types of the deck and of `@maroid/plugin-sdk`.
- The deck: `displayNameOf` reads the entry, and drops the fallback to the identifier.
- Every plugin of the repository declares its name.

### Result

- Positive: every plugin shows a name that its author chose, in every client.
- Positive: one source of the name, and the loader checks it.
- Negative: every plugin changes in one release, and a plugin built before it fails to load.
- Work that follows: the change to `PCAP`, then the build of every plugin under `BLD-004`.
