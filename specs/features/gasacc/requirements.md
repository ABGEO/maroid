---
id: GASACC
title: The gas account of a workspace
type: requirements
status: approved
created: 2026-10-13
updated: 2026-10-14
approved_by: Temuri
approved_on: 2026-10-13
constrained_by: [PLG, ARC, OWN, DAT, CFG, EXT, LOG, SEC, UI, JOB, NTF, CLI]
---

# Requirements: The gas account of a workspace

## 1. Problem

The gas plugin signs in to the gas service each month with the username, the password,
and the customer number that a workspace stores. It keeps the transactions of the last
month and sends the bill. Nothing shows a transaction, and a person must find the
customer number at the gas service by hand.

One account of the gas service holds one customer for each house. A household that keeps
one workspace for each house wants the transactions of that house only.

The gas service gives a session token for a username and a password, with no code by
phone. The sign in can therefore run with no person present.

## 2. Users

- An editor of a workspace: connects the gas account one time, and assigns the customer
  of the house.
- An administrator of the instance: collects the transactions of an earlier period for one
  workspace, from the command line.
- A member of the workspace: reads the last reading on the home page, reads the
  transactions and the bills on the transactions page, and receives the bill.

## 3. Out of scope

- A payment through Maroid.
- The balance or the status of a customer on the connection page.
- The debt report, the registration report, and the statement of the gas service.
- A copy of a bill or of a meter photo in Maroid. The bill notification downloads its files, and keeps none.
- The transactions of a customer that the workspace assigned before.
- A collection of an earlier period from the connection page or from a bot command.
- A chart of a range other than the last 12 months.
- A reading that a person submits to the gas service.
- A gas account of one user. The connection belongs to the workspace.
- Two or more customers in one workspace.
- A bot command or a tool of the plugin.
- A change of the schedule of the collection.

## 4. Definitions

The glossary holds each term of this feature: `GLO-gas-service`, `GLO-customer`,
`GLO-assigned-customer`, `GLO-meter`, `GLO-reading`, `GLO-usage`, `GLO-transaction`,
`GLO-bill`, `GLO-meter-photo`. The gas service reports the usage of a reading, and
Maroid computes none.

## 5. Functional requirements

### Credentials

### `GASACC-FR-001`

An editor must store the username and the password of the gas account as settings of the workspace.

### `GASACC-FR-002`

The plugin must declare the password as a secret field.

### `GASACC-FR-003`

The settings must hold no customer number.

**Why:** The editor assigns the customer from the list of the account (`GASACC-FR-014`).

### The connection page

### `GASACC-FR-004`

The connection page must show "not connected" when Maroid holds no connection for the acting workspace.

### `GASACC-FR-005`

The connection page must name the settings that an editor must fill, when the settings of the workspace are absent.

### `GASACC-FR-006`

The connection page must list the customers of the account when Maroid holds a connection.
Each customer shows its customer number, its name, and its address.

**Why:** A read of the customers proves that the connection still works. A new customer
at the gas service appears with no further step.

### `GASACC-FR-007`

The connection page must mark the assigned customer.

### `GASACC-FR-008`

The connection page must report the gas service as unavailable when the gas service fails with no rejection of the connection.

### Connect and disconnect

### `GASACC-FR-009`

An editor must start a connection from the connection page.

### `GASACC-FR-010`

The plugin must sign in to the gas service with the stored credentials, and store the connection, when an editor starts a connection.

**Examples:**

- Normal case: the credentials are correct. The page lists the customers.
- Limit case: the workspace was connected before. The new connection replaces the old one. The assigned customer stays.
- Unwanted case: the password is wrong. The page reports the credentials rejected. Maroid holds no connection.
- Unwanted case: the gas service asks for a human check before the sign in. The page reports the sign in blocked. Maroid holds no connection.

### `GASACC-FR-011`

An editor must end the connection from the connection page.

### `GASACC-FR-012`

The end of a connection must remove the assigned customer.

**Why:** A new connection can name a different account, which does not hold that customer.

### `GASACC-FR-013`

The plugin must remove the connection when the gas service rejects the stored credentials.

**Examples:**

- Normal case: the password changed at the gas service. The token ends, and the new sign in of `GASACC-FR-019` fails. The page shows "not connected", and an editor connects again.

### The customer

### `GASACC-FR-014`

An editor must assign one customer of the account to the workspace.

**Examples:**

- Normal case: the account holds two customers. The editor assigns the second one. The page marks it.
- Limit case: the account holds one customer. The editor still assigns it.
- Limit case: the editor assigns a different customer. It replaces the first one.

### `GASACC-FR-015`

The plugin must refuse a customer that the account does not hold.

### `GASACC-FR-043`

The plugin must store the assignment of a customer, and call no collection, when an editor assigns it.

**Why:** An assignment answers at once. An administrator collects the history of the
customer from the command line after the assignment (`GASACC-FR-023`), and the schedule
collects each month after that.

**Examples:**

- Normal case: the editor assigns the customer on 2026-10-14. The page marks it. The plugin reads no transaction, and the workspace receives no bill.
- Limit case: the editor assigns a different customer. It replaces the first one, and the plugin reads nothing of it.

### The collection

### `GASACC-FR-017`

The plugin must collect the transactions of the previous calendar month of the assigned customer of each workspace on the schedule of the collection.

**Examples:**

- Normal case: the collection runs on 2026-11-01. It reads the transactions from 2026-10-01 to 2026-10-31.

### `GASACC-FR-018`

The collection must skip a workspace that holds no connection or no assigned customer.

### `GASACC-FR-019`

The plugin must sign in again with the stored credentials, once, and replace the connection, when the gas service rejects the connection during any action.

**Why:** No code by phone is needed, so every action recovers with no person present.

**Examples:**

- Normal case: the token ended. A member opens the connection page. The plugin signs in, and the page lists the customers.
- Normal case: the token ended. The collection signs in, stores the new token, and collects the transactions.
- Limit case: Maroid holds no connection. The page shows "not connected" (`GASACC-FR-004`), and the plugin does not sign in.
- Unwanted case: the new sign in fails too. The action fails with the reason of `GASACC-FR-013` or `GASACC-FR-008`.

### `GASACC-FR-020`

The collection must fail with an error, and store nothing, when the account no longer holds the assigned customer.

### `GASACC-FR-021`

Maroid must keep one transaction for one kind of operation of one customer on one date, whatever the count of the collections that read it.

**Examples:**

- Normal case: two collections read the same payment. Maroid holds one transaction.
- Limit case: a reading and a payment on the same date. Maroid holds two transactions.
- Limit case: the gas service corrects the amount of a reading. Maroid holds one transaction of that date, with the corrected amount.

### `GASACC-FR-022`

Maroid must keep a transaction after a change of the assigned customer or the end of the connection.

### The collection of an earlier period

### `GASACC-FR-023`

An administrator must collect the transactions of one workspace for a period from the command line, with the first and the last date of the period.

**Why:** The scheduled collection reads one month, and an assignment reads none
(`GASACC-FR-043`). The chart shows 12. An administrator fills the history once, for each
workspace, after the assignment.

**Examples:**

- Normal case: the period from 2025-10-01 to 2026-09-30. Maroid stores the transactions of the assigned customer in that period.
- Limit case: the period holds transactions that Maroid holds. Maroid holds one of each (`GASACC-FR-021`).
- Unwanted case: the first date is after the last date. The command fails, and calls no service.

### `GASACC-FR-024`

The command line collection must send no bill.

**Why:** A period of 12 months holds 12 readings. A bill for each one floods the chat.

### `GASACC-FR-025`

The command line collection must fail with an error when the workspace holds no connection or no assigned customer.

**Why:** An administrator runs it by hand, so a silent skip hides the cause.

### The bill

### `GASACC-FR-026`

The plugin must declare the bill as a notification topic, which a manager routes in each workspace.

**Why:** A workspace chooses the chat that receives the bill. A workspace with no route receives nothing.

### `GASACC-FR-027`

A collection of the previous calendar month must send the bill of the latest reading that it stored for the first time, with the bill and the meter photo attached.

**Examples:**

- Normal case: the run stores the reading of 2026.09.28. The workspace receives its bill.
- Limit case: the run stores no new reading. The workspace receives nothing.
- Limit case: the run stores two new readings. The workspace receives the bill of the latest one.

### `GASACC-FR-028`

The bill must show the date of the reading, the meter value, the usage, and the amount.

**Examples:**

- Normal case: the reading of 2026.09.28. The bill shows 2026.09.28, 1532 m³, 21 m³, and 11.33 ₾.

### The widget

### `GASACC-FR-029`

The plugin must declare a widget that shows the latest reading of the assigned customer: its date, its usage, its meter value, and its charge.

### `GASACC-FR-030`

The widget must answer "not set up", with a link to the connection page, when the workspace holds no assigned customer.

### `GASACC-FR-031`

The widget must answer "empty" when Maroid holds no reading of the assigned customer.

### `GASACC-FR-032`

The widget must read the transactions that Maroid holds, with no call to the gas service.

### `GASACC-FR-033`

The widget must link to the transactions page.

### The transactions page

### `GASACC-FR-034`

The transactions page must list the transactions of the assigned customer, the latest date first.
Each row shows the date, the name of the operation, the amount, and the balance.
A reading also shows its usage and its meter value.

### `GASACC-FR-035`

The transactions page must show at most 20 transactions at a time, with a way to the next and the previous transactions.

### `GASACC-FR-036`

A row must link to the bill and to the meter photo at the gas service, when the gas service holds them for that transaction.

**Why:** The gas service serves each file at its own address, with no session. Maroid stores no file.

**Examples:**

- Normal case: a reading with both files. The row offers two links.
- Limit case: a payment. The row offers no link.

### `GASACC-FR-037`

The transactions page must show a chart of the readings dated within the last 12 months:
the usage of each reading as a column, and its charge as a line.

**Examples:**

- Normal case: 12 readings and 12 payments. The chart shows 12 columns and one line of 12 points.
- Limit case: a reading dated 13 months ago. The list shows it. The chart does not.

### `GASACC-FR-038`

The transactions page must ask for the setup, with a link to the connection page, when the workspace holds no assigned customer.

### `GASACC-FR-039`

The transactions page must report "no transaction" when Maroid holds no transaction of the assigned customer.

### `GASACC-FR-040`

The list and the chart must read the transactions that Maroid holds, with no call to the gas service.

### Permissions

### `GASACC-FR-041`

A member must hold a permission whose lowest role is viewer to read the connection page, the widget, and the transactions page.

### `GASACC-FR-042`

A member must hold a permission whose lowest role is editor to start a connection, to end it, and to assign the customer.

## 6. Invariants

### `GASACC-INV-001`

One workspace holds at most one connection and at most one assigned customer.

### `GASACC-INV-002`

A transaction, a connection, and an assigned customer of one workspace serve no other workspace.

### `GASACC-INV-003`

No answer of Maroid to a client holds a session token or a password of a gas account.

### `GASACC-INV-004`

No answer of Maroid to a client holds the personal number of a customer or of the account holder.

## 7. Constraints from the guidelines

| Rule      | Guideline         | Effect on this feature |
| --------- | ----------------- | ---------------------- |
| `PLG-007` | Plugin model      | The plugin reads the settings, the connection, and the database through the host. |
| `ARC-008` | Architecture      | The hub holds no rule that names the gas plugin. |
| `OWN-004` | Record ownership  | The connection, the assigned customer, and the transactions belong to one workspace. |
| `DAT-001` | Data              | The transactions live in the schema of the plugin. |
| `CFG-007` | Configuration     | No credential of a workspace stays in the configuration file. |
| `EXT-005` | External services | A failure of the gas service is an error, never an empty result. |
| `LOG-008` | Logging           | No log holds a password or a session token. |
| `SEC-013` | Security          | Each route and the widget of the plugin name a permission. |
| `UI-006`  | Web UI            | The pages import no module of the deck. |
| `JOB-007` | Jobs              | A failed collection of one workspace does not stop the collection of another. |
| `NTF-002` | Notifications     | The bill keeps its notification topic. The plugin names no destination. |
| `CLI-005` | Command line      | The plugin adds the command through the command registry. |
| `OWN-003` | Ownership         | The command takes the acting workspace from a flag, and carries no acting user. |

## 8. Open questions

| #   | Question | Owner | Answer |
| --- | -------- | ----- | ------ |
| 1   | Can one customer hold two transactions of one operation on one date, such as two payments? `GASACC-FR-021` merges them. | Temuri | No. The rule stays. `GASACC-FR-021`. |
| 2   | Does the path of a bill or of a meter photo stay valid for months? If not, a download of an old transaction fails. | Temuri | Yes. Maroid stores each path once. |
| 3   | Which range does a collection read? | Temuri | The previous calendar month, on the schedule (`GASACC-FR-017`). The assignment collects nothing, changed on 2026-10-14 (`GASACC-FR-043`). An administrator collects an earlier period from the command line (`GASACC-FR-023`). |
| 4   | Does a customer ever hold two meters? The answer names no meter. | Temuri | No. One meter for each customer. The chart shows one series. |

## Retired identifiers

| ID              | Retired    | Reason                                                                |
| --------------- | ---------- | --------------------------------------------------------------------- |
| `GASACC-FR-016` | 2026-10-14 | An assignment collects nothing. `GASACC-FR-043` replaces the statement. |
