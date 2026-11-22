---
id: PROFILE
title: The profile that a person changes on their own record
type: requirements
status: approved
created: 2026-10-09
updated: 2026-11-22
approved_by: Temuri
approved_on: 2026-11-22
constrained_by: [SEC, API, LOG, OWN, RES, DAT]
---

# Requirements: The profile that a person changes on their own record

## 1. Problem

A person sees a misspelled name on their own record and asks an administrator to fix
it. Only an administrator changes a name, under `PLUGACC-FR-034`.

A person with a local account who wants a new password asks an administrator too.
The administrator then chooses the password and knows it, under `IDPROV-FR-019`.

Maroid counts the day of a person in UTC, and the web shell shows a moment in the
zone of the browser. A job and a Telegram update have no browser. A plugin that asks
a person a question at 09:00 cannot tell whose 09:00 it is. A person who reads the
shell on a device set to another zone sees the time of that device, not their own.

## 2. Users

- A person who signed in: corrects their own name, changes the password of their own
  local account, and sets their own time zone.
- A plugin: reads the time zone of the acting user, to compute a local hour or a
  local day.

## 3. Out of scope

- A change to any field of the record other than the first name, the last name, and
  the time zone.
- A change of the time zone through the bot, or from a shared location.
- A time zone that Maroid detects and sets with no action of the person.
- A time zone of a workspace.
- A read by a plugin of the time zone of a person other than the acting user.
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

### `PROFILE-FR-011`

Maroid must refuse a change by a user to any field of their own record other than the
first name, the last name, and the time zone.

**Why:** The administrator flag, the block, and the plugin allowlist are decisions of
an administrator, as `SEC-011` gives.

**Examples:**

- Unwanted case: a user who is no administrator asks to become one. Maroid refuses
  the change and changes no field.
- Unwanted case: Nina sends a new time zone and the administrator flag in one
  request. Maroid refuses the request and changes neither field.

### `PROFILE-FR-012`

A user must read their own time zone from the web shell.

### `PROFILE-FR-013`

A user must set their own time zone to a zone that the IANA time zone database names.

**Why:** The database names each zone with its history of offsets and of daylight
saving time. A fixed offset does not follow daylight saving time.

**Examples:**

- Normal case: Nina sets `Asia/Tbilisi`.
- Limit case: Nina sets `UTC`. The record holds `UTC`, which differs from no zone in
  the web shell.

### `PROFILE-FR-014`

Maroid must refuse a time zone that the IANA time zone database does not name.

**Examples:**

- Unwanted case: Nina sends `+04:00`. Maroid refuses it and keeps the zone.
- Unwanted case: Nina sends `Asia/Tbilsi`. Maroid refuses it and keeps the zone.

### `PROFILE-FR-015`

A user must clear their own time zone.

**Examples:**

- Normal case: Nina clears `Asia/Tbilisi`. The record holds no time zone.
- Limit case: Beka holds no time zone and clears it. Nothing changes.

### `PROFILE-FR-016`

When the record of the acting user holds a time zone, the web shell must show each
moment in that zone.

**Why:** The person sees their own time on any device.

**Examples:**

- Normal case: Nina holds `Asia/Tbilisi` and opens the shell in Berlin in winter.
  A moment at 18:30 UTC shows 22:30.
- Limit case: Nina holds `UTC`. The same moment shows 18:30 in a browser at UTC+4.

### `PROFILE-FR-017`

When the record of the acting user holds no time zone, the web shell must show each
moment in the zone of the browser.

**Why:** A person who sets nothing sees the shell as before this feature.

**Examples:**

- Normal case: Beka holds no zone, and the browser runs at UTC+1. A moment at
  18:30 UTC shows 19:30.

### `PROFILE-FR-018`

The web shell must read a moment that a person types in the zone that it shows.

**Why:** A person who types 22:00 means the 22:00 that the shell shows them.

**Examples:**

- Normal case: Nina holds `Asia/Tbilisi` and types 22:00 in a browser at UTC+1.
  Maroid stores 18:00 UTC.
- Limit case: Nina holds `Europe/Berlin` and types 02:30 on the night that the
  clocks jump from 02:00 to 03:00. Maroid stores the moment one hour later, 03:30
  local time.

### `PROFILE-FR-019`

A plugin must read the time zone of the acting user of its unit of work.

**Why:** A job and a Telegram update have no browser. The record is the one source
that they share.

**Examples:**

- Normal case: a job runs for Nina, who holds `Asia/Tbilisi`. The plugin reads
  `Asia/Tbilisi`.
- Normal case: Nina sends a command to the bot. The plugin reads `Asia/Tbilisi`.

### `PROFILE-FR-020`

A plugin must read UTC when the record of the acting user holds no time zone, or
when the unit of work has no acting user.

**Why:** A computation that depends on no browser stays the same on every run.

**Examples:**

- Normal case: a job runs for Beka, who holds no zone. The plugin reads UTC.
- Limit case: a job runs for a workspace, with no acting user. The plugin reads UTC.

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

### `PROFILE-INV-002`

Every moment that Maroid stores or sends is in UTC. The time zone of a person changes
how a moment shows and which hour a plugin computes, never the moment itself.

**Why:** One instant has one value, whoever reads it.

### `PROFILE-INV-003`

A change of the time zone of a person changes no moment that Maroid holds.

**Examples:**

- Normal case: Nina logs a watering at 18:00 UTC with `Asia/Tbilisi`, then moves to
  `Europe/Berlin`. The entry still holds 18:00 UTC, and the shell shows 19:00.

### `PROFILE-INV-004`

A plugin reads the time zone of the acting user alone.

**Why:** The zone tells where a person lives or travels. A plugin needs no zone of
another person.

## 7. Constraints from the guidelines

| Rule      | Guideline | Effect on this feature                                                                 |
| --------- | --------- | -------------------------------------------------------------------------------------- |
| `SEC-001` | Security  | Maroid stores no password and no hash. A password passes through in memory.            |
| `SEC-004` | Security  | A blocked record reaches nothing, this feature included.                               |
| `SEC-011` | Security  | `PROFILE-FR-011`. Only an administrator changes the other fields.                      |
| `API-003` | HTTP API  | `/users*` is for an administrator. The route of the acting user is an exception there. |
| `LOG-008` | Logging   | No log line carries the current password or the new one.                               |
| `OWN-002` | Ownership | No statement here creates a user record.                                               |
| `OWN-003` | Ownership | `PROFILE-FR-019`. Each unit of work has at most one acting user, whose zone it reads.  |
| `RES-001` | REST      | `PROFILE-INV-002`. A moment travels as UTC. `ADR-0024` lets a client show it locally.  |
| `DAT-010` | Data      | `PROFILE-INV-002`. A column stores UTC. `ADR-0024` adds the column of a time zone.     |

## 8. Open questions

| #   | Question                                                       | Owner  | Answer                                                        |
| --- | -------------------------------------------------------------- | ------ | ------------------------------------------------------------- |
| 1   | Does a person change their name without an administrator?      | Temuri | Yes. The first name and the last name only.                   |
| 2   | Who sees the password section?                                 | Temuri | A holder of a local account, while the local provider exists. |
| 3   | Does a password change need the current password?              | Temuri | Yes.                                                          |
| 4   | Does a password change end the other sessions?                 | Temuri | No.                                                           |
| 5   | Does a person change the email address of their local account? | Temuri | No. It shows read only.                                       |
| 6   | What does a record hold before the person sets a zone?         | Temuri | No zone. A computation uses UTC, and the shell the browser.   |
| 7   | Where does a person change the time zone?                      | Temuri | The profile page of the web shell alone.                      |
| 8   | Which zone shows a moment in the web shell?                    | Temuri | The zone of the record, else the zone of the browser.         |
| 9   | Whose time zone does a plugin read?                            | Temuri | The acting user's alone.                                      |

## Retired identifiers

| ID               | Retired    | Reason                                                          |
| ---------------- | ---------- | --------------------------------------------------------------- |
| `PROFILE-FR-003` | 2026-11-22 | A user changes the time zone too. `PROFILE-FR-011` replaces it. |
