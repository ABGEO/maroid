---
id: MAGTIACC
title: The telecom account of a workspace
type: requirements
status: approved
created: 2026-11-24
updated: 2026-11-24
approved_by: Temuri
approved_on: 2026-11-24
constrained_by: [PLG, ARC, OWN, DAT, CFG, EXT, LOG, SEC, UI, JOB, CLI]
---

# Requirements: The telecom account of a workspace

## 1. Problem

The household pays the telecom service for a mobile number of each person and for the
internet and television of the house. The balance, the allowances left, and the date of the
next renewal are visible only at the site of the telecom service, one subscriber at a time.
The traffic of a day and the money that a number spends are visible there for a period that
a person types each time.

One account of the telecom service holds every subscriber of one person: the mobile number of
that person and the home line of the house. A household keeps the house in one workspace, and
the mobile number in the workspace of one person.

## 2. Users

- An editor of a workspace: connects the telecom account one time, and assigns the subscriber of the workspace.
- An administrator of the instance: collects the traffic and the money entries of an earlier period for one workspace, from the command line.
- A member of the workspace: reads the state of the subscriber on the home page and on the
  subscriber page, and reads the traffic and the money entries of a mobile subscriber.

## 3. Out of scope

- A top up, a purchase of a package, or a change of the plan through Maroid.
- A domain, or any subscriber that is not a mobile subscriber or a home line.
- The traffic or the money entries of a home line.
- A history of the state of a subscriber. Maroid reads the state, and keeps none.
- The online payments list of the telecom service.
- A sign in that needs a code from a phone.
- A notification of any kind.
- A bot command or a tool of the plugin.
- Two or more subscribers in one workspace.
- A telecom account of one user. The connection belongs to the workspace.
- A move of traffic that another system collected before.

## 4. Definitions

The glossary holds each term of this feature: `GLO-telecom-service`, `GLO-telecom-account`,
`GLO-subscriber`, `GLO-mobile-subscriber`, `GLO-home-line`, `GLO-assigned-subscriber`,
`GLO-plan`, `GLO-allowance`, `GLO-subscriber-state`, `GLO-traffic`, `GLO-money-entry`, and
`GLO-balance`. `GLO-connection`, `GLO-connection-page`, `GLO-plugin-secret`, and `GLO-widget`
apply as they stand.

## 5. Functional requirements

`requirements-pages.md` holds `MAGTIACC-FR-032` to `MAGTIACC-FR-051`: the subscriber page,
the widget, the traffic page, and the money page.

### Credentials

### `MAGTIACC-FR-001`

An editor must store the username and the password of the telecom account as settings of the workspace.

### `MAGTIACC-FR-002`

The plugin must declare the password as a secret field.

### The connection page

### `MAGTIACC-FR-003`

The connection page must show "not connected" when Maroid holds no connection for the acting workspace.

### `MAGTIACC-FR-004`

The connection page must name the settings that an editor must fill, when the settings of the workspace are absent.

### `MAGTIACC-FR-005`

The connection page must list the mobile subscribers and the home lines of the account when Maroid holds a connection.
Each subscriber shows its number, the label that the holder gave it, and its kind.

**Why:** A read of the subscribers proves that the connection still works. A new subscriber
at the telecom service appears with no further step. A domain is out of scope, so the page
lists none.

**Examples:**

- Normal case: the account holds a mobile subscriber, a home line, and a domain. The page lists the mobile subscriber and the home line.
- Limit case: the account holds a domain alone. The page reports that the account holds no subscriber.

### `MAGTIACC-FR-006`

The connection page must mark the assigned subscriber.

### `MAGTIACC-FR-007`

The connection page must report the telecom service as unavailable when the telecom service fails with no rejection of the credentials.

### Connect and disconnect

### `MAGTIACC-FR-008`

An editor must start a connection from the connection page.

### `MAGTIACC-FR-009`

The plugin must sign in to the telecom service with the stored credentials, and store the connection, when an editor starts a connection.

**Examples:**

- Normal case: the credentials are correct. The page lists two subscribers: a mobile subscriber and a home line.
- Limit case: the workspace was connected before. The new connection replaces the old one. The assigned subscriber stays.
- Unwanted case: the password is wrong. The page reports the credentials rejected. Maroid holds no connection.

### `MAGTIACC-FR-010`

The plugin must refuse a connection, and report that the account asks for a code, when the telecom service asks for a code at the sign in.

**Why:** A collection runs with no person present, so no person can type the code.

**Examples:**

- Normal case: the holder turns on the sign in with a code at the telecom service. The editor starts a connection. The page reports the code, and Maroid holds no connection.

### `MAGTIACC-FR-011`

The plugin must refuse a connection, and report the account locked, when the telecom service reports the account locked.

### `MAGTIACC-FR-012`

An editor must end the connection from the connection page.

### `MAGTIACC-FR-013`

The end of a connection must remove the assigned subscriber.

**Why:** A new connection can name a different account, which does not hold that subscriber.

### `MAGTIACC-FR-014`

The plugin must sign in again with the stored credentials, once, and replace the connection, when the telecom service rejects the session during any action.

**Examples:**

- Normal case: the session ended overnight. A member opens the home page. The plugin signs in, and the widget shows the balance.
- Unwanted case: the new sign in fails too. The action fails with the reason of `MAGTIACC-FR-015` or `MAGTIACC-FR-007`.

### `MAGTIACC-FR-015`

The plugin must remove the connection when a sign in with the stored credentials meets a rejection of the credentials, a request for a code, or a locked account.

**Examples:**

- Normal case: the owner changed the password at the telecom service. The next read of the connection page shows "not connected". The assigned subscriber stays.

### The subscriber

### `MAGTIACC-FR-016`

An editor must assign one subscriber of the account to the workspace.

**Examples:**

- Normal case: the account holds a mobile subscriber and a home line. The editor of the house assigns the home line. The page marks it.
- Normal case: the editor of a personal workspace connects the same account and assigns the mobile subscriber.
- Limit case: the editor assigns a different subscriber. It replaces the first one.

### `MAGTIACC-FR-017`

The plugin must refuse a subscriber that the account does not hold.

### `MAGTIACC-FR-018`

The plugin must refuse a subscriber that is not a mobile subscriber or a home line.

**Examples:**

- Unwanted case: the editor assigns the domain `example.ge`. The plugin refuses it, and the assigned subscriber stays as it was.

### `MAGTIACC-FR-019`

The plugin must store the assignment of a subscriber, and call no collection, when an editor assigns it.

**Why:** An assignment answers at once. An administrator collects the history from the
command line (`MAGTIACC-FR-029`), and the schedule collects each day after that.

### The collection

### `MAGTIACC-FR-020`

The plugin must collect the traffic and the money entries of the previous day of the assigned mobile subscriber of each workspace on the schedule of the collection.

**Examples:**

- Normal case: the collection runs on 2026-11-25. It reads the traffic and the money entries of 2026-11-24.

### `MAGTIACC-FR-021`

The schedule of the collection must default to 09:00 each day.

**Why:** The telecom service closes the traffic of a day after midnight. One run each day keeps the pages one day behind at most.

### `MAGTIACC-FR-022`

The collection must skip a workspace that holds no connection, no assigned subscriber, or an assigned home line.

### `MAGTIACC-FR-023`

The collection must fail with an error that names the reason when a sign in meets a rejection of the credentials, a request for a code, or a locked account.

### `MAGTIACC-FR-024`

The collection must fail with an error, and store nothing, when the account no longer holds the assigned subscriber.

### `MAGTIACC-FR-025`

Maroid must keep one traffic row for one subscriber on one day, with the values of the latest collection that read it.

**Examples:**

- Normal case: two collections read 2026-11-24. Maroid holds one traffic row of that day.
- Limit case: the telecom service reports 812.4 MB on the first read and 815.0 MB on the second. Maroid holds 815.0 MB.

### `MAGTIACC-FR-026`

Maroid must keep one money entry for one subscriber, one moment, one kind, one direction, and one amount, whatever the count of the collections that read it.

**Examples:**

- Normal case: two collections read the top up of 17.30 lari at 2026-11-13 01:50:48. Maroid holds one money entry.
- Limit case: two charges of 2.00 lari of different kinds at the same second. Maroid holds two money entries.

### `MAGTIACC-FR-027`

Maroid must keep the traffic and the money entries after a change of the assigned subscriber or the end of the connection.

### `MAGTIACC-FR-028`

A day of the traffic and the moment of a money entry must be the day and the time in Tbilisi.

**Why:** The telecom service reports its days in the time of Tbilisi.

### The collection of an earlier period

### `MAGTIACC-FR-029`

An administrator must collect the traffic and the money entries of one workspace for a period from the command line, with the first and the last day of the period.

**Examples:**

- Normal case: the period from 2025-11-01 to 2026-10-31. Maroid stores the traffic and the money entries of the assigned mobile subscriber in that period.
- Limit case: the period holds days that Maroid holds. Maroid holds one row of each (`MAGTIACC-FR-025`, `MAGTIACC-FR-026`).
- Unwanted case: the first day is after the last day. The command fails, and calls no service.

### `MAGTIACC-FR-030`

The command line collection must fail with an error when the workspace holds no connection, no assigned subscriber, or an assigned home line.

### `MAGTIACC-FR-031`

The command line collection must accept a period of any length, and store nothing for a day that the telecom service reports no data for.

**Why:** The depth of the history at the telecom service is not known, and it can change.

**Examples:**

- Limit case: the period starts in 2020. The telecom service holds data from 2024. Maroid stores the days from 2024, and the command succeeds.

### Permissions

### `MAGTIACC-FR-052`

A member must hold a permission whose lowest role is viewer to read the connection page, the subscriber page, the widget, the traffic page, and the money page.

### `MAGTIACC-FR-053`

A member must hold a permission whose lowest role is editor to start a connection, to end it, and to assign the subscriber.

## 6. Non-functional requirements

### `MAGTIACC-NFR-001`

The plugin must answer a read of the subscriber state, with the state or an error, no later than 4 seconds after the read starts, a new sign in included.
Measure in the plugin, from the start of the read to its answer.

**Why:** The hub shows a widget as failed after 5 seconds (`WIDGET-NFR-001`). A plugin error names the cause. A time limit of the hub does not.

## 7. Invariants

### `MAGTIACC-INV-001`

One workspace holds at most one connection and at most one assigned subscriber.

### `MAGTIACC-INV-002`

The connection, the assigned subscriber, the traffic, and the money entries of one workspace serve no other workspace.

### `MAGTIACC-INV-003`

No answer of Maroid to a client holds a session or a password of a telecom account.

### `MAGTIACC-INV-004`

No answer of the plugin to a client holds the name, the personal number, or the contacts of the account holder.

## 8. Constraints from the guidelines

| Rule                 | Guideline         | Effect on this feature                                                                                 |
| -------------------- | ----------------- | ------------------------------------------------------------------------------------------------------ |
| `PLG-007`            | Plugin model      | The plugin reads the settings, the connection, and the database through the host.                      |
| `PLG-008`            | Plugin model      | The plugin imports no other plugin.                                                                    |
| `ARC-008`            | Architecture      | The hub holds no rule that names the telecom plugin.                                                   |
| `OWN-004`            | Record ownership  | The connection, the assigned subscriber, the traffic, and the money entries belong to one workspace.   |
| `DAT-001`            | Data              | The traffic and the money entries live in the schema of the plugin.                                    |
| `CFG-007`            | Configuration     | No credential of a workspace stays in the configuration file.                                          |
| `EXT-005`            | External services | A failure of the telecom service is an error, never an empty result.                                   |
| `LOG-008`            | Logging           | No log holds a password or a session.                                                                  |
| `SEC-013`            | Security          | Each route and the widget of the plugin name a permission.                                             |
| `UI-005`             | Web UI            | The plugin declares fixed navigation entries, so the traffic and the money pages stay for a home line. |
| `UI-006`             | Web UI            | The pages import no module of the deck.                                                                |
| `JOB-007`            | Jobs              | A failed collection of one workspace does not stop the collection of another.                          |
| `CLI-005`, `OWN-003` | Command line      | The plugin adds the command through the registry. It takes the workspace from a flag, and no user.     |

## 9. Open questions

| #   | Question                                                              | Owner  | Answer                                                                          |
| --- | --------------------------------------------------------------------- | ------ | ------------------------------------------------------------------------------- |
| 1   | Does the statistics page of the telecom service send a code by phone? | Temuri | No. The text of the page names a code, but the data appears with none.          |
| 2   | Which subscribers can a workspace follow?                             | Temuri | One, a mobile subscriber or a home line (`MAGTIACC-FR-016`, `MAGTIACC-FR-018`). |
| 3   | Does a home line hide the traffic and the money pages?                | Temuri | No. The navigation is fixed (`UI-005`). The pages say so (`MAGTIACC-FR-049`).   |
| 4   | How far back does the command line collection read?                   | Temuri | With no limit. The telecom service decides (`MAGTIACC-FR-031`).                 |

## Retired identifiers

This file has no retired identifier.
