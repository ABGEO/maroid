---
id: IDPROV
title: The providers of the instance and the local account
type: requirements
status: approved
created: 2026-10-08
updated: 2026-10-08
approved_by: Temuri
approved_on: 2026-10-08
constrained_by: [SEC, OWN, API, CLI, CFG, LOG]
---

# Requirements: The providers of the instance and the local account

## 1. Problem

The owner adds a provider in two places. The configuration of the IdP holds the
connector, and the configuration of the hub repeats its identifier and its name so
that `EXTID-FR-009` reports it. Each change edits two files and restarts two
processes.

A new instance holds no provider until the owner writes one. Nobody signs in, so
nobody redeems the invitation of the first administrator.

## 2. Users

- An administrator: adds, changes, and removes the providers of the instance, and
  gives a person a local account.
- The person who installs the instance: signs in for the first time with nothing but
  access to the host.
- A household member: signs in with the account that an administrator added.

## 3. Out of scope

- A provider kind other than the three presets. Generic OIDC covers any OIDC service.
- A page where a person sets or changes their own local password.
- A lockout after failed local sign ins.
- A second factor for a local account.
- Ending the live sessions of a removed provider.
- Changing the email address of a local account. An administrator removes the
  account and gives a new one.
- The order of the providers on the sign in page of the IdP.
- Changing a static provider from Maroid.

## 4. Functional requirements

### `IDPROV-FR-001`

An administrator must list every provider of the instance, with its name, its preset,
and whether it is a static provider.

**Examples:**

- Normal case: the list holds `local`, `telegram`, and `abgeo-cloud`.
- Limit case: the file of the IdP holds `mock`. The list marks it static.

### `IDPROV-FR-002`

An administrator must add a provider of the generic OIDC preset, with an identifier,
a name, an issuer, a client identifier, a client secret, the scopes, and a user
identifier claim.

**Examples:**

- Normal case: an administrator adds `abgeo-cloud` with its issuer. A person signs in
  with it at the next attempt.
- Unwanted case: the identifier is `telegram` or `local`. Maroid refuses it.
- Unwanted case: the identifier names a static provider. Maroid refuses it.

### `IDPROV-FR-003`

Maroid must give a generic OIDC provider the user identifier claim `sub`, and the
scopes `openid`, `profile`, and `email`, when the administrator names none.

### `IDPROV-FR-004`

An administrator must add the Telegram provider with a client identifier and a client
secret. Maroid names it `Telegram`.

**Why:** The bot resolves an update by the Telegram user identifier. `ADR-0002` fixes
the identifier of the provider, the issuer, and the claim that carries that number.

**Examples:**

- Normal case: the administrator enters the secret from BotFather. A person signs in
  with Telegram, and the bot resolves their updates to the same record.
- Unwanted case: a Telegram provider exists. Maroid refuses a second one.

### `IDPROV-FR-005`

Maroid must offer the numeric prefix of the bot token as the client identifier of a
new Telegram provider, when the hub holds a bot token.

**Why:** The login bot of Telegram is most often the bot of the hub. The administrator
then types one value less. The value stays editable, because a Telegram user
identifier is the same at every bot.

### `IDPROV-FR-006`

An administrator must add the local provider with no input. Maroid names it `Maroid`.

**Why:** The local provider holds no setting. The sign in page of the IdP reads
"Log in with Maroid", which sets an account of the instance apart from an external one.

**Examples:**

- Unwanted case: a local provider exists. Maroid refuses a second one.

### `IDPROV-FR-007`

Maroid must show the redirect address that the upstream provider needs, for a
provider of the generic OIDC preset or the Telegram preset.

**Why:** The administrator registers that address at the upstream provider before
the first sign in.

### `IDPROV-FR-008`

An administrator must give further options of the IdP to a generic OIDC provider or
the Telegram provider, as one free text block in a structured format.

**Why:** The IdP offers options that no preset field covers.

**Examples:**

- Normal case: the block turns on the read of the user information endpoint. The next
  sign in reads it.
- Unwanted case: the block sets a key that the preset fixes. Maroid refuses the
  change, names the key, and changes nothing.
- Unwanted case: the block sets a key that the IdP does not know. Maroid refuses it
  and names the key.

### `IDPROV-FR-009`

An administrator must rename a provider. An administrator must change the client
identifier and the further options of a generic OIDC provider or the Telegram
provider, and the scopes of a generic OIDC provider.

**Why:** The name is the label of the button on the sign in page of the IdP.

### `IDPROV-FR-010`

An administrator must replace the client secret of a provider. A change that names no
secret keeps the secret that the provider holds.

### `IDPROV-FR-011`

Maroid must refuse a provider whose issuer publishes no OIDC discovery document.

**Why:** The IdP accepts such a provider, and it then fails at each sign in.

**Examples:**

- Unwanted case: the issuer does not resolve. Maroid refuses it, names the issuer, and
  creates nothing.

### `IDPROV-FR-012`

Before an administrator removes a provider, Maroid must report the count of identities
that the removal deletes, and each administrator who then holds no identity at any
provider that remains.

**Why:** A removal can lock an administrator out. `IDPROV-FR-025` is the way back in.

**Examples:**

- Normal case: removing `abgeo-cloud` deletes four identities. Every administrator
  keeps a Telegram identity, so the report names nobody.
- Limit case: the only administrator holds a local identity only. Removing `local`
  names that administrator. The administrator confirms, and the removal proceeds.

### `IDPROV-FR-013`

An administrator must remove a provider after the report of `IDPROV-FR-012`.

### `IDPROV-FR-014`

Maroid must delete every identity of a provider when it removes the provider.

**Why:** An identity of a removed provider names an account that can never sign in.
The removal can leave a record with no identity, which `EXTID-FR-008` refuses for a
detach. An administrator then issues a new invitation.

### `IDPROV-FR-015`

Maroid must delete every local account when it removes the local provider.

**Why:** A password left behind signs in again the moment a local provider returns.

### `IDPROV-FR-016`

Maroid must refuse a change and a removal of a static provider.

### `IDPROV-FR-017`

Maroid must report under `EXTID-FR-009` each provider that the IdP holds, static ones
included.

**Why:** The configuration of the hub stops listing the providers.

### `IDPROV-FR-018`

An administrator must give a user record a local account, with an email address and a
password.

**Examples:**

- Normal case: the administrator gives Nina `nina@home.example`. Nina signs in with it.
- Unwanted case: another local account holds the address. Maroid refuses it.
- Unwanted case: no local provider exists. Maroid refuses it, and says to add the
  local provider first.

### `IDPROV-FR-019`

An administrator must set a new password for a local account.

**Examples:**

- Normal case: Nina forgot her password. The administrator sets one. Her old one
  stops working at the next sign in.

### `IDPROV-FR-020`

An administrator must remove the local account of a user record.

**Examples:**

- Unwanted case: the local account is the last identity of the record. Maroid refuses
  it, as `EXTID-FR-008` gives.

### `IDPROV-FR-021`

Maroid must remove the password of a local account when a person detaches it under
`EXTID-FR-007`.

### `IDPROV-FR-022`

Maroid must refuse an attach through the local provider.

**Why:** Only an administrator gives a local account. A person who signs in with one
already holds the identity.

### `IDPROV-FR-023`

Maroid must refuse a password shorter than 12 characters or longer than 72 bytes.

**Why:** A local account carries no lockout. The IdP reads the first 72 bytes only, so
a longer password differs from what the person typed.

### `IDPROV-FR-024`

Maroid must return no password, no password hash, and no client secret in any answer.

### `IDPROV-FR-025`

The command that sets a local account must give an existing user record a local
account, or set a new password on the one it holds.

**Why:** The first administrator signs in with no external provider. The same command
restores access after a lockout.

**Examples:**

- Normal case: on a new instance, the installer creates an administrator record, then
  gives it a local account. The installer signs in at the web shell.
- Limit case: the administrator removed every provider. The command restores the local
  provider and a password.

### `IDPROV-FR-026`

The command that sets a local account must add the local provider when none exists.

### `IDPROV-FR-027`

The command that sets a local account must read the password from an interactive
prompt, twice, or from the standard input. It must refuse a password on the command
line.

**Why:** A password on the command line reaches the shell history and the process list.

### `IDPROV-FR-028`

The command that creates a user record must print the identifier of the record.

**Why:** `IDPROV-FR-025` names the record by that identifier.

## 5. Non-functional requirements

### `IDPROV-NFR-001`

An added, changed, or removed provider takes effect at the next sign in that starts
after the administrator receives the answer. Neither the hub nor the IdP restarts.
Measured at the sign in page of the IdP.

### `IDPROV-NFR-002`

When the IdP does not answer within 5 seconds, a change to a provider or a local account
fails with an answer that names the IdP, and Maroid changes nothing. Measured at the
answer to the administrator.

## 6. Invariants

### `IDPROV-INV-001`

A provider keeps its identifier for its life. A provider of the generic OIDC preset or
the Telegram preset keeps its issuer and its user identifier claim for its life.

**Why:** `SEC-003` joins an identity on the user identifier that these values decide.

### `IDPROV-INV-002`

The Telegram provider has the identifier `telegram`, the issuer of Telegram, and the
user identifier claim `id`.

### `IDPROV-INV-003`

An instance holds at most one Telegram provider and at most one local provider.

### `IDPROV-INV-004`

One email address names at most one local account.

### `IDPROV-INV-005`

Every local account belongs to exactly one user record, through one identity.

## 7. Constraints from the guidelines

| Rule      | Guideline  | Effect on this feature                                                         |
| --------- | ---------- | ------------------------------------------------------------------------------ |
| `SEC-001` | Security   | Maroid stores no password and no hash. A password passes through in memory.    |
| `SEC-003` | Security   | `IDPROV-INV-001` and `IDPROV-FR-014`.                                          |
| `SEC-004` | Security   | A blocked record keeps its local account and still reaches nothing.            |
| `SEC-011` | Security   | Only an administrator adds a provider or gives a local account.                |
| `LOG-008` | Logging    | No log line carries a password, a hash, or a client secret.                    |
| `OWN-002` | Ownership  | An administrator creates a user record. No statement here creates one.         |
| `API-003` | HTTP API   | The provider routes live under `/providers*`. The local account under `/users*`. |
| `CLI-001` | CLI        | `maroid user password` serves `IDPROV-FR-025` to `IDPROV-FR-027`.              |
| `CFG-004` | Configuration | The client certificate of the hub stays out of the repository.             |

## 8. Open questions

| #   | Question                                                         | Owner  | Answer |
| --- | ---------------------------------------------------------------- | ------ | ------ |
| 1   | What happens when the web shell sets a password and no local provider exists? | Temuri | Maroid refuses. Only the command line adds the local provider by itself. |
| 2   | What length does a local password take?                          | Temuri | 12 characters to 72 bytes.                                                |
| 3   | Does blocking a record remove its local account?                 | Temuri | No. `SEC-004` already stops the person.                                   |
| 4   | Does removing a provider end its live sessions?                  | Temuri | No, not in this feature.                                                  |
| 5   | Does a local user change their own password?                     | Temuri | No, not in this feature. An administrator sets it.                        |

## Retired identifiers

This file has no retired identifier.
