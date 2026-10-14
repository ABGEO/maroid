---
id: GWPACC
title: The water account of a workspace
type: requirements
status: approved
created: 2026-10-12
updated: 2026-10-14
approved_by: Temuri
approved_on: 2026-10-12
constrained_by: [PLG, ARC, OWN, DAT, CFG, EXT, LOG, SEC, UI, JOB]
---

# Requirements: The water account of a workspace

## 1. Problem

The water plugin signs in to the water service each month, reads every customer of the
account, and writes the readings to the log. Nothing keeps a reading, and nothing shows one.

One account of the water service holds one customer for each house. A household that keeps
one workspace for each house wants the readings of that house only. Today the plugin
cannot tell which customer belongs to which workspace.

The water service gives a session token for a username and a password, with no code by
phone. The sign in can therefore run with no person present.

## 2. Users

- An editor of a workspace: connects the water account one time, and assigns the customer
  of the house.
- A member of the workspace: reads the usage of the last month for each meter on the home page,
  and compares the months of the last year on the readings page.

## 3. Out of scope

- A price, a bill, a balance, or a payment.
- The readings of a customer that the workspace assigned before.
- A chart of a range other than the last 12 months.
- A reading that a person submits to the water service.
- A notification about a reading.
- A water account of one user. The connection belongs to the workspace.
- Two or more customers in one workspace.
- A bot command or a tool of the plugin.
- A change of the schedule of the collection.

## 4. Definitions

The glossary holds each term of this feature: `GLO-water-service`, `GLO-customer`,
`GLO-communal-number`, `GLO-assigned-customer`, `GLO-meter`, `GLO-reading`, `GLO-usage`.

## 5. Functional requirements

### Credentials

### `GWPACC-FR-001`

An editor must store the username and the password of the water account as settings of the workspace.

### `GWPACC-FR-002`

The plugin must declare the password as a secret field.

### The connection page

### `GWPACC-FR-003`

The connection page must show "not connected" when Maroid holds no connection for the acting workspace.

### `GWPACC-FR-004`

The connection page must name the settings that an editor must fill, when the settings of the workspace are absent.

### `GWPACC-FR-005`

The connection page must list the customers of the account when Maroid holds a connection.
Each customer shows its customer number, its communal number, and its address.

**Why:** A read of the customers proves that the connection still works. Maroid holds no
copy of the list, so a new customer at the water service appears with no further step.

### `GWPACC-FR-006`

The connection page must mark the assigned customer.

### `GWPACC-FR-007`

The connection page must report the water service as unavailable when the water service fails with no rejection of the connection.

**Examples:**

- Normal case: the water service answers with status 503. The page reports it unavailable. The connection stays.

### Connect and disconnect

### `GWPACC-FR-008`

An editor must start a connection from the connection page.

### `GWPACC-FR-009`

The plugin must sign in to the water service with the stored credentials, and store the connection, when an editor starts a connection.

**Examples:**

- Normal case: the credentials are correct. The page lists the customers.
- Limit case: the workspace was connected before. The new connection replaces the old one. The assigned customer stays.
- Unwanted case: the password is wrong. The page reports the credentials rejected. Maroid holds no connection.

### `GWPACC-FR-010`

An editor must end the connection from the connection page.

### `GWPACC-FR-011`

The end of a connection must remove the assigned customer.

**Why:** A new connection can name a different account, which does not hold that customer.

### `GWPACC-FR-012`

The plugin must remove the connection when the water service rejects the stored credentials.

**Examples:**

- Normal case: the password changed at the water service. The token ends, and the new sign in of `GWPACC-FR-018` fails. The page shows "not connected", and an editor connects again.

### The customer

### `GWPACC-FR-013`

An editor must assign one customer of the account to the workspace.

**Examples:**

- Normal case: the account holds three customers. The editor assigns the second one. The page marks it.
- Limit case: the account holds one customer. The editor still assigns it.
- Limit case: the editor assigns a different customer. It replaces the first one.

### `GWPACC-FR-014`

The plugin must refuse a customer that the account does not hold.

### `GWPACC-FR-015`

The plugin must collect the readings of the assigned customer when an editor assigns it.

**Why:** The scheduled collection runs once a month. The home page would show nothing until then.

### The collection

### `GWPACC-FR-016`

The plugin must collect the readings of the assigned customer of each workspace on the schedule of the collection.

### `GWPACC-FR-017`

The collection must skip a workspace that holds no connection or no assigned customer.

### `GWPACC-FR-018`

The plugin must sign in again with the stored credentials, once, and replace the connection, when the water service rejects the connection during any action.

**Why:** No code by phone is needed, so every action recovers with no person present.

**Examples:**

- Normal case: the token ended. A member opens the connection page. The plugin signs in, and the page lists the customers.
- Normal case: the token ended. The collection signs in, stores the new token, and collects the readings.
- Limit case: Maroid holds no connection. The page shows "not connected" (`GWPACC-FR-003`), and the plugin does not sign in.
- Unwanted case: the new sign in fails too. The action fails with the reason of `GWPACC-FR-012` or `GWPACC-FR-007`.

### `GWPACC-FR-019`

The collection must fail with an error when the account no longer holds the assigned customer.

**Examples:**

- Unwanted case: the water service removed the customer. The run fails, and stores nothing. The page no longer marks a assigned customer in the list.

### `GWPACC-FR-020`

Maroid must keep one reading for one meter on one date, whatever the count of the collections that read it.

**Examples:**

- Normal case: two collections read the same value of the same meter on the same date. Maroid holds one reading.
- Limit case: the same meter shows a new value on a new date. Maroid holds two readings.
- Limit case: the water service corrects the value of a date, for example an estimate that a real reading replaces. Maroid holds one reading of that date, with the corrected value.

### `GWPACC-FR-021`

Maroid must keep a reading after a change of the assigned customer or the end of the connection.

**Why:** A mistaken click must not lose the history. Each reading names its customer.

### The widget

### `GWPACC-FR-022`

The plugin must declare a widget that lists the last reading of each meter of the assigned customer.
Each line shows the serial number of the meter, the date of the reading, the value, and the usage.

**Why:** The usage tells the household how much water the house used in the last month.

**Examples:**

- Normal case: one meter, 1204.0 on 15/08/2026 and 1216.5 on 15/09/2026. The line shows 1216.5 on 15/09/2026, and a usage of 12.5.
- Limit case: the first collection after the assignment of the customer holds one reading of the meter. The line shows the value and the date, and no usage.
- Limit case: the customer has two meters. The widget shows two lines.
- Unwanted case: a value below the one before it, after a change of the meter. The line shows the negative usage as it is.

### `GWPACC-FR-023`

The widget must answer "not set up", with a link to the connection page, when the workspace holds no assigned customer.

**Why:** The end of a connection removes the assigned customer, so a workspace that an
editor disconnected answers "not set up". A token that the water service rejects leaves the
assigned customer, and the readings that Maroid holds stay correct, so the widget still shows them.

### `GWPACC-FR-024`

The widget must answer "empty" when Maroid holds no reading of the assigned customer.

### `GWPACC-FR-025`

The widget must read the readings that Maroid holds, with no call to the water service.

**Why:** The home page reads every widget at each visit. A call to the water service would slow it.

### The readings page

### `GWPACC-FR-028`

The readings page must list the readings of the assigned customer, the latest date first.
Each row shows the serial number of the meter, the date, the value, and the usage.

**Examples:**

- Normal case: two meters and 24 months of readings. The page shows the readings of both meters, the latest first.
- Limit case: the earliest reading of a meter. The row shows no usage.
- Limit case: the workspace assigned a different customer. The page lists the readings of the new customer only.

### `GWPACC-FR-029`

The readings page must show at most 20 readings at a time, with a way to the next and the previous readings.

**Why:** Each meter adds one reading a month, so the list grows without end.

**Examples:**

- Normal case: 45 readings. The first view shows the 20 latest. The next view shows the following 20.
- Limit case: 20 readings. The page shows all 20, and no way to further readings.

### `GWPACC-FR-030`

The readings page must show a chart of the readings dated within the last 12 months: the usage of
each reading as a column, and the value of the meter as a line, for each meter.

**Why:** The columns compare the months. The line shows a change of the meter, such as a replacement.

**Examples:**

- Normal case: one meter, 12 readings. The chart shows 12 columns and one line.
- Limit case: two meters. The chart shows the columns and the line of each meter, in two colors.
- Limit case: a reading dated 13 months ago. The list shows it. The chart does not, and the usage of the next reading still counts from it.

### `GWPACC-FR-031`

The readings page must ask for the setup, with a link to the connection page, when the workspace holds no assigned customer.

### `GWPACC-FR-032`

The readings page must report "no reading" when Maroid holds no reading of the assigned customer.

### `GWPACC-FR-033`

The readings page must read the readings that Maroid holds, with no call to the water service.

### `GWPACC-FR-034`

The widget must link to the readings page.

**Why:** The widget shows the last reading of two meters at most. The page shows the rest.

### Permissions

### `GWPACC-FR-026`

A member must hold a permission whose lowest role is viewer to read the connection page, the widget, and the readings page.

### `GWPACC-FR-027`

A member must hold a permission whose lowest role is editor to start a connection, to end it, and to assign the customer.

## 6. Invariants

### `GWPACC-INV-001`

One workspace holds at most one connection and at most one assigned customer.

### `GWPACC-INV-002`

A reading, a connection, and a assigned customer of one workspace serve no other workspace.

### `GWPACC-INV-003`

No answer of Maroid to a client holds a session token or a password of a water account.

## 7. Constraints from the guidelines

| Rule      | Guideline         | Effect on this feature |
| --------- | ----------------- | ---------------------- |
| `PLG-007` | Plugin model      | The plugin reads the settings, the connection, and the database through the host. |
| `ARC-008` | Architecture      | The hub holds no rule that names the water plugin. |
| `OWN-004` | Record ownership  | The connection, the assigned customer, and the readings belong to one workspace. |
| `DAT-001` | Data              | The readings live in the schema of the plugin. |
| `CFG-007` | Configuration     | No credential of a workspace stays in the configuration file. |
| `EXT-005` | External services | A failure of the water service is an error, never an empty result. |
| `LOG-008` | Logging           | No log holds a password or a session token. |
| `SEC-013` | Security          | Each route and the widget of the plugin name a permission. |
| `UI-006`  | Web UI            | The connection page imports no module of the deck. |
| `JOB-007` | Jobs              | A failed collection of one workspace does not stop the collection of another. |

## 8. Open questions

| #   | Question | Owner | Answer |
| --- | -------- | ----- | ------ |
| 1   | What happens when the water service rejects the token? | Temuri | Every action signs in again once, changed on 2026-10-14 (`GWPACC-FR-018`). A rejected password removes the connection (`GWPACC-FR-012`). |
| 2   | What does the widget show for two meters? | Temuri | One line for each meter. `GWPACC-FR-022`. |
| 3   | Does a change of customer, or the end of a connection, remove the readings? | Temuri | No. `GWPACC-FR-021`. |
| 4   | Does the plugin assign the customer when the account holds one? | Temuri | No. The editor assigns it. `GWPACC-FR-013`. |
| 5   | Which role connects and assigns the customer? | Temuri | Editor. `GWPACC-FR-027`. |
| 6   | How long does Maroid keep a reading? | Temuri | As long as the workspace exists. |

## Retired identifiers

This file has no retired identifier.
