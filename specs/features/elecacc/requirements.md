---
id: ELECACC
title: The electricity account of a workspace
type: requirements
status: approved
created: 2026-10-14
updated: 2026-10-14
approved_by: Temuri
approved_on: 2026-10-14
constrained_by: [PLG, ARC, OWN, DAT, CFG, EXT, LOG, SEC, UI, JOB, NTF, CLI]
---

# Requirements: The electricity account of a workspace

## 1. Problem

The electricity plugin of today signs in to the distributor of the city. The distributor now
asks for a code by phone at each sign in, and its session ends within hours, so a monthly
collection cannot run with no person present.

The electricity service is the supplier that now bills the household. It signs a person in
with a username and a password, with no code by phone. A plugin of its own collects the
transactions and the bills of the electricity service, as the gas plugin does for gas.

One account of the electricity service holds one customer for each house. A household that
keeps one workspace for each house wants the transactions of that house only.

## 2. Users

- An editor of a workspace: connects the electricity account one time, and assigns the customer of the house.
- An administrator of the instance: collects the transactions of an earlier period for one workspace, from the command line.
- A member of the workspace: reads the last reading on the home page, reads the transactions
  and the bills, downloads a bill, and receives the bill each month.

## 3. Out of scope

- A payment through Maroid.
- The balance, the tariff group, or the deposit of a customer on the connection page.
- The monthly summary of the electricity service.
- A copy of a bill in Maroid. The notification and the download fetch it, and keep none.
- The transactions or the bills of a customer that the workspace assigned before.
- A collection of an earlier period from a page or from a bot command.
- A chart of a range other than the last 12 months.
- A reading that a person submits to the electricity service.
- An electricity account of one user. The connection belongs to the workspace.
- Two or more customers in one workspace.
- A bot command or a tool of the plugin.
- The removal of the plugin of the distributor. The owner removes it in a separate change.
- A move of the transactions that the plugin of the distributor stored.

## 4. Definitions

The glossary holds each term of this feature: `GLO-electricity-service`, `GLO-customer`,
`GLO-assigned-customer`, `GLO-meter`, `GLO-reading`, `GLO-usage`, `GLO-transaction`, `GLO-bill`.

The electricity service splits a reading in two charges on one date when the tariff changes
within the month. The charges of one customer on one date form one reading. The usage and the
charge of the reading are the sums of those charges. Its meter value is the highest of them.
The electricity service reports the usage of a charge, and Maroid computes none.

## 5. Functional requirements

`requirements-pages.md` holds `ELECACC-FR-032` to `ELECACC-FR-045`: the widget, the
transactions page, and the bills page.

### Credentials

### `ELECACC-FR-001`

An editor must store the username and the password of the electricity account as settings of the workspace.

### `ELECACC-FR-002`

The plugin must declare the password as a secret field.

### The connection page

### `ELECACC-FR-003`

The connection page must show "not connected" when Maroid holds no connection for the acting workspace.

### `ELECACC-FR-004`

The connection page must name the settings that an editor must fill, when the settings of the workspace are absent.

### `ELECACC-FR-005`

The connection page must list the customers of the account when Maroid holds a connection.
Each customer shows its customer number, its name, and its address.

**Why:** A read of the customers proves that the connection still works. A new customer
at the electricity service appears with no further step.

### `ELECACC-FR-006`

The connection page must mark the assigned customer.

### `ELECACC-FR-007`

The connection page must report the electricity service as unavailable when the electricity service fails with no rejection of the credentials.

### Connect and disconnect

### `ELECACC-FR-008`

An editor must start a connection from the connection page.

### `ELECACC-FR-009`

The plugin must sign in to the electricity service with the stored credentials, and store the connection, when an editor starts a connection.

**Examples:**

- Normal case: the credentials are correct. The page lists the customers.
- Limit case: the workspace was connected before. The new connection replaces the old one. The assigned customer stays.
- Unwanted case: the password is wrong. The page reports the credentials rejected. Maroid holds no connection.

### `ELECACC-FR-010`

An editor must end the connection from the connection page.

### `ELECACC-FR-011`

The end of a connection must remove the assigned customer.

**Why:** A new connection can name a different account, which does not hold that customer.

### `ELECACC-FR-012`

The plugin must sign in again with the stored credentials, once, and replace the connection, when the electricity service rejects the session during any action.

**Why:** The session of the electricity service ends after a short time. No code by phone is
needed, so every action recovers with no person present.

**Examples:**

- Normal case: the session ended an hour ago. A member opens the connection page. The plugin signs in, and the page lists the customers.
- Unwanted case: the new sign in fails too. The action fails with the reason of `ELECACC-FR-013` or `ELECACC-FR-007`.

### `ELECACC-FR-013`

The plugin must remove the connection when the electricity service rejects the stored credentials.

**Examples:**

- Normal case: the owner changed the password at the electricity service. The next read of the connection page shows "not connected". The assigned customer stays.

### The customer

### `ELECACC-FR-014`

An editor must assign one customer of the account to the workspace.

**Examples:**

- Normal case: the account holds two customers. The editor assigns the second one. The page marks it.
- Limit case: the account holds one customer. The editor still assigns it.
- Limit case: the editor assigns a different customer. It replaces the first one.

### `ELECACC-FR-015`

The plugin must refuse a customer that the account does not hold.

### `ELECACC-FR-048`

The plugin must store the assignment of a customer, and call no collection, when an editor assigns it.

**Why:** An assignment answers at once. An administrator collects the history of the
customer from the command line after the assignment (`ELECACC-FR-025`), and the schedule
collects each month after that.

**Examples:**

- Normal case: the editor assigns the customer on 2026-10-14. The page marks it. The plugin reads no transaction and no bill, and the workspace receives no bill.
- Limit case: the editor assigns a different customer. It replaces the first one, and the plugin reads nothing of it.

### The collection

### `ELECACC-FR-017`

The plugin must collect the transactions and the bills of the previous calendar month of the assigned customer of each workspace on the schedule of the collection.

**Examples:**

- Normal case: the collection runs on 2026-11-01. It reads the transactions and the bills of October 2026.

### `ELECACC-FR-018`

The schedule of the collection must default to 10:00 on the first day of each month.

**Why:** The plugin of the distributor ran then, and the electricity service issues the bill of a month on its last day.

### `ELECACC-FR-019`

The collection must skip a workspace that holds no connection or no assigned customer.

### `ELECACC-FR-020`

The collection must fail with an error that names the rejection when the electricity service rejects the stored credentials.

### `ELECACC-FR-021`

The collection must fail with an error, and store nothing, when the account no longer holds the assigned customer.

### `ELECACC-FR-022`

Maroid must keep one transaction for one kind of operation of one customer on one date with one meter value, whatever the count of the collections that read it.

**Examples:**

- Normal case: two collections read the same payment. Maroid holds one transaction.
- Limit case: two charges on 2026-04-28, at the meter values 1163 and 1317. Maroid holds two transactions.
- Limit case: the electricity service corrects the amount of a charge. Maroid holds one transaction, with the corrected amount.

### `ELECACC-FR-023`

Maroid must keep one bill for one bill number.

**Examples:**

- Normal case: two collections read the bill `8370764-2026-9`. Maroid holds one bill.

### `ELECACC-FR-024`

Maroid must keep a transaction and a bill after a change of the assigned customer or the end of the connection.

### The collection of an earlier period

### `ELECACC-FR-025`

An administrator must collect the transactions and the bills of one workspace for a period from the command line, with the first and the last month of the period.

**Why:** The scheduled collection reads one month, and an assignment reads none
(`ELECACC-FR-048`). The chart shows 12. An administrator fills the history once, for each
workspace, after the assignment.

**Examples:**

- Normal case: the period from 2025-10 to 2026-09. Maroid stores the transactions and the bills of the assigned customer in that period.
- Limit case: the period holds transactions that Maroid holds. Maroid holds one of each (`ELECACC-FR-022`).
- Unwanted case: the first month is after the last month. The command fails, and calls no service.

### `ELECACC-FR-026`

The command line collection must send no bill.

**Why:** A period of 12 months holds 12 readings. A bill for each one floods the chat.

### `ELECACC-FR-027`

The command line collection must fail with an error when the workspace holds no connection or no assigned customer.

### The bill notification

### `ELECACC-FR-028`

The plugin must declare the bill as a notification topic, which a manager routes in each workspace.

### `ELECACC-FR-029`

A collection of the previous calendar month must send the bill of the latest reading that it stored for the first time.

**Examples:**

- Normal case: the run stores the reading of 2026-09-28. The workspace receives its bill.
- Limit case: the run stores no new charge. The workspace receives nothing.
- Limit case: the run stores the two charges of 2026-04-28. The workspace receives one bill.

### `ELECACC-FR-030`

The bill notification must show the date of the reading, the meter value, the usage, and the charge.

**Examples:**

- Normal case: the reading of 2026-04-28. The message shows 2026-04-28, 1317 kWh, 166.5 kWh, and 39.39 ₾.

### `ELECACC-FR-031`

The bill notification must attach the bill of the month of the reading, when the electricity service holds it.

**Examples:**

- Normal case: the reading of 2026-09-28 and the bill `8370764-2026-9`. The message carries the document.
- Unwanted case: the download of the document fails. The message goes without it.

### Permissions

### `ELECACC-FR-046`

A member must hold a permission whose lowest role is viewer to read the connection page, the widget, the transactions page, and the bills page, and to download a bill.

### `ELECACC-FR-047`

A member must hold a permission whose lowest role is editor to start a connection, to end it, and to assign the customer.

## 6. Invariants

### `ELECACC-INV-001`

One workspace holds at most one connection and at most one assigned customer.

### `ELECACC-INV-002`

A transaction, a bill, a connection, and an assigned customer of one workspace serve no other workspace.

### `ELECACC-INV-003`

No answer of Maroid to a client holds a session or a password of an electricity account.

### `ELECACC-INV-004`

No answer of the plugin to a client holds the name or the personal number of the account holder.

### `ELECACC-INV-005`

A member downloads only a bill that Maroid holds for the assigned customer of the acting workspace.

## 7. Constraints from the guidelines

| Rule      | Guideline         | Effect on this feature                                                                          |
| --------- | ----------------- | ----------------------------------------------------------------------------------------------- |
| `PLG-007` | Plugin model      | The plugin reads the settings, the connection, and the database through the host.               |
| `PLG-008` | Plugin model      | The plugin imports neither the gas plugin nor the plugin of the distributor.                    |
| `ARC-008` | Architecture      | The hub holds no rule that names the electricity plugin.                                        |
| `OWN-004` | Record ownership  | The connection, the assigned customer, the transactions, and the bills belong to one workspace. |
| `DAT-001` | Data              | The transactions and the bills live in the schema of the plugin.                                |
| `CFG-007` | Configuration     | No credential of a workspace stays in the configuration file.                                   |
| `EXT-005` | External services | A failure of the electricity service is an error, never an empty result.                        |
| `LOG-008` | Logging           | No log holds a password or a session.                                                           |
| `SEC-013` | Security          | Each route and the widget of the plugin name a permission.                                      |
| `UI-006`  | Web UI            | The pages import no module of the deck.                                                         |
| `JOB-007` | Jobs              | A failed collection of one workspace does not stop the collection of another.                   |
| `NTF-002` | Notifications     | The bill keeps its notification topic. The plugin names no destination.                         |
| `CLI-005` | Command line      | The plugin adds the command through the command registry.                                       |
| `OWN-003` | Ownership         | The command takes the acting workspace from a flag, and carries no acting user.                 |

## 8. Open questions

| #   | Question                                                                                                              | Owner  | Answer                                                                                                                                                                                    |
| --- | --------------------------------------------------------------------------------------------------------------------- | ------ | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1   | What does an assignment collect?                                                                                      | Temuri | Nothing, changed on 2026-10-14. An administrator collects the history from the command line, and the schedule collects each month (`ELECACC-FR-048`, `ELECACC-FR-025`, `ELECACC-FR-017`). |
| 2   | How does a member reach the document of a bill?                                                                       | Temuri | Through Maroid, which signs in and fetches it (`ELECACC-FR-042`).                                                                                                                         |
| 3   | Can one customer hold two payments on one date? `ELECACC-FR-022` merges them, because a payment holds no meter value. | Temuri | No. The rule stays. `ELECACC-FR-022`.                                                                                                                                                     |
| 4   | Does the electricity service ever issue two bills for one month? The bill number names the month.                     | Temuri | No. One bill for each month. `ELECACC-FR-023`, `ELECACC-DD-010`.                                                                                                                          |
| 5   | What happens to the plugin of the distributor?                                                                        | Temuri | It stays. The owner removes it in a separate change.                                                                                                                                      |

## Retired identifiers

| ID               | Retired    | Reason                                                                   |
| ---------------- | ---------- | ------------------------------------------------------------------------ |
| `ELECACC-FR-016` | 2026-10-14 | An assignment collects nothing. `ELECACC-FR-048` replaces the statement. |
