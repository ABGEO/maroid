---
title: The Maroid guidelines
type: index
status: active
created: 2026-09-11
updated: 2026-09-23
---

# The Maroid guidelines

The guidelines are the files in this directory. They hold the rules of the whole system.
A rule applies to every feature, to every plugin, and to every line of code.
A specification must not contradict a rule.

A rule is not about one feature. A statement about one feature belongs in
`specs/features/<key>/requirements.md`.

This file is an index. It holds no rule.

## The guidelines

A file name holds the slug only. The frontmatter of the file holds the identifier.
The identifier is the prefix of every rule in that file.
`PLG-004` is the fourth rule in `plugin-model.md`.

| ID    | File                                           | Governs                                                     | Rules |
| ----- | ---------------------------------------------- | ----------------------------------------------------------- | ----- |
| `ARC` | [architecture.md](./architecture.md)           | The shape of the system. The technology of each layer.      | 10    |
| `PLG` | [plugin-model.md](./plugin-model.md)           | The contract between the hub and a plugin.                  | 11    |
| `DAT` | [data.md](./data.md)                           | The schemas, the migrations, the database access.           | 11    |
| `OWN` | [ownership.md](./ownership.md)                 | The user record, the acting user, the isolation of a row.   | 9     |
| `UI`  | [web-ui.md](./web-ui.md)                       | The contract between the shell and a plugin user interface. | 9     |
| `CFG` | [configuration.md](./configuration.md)         | The configuration and the secrets.                          | 7     |
| `API` | [http-api.md](./http-api.md)                   | The HTTP routes, the middleware, the responses.             | 7     |
| `ERR` | [errors.md](./errors.md)                       | The shape of a failure. The problem types.                  | 7     |
| `RES` | [rest.md](./rest.md)                           | The REST standard, the deviations, the page, the payload.   | 11    |
| `SEC` | [security.md](./security.md)                   | The identity, the token, the allowlists.                    | 10    |
| `TG`  | [telegram.md](./telegram.md)                   | The bot, the commands, the conversations.                   | 7     |
| `MQT` | [mqtt.md](./mqtt.md)                           | The topics, the namespaces, the delivery.                   | 7     |
| `JOB` | [jobs.md](./jobs.md)                           | The workers and the cron jobs.                              | 8     |
| `NTF` | [notifications.md](./notifications.md)         | The channels, the transports, the failover.                 | 6     |
| `PKG` | [package-layout.md](./package-layout.md)       | The directories of a plugin and of the hub.                 | 5     |
| `DEP` | [dependencies.md](./dependencies.md)           | How the container builds and caches a dependency.           | 8     |
| `LIF` | [lifecycle.md](./lifecycle.md)                 | The start order and the shutdown order.                     | 6     |
| `CLI` | [cli.md](./cli.md)                             | The command tree and the command contract.                  | 6     |
| `REP` | [repository.md](./repository.md)               | The Go code that reads and writes the database.             | 7     |
| `EXT` | [external-services.md](./external-services.md) | The clients of a third-party API.                           | 7     |
| `LOG` | [logging.md](./logging.md)                     | The logger, the attributes, the redaction.                  | 10    |
| `GO`  | [go-style.md](./go-style.md)                   | The Go code style.                                          | 10    |
| `TS`  | [frontend-style.md](./frontend-style.md)       | The TypeScript and Svelte code style.                       | 6     |
| `TST` | [testing.md](./testing.md)                     | The tests.                                                  | 4     |
| `BLD` | [build-release.md](./build-release.md)         | The build and the release.                                  | 4     |
| `PRC` | [process.md](./process.md)                     | How a change moves from an idea to the code.                | 3     |
| `SPC` | [specification.md](./specification.md)         | The artifacts of a specification and each declaration.      | 6     |
| `GIT` | [git.md](./git.md)                             | The commits. The format, the scope, the authorship.         | 6     |
| `TRC` | [traceability.md](./traceability.md)           | The identifiers and the frontmatter.                        | 8     |
| `LNG` | [language.md](./language.md)                   | How to write every document, comment, and commit message.   | 13    |
| `GLO` | [glossary.md](./glossary.md)                   | The terms. One term has one meaning.                        | None  |

## The identifiers

A guideline rule has two segments: `<GID>-<NNN>`.
A feature statement has three segments: `<KEY>-<TYPE>-<NNN>`.
The count of the segments tells you the level. See `TRC-001`.

A guideline identifier has two or three characters.
A feature key has four to eight characters. The two sets cannot collide.

A guideline identifier is permanent. A retired guideline identifier never repeats.

## Where to put a new rule

| The rule is about                               | Put it in |
| ----------------------------------------------- | --------- |
| The shape of the system or a technology choice  | `ARC`     |
| What a plugin must do to load and to register   | `PLG`     |
| A schema, a migration, or a query               | `DAT`     |
| A page, a remote, or the navigation             | `UI`      |
| A configuration value or a secret               | `CFG`     |
| An HTTP route, a prefix, or the middleware      | `API`     |
| An error response or a problem type             | `ERR`     |
| The API standard, a deviation from it, a success body, a page, a cursor, a property name, an OpenAPI document, or a client of the API | `RES` |
| A token, an identity, or an allowlist           | `SEC`     |
| A user, the owner of a row, or row isolation    | `OWN`     |
| A bot command or a conversation                 | `TG`      |
| An MQTT topic or a subscriber                   | `MQT`     |
| A cron job or a background worker               | `JOB`     |
| A notification channel or transport             | `NTF`     |
| A directory or a package name                   | `PKG`     |
| A shared dependency or a provider               | `DEP`     |
| The start order or the shutdown                 | `LIF`     |
| A CLI command or a flag                         | `CLI`     |
| A query, a repository, or an entity             | `REP`     |
| A call to a third-party API                     | `EXT`     |
| A log message or a log attribute                | `LOG`     |
| How to write Go                                 | `GO`      |
| How to write TypeScript or Svelte               | `TS`      |
| A test                                          | `TST`     |
| A build step or a release step                  | `BLD`     |
| How we work                                     | `PRC`     |
| A commit, a branch, or a tag                    | `GIT`     |
| An identifier or a frontmatter field            | `TRC`     |
| A word, a sentence, or a document               | `LNG`     |
| An artifact or a declaration in a specification | `SPC`     |

A rule that fits no guideline needs a new guideline. A new guideline needs an ADR.

## How a rule changes

An accepted ADR in [`specs/adr/`](../adr/) changes a rule. Nothing else changes
a rule.

1. Write an ADR. Use [`specs/adr/template.md`](../adr/template.md).
2. Name each rule identifier that changes.
3. After the owner accepts the ADR, correct the guideline file.
4. Set the `updated` field in the frontmatter of the guideline.
5. Find each specification that cites the rule. Examine each one.
6. Add a row to the `Retired identifiers` section of the guideline for a rule that you remove.

Each guideline file holds its own `Retired identifiers` section.
