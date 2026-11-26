---
id: PENSION
title: The pension account of a user
type: requirements
status: approved
created: 2026-11-26
updated: 2026-11-27
approved_by: Temuri
approved_on: 2026-11-27
constrained_by: [PLG, ARC, OWN, DAT, CFG, EXT, LOG, SEC, JOB, CLI]
---

# Requirements: The pension account of a user

## 1. Problem

The pension plugin reads the contributions of a person once each month, and stores them.
No page shows them, so a person cannot see the result, and no workspace enables the plugin.
The plugin reads the first 10 contributions of a month only, and drops the rest with no error.
It reads no asset value and no portfolio, which the pension service shows on its own home page.

A person can find out the state of their account only at the pension service. They also
cannot tell whether Maroid still signs in with their account.

The pension service signs a person in with a username and a password, with no code by phone.
A collection can run with no person present.

## 2. Users

- A member who saves for a pension: connects their own pension account one time, then reads
  their contributions and their asset value in Maroid.
- A member of a shared workspace whose partner also saves: each connects their own account.
  Neither sees the account of the other.
- An administrator: fills the months that a scheduled collection did not read.

## 3. Out of scope

- The pages and the widget that show the collected data. They are in `requirements-pages.md`.
- A collection when a member starts a connection.
- A scheduled collection of a month before the previous calendar month.
- A notification of a collection.
- A pension account that the members of a workspace share.
- The letters, the notices, and the statements that the pension service holds.
- A change of the portfolio choice, or any other write to the pension service.
- The penalty contributions and the cash deposits that the pension service reports.

## 4. Definitions

The glossary holds each term of this feature: `GLO-pension-service`, `GLO-pension-account`, `GLO-contribution`, `GLO-source`, `GLO-basis`, `GLO-employer`, `GLO-asset-value`, `GLO-gain`, `GLO-withheld-amount`, `GLO-portfolio`, `GLO-asset-manager`, `GLO-portfolio-share`, `GLO-unit-price`, and `GLO-account-summary`. `GLO-connection`, `GLO-connection-page`, `GLO-plugin-secret`, `GLO-secret-field`, and
`GLO-time-zone` apply as they stand.

## 5. Functional requirements

### Credentials

### `PENSION-FR-001`

A user must store the username and the password of their pension account as settings of their own.

**Why:** A second member of the workspace saves with their own pension account.

### `PENSION-FR-002`

The plugin must declare the password as a secret field.

### The connection page

### `PENSION-FR-003`

The connection page must show "not connected" when Maroid holds no connection for the acting user.

### `PENSION-FR-004`

The connection page must name the settings that the acting user must fill, when the settings of the acting user are absent.

### `PENSION-FR-005`

The connection page must show the full name of the account holder and the asset value, when Maroid holds a connection.

**Why:** A read of the pension account proves that the connection still works.

**Examples:**

- Normal case: the pension service answers. The page shows "Connected as Nino Beridze" and 12,480.55 GEL.
- Unwanted case: the pension service rejects the stored credentials. `PENSION-FR-011` applies, and the page shows "not connected".

### `PENSION-FR-006`

The connection page must report the pension service as unavailable when the pension service fails with no rejection of the credentials.

**Why:** An outage at the pension service is no reason to end a working connection.

**Examples:**

- Normal case: the pension service answers with status 503. The page reports the service unavailable. The connection stays.

### Connect and disconnect

### `PENSION-FR-007`

A member must start a connection from the connection page.

### `PENSION-FR-008`

The plugin must sign in to the pension service with the stored credentials, and store the connection, when a member starts a connection.

**Examples:**

- Normal case: the credentials are correct. The page shows the connected state of `PENSION-FR-005`. The plugin collects nothing.
- Limit case: the member was connected before. The new connection replaces the old one.
- Unwanted case: the password is wrong. The page reports the credentials rejected. Maroid holds no connection.

### `PENSION-FR-009`

A member must end their connection from the connection page.

### `PENSION-FR-010`

The plugin must sign in again with the stored credentials, once, and replace the connection, when the pension service rejects the session during any action.

**Why:** No code by phone is needed, so every action recovers with no person present.

**Examples:**

- Normal case: the session ended a day ago. The collection runs. The plugin signs in, and the collection continues.
- Unwanted case: the new sign in fails too. The action fails with the reason of `PENSION-FR-011` or `PENSION-FR-006`.

### `PENSION-FR-011`

The plugin must remove the connection when the pension service rejects the stored credentials.

**Examples:**

- Normal case: the member changed the password at the pension service. The next read of the connection page shows "not connected". The collected data stays.

### `PENSION-FR-012`

Maroid must keep the collected data of a user after the end of their connection.

**Examples:**

- Normal case: the member ends the connection, and connects again a month later. Maroid holds the contributions of both periods.

### The collection

### `PENSION-FR-013`

The plugin must collect the pension account of each connected user on the schedule of the collection.

**Why:** The owner decides who uses the plugin through the enablement. A user who never opened
it in an enabled workspace stores no credentials, and holds no connection.

### `PENSION-FR-014`

The schedule of the collection must default to 10:00 on the first day of each month.

### `PENSION-FR-015`

The collection must skip a user who holds no connection.

**Examples:**

- Normal case: a user stored the credentials but never started a connection. The collection reads nothing for that user.

### `PENSION-FR-016`

The collection must read every contribution that the pension service dates in the previous calendar month of the user.

**Examples:**

- Normal case: the collection runs on 2026-11-01 for a user in Asia/Tbilisi. It reads the contributions dated from 2026-10-01 to 2026-10-31.
- Limit case: the month holds 23 contributions. Maroid holds all 23.
- Limit case: the month holds no contribution. The collection stores no contribution, and succeeds.

### `PENSION-FR-017`

The collection must read the asset value and the unit price of the pension account for each date in the previous calendar month that the pension service reports.

**Examples:**

- Normal case: the pension service reports two dates in October 2026. Maroid holds two asset values for October.
- Limit case: the pension service reports no date in the month. The collection stores no asset value, and succeeds.

### `PENSION-FR-018`

The collection must read every unit price of each portfolio of the pension account that the pension service reports, whatever its date.

**Why:** The pension service answers with the full history of a portfolio in one answer, with
no period. A portfolio holds one unit price for each working day since its start.

**Examples:**

- Normal case: the first collection reads 1964 unit prices of one portfolio. The next collection reads 1985, and Maroid holds 1985.

### `PENSION-FR-019`

The collection must replace the account summary of the user with the one that the pension service reports at the time of the collection.

**Why:** The pension service reports the current summary only. The asset values of
`PENSION-FR-017` give the history.

### `PENSION-FR-020`

The collection must fail with an error, and change no stored data, when the pension service fails during the collection of one user.

**Why:** A silent part of a month becomes a missing contribution, not a visible failure.

**Examples:**

- Normal case: the second page of contributions answers with status 500. The run of that user fails. The rows of the month stay as they were.
- Limit case: the collection of one user fails. The collection of the next user runs.

### `PENSION-FR-021`

Maroid must keep one contribution for one basis and one source, whatever the count of the collections that read it.

**Examples:**

- Normal case: one basis holds a contribution of the person, of the employer, and of the state. Maroid holds three contributions.
- Limit case: two collections read the same contribution. Maroid holds one contribution.
- Limit case: the pension service corrects the amount of a contribution. Maroid holds one contribution, with the corrected amount.

### `PENSION-FR-022`

Maroid must keep one asset value for one pension account on one day, and one unit price for one portfolio on one day.

**Examples:**

- Normal case: a later collection reads a different asset value for a day that Maroid holds. Maroid holds one asset value for that day, with the later value.

### The collection of an earlier period

### `PENSION-FR-023`

An administrator must collect the pension account of one user for a period from the command line, with the first and the last month of the period.

**Why:** The scheduled collection reads one month, and a connection reads none. An administrator
fills the history once, for each user, after the connection.

**Examples:**

- Normal case: the period from 2024-01 to 2026-10. Maroid stores the contributions, the asset values, and the unit prices of that period.
- Limit case: the period holds contributions that Maroid holds. Maroid holds one of each (`PENSION-FR-021`).
- Unwanted case: the first month is after the last month. The command fails, and calls no service.

### `PENSION-FR-024`

The command line collection must refuse a last month after the previous calendar month of the user.

**Why:** A month that has not ended holds part of its contributions.

### `PENSION-FR-025`

The command line collection must fail with an error when the user holds no connection.

### Permissions

### `PENSION-FR-026`

A member must hold a permission whose lowest role is viewer to read the connection page, to start a connection, and to end it.

**Why:** The connection belongs to the member, not to the workspace. A viewer saves for a pension too.

## 6. Invariants

### `PENSION-INV-001`

One user holds at most one connection.

### `PENSION-INV-002`

A contribution, an asset value, an account summary, and a connection of one user serve no other user, in any workspace.

### `PENSION-INV-003`

No answer of Maroid to a client holds a session or a password of a pension account.

### `PENSION-INV-004`

No answer of the plugin to a client holds the personal number of the account holder.

## 7. Constraints from the guidelines

| Rule      | Guideline         | Effect on this feature                                                        |
| --------- | ----------------- | ----------------------------------------------------------------------------- |
| `PLG-007` | Plugin model      | The plugin reads the settings, the connection, and the data through the host. |
| `ARC-008` | Architecture      | The hub names no pension rule. The collection runs as a job for each user.    |
| `OWN-004` | Record ownership  | The credentials, the connection, and the collected data belong to one user.   |
| `DAT-001` | Data              | The plugin keeps its data in one schema of its own.                           |
| `CFG-007` | Configuration     | No credential of a person stays in the configuration file.                    |
| `EXT-005` | External services | A failure of the pension service is an error, never an empty result.          |
| `LOG-008` | Logging           | No log holds a password or a session token.                                   |
| `SEC-013` | Security          | Each route of the plugin names a permission.                                  |
| `JOB-005` | Background jobs   | The collection is a cron job of the plugin, with a schedule in the config.    |
| `CLI-005` | Command line      | The plugin adds the command line collection through the command registry.     |

## 8. Open questions

| #   | Question                                                         | Owner  | Answer                                                     |
| --- | ---------------------------------------------------------------- | ------ | ---------------------------------------------------------- |
| 1   | Does sign in need a code by phone?                               | Temuri | No. A password only.                                       |
| 2   | Does a connection collect at once?                               | Temuri | No. `PENSION-FR-008`.                                      |
| 3   | How far back does a scheduled collection read?                   | Temuri | The previous calendar month. `PENSION-FR-016`.             |
| 4   | Does the collection skip a user with no enabled workspace?       | Temuri | No. It runs for each connected user. `PENSION-FR-013`.     |
| 5   | How does Maroid fill a missed month?                             | Temuri | An administrator, from the command line. `PENSION-FR-023`. |
| 6   | Does a collection send a notification?                           | Temuri | No. Out of scope.                                          |
| 7   | Does the pension service give one identifier for a contribution? | Temuri | No. One basis and one source give one. `PENSION-FR-021`.   |

## Retired identifiers

This file has no retired identifier.
