---
id: EXTID
title: External identities and the delegated sign in
type: requirements
status: approved
created: 2026-09-15
updated: 2026-09-15
approved_by: Temuri
approved_on: 2026-09-15
constrained_by: [OWN, SEC, API, TG, CLI, DAT]
---

# Requirements: External identities and the delegated sign in

## 1. Problem

Maroid ties a person to one Telegram account. The user record holds the Telegram
identifier, the web login reads it, and the bot resolves an update by it. A person who holds a second account cannot sign in with it. A second account needs a
second user record, and the rows of one person then split across two owners.

The authentication itself lives inside the hub. The hub runs the login flow, mints
its own token, and holds the signing key. A second provider therefore costs a second
flow inside the hub, and the hub keeps a signing key that nothing else needs.

## 2. Users

- The owner: gives a person access before that person ever opened Maroid.
- A household member: signs in with whichever account they hold, and reaches the same
  data every time.
- A person who lost an account: signs in with a second one and loses nothing.

## 3. Out of scope

- The page that shows the attached accounts. This feature serves the data behind it.
- Renewing a session without a sign in.
- An agent, a machine token, and the scopes that limit one.
- A capability that a plugin asks for, and the choice of the provider that carries a notification.
- Merging two user records into one.
- Registration that the owner did not start.
- Copying the name of a user from a provider.
- Deleting a user record.

## 4. Functional requirements

### `EXTID-FR-001`

Maroid must accept a sign in only when the external account holds an identity. That
identity must name an active user record.

**Why:** The identity provider federates a provider that accepts every account of the
public. The user record is the allowlist. See `SEC-004`.

**Examples:**

- Normal case: a person whose Telegram account holds an identity signs in and reaches their data.
- Limit case: the identity names a blocked record. The sign in fails.
- Unwanted case: a Google account that no identity names signs in and reaches the shell.

### `EXTID-FR-002`

Maroid must tell a person whose external account holds no identity that they hold no
access. That answer differs from the answer to a failed sign in.

**Why:** The person then asks the owner for an invitation, instead of trying again.

### `EXTID-FR-003`

Maroid must create no user record and no identity from a sign in. See `OWN-002`.

### `EXTID-FR-004`

A signed in user must attach a further external account to their own record.

**Examples:**

- Normal case: a person signed in through Telegram attaches their Google account, then signs in with either one.
- Limit case: the account already holds an identity that names the same record. Maroid reports success and changes nothing.

### `EXTID-FR-005`

Maroid must refuse an attachment when the external account already holds an identity
that names another user record. Both records stay unchanged.

**Why:** One external account reaching two records lets one person read the data of
another.

### `EXTID-FR-006`

Maroid must attach the external account to the record of the person who started the
attachment. The request that finishes it names no record.

**Why:** A value that the browser carries is a value that the person can rewrite. A
rewritten value attaches an account to a record that the person does not own.

### `EXTID-FR-007`

A signed in user must detach an external account from their own record.

### `EXTID-FR-008`

Maroid must refuse to detach the last external account of a user record.

**Why:** Nobody reaches a record that holds no identity, and only a new invitation
restores it.

### `EXTID-FR-009`

Maroid must report each provider to a signed in user. The report states whether an
external account of that provider holds an identity for their record.

### `EXTID-FR-010`

The owner must create a user record and obtain an invitation for it in one action.

**Why:** The identifier of an external account exists only after that person signs in
one time. The owner cannot write the first identity by hand.

### `EXTID-FR-011`

The owner must obtain an invitation for a user record that already exists.

**Why:** An invitation expires, and a person who lost their only external account
needs a second one.

### `EXTID-FR-012`

Maroid must create the first identity of a user record only from a valid invitation
for that record.

### `EXTID-FR-013`

Maroid must accept one invitation one time.

**Examples:**

- Normal case: the person opens the invitation, signs in, and Maroid attaches the account.
- Unwanted case: the person opens the same invitation again and attaches a second account.

### `EXTID-FR-014`

Maroid must refuse an invitation after its validity ends.

### `EXTID-FR-015`

Maroid must keep the name of a user unchanged when that person signs in.

**Why:** Two providers give two different names for one person. The owner chose the
one that Maroid holds.

### `EXTID-FR-016`

Maroid must hold the account handle, the name, and the picture of each external
account. The provider gives all three at each sign in.

**Why:** `EXTID-FR-009` shows a person which account is attached. The handle names it.

### `EXTID-FR-017`

A person must reach the same user record from the bot and from the web shell.

**Examples:**

- Normal case: a person attaches their Telegram account, writes to the bot, and reads the same plants in the shell.
- Unwanted case: the bot serves a person whose record holds no Telegram identity.

## 5. Non-functional requirements

### `EXTID-NFR-001`

Over 1000 consecutive requests that carry a valid session, no more than one waits
for the identity provider. Measure at the hub.

**Why:** Every request verifies a token that another service issued. A call to that
service on each request makes it a dependency of every read.

### `EXTID-NFR-002`

A person who uses Maroid every day must sign in no more often than one time in seven
days.

**Why:** This is the lifetime that the hub gives today.

## 6. Invariants

### `EXTID-INV-001`

One external account names at most one user record.

### `EXTID-INV-002`

A user record that held an identity holds at least one identity.

**Why:** A record at zero identities reaches nobody, and no operation of a user
restores it.

### `EXTID-INV-003`

One session and one bot conversation each resolve to exactly one user record.

## 7. Constraints from the guidelines

| Rule      | Guideline        | Effect on this feature                                                          |
| --------- | ---------------- | --------------------------------------------------------------------------------- |
| `OWN-002` | Record ownership | The owner creates every record. `EXTID-FR-010` is an action of the owner, not of the person who signs in. |
| `OWN-003` | Record ownership | Each entry point carries one acting user, and the identity resolves it.          |
| `OWN-004` | Record ownership | A table that the resolver reads before an acting user exists is shared, and states its reason. |
| `SEC-004` | Security         | The user record is the allowlist, and the hub reads it on each request.          |
| `API-003` | HTTP API         | Each new route takes a place in the fixed list of prefixes.                      |
| `TG-006`  | Telegram         | The conversation store keeps its key.                                            |

`ADR-0002` changes `SEC-001`, `SEC-002`, `SEC-003`, `OWN-001`, `API-003`, and
`GLO-user`. The owner accepted that decision on 2026-09-15.

`IDENT-FR-002`, `IDENT-FR-010`, and `IDENT-FR-011` stay true. This feature changes how
the acting user resolves under them. `IDENT-INV-003` retires, and `EXTID-INV-001`
succeeds it.

## 8. Open questions

| #   | Question                                                                            | Owner  | Answer                                                                                                                                                        |
| --- | ------------------------------------------------------------------------------------- | ------ | --------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1   | What happens when the token of the identity provider expires during a session?      | Temuri | the person signs in again. Maroid holds no renewal and stores no second token. The identity provider carries the lifetime that `EXTID-NFR-002` gives. |
| 2   | Which providers does the first iteration offer?                                     | Temuri | the providers that the identity provider federates today. A further one costs configuration only, because `EXTID-FR-009` reads the list. A provider that holds a password is not one of them, because no statement asks Maroid to serve a password. |
| 3   | Does an invitation name the provider that redeems it?                                | Temuri | no. The person uses whichever account they hold.                                                                                                     |
| 4   | What becomes of a user record whose invitation expired with no sign in?             | Temuri | nothing. The record holds no identity, so it reaches nobody. The owner issues a second invitation or blocks the record.                              |
| 5   | How does the invitation reach the person?                                           | Temuri | the owner sends it by hand. Maroid delivers nothing.                                                                                                 |
| 6   | Does a detached external account keep the handle and the picture that it gave?       | Temuri | no. A detach removes the identity and everything on it.                                                                                              |

## Retired identifiers

This file has no retired identifier.
