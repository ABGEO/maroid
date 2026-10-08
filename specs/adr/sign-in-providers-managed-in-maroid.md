---
id: ADR-0010
title: Maroid manages the providers of Dex
type: adr
status: accepted
created: 2026-10-08
updated: 2026-10-08
decided: 2026-10-08
changes: [SEC-001, SEC-003, SEC-011, API-003, CLI-001, GLO-provider,
  GLO-provider-preset, GLO-static-provider, GLO-local-provider]
supersedes:
superseded_by:
---

# Maroid manages the providers of Dex

## Context

A provider lives in two places today. The owner writes a connector into the
configuration file of Dex, then writes the same identifier and name into
`auth.providers` of the hub, so that `EXTID-FR-009` reports it. A change edits two
files and restarts two processes.

`ADR-0002` and `SEC-001` assume this. `SEC-001` says that a further connector costs
one entry in the configuration of Dex, and that Maroid holds no password.
`EXTID` question 2 excludes a provider that holds a password.

A new instance has a second problem. With no connector, nobody signs in, so nobody
redeems the invitation of the first administrator.

A spike against Dex `v2.46.0` (API 4) on 2026-10-08 found these facts.

| Fact of Dex                                                                                       | Consequence for Maroid                                                  |
| ------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------- |
| The gRPC API creates, updates, lists, and deletes a connector when `DEX_API_CONNECTORS_CRUD=true`. A change takes effect with no restart. | The hub can own the lifecycle of a provider.                              |
| `ListConnectors` returns the connectors of the file with the stored ones, and marks neither. A write to a connector of the file fails with `read-only`. | A connector of the file stays possible, and the hub cannot change it.     |
| Dex validates nothing on a write. It accepted an unknown type, an issuer that does not resolve, and an unknown key. The login page then lists the broken connector, and a click answers 400. | The hub validates a connector before it writes it.                        |
| Dex returns a stored connector config as the exact JSON that it received.                         | The hub reads back what it wrote, and no hub table is needed.             |
| A connector of type `local` reads one password database that every `local` connector shares. Its login form asks for an email address. | One local provider is the only sensible count.                            |
| `CreatePassword` takes a bcrypt hash and a `user_id` that the caller chooses. Dex has no page where a person sets or changes a password. | A password reaches Dex only through a caller of the API.                  |
| Seven wrong passwords in a row did not block a local account.                                     | A local account carries no lockout today.                                 |
| The gRPC port of the compose file is published to the host in plain text, with no client authentication. | Anyone who reaches the port sets a password and signs in as that person.  |

## Decision

Dex stays the store of every provider. The hub manages a provider through the gRPC
API of Dex, over mutual TLS, and offers three presets: local, Telegram, and generic
OIDC. An administrator sets the password of a local account through the hub, and
the CLI gives the first administrator a local account.

## Rationale

Dex already stores the connector and returns it as written. A second store in the
hub only adds a copy that drifts.

A preset fixes the fields that decide the upstream user identifier. `SEC-003` joins
an identity on that identifier, so a changed issuer or `userIDKey` orphans every
identity of the provider. A free YAML field carries the other options of Dex, and
the hub refuses a key that the preset fixes.

The Telegram preset fixes the identifier `telegram`, the issuer, and `userIDKey: id`.
The bot resolves an update by that identifier and that number. `ADR-0002` gives both.

A local account solves the first sign in with no service outside the deployment.
The same CLI command restores access after a lockout, because it needs only the
database and the API of Dex. A connector in the file of Dex remains a second way in.

The API of Dex writes a password and a connector. A caller of it signs in as any
person, so it takes the same guard as a signing key.

## Alternatives

| Alternative                                                       | Why we did not select it                                                                                          |
| ----------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------- |
| Keep the connectors in the file of Dex and in `auth.providers`.   | Two files for one fact, and a restart for each change.                                                             |
| A hub table that holds the providers and writes them to Dex.       | Two stores for one fact. A failed write leaves them apart.                                                         |
| One raw JSON field and no preset.                                 | Nothing then fixes the issuer or `userIDKey`, and one edit orphans every identity of the provider.                 |
| A web wizard that a one-time code opens on a new instance.         | Anyone who reaches the instance first claims it. The CLI needs access to the host.                                 |
| `staticPasswords` in the file of Dex for the first administrator. | The hash lives in a file, and nobody changes it from Maroid. It stays possible, because the file keeps working.   |
| A page where a local user sets or changes the password.            | Dex has none, so the hub would build it. Deferred until a second local user needs it.                              |
| The plain gRPC port inside the network of the deployment.         | Any process on that network then sets a password.                                                                  |
| Client certificates that OpenBao PKI issues to the hub.            | A second moving part on the sign in path. A file path accepts a certificate from any issuer, OpenBao included.     |

## Consequences

### Rules that change

`SEC-001` loses "holds no password". The new text:

> Identity comes from Dex. Dex is the authorization server, and the hub is its
> consumer. Telegram is one provider of Dex, and it is not the only one.
>
> Maroid stores no password and no password hash. An administrator sets the
> password of a local account through the hub. The hub hashes it with bcrypt in
> memory, sends the hash to Dex, and logs neither value.
>
> Maroid holds a local user record. The record carries no credential. A workspace
> owns the data, and `OWN-010` gives it.
>
> **Why:** One person holds several external accounts. A further provider costs one
> form in the deck, and no flow inside the hub.

`SEC-003` gains a second paragraph after the one on `federated_claims`:

> A provider keeps its identifier for its life. A provider of the OIDC kind also
> keeps its issuer and its `userIDKey`. Deleting a provider deletes its identities.

`SEC-011` gains a sentence after the one on the management API:

> An administrator manages the providers of the instance, and sets the password
> of a local account.

The last sentence of `SEC-011` becomes:

> The CLI makes the first administrator, and gives it a local account.

`API-003` gains one row:

> | Method and path | Serves                       | Access        |
> | --------------- | ---------------------------- | ------------- |
> | `/providers*`   | The providers of the instance | Administrator |

`/users*` already carries the password of a local account.

`CLI-001` gains one line in the tree:

> ```
> maroid user password    The local account of a user record. See SEC-011.
> ```

The glossary changes `GLO-provider` and gains three terms:

> | ID                    | Term            | Definition                                                                                  |
> | --------------------- | --------------- | ------------------------------------------------------------------------------------------- |
> | `GLO-provider`        | provider        | One account system that Dex federates. Telegram is one. An administrator manages it, or the file of Dex holds it. |
> | `GLO-provider-preset` | provider preset | The kind of a provider that the hub offers: local, Telegram, or generic OIDC. It fixes the fields that decide the user identifier. |
> | `GLO-static-provider` | static provider | A provider that the file of Dex holds. The hub lists it and changes nothing on it.         |
> | `GLO-local-provider`  | local provider  | The provider whose accounts sign in with an email address and a password that Dex holds.   |

### Configuration

`auth.providers` goes. The hub reads the providers from Dex.

The hub gains the address of the gRPC API of Dex, and the paths of its client
certificate, its key, and the authority of Dex. Dex sets `DEX_API_CONNECTORS_CRUD`,
a server certificate, and `tlsClientCA`. `docker-compose.yaml` stops publishing the
port, and `chart/` follows.

### Specifications to examine

| Document                           | Why                                                                                               |
| ---------------------------------- | ------------------------------------------------------------------------------------------------- |
| `features/extid/requirements.md`   | Question 2 excludes a provider that holds a password. `EXTID-FR-009` reads the providers from Dex. |
| `features/extid/spec.md`           | The configuration table names `auth.providers`.                                                    |
| `features/plugacc/spec.md`         | The user management of the deck gains the password of a local account.                             |

### Result

- Positive: one administrator action adds, changes, or removes a provider, with no restart.
- Positive: a new instance signs in its first administrator with no external provider.
- Negative: Dex stores each connector secret as plain JSON in its database. Maroid
  stores none, and cannot change that.
- Negative: a password passes through the hub in memory. A defect that logs a request
  body leaks it.
- Negative: a local account has no lockout after failed attempts.
- Negative: a delete leaves a user record with no identity, against the last identity
  rule of a detach. The administrator issues a new invitation.
- Work that follows: a requirements document for the providers of the instance, then
  its specification. The specification gives the mutual TLS of the API of Dex as a
  design decision, because this feature is its only caller.
