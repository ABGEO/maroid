---
id: PROFILE
title: The profile that a person changes on their own record
type: requirements
status: approved
created: 2026-10-09
updated: 2026-10-09
approved_by: Temuri
approved_on: 2026-10-09
constrained_by: [SEC, API, LOG, OWN]
---

# Requirements: The profile that a person changes on their own record

## 1. Problem

A person sees a misspelled name on their own record and asks an administrator to fix
it. Only an administrator changes a name, under `PLUGACC-FR-034`.

A person with a local account who wants a new password asks an administrator too.
The administrator then chooses the password and knows it, under `IDPROV-FR-019`.

## 2. Users

- A person who signed in: corrects their own name, and changes the password of their
  own local account.

## 3. Out of scope

- A change to any field of the record other than the first name and the last name.
- A local account that a person gives themselves. `IDPROV-FR-022` still refuses it.
- A change of the email address of a local account.
- A reset of a forgotten password by the person. An administrator sets it, as
  `IDPROV-FR-019` gives.
- Ending the other sessions of a person after a password change.
- A lockout after failed attempts at the current password.

## 4. Functional requirements

### `PROFILE-FR-001`

A user must read their own first name and last name from the web shell.

### `PROFILE-FR-002`

A user must change their own first name and their own last name.

**Why:** A person knows the spelling of their own name. An administrator no longer
relays the correction.

**Examples:**

- Normal case: Nina corrects "Nnia" to "Nina". The record keeps its identities, its
  workspaces, and its allowlist.
- Limit case: Nina clears her last name. The record holds no last name.
- Unwanted case: Nina names the record of Beka in the request. Maroid changes nothing
  on the record of Beka.

### `PROFILE-FR-003`

Maroid must refuse a change by a user to any field of their own record other than the
first name and the last name.

**Why:** The administrator flag, the block, and the plugin allowlist are decisions of
an administrator, as `SEC-011` gives.

**Examples:**

- Unwanted case: a user who is no administrator asks to become one. Maroid refuses
  the change and changes no field.

### `PROFILE-FR-004`

The web shell must show the password section only when the instance holds a local
provider and the acting user holds a local account.

**Why:** A person without a local account has no password to change.

**Examples:**

- Normal case: Nina holds a local account and a Telegram identity. She signed in with
  Telegram. The section shows.
- Limit case: the administrator removed the local provider. The section is absent.
- Unwanted case: Beka holds a Telegram identity only. The section is absent.

### `PROFILE-FR-005`

The password section must show the email address of the local account of the acting
user, read only.

**Why:** The person sees which account the password belongs to.

### `PROFILE-FR-006`

A user must set a new password on their own local account.

**Examples:**

- Normal case: Nina sets a new password. Her old one stops working at the next sign
  in.
- Unwanted case: Nina holds no local account. Maroid refuses it.

### `PROFILE-FR-007`

Maroid must refuse a new password unless the request carries the current password of
the local account.

**Why:** A person who takes over an open session must not take over the local account
as well.

**Examples:**

- Unwanted case: the current password is wrong. Maroid refuses it, says that the
  current password does not match, and keeps the old password.
- Unwanted case: the request carries no current password. Maroid refuses it.

### `PROFILE-FR-008`

The web shell must ask for the new password twice, and send it only when the two
entries match.

**Why:** A typing error would lock the person out of the local account.

### `PROFILE-FR-009`

Maroid must refuse a new password that `IDPROV-FR-023` refuses.

### `PROFILE-FR-010`

Maroid must keep every session of the acting user after a password change.

**Why:** `IDPROV-FR-019` ends no session either. Ending sessions is a separate feature.

## 5. Non-functional requirements

### `PROFILE-NFR-001`

When the IdP does not answer within 5 seconds, a password change fails with an answer
that names the IdP, and Maroid changes nothing. Measured at the answer to the user.

**Why:** The same limit as `IDPROV-NFR-002`.

## 6. Invariants

### `PROFILE-INV-001`

A user changes only their own record and their own local account through this
feature. The request names no record. The session decides it.

**Why:** A request that names the record invites a request that names another one.

## 7. Constraints from the guidelines

| Rule      | Guideline | Effect on this feature                                                   |
| --------- | --------- | ------------------------------------------------------------------------ |
| `SEC-001` | Security  | Maroid stores no password and no hash. A password passes through in memory. |
| `SEC-004` | Security  | A blocked record reaches nothing, this feature included.                 |
| `SEC-011` | Security  | `PROFILE-FR-003`. Only an administrator changes the other fields.        |
| `API-003` | HTTP API  | `/users*` is for an administrator. The route of the acting user is an exception there. |
| `LOG-008` | Logging   | No log line carries the current password or the new one.                 |
| `OWN-002` | Ownership | No statement here creates a user record.                                 |

## 8. Open questions

| #   | Question                                                         | Owner  | Answer |
| --- | ---------------------------------------------------------------- | ------ | ------ |
| 1   | Does a person change their name without an administrator?        | Temuri | Yes. The first name and the last name only. |
| 2   | Who sees the password section?                                   | Temuri | A holder of a local account, while the local provider exists. |
| 3   | Does a password change need the current password?                | Temuri | Yes.   |
| 4   | Does a password change end the other sessions?                   | Temuri | No.    |
| 5   | Does a person change the email address of their local account?   | Temuri | No. It shows read only. |

## Retired identifiers

This file has no retired identifier.
