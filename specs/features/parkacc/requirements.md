---
id: PARKACC
title: The parking account of a user
type: requirements
status: approved
created: 2026-10-12
updated: 2026-10-12
approved_by: Temuri
approved_on: 2026-10-12
constrained_by: [PLG, ARC, OWN, CFG, EXT, LOG, SEC, UI, TG]
---

# Requirements: The parking account of a user

## 1. Problem

The parking plugin asks a person for two values that the person cannot find in the
parking service: a session token and a vehicle identifier. The person copies the token
out of the traffic of the mobile application, and enters it as a setting. When the token
ends, every command fails with an error that names no cause.

The parking service gives a token only after a code that it sends by SMS. The person
must type that code, so the sign in cannot run with no person present. A token lives
for weeks, so one sign in serves many commands.

## 2. Users

- A member who parks: connects their parking account one time, then starts and stops a
  parking session from the bot with no further sign in.
- A member who owns two vehicles: chooses the vehicle at the start of each parking session.

## 3. Out of scope

- A bot command that connects the parking account.
- A notification before a connection ends. The parking service gives no end time.
- A parking account that the parking service confirms by email, or with no code.
- A parking account that the members of a workspace share.
- A default vehicle, or a vehicle that Maroid remembers between two parking sessions.
- A change to the commands that read the balance, read the parking session, or stop it,
  beyond `PARKACC-FR-019` and `PARKACC-FR-021`.

## 4. Functional requirements

### Credentials

### `PARKACC-FR-001`

A user must store the username and the password of their parking account as settings of their own.

**Why:** A second member of the workspace parks with their own parking account.

### `PARKACC-FR-002`

The plugin must declare the password as a secret field.

### `PARKACC-FR-003`

The plugin must ask no person for a session token or for a vehicle identifier.

### The connection page

### `PARKACC-FR-004`

The connection page must show "not connected" when Maroid holds no connection for the acting user.

### `PARKACC-FR-005`

The connection page must show the first name, the last name, the balance, and the count of the free parkings left today, when Maroid holds a connection.

**Why:** A read of the parking account proves that the connection still works.

**Examples:**

- Normal case: the parking service answers. The page shows "Connected as Nino Beridze", 4.52 GEL, and 4.
- Unwanted case: the parking service rejects the connection. `PARKACC-FR-019` applies, and the page shows "not connected".

### `PARKACC-FR-006`

The connection page must report the parking service as unavailable when the parking service fails with no rejection of the connection.

**Why:** An outage at the parking service is no reason to end a working connection.

**Examples:**

- Normal case: the parking service answers with status 503. The page reports the service unavailable. The connection stays.

### `PARKACC-FR-007`

The connection page must name the settings that the acting user must fill, when the settings of the acting user are absent.

### Connect

### `PARKACC-FR-008`

A member must start a connection from the connection page.

### `PARKACC-FR-009`

The plugin must sign in to the parking service with the stored credentials when a member starts a connection.

### `PARKACC-FR-010`

The plugin must ask the parking service to send a code to the phone of the parking account after the sign in.

### `PARKACC-FR-011`

The connection page must show the masked phone number that the parking service gives for the code.

### `PARKACC-FR-012`

The connection page must report rejected credentials when the parking service rejects the sign in.

**Examples:**

- Unwanted case: the password is wrong. The page reports the credentials rejected. Maroid holds no pending connection.

### `PARKACC-FR-013`

The plugin must refuse a connection when the parking service asks for no code by phone.

**Why:** Out of scope. The page names the reason, so the person knows that the plugin cannot connect this parking account.

### `PARKACC-FR-014`

The plugin must store the connection when the parking service accepts the code.

**Examples:**

- Normal case: the member types the code within 5 minutes. The page shows the connected state of `PARKACC-FR-005`.
- Limit case: the member was connected before. The new connection replaces the old one.
  The old one serves every command until the parking service accepts the new code.

### `PARKACC-FR-015`

The connection page must keep the pending connection when the parking service rejects the code.

**Examples:**

- Normal case: the member types a wrong code. The page reports the code wrong. The member types the correct code, and the page shows the connected state.
- Limit case: the parking service refuses further attempts. The page shows the answer of the parking service.

### `PARKACC-FR-016`

Maroid must end a pending connection 5 minutes after the request of its code.

**Examples:**

- Normal case: the member types the code after 4 minutes. The parking service accepts it.
- Unwanted case: the member types the code after 6 minutes. The page asks the member to start a connection again.

### `PARKACC-FR-017`

A member must ask for a new code when the wait that the parking service gives ends.

**Examples:**

- Normal case: the parking service gives a wait of 60 seconds. After 61 seconds the member asks for a new code. The parking service sends it.
- Unwanted case: the member asks after 30 seconds. Maroid refuses, and names the seconds left.

### Disconnect and the end of a connection

### `PARKACC-FR-018`

A member must end their connection from the connection page.

### `PARKACC-FR-019`

The plugin must remove the connection when the parking service rejects it.

**Why:** The parking service gives no end time. A rejection is the only signal that a connection ended.

### `PARKACC-FR-020`

Maroid must keep a connection until the member ends it or the parking service rejects it.

**Examples:**

- Normal case: the member changes the password in the settings. The connection stays.

### `PARKACC-FR-021`

The bot must answer a command of the plugin with "not connected" and a link to the connection page, when Maroid holds no connection for the acting user.

**Examples:**

- Normal case: a member who never connected sends `/balance`. The bot answers with the link.
- Unwanted case: the parking service rejects the connection during `/status`. The plugin removes it, and the bot answers with the link.

### `PARKACC-FR-022`

Maroid must give a plugin the address of a page of that plugin in the deck.

**Why:** `PARKACC-FR-021` links to the connection page, and only the hub knows the address of the deck.

### `PARKACC-FR-023`

A member must hold a permission whose lowest role is viewer to read the connection page, to start a connection, and to end it.

**Why:** The connection belongs to the member, not to the workspace. A viewer who parks
with their own account needs it before an editor role lets them start a parking session.

### Vehicles

### `PARKACC-FR-024`

The plugin must read the vehicles of the parking account at the start of each parking session.

**Why:** A person adds a vehicle in the mobile application of the parking service. Maroid
reads the list each time, and holds no copy that can become stale.

### `PARKACC-FR-025`

The plugin must select the vehicle with no question when the parking account holds one vehicle.

### `PARKACC-FR-026`

The bot must ask the member to choose a vehicle when the parking account holds two or more vehicles.
Each choice shows the name and the plate number of the vehicle.

### `PARKACC-FR-027`

The bot must refuse to start a parking session when the parking account holds no vehicle.

### `PARKACC-FR-028`

The plugin must start the parking session for the vehicle that the member chose.

## 5. Invariants

### `PARKACC-INV-001`

One user holds at most one connection.

### `PARKACC-INV-002`

The connection of one user serves no command of another user.

### `PARKACC-INV-003`

No answer of Maroid to a client holds a session token or a password of a parking account.

## 6. Constraints from the guidelines

| Rule      | Guideline         | Effect on this feature                                                                |
| --------- | ----------------- | ------------------------------------------------------------------------------------- |
| `PLG-007` | Plugin model      | The plugin reads the settings, the connection, and the deck address through the host. |
| `ARC-008` | Architecture      | The deck address that the hub gives names no plugin.                                  |
| `OWN-004` | Record ownership  | The credentials and the connection belong to one user.                                |
| `CFG-007` | Configuration     | No credential of a person stays in the configuration file.                            |
| `EXT-005` | External services | A failure of the parking service is an error, never an empty result.                  |
| `LOG-008` | Logging           | No log holds a password, a code, or a session token.                                  |
| `SEC-013` | Security          | Each route and each command of the plugin names a permission.                         |
| `UI-006`  | Web UI            | The connection page imports from `@maroid/plugin-sdk` only.                           |
| `TG-005`  | Telegram          | The vehicle choice is a step of the conversation that starts a parking session.       |

## 7. Open questions

| #   | Question                                             | Owner  | Answer                                                  |
| --- | ---------------------------------------------------- | ------ | ------------------------------------------------------- |
| 1   | Does a change of the credentials end the connection? | Temuri | No. `PARKACC-FR-020`.                                   |
| 2   | Does a wrong code end the pending connection?        | Temuri | No. The member retries until it ends. `PARKACC-FR-015`. |
| 3   | Does a connected member start a second connection?   | Temuri | Yes. It replaces the first one. `PARKACC-FR-014`.       |
| 4   | Does the end of a connection remove the credentials? | Temuri | No. Only the connection.                                |
| 5   | Which role reads the connection page?                | Temuri | Viewer. `PARKACC-FR-023`.                               |

## Retired identifiers

This file has no retired identifier.
