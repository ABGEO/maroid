---
id: DRIVVO
title: The car log account of a workspace
type: requirements
status: approved
created: 2026-11-25
updated: 2026-11-25
approved_by: Temuri
approved_on: 2026-11-25
constrained_by: [PLG, ARC, OWN, DAT, CFG, EXT, LOG, SEC, UI, JOB]
---

# Requirements: The car log account of a workspace

## 1. Problem

The household logs each refuel, maintenance, and cost of its cars in the mobile application
of the car log service. The figures are visible only in that application and at its web site,
one vehicle at a time. Maroid shows none of them next to the other costs of the house.

One car log account holds every vehicle of the household, the sold ones too. A vehicle holds
some hundreds of log entries over its life.

## 2. Users

- An editor of a workspace: connects the car log account one time.
- A member of the workspace: reads the figures, the log entries, and the reminders of each vehicle.

## 3. Out of scope

- A write to the car log account: a new, a changed, or a deleted log entry, reminder, or vehicle.
- A sign in with Google, or any sign in that is not an email address and a password.
- The trips, the checklists, the forms, the drivers, and the attached files of the account.
- The fuel prices of the stations near a place.
- A notification of any kind: a due reminder, or a failed collection.
- A bot command or an MCP tool of the plugin.
- A collection that a person or a new connection starts. The schedule runs every collection.
- The time of the latest collection on a page.
- A conversion of a unit or of a currency.
- The discount of a log entry. An amount is the sum of its lines (`DRIVVO-FR-037`).
- A separate consumption for each fuel of a vehicle with two or three fuels.
- An estimate of the date on which a reminder with an odometer due point comes due.
- An assignment of a vehicle to a workspace. Every vehicle of the account serves the workspace.
- A choice of vehicle that Maroid remembers for a member.
- A car log account of one user. The connection belongs to the workspace.
- A move of log entries that another system collected before.

## 4. Definitions

The glossary holds each term of this feature: `GLO-car-log-service`, `GLO-car-log-account`, `GLO-vehicle`, `GLO-principal-vehicle`, `GLO-inactive-vehicle`, `GLO-log-entry`, `GLO-refuel`, `GLO-fuel-line`, `GLO-full-tank-mark`, `GLO-free-mark`, `GLO-missed-refuel`, `GLO-maintenance`, `GLO-expense`, `GLO-income`, `GLO-odometer-entry`, `GLO-instalment`, `GLO-reference-item`, `GLO-reminder`, `GLO-due-point`, `GLO-latest-odometer`, `GLO-car-log-settings`, `GLO-car-log-copy`, `GLO-spend`, and `GLO-consumption`. `GLO-connection`, `GLO-connection-page`, `GLO-plugin-secret`, `GLO-widget`, and
`GLO-time-zone` apply as they stand.

## 5. Functional requirements

`requirements-pages.md` holds `DRIVVO-FR-027` to `DRIVVO-FR-068`: the choice of the vehicle,
the overview, the lists, the detail of a log entry, the reminders, and the widget.

### Credentials

### `DRIVVO-FR-001`

An editor must store the email address and the password of the car log account as settings of the workspace.

### `DRIVVO-FR-002`

The plugin must declare the password as a secret field.

### The connection page

### `DRIVVO-FR-003`

The connection page must show "not connected" when Maroid holds no connection for the acting workspace.

### `DRIVVO-FR-004`

The connection page must name the settings that an editor must fill, when the settings of the workspace are absent.

### `DRIVVO-FR-005`

The connection page must list the vehicles of the account, read from the car log service, when Maroid holds a connection.
Each vehicle shows its name, its make, its model, and an inactive vehicle mark.

**Why:** A read of the vehicles proves that the connection still works.

### `DRIVVO-FR-006`

The connection page must report the car log service as unavailable when the car log service fails with no rejection of the credentials.

### Connect and disconnect

### `DRIVVO-FR-007`

An editor must start a connection from the connection page.

### `DRIVVO-FR-008`

The plugin must sign in to the car log service with the stored credentials, and store the connection, when an editor starts a connection.

**Examples:**

- Normal case: the credentials are correct. The page lists two vehicles.
- Limit case: the workspace was connected before. The new connection replaces the old one.
- Unwanted case: the password is wrong. The page reports the credentials rejected. Maroid holds no connection.
- Unwanted case: the account signs in with Google and holds no password. The page reports the credentials rejected.

### `DRIVVO-FR-009`

The plugin must refuse a connection, and report that the account asks for a new password, when the car log service asks for a new password at the sign in.

**Why:** A collection runs with no person present, so no person can choose the new password.

### `DRIVVO-FR-011`

An editor must end the connection from the connection page.

### `DRIVVO-FR-012`

The plugin must sign in again with the stored credentials, once, and replace the connection, when the car log service rejects the session during any action.

**Examples:**

- Normal case: the session ended overnight. The collection signs in, and reads the account.
- Unwanted case: the new sign in fails too. The action fails with the reason of `DRIVVO-FR-013` or `DRIVVO-FR-006`.

### `DRIVVO-FR-013`

The plugin must remove the connection when a sign in with the stored credentials meets a rejection of the credentials or a request for a new password.

**Examples:**

- Normal case: the owner changed the password at the car log service. The next read of the connection page shows "not connected".

### `DRIVVO-FR-014`

Maroid must keep the car log copy when the connection ends, for any reason.

**Examples:**

- Normal case: an editor ends the connection. The pages show the same log entries as before.
- Limit case: an editor connects a different car log account. Its first collection replaces the car log copy (`DRIVVO-FR-019`).

### The collection

### `DRIVVO-FR-015`

The plugin must collect the vehicles, the log entries, the reminders, the reference items, and the car log settings of the account of each connected workspace on the schedule of the collection.

### `DRIVVO-FR-016`

The schedule of the collection must default to the start of each hour.

**Why:** A refuel in the mobile application shows in Maroid within one hour.

### `DRIVVO-FR-017`

The collection must skip a workspace that holds no connection.

### `DRIVVO-FR-018`

A collection must collect every vehicle of the account, an inactive vehicle too, with every log entry and every reminder of each vehicle.

**Examples:**

- Normal case: the account holds an active car and a sold car. The car log copy holds both, with every log entry of each.

### `DRIVVO-FR-019`

After a successful collection, the car log copy must hold each item that the account held at the read, with the values of that read, and no other item.

**Why:** The car log service names no deleted item. A read of the full account is the one way to find one.

**Examples:**

- Normal case: a member adds a refuel in the mobile application. The next collection adds it to the car log copy.
- Normal case: a member changes the amount of a maintenance from 120.00 to 150.00. The car log copy holds 150.00.
- Limit case: a member deletes a refuel in the mobile application. The next collection removes it from the car log copy.

### `DRIVVO-FR-020`

A collection that fails must leave the car log copy of the workspace as it was before the collection.

**Why:** A partial copy shows a wrong total, and no page tells the member that it is partial.

**Examples:**

- Unwanted case: the car log service fails on the read of the expenses of the second vehicle. The car log copy holds the values of the previous collection, for every vehicle.

### `DRIVVO-FR-021`

A failed collection must log an error that names the workspace and the reason.

### `DRIVVO-FR-022`

The plugin must read each day and time of a log entry from the car log service as a time in Tbilisi.

**Why:** The car log service reports its times in the time of Tbilisi, with no zone.

**Examples:**

- Normal case: the service reports a refuel at `2026-11-24 18:30:00`. Maroid holds the moment 2026-11-24 14:30 UTC.

### `DRIVVO-FR-023`

The plugin must store each fuel line of a refuel, up to three, with the start and the end charge of the battery and the minutes of the charge when the refuel holds them.

### `DRIVVO-FR-024`

The plugin must store each instalment of a maintenance, an expense, and an income.

### Permissions

### `DRIVVO-FR-025`

A member must hold a permission whose lowest role is viewer to read the connection page, every page of `requirements-pages.md`, and the widget.

### `DRIVVO-FR-026`

A member must hold a permission whose lowest role is editor to start a connection and to end it.

## 6. Non-functional requirements

### `DRIVVO-NFR-001`

A collection of one workspace whose account holds 3 vehicles and 2,000 log entries must finish in less than 60 seconds.
Measure in the plugin, from the start of the collection of the workspace to the store of the car log copy.

**Why:** The schedule runs each hour. A collection of one workspace must not delay the collections of the other workspaces.

### `DRIVVO-NFR-002`

The plugin must answer a read of any page of `requirements-pages.md`, or of the widget, in less than 1 second when the car log copy holds 2,000 log entries.
Measure in the plugin, from the receipt of the request to the answer.

**Why:** A page reads the car log copy alone (`DRIVVO-FR-033`). The hub shows a widget as failed after 5 seconds (`WIDGET-NFR-001`).

## 7. Invariants

### `DRIVVO-INV-001`

One workspace holds at most one connection.

### `DRIVVO-INV-002`

The connection and the car log copy of one workspace serve no other workspace.

### `DRIVVO-INV-003`

The car log copy of one workspace holds one item for each item of the car log service.

### `DRIVVO-INV-004`

Each log entry and each reminder of the car log copy belongs to a vehicle that the car log copy holds.

### `DRIVVO-INV-005`

No answer of Maroid to a client holds a session or a password of a car log account.

### `DRIVVO-INV-006`

No answer of the plugin to a client holds the name, the email address, the phone number, or the driver licence of the account holder.

## 8. Constraints from the guidelines

| Rule      | Guideline         | Effect on this feature                                                                 |
| --------- | ----------------- | -------------------------------------------------------------------------------------- |
| `PLG-007` | Plugin model      | The plugin reads the settings, the connection, and the database through the host.      |
| `PLG-008` | Plugin model      | The plugin imports no other plugin.                                                    |
| `ARC-008` | Architecture      | The hub holds no rule that names the car log plugin.                                   |
| `OWN-004` | Record ownership  | The connection and the car log copy belong to one workspace.                           |
| `DAT-001` | Data              | The car log copy lives in the schema of the plugin.                                    |
| `DAT-010` | Data              | A moment of a log entry is stored in UTC.                                              |
| `CFG-007` | Configuration     | No credential of a workspace stays in the configuration file.                          |
| `EXT-005` | External services | A failure of the car log service is an error, never an empty result.                   |
| `LOG-008` | Logging           | No log holds a password or a session.                                                  |
| `SEC-013` | Security          | Each route and the widget of the plugin name a permission.                             |
| `UI-005`  | Web UI            | A navigation entry names no vehicle. It opens the principal vehicle (`DRIVVO-FR-029`). |
| `UI-006`  | Web UI            | The pages import no module of the deck.                                                |
| `JOB-006` | Jobs              | A collection that runs longer than one hour skips the next run.                        |
| `JOB-007` | Jobs              | A failed collection of one workspace does not stop the collection of another.          |

## 9. Open questions

| #   | Question                                                          | Owner  | Answer                                                                                  |
| --- | ----------------------------------------------------------------- | ------ | --------------------------------------------------------------------------------------- |
| 1   | Does the plugin write to the car log account?                     | Temuri | No. Read only. A later feature adds the writes (section 3).                             |
| 2   | Which vehicles serve a workspace?                                 | Temuri | Every vehicle of the account. A member chooses one on each page (`DRIVVO-FR-027`).      |
| 3   | What happens to an item that the car log service no longer holds? | Temuri | The next collection removes it (`DRIVVO-FR-019`).                                       |
| 4   | What happens to the car log copy at the end of the connection?    | Temuri | It stays (`DRIVVO-FR-014`).                                                             |
| 5   | In which zone does the car log service report a time?             | Temuri | Tbilisi. Maroid stores UTC (`DRIVVO-FR-022`, `DAT-010`).                                |
| 6   | Where does a repeated reminder count from?                        | Temuri | From the latest log entry of its type, as the car log service counts (`DRIVVO-FR-062`). |
| 7   | Does Maroid count the discount of a log entry?                    | Temuri | No. The household enters no discount, and the web site shows none (section 3).          |

## Retired identifiers

| ID              | Retired    | Reason                                                                                            |
| --------------- | ---------- | ------------------------------------------------------------------------------------------------- |
| `DRIVVO-FR-010` | 2026-11-25 | The owner removed the collection at a connection. The schedule runs every collection (section 3). |
