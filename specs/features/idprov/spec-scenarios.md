---
id: IDPROV
title: The scenarios of the providers and the local account
type: spec
status: approved
created: 2026-10-08
updated: 2026-10-09
approved_by: Temuri
approved_on: 2026-10-08
constrained_by: [TST, TRC]
requirements: features/idprov/requirements.md
---

# Specification: The scenarios of the providers and the local account

`spec.md` holds the design. This file holds the scenarios. `SPC-001` divides them.

A unit scenario runs against a fake `dex.Client` that holds connectors and passwords in
memory. An integration scenario adds the container that `testdb.Start` gives, with the
migrations of the hub applied. A manual scenario runs against the compose deployment
with Dex v2.46.0.

Unless a scenario says otherwise, Zura is the only active administrator, and Dex holds
the static connector `mock` and no other. `telegram.token` starts with `8514702227:`.

## `IDPROV-SC-001`

**Verifies:** `IDPROV-FR-001`, `IDPROV-DD-002`
**Layer:** unit

**Given** Dex holds `mock` with no `maroidPreset`, and `telegram` with
`maroidPreset: telegram`.
**When** Zura sends `GET /providers`.
**Then** the page holds `mock` with `static: true` and no preset, and `telegram` with
`static: false` and `preset: telegram`.

## `IDPROV-SC-002`

**Verifies:** `IDPROV-FR-002`, `IDPROV-FR-003`
**Layer:** unit

**Given** an issuer that publishes a discovery document.
**When** Zura adds `abgeo-cloud` with a name, the issuer, a client identifier, and a
secret, and names no claim and no scopes.
**Then** the answer is 201 with `user_id_key: sub` and the scopes `openid`, `profile`,
and `email`. Dex holds the connector of type `oidc`, with `maroidPreset: oidc`.

## `IDPROV-SC-003`

**Verifies:** `IDPROV-FR-002`
**Layer:** unit

**Given** Dex holds `mock` and `abgeo-cloud`.
**When** Zura adds a generic OIDC provider named `telegram`, then `local`, then `mock`,
then `abgeo-cloud`, then `Bad_Id`.
**Then** the first two and the last answer 422 at `/id`. `mock` and `abgeo-cloud`
answer `provider-exists`. Dex holds no new connector.

## `IDPROV-SC-004`

**Verifies:** `IDPROV-FR-004`, `IDPROV-FR-005`, `IDPROV-INV-002`
**Layer:** unit

**Given** no Telegram provider.
**When** Zura adds the Telegram preset with a secret and no client identifier.
**Then** Dex holds `telegram` named `Telegram`, with the issuer of Telegram,
`userIDKey: id`, the scopes `openid` and `profile`, and `clientID: 8514702227`. The
answer shows that client identifier.

## `IDPROV-SC-005`

**Verifies:** `IDPROV-INV-003`
**Layer:** unit

**Given** Dex holds `telegram` and `local`.
**When** Zura adds the Telegram preset, then the local preset.
**Then** both answer `provider-exists`.

## `IDPROV-SC-006`

**Verifies:** `IDPROV-FR-006`
**Layer:** unit

**Given** no local provider.
**When** Zura adds the local preset with an empty body beyond the preset.
**Then** Dex holds `local` of type `local`, named `Email`.

## `IDPROV-SC-007`

**Verifies:** `IDPROV-FR-007`
**Layer:** unit

**Given** `oidc.issuer` is `https://auth.maroid.localhost`.
**When** Zura reads `abgeo-cloud`, then `local`.
**Then** both carry `redirect_uri: https://auth.maroid.localhost/callback`, so the form
of a new provider reads the address from any provider of the list.

## `IDPROV-SC-008`

**Verifies:** `IDPROV-FR-008`, `IDPROV-DD-005`
**Layer:** unit

**Given** the provider `abgeo-cloud`.
**When** Zura changes its options to `{getUserInfo: true}`, then to `{userIDKey: email}`,
then to `{getUserinfo: true}`, then to `{getUserInfo: "yes"}`.
**Then** the first lands, and the stored config holds `getUserInfo: true`. The other
three answer 422 at `/options/userIDKey`, `/options/getUserinfo`, and
`/options/getUserInfo`, and the stored config keeps the first change.

## `IDPROV-SC-009`

**Verifies:** `IDPROV-FR-009`, `IDPROV-INV-001`
**Layer:** unit

**Given** the providers `abgeo-cloud` and `telegram`.
**When** Zura renames `abgeo-cloud`, changes its client identifier and its scopes,
then sends a change of its `issuer`, then of its `user_id_key`, then a change of the
scopes of `telegram`.
**Then** the first change lands. The other three answer 422 at the field they name,
and Dex keeps the issuer, the claim, and the scopes.

## `IDPROV-SC-010`

**Verifies:** `IDPROV-FR-010`, `IDPROV-FR-024`
**Layer:** unit

**Given** `abgeo-cloud` with the secret `s1`.
**When** Zura renames it with no secret, then sends the secret `s2`.
**Then** Dex holds `s1` after the first change and `s2` after the second. Each answer
carries `client_secret_set: true` and no secret.

## `IDPROV-SC-011`

**Verifies:** `IDPROV-FR-011`
**Layer:** unit

**Given** a test server that answers 404 at the discovery path.
**When** Zura adds a generic OIDC provider with that issuer, then with
`https://nowhere.invalid`.
**Then** both answer 422 at `/issuer`, and Dex holds no new connector.

## `IDPROV-SC-012`

**Verifies:** `IDPROV-FR-012`, `IDPROV-DD-007`
**Layer:** integration

**Given** Zura and Nino are active administrators. Zura holds identities at `local`
and `telegram`. Nino holds one at `local`. Ana, no administrator, holds one at `local`.
Blocked administrator Gio holds one at `local`.
**When** Zura reads `local`, then `telegram`.
**Then** `local` carries `identity_count: 4` and names Nino alone. `telegram` carries
`identity_count: 1` and names nobody.

## `IDPROV-SC-013`

**Verifies:** `IDPROV-FR-013`, `IDPROV-FR-014`, `IDPROV-DD-003`
**Layer:** integration

**Given** `abgeo-cloud` with two identities, and a fake Dex that fails the first
`DeleteConnector`.
**When** Zura removes `abgeo-cloud`, then removes it again.
**Then** the first answers `not-ready`, and both identities and the connector remain.
The second answers 204, and neither remains.

## `IDPROV-SC-014`

**Verifies:** `IDPROV-FR-015`
**Layer:** unit

**Given** `local` with the passwords of Zura and Ana.
**When** Zura removes `local`.
**Then** Dex holds no `local` connector and no password.

## `IDPROV-SC-015`

**Verifies:** `IDPROV-FR-016`
**Layer:** unit

**When** Zura renames `mock`, then removes it.
**Then** both answer `provider-static`, and Dex keeps `mock`.

## `IDPROV-SC-016`

**Verifies:** `IDPROV-FR-017`
**Layer:** unit

**Given** Dex holds `mock`, `telegram`, and `local`. Ana holds an identity at `telegram`.
**When** Ana sends `GET /auth/identities`.
**Then** the page holds the three providers by their names in Dex, and marks `telegram`
attached.

## `IDPROV-SC-017`

**Verifies:** `IDPROV-FR-018`, `IDPROV-INV-004`
**Layer:** integration

**Given** `local` exists, and Ana holds a local account at `ana@home.example`.
**When** Zura gives Nina `nina@home.example`, then gives Beka `ana@home.example`.
**Then** the first answers 201, and Dex holds the password with the `user_id` of Nina.
The second answers `local-account-exists`, and Beka holds no identity.

## `IDPROV-SC-018`

**Verifies:** `IDPROV-FR-019`
**Layer:** unit

**Given** Nina holds a local account.
**When** Zura sets a new password for Nina.
**Then** the hash that Dex holds matches the new password and not the old one.

## `IDPROV-SC-019`

**Verifies:** `IDPROV-FR-020`
**Layer:** integration

**Given** Nina holds identities at `local` and `telegram`. Beka holds one at `local` only.
**When** Zura removes the local account of Nina, then of Beka.
**Then** the first answers 204, and Dex holds no password for Nina. The second answers
`identity-last`, and Dex keeps the password of Beka.

## `IDPROV-SC-020`

**Verifies:** `IDPROV-FR-021`
**Layer:** integration

**Given** Nina holds identities at `local` and `telegram`.
**When** Nina sends `DELETE /auth/identities/local`.
**Then** the identity and the password in Dex are both gone.

## `IDPROV-SC-021`

**Verifies:** `IDPROV-FR-022`
**Layer:** unit

**When** Nina starts an attach with the provider `local`.
**Then** the answer is 400, `request-invalid`, and names the parameter `provider`. No
flow starts.

## `IDPROV-SC-022`

**Verifies:** `IDPROV-FR-023`
**Layer:** unit

**When** Zura gives a local account with a password of 11 characters, of 12, of 24
characters of three bytes each, and of 73 bytes.
**Then** 12 characters and 72 bytes land. 11 characters and 73 bytes answer 422 at
`/password`.

## `IDPROV-SC-023`

**Verifies:** `IDPROV-FR-024`
**Layer:** unit

**Given** every route of section 4.3 of `spec.md` answers once, and a log handler that
records every line.
**Then** no body and no log line holds the password, its hash, or a client secret.

## `IDPROV-SC-024`

**Verifies:** `IDPROV-FR-025`, `IDPROV-FR-026`
**Layer:** manual

**Given** a new instance, and a Dex file with no connector.
**When** the operator runs `maroid user invite --first-name Zura --admin`, then
`maroid user password --user <the first line> --email zura@home.example`, and types
the password twice.
**Then** the sign in page of Dex shows "Log in with Email". Zura signs in and reaches
`/admin/providers`.

## `IDPROV-SC-025`

**Verifies:** `IDPROV-FR-025`
**Layer:** unit

**Given** Zura holds a local account, and Dex holds no `local` connector.
**When** the operator runs `maroid user password --user <Zura>` with a new password.
**Then** Dex holds `local` again and the new password. A run that also names
`--email` fails, and changes nothing.

## `IDPROV-SC-026`

**Verifies:** `IDPROV-FR-027`
**Layer:** unit

**When** the command reads two different values at the prompt, then reads a password
from the standard input with no terminal.
**Then** the first run fails and changes nothing. The second sets the password. The
command declares no flag that takes a password.

## `IDPROV-SC-027`

**Verifies:** `IDPROV-FR-028`
**Layer:** unit

**When** the operator runs `maroid user invite --first-name Nina`.
**Then** the first line is the identifier of the new record, and the second line is
the address of the invitation.

## `IDPROV-SC-028`

**Verifies:** `IDPROV-NFR-001`
**Layer:** manual

**Given** the compose deployment, running.
**When** Zura adds `abgeo-cloud` in the deck, then opens the sign in page of Dex.
**Then** the page shows the button. After a rename, a reload shows the new name. After
a removal, a reload shows no button. Neither process restarted.

## `IDPROV-SC-029`

**Verifies:** `IDPROV-NFR-002`
**Layer:** unit

**Given** a fake Dex that holds each call for 6 seconds, and `dex.timeout` of 5 seconds.
**When** Zura adds a provider, then gives Nina a local account.
**Then** each answer arrives between 5 and 5.5 seconds after the request, as
`not-ready` with the dependency `dex`. Nina holds no identity.

## `IDPROV-SC-030`

**Verifies:** `IDPROV-INV-005`
**Layer:** integration

**When** Zura gives Nina a local account.
**Then** the identity is `(local, <the identifier of Nina>)`, and the password in Dex
carries the same `user_id`.

## `IDPROV-SC-031`

**Verifies:** `IDPROV-DD-001`, `IDPROV-DD-002`
**Layer:** manual

**Given** Dex with `tlsClientCA` set.
**When** `grpcurl` calls `ListConnectors` with no client certificate, then the hub adds
`local` and a generic OIDC provider.
**Then** the call with no certificate fails at the handshake. Dex opens both stored
connectors, with `maroidPreset` in their config.

## Retired identifiers

This file has no retired identifier.
