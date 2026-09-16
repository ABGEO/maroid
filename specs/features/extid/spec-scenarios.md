---
id: EXTID
title: The scenarios of the external identities and the delegated sign in
type: spec
status: approved
created: 2026-09-15
updated: 2026-09-16
approved_by: Temuri
approved_on: 2026-09-16
constrained_by: [TST, SEC, OWN, TRC]
requirements: features/extid/requirements.md
---

# Specification: The scenarios of the external identities and the delegated sign in

`spec.md` holds the design. This file holds the scenarios, because the two together
pass the size that `LNG-013` gives. See `SPC-001`.

Every scenario below starts from two user records, U and V. U holds one identity at
the provider `telegram`. V holds one identity at the provider `cloud`. A test that
needs a token builds one that a local key set signs, and points the verifier at that
key set.

## `EXTID-SC-001`

**Verifies:** `EXTID-FR-001`
**Layer:** integration

**Given** a token whose `federated_claims` names `telegram` and the external account
of U.
**When** the request reaches the middleware.
**Then** the acting user is U, and the handler runs.

## `EXTID-SC-002`

**Verifies:** `EXTID-FR-001`, `EXTID-FR-003`
**Layer:** integration

**Given** a valid token whose `federated_claims` names an external account that no
identity holds.
**When** the sign in finishes at the callback.
**Then** the answer is a redirect with a failure, and the counts of the rows in
`public.users` and `public.identities` do not change.

## `EXTID-SC-003`

**Verifies:** `EXTID-FR-001`
**Layer:** integration

**Given** the identity of U, and U with `status = 'blocked'`.
**When** U sends a request with a token that is still valid.
**Then** the answer is 401.

## `EXTID-SC-004`

**Verifies:** `EXTID-FR-002`
**Layer:** integration

**Given** a sign in whose external account holds no identity.
**When** the callback finishes.
**Then** the redirect carries `error=no_identity`, which differs from the
`error=auth_failed` of a failed exchange.

## `EXTID-SC-005`

**Verifies:** `EXTID-FR-004`
**Layer:** integration

**Given** U signed in, and an external account at `cloud` that no identity holds.
**When** U runs the attach for `cloud` and the callback finishes.
**Then** U holds two identities, and a later sign in with either one resolves to U.

## `EXTID-SC-006`

**Verifies:** `EXTID-FR-004`
**Layer:** integration

**Given** U signed in, and the attach for `telegram` with the external account that U
already holds.
**When** the callback finishes.
**Then** the answer reports success, and U still holds exactly one identity at
`telegram` with the same identifier.

## `EXTID-SC-007`

**Verifies:** `EXTID-FR-005`, `EXTID-INV-001`
**Layer:** integration

**Given** U signed in, and the external account that the identity of V names.
**When** U runs the attach for that account and the callback finishes.
**Then** the redirect carries `error=identity_taken`, U holds one identity, and V
holds one identity.

## `EXTID-SC-008`

**Verifies:** `EXTID-FR-006`
**Layer:** integration

**Given** U started an attach, and the flow row that names U.
**When** the callback arrives with that state, with every cookie removed, and with a
forged cookie that names V.
**Then** the identity names U.

## `EXTID-SC-009`

**Verifies:** `EXTID-FR-007`
**Layer:** integration

**Given** U holds two identities.
**When** U detaches `cloud`.
**Then** U holds one identity, and a sign in with the detached account finds no
identity.

## `EXTID-SC-010`

**Verifies:** `EXTID-FR-008`
**Layer:** integration

**Given** U holds one identity.
**When** U detaches it.
**Then** the answer is 409, and U still holds one identity.

## `EXTID-SC-011`

**Verifies:** `EXTID-FR-009`
**Layer:** integration

**Given** `auth.providers` lists `telegram` and `cloud`, and U holds an identity at
`telegram` only.
**When** U reads `/auth/identities`.
**Then** the body names both providers, marks `telegram` as attached with its handle,
and marks `cloud` as not attached.

## `EXTID-SC-012`

**Verifies:** `EXTID-FR-010`
**Layer:** integration

**Given** a database with no record for the person.
**When** the owner runs `maroid user invite --first-name Nino --last-name Beridze`.
**Then** `public.users` holds one new record, `public.invitations` holds one valid row
for it, and the command prints an address that carries the token.

## `EXTID-SC-013`

**Verifies:** `EXTID-FR-011`
**Layer:** integration

**Given** the record U, whose only invitation expired.
**When** the owner runs `maroid user invite --user <the identifier of U>`.
**Then** `public.invitations` holds a second valid row for U, and the count of the
records in `public.users` does not change.

## `EXTID-SC-014`

**Verifies:** `EXTID-FR-012`
**Layer:** integration

**Given** a record with no identity and a valid invitation for it.
**When** the person opens the address, signs in, and the callback finishes.
**Then** that record holds one identity, the invitation carries a `consumed_at`, and
the answer sets the cookie.

## `EXTID-SC-015`

**Verifies:** `EXTID-FR-013`
**Layer:** integration

**Given** an invitation that one sign in already consumed.
**When** a second person opens the same address.
**Then** the answer is 400, and the record still holds one identity.

## `EXTID-SC-016`

**Verifies:** `EXTID-FR-014`
**Layer:** integration

**Given** an invitation whose `expires_at` is in the past.
**When** the person opens the address.
**Then** the answer is 400, and no request reaches Dex.

## `EXTID-SC-017`

**Verifies:** `EXTID-FR-015`
**Layer:** integration

**Given** U with `first_name = 'Temuri'`, and a token whose `name` claim is
`Someone Else`.
**When** U signs in.
**Then** `first_name` and `last_name` of U do not change.

## `EXTID-SC-018`

**Verifies:** `EXTID-FR-016`
**Layer:** integration

**Given** the identity of U at `telegram` with the handle `old_handle`.
**When** U signs in with a token whose `preferred_username` claim is `new_handle`.
**Then** that identity carries `new_handle`, and the identity of U at `cloud` does
not change.

## `EXTID-SC-019`

**Verifies:** `EXTID-FR-017`
**Layer:** manual

**Given** U holds an identity at `telegram`, and U created one scoped record in the
web shell.
**When** U sends an update to the bot that lists that record.
**Then** the bot lists the record that U created in the shell.

## `EXTID-SC-020`

**Verifies:** `EXTID-NFR-001`
**Layer:** integration

**Given** a verifier pointed at a key set server that counts each request, and a
token that its current key signs.
**When** 1000 requests arrive with that token.
**Then** the count of the requests at the key set server is one or less.

## `EXTID-SC-021`

**Verifies:** `EXTID-NFR-002`
**Layer:** manual

**Given** Dex configured with the lifetime that `ADR-0002` binds, and a person who
signs in.
**When** that person sends one request each day for seven days.
**Then** no request redirects them to the sign in.

## `EXTID-SC-022`

**Verifies:** `EXTID-FR-008`, `EXTID-INV-002`
**Layer:** integration

**Given** U holds exactly two identities.
**When** two transactions detach the two providers at the same time.
**Then** one detach succeeds, the other reports the last identity, and U holds one
identity.

## `EXTID-SC-023`

**Verifies:** `EXTID-INV-003`
**Layer:** unit

**Given** a resolver with the identity of U at `telegram`.
**When** the HTTP middleware resolves a token for that account, and the Telegram
middleware resolves an update from the same sender.
**Then** both put the identifier of U into the context.

## `EXTID-SC-024`

**Verifies:** `EXTID-FR-017`
**Layer:** integration

**Given** two records in `public.users` at the migration before this feature, with
the Telegram identifiers 111 and 222, and the display names `Temuri Takalandze` and
`Nino`.
**When** every migration of this feature runs.
**Then** `public.identities` holds two rows with the provider `telegram` and the
identifiers `111` and `222`, each one naming the record that it came from.
**And** the first record holds `Temuri` and `Takalandze`, and the second holds `Nino`
with a null last name.

## Retired identifiers

This file has no retired identifier.
