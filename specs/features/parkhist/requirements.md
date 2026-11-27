---
id: PARKHIST
title: The parking history of a user
type: requirements
status: approved
created: 2026-11-27
updated: 2026-11-27
approved_by: Temuri
approved_on: 2026-11-27
constrained_by: [PLG, ARC, OWN, DAT, EXT, LOG, SEC, JOB, CLI]
---

# Requirements: The parking history of a user

## 1. Problem

A member of Maroid starts and stops a session from the bot, and reads the balance on
the home page. Maroid shows no past session and no past top up. To learn where they
parked, or how much they paid last month, the member scrolls the mobile application of the
parking service, 20 rows at a time.

The parking service reports the past sessions and the transactions of the parking
account, newest first, 20 rows on each page. An account of three years holds about 500 parking
sessions and 400 transactions. The service reports no sum, no filter, and no total of a month.

A connection lives for about 30 days. A new one needs a code by SMS, so a collection cannot
renew it with no person present (`PARKACC-FR-014`).

## 2. Users

- A member who parks: reads their sessions, their transactions, and the charges of each month.
- A member of a shared workspace whose partner also parks: each reads their own history. Neither
  sees the history of the other.
- An administrator: collects the full history of one user one time, after the connection.

## 3. Out of scope

- The pages of this feature. They are in `requirements-pages.md`.
- A collection when a member starts a connection.
- A call to the parking service from the sessions page or the transactions page.
- The active session on a page of this feature.
- The fines of a vehicle, their count, and a notification of a new fine.
- The permits of a vehicle, and the catalogue of the permits for sale.
- The map of the parking lots.
- A purchase of a permit, a top up, or any other write to the parking service.
- The saved bank card of the parking account.
- A notification of a collection.

## 4. Definitions

These terms hold in the parking plugin, and override the glossary there. They stay in this
document, and do not move to `specs/guidelines/glossary.md`. `GLO-parking-service`,
`GLO-parking-account`, `GLO-connection`, `GLO-vehicle`, and `GLO-time-zone` apply as they stand.
"Session token" and the plugin secret `session` keep the meaning of `PARKACC`.

| Term         | Meaning                                                                                                                                                           |
| ------------ | ----------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| session      | Overrides `GLO-session`. One stay of a vehicle on one parking lot, from its start to its end, as the parking service reports it.                                  |
| transaction  | Overrides `GLO-transaction`. One top up or one charge on the balance of a parking account, with its moment, its kind, and its amount.                             |
| parking lot  | A place for parking that the parking service names with a number and an address.                                                                                  |
| free parking | A session of at most 15 minutes that the parking service does not charge. A parking account gets a count of them each day.                                        |
| permit       | A pass that the parking service sells for one vehicle for a period.                                                                                               |
| balance      | Extends `GLO-balance`: also the amount of a parking account at the parking service, in lari.                                                                      |
| charge       | A transaction that takes an amount from the balance: a parking payment or a permit purchase.                                                                      |
| collection   | One run that reads the sessions and the transactions of one user from the parking service, and stores the new ones. A run of the schedule or of the command line. |

## 5. Functional requirements

### The scheduled collection

### `PARKHIST-FR-001`

The plugin must collect the sessions and the transactions of each connected user on the schedule of the collection.

**Why:** The owner decides who uses the plugin through the enablement. A user who never connected holds no connection.

### `PARKHIST-FR-002`

The schedule of the collection must default to once each hour.

**Why:** A member who parked at noon finds the session on the page within the hour.

### `PARKHIST-FR-003`

The collection must skip a user who holds no connection.

**Examples:**

- Normal case: the connection of a user ended 2 days ago. The collection makes no call to the parking service for that user.

### `PARKHIST-FR-004`

A connection must start no collection.

**Why:** The owner decided that a collection of the full history runs only from the command line.

### `PARKHIST-FR-005`

A scheduled collection must stop at the first page that holds no session that Maroid does not hold. The same rule applies to the transactions.

**Why:** A user parks a few times a day. One page of 20 rows holds every new row of one hour.

**Examples:**

- Normal case: the user parked once since the last collection. The collection reads two pages of sessions: the first holds one new row, the second holds none.
- Limit case: the user did not park. The collection reads one page of sessions.

### `PARKHIST-FR-006`

A scheduled collection must read at most 5 pages of sessions and at most 5 pages of transactions for one user.

**Why:** A user who never ran the command line collection holds no row. With no limit, the first scheduled collection reads the full history.

**Examples:**

- Normal case: the user holds no row, and the parking service reports 24 pages. The collection reads 5 pages, and stores 100 sessions.
- Unwanted case: the connection was absent for two months. The collection stores the newest 100 rows. The older rows of the gap stay absent until the command line collection runs.

### `PARKHIST-FR-007`

A scheduled collection must write a warning to the log when it stops at the limit of `PARKHIST-FR-006`, and the last page it read holds a new row.

**Why:** Only the command line collection fills the gap, and the administrator must learn of it.

### `PARKHIST-FR-008`

The collection must store a session only after the parking service reports its end.

**Why:** The amount and the end of an active session change until the vehicle leaves.

**Examples:**

- Normal case: the newest session is active. The collection stores none of it. The next collection after its end stores it.

### `PARKHIST-FR-009`

A collection must fail with an error, and change no stored data, when the parking service fails during the collection of one user.

**Why:** A stored first page hides a missing third page from the next collection, which stops at the first page (`PARKHIST-FR-005`).

**Examples:**

- Normal case: the third page of transactions answers with status 500. The collection of that user fails. The rows of that user stay as they were.
- Limit case: the collection of one user fails. The collection of the next user runs.

### `PARKHIST-FR-010`

A collection must end the connection when the parking service rejects it, as `PARKACC-FR-019` gives.

**Examples:**

- Normal case: the connection ended at the parking service after 30 days. The collection stores nothing for that user, and removes the connection. The next collection skips the user (`PARKHIST-FR-003`).

### The stored data

### `PARKHIST-FR-011`

Maroid must keep one session for one identifier that the parking service gives it, and one transaction for one identifier, whatever the count of the collections that read it.

**Examples:**

- Normal case: two collections read the same session. Maroid holds one.
- Limit case: the parking service corrects the amount of a session. Maroid holds one, with the corrected amount.

### `PARKHIST-FR-012`

Maroid must store, for each session: the number and the address of the parking lot, the plate of the vehicle, the start, the end, the amount, and whether it was a free parking.

### `PARKHIST-FR-013`

Maroid must read the start and the end of a session as times of the Asia/Tbilisi time zone.

**Why:** The parking service gives the local time of Tbilisi with no zone.

**Examples:**

- Normal case: the parking service gives the start `2026-11-21T13:26:08`. Maroid stores 09:26:08 UTC.

### `PARKHIST-FR-014`

Maroid must store, for each transaction: the moment, the kind, the amount, the balance after the transaction, and the English description that the parking service gives.

### `PARKHIST-FR-015`

Maroid must store the kind of a transaction as a top up, a parking payment, a permit purchase, or another kind.

**Examples:**

- Normal case: the parking service names the kind `FILL_BALANCE`. Maroid stores a top up.
- Unwanted case: the parking service names a kind that Maroid does not know. Maroid stores another kind, with the description of the parking service.

### `PARKHIST-FR-016`

Maroid must link a parking payment to the session that it pays, when the parking service names that session.

**Examples:**

- Normal case: a payment of 0.55 GEL names the session 45304769. The transactions page names its parking lot.
- Limit case: Maroid does not hold the named session yet. The transaction stays, with no parking lot.

### `PARKHIST-FR-017`

Maroid must keep the collected data of a user after the end of their connection.

**Examples:**

- Normal case: the connection ended. The pages show every stored row. The next connection adds the new rows to them.

### The command line collection

### `PARKHIST-FR-018`

An administrator must collect every session and every transaction of one user from the command line.

**Why:** The scheduled collection reads at most 5 pages (`PARKHIST-FR-006`), and a connection reads none.

**Examples:**

- Normal case: the parking service reports 24 pages of sessions and 20 pages of transactions. Maroid stores every row of all 44 pages.
- Limit case: Maroid holds part of the rows. Maroid holds one of each (`PARKHIST-FR-011`).

### `PARKHIST-FR-019`

The command line collection must report the count of the new sessions and the count of the new transactions that it stored.

### `PARKHIST-FR-020`

The command line collection must fail with an error when the user holds no connection.

## 6. Invariants

### `PARKHIST-INV-001`

A session and a transaction of one user serve no other user, in any workspace.

### `PARKHIST-INV-002`

No answer of the plugin to a client holds the wallet number or the personal identifier of a parking account.

**Why:** The parking service sends both with each row. A member reads neither.

## 7. Constraints from the guidelines

| Rule      | Guideline         | Effect on this feature                                                     |
| --------- | ----------------- | -------------------------------------------------------------------------- |
| `PLG-007` | Plugin model      | The plugin reads the connection and the data through the host.             |
| `ARC-008` | Architecture      | The hub names no parking rule. The collection runs as a job for each user. |
| `OWN-004` | Record ownership  | The sessions and the transactions belong to one user.                      |
| `DAT-001` | Data              | The plugin keeps its data in one schema of its own.                        |
| `DAT-010` | Data              | A start, an end, and a moment are points in time, not local times.         |
| `EXT-005` | External services | A failure of the parking service is an error, never an empty result.       |
| `LOG-008` | Logging           | No log holds a session token.                                              |
| `SEC-013` | Security          | Each route of the plugin names a permission.                               |
| `JOB-005` | Background jobs   | The collection is a cron job of the plugin, with a schedule in the config. |
| `CLI-005` | Command line      | The plugin adds the command line collection through the command registry.  |

## 8. Open questions

| #   | Question                                              | Owner  | Answer                                                                            |
| --- | ----------------------------------------------------- | ------ | --------------------------------------------------------------------------------- |
| 1   | Does a connection collect at once?                    | Temuri | No. `PARKHIST-FR-004`.                                                            |
| 2   | Which data does Maroid store?                         | Temuri | The sessions and the transactions alone.                                          |
| 3   | How often does the collection run?                    | Temuri | Each hour. `PARKHIST-FR-002`.                                                     |
| 4   | How far does one scheduled collection read?           | Temuri | At most 5 pages of each. `PARKHIST-FR-006`.                                       |
| 5   | Who does the command line collection read?            | Temuri | One user, named by a flag. `PARKHIST-FR-018`.                                     |
| 6   | Are the fines in scope?                               | Temuri | No. Out of scope, with their count and their notification.                        |
| 7   | Does a page of this feature call the parking service? | Temuri | Only the overview, for the balance and the free parkings left. `PARKHIST-FR-044`. |

## Retired identifiers

This file has no retired identifier.
