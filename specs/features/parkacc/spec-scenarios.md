---
id: PARKACC
title: The scenarios of the parking account of a user
type: spec
status: approved
created: 2026-10-12
updated: 2026-10-12
approved_by: Temuri
approved_on: 2026-10-12
constrained_by: [TST, TRC]
requirements: features/parkacc/requirements.md
---

# Specification: The scenarios of the parking account of a user

A unit scenario runs the plugin against an `httptest` server that plays the parking
service, a fake `SecretProvider`, a fake `SettingsProvider`, and a fake `PageLinker`.
"The fake service" is that server. User A acts in workspace W unless the scenario says otherwise.

## `PARKACC-SC-001`

**Verifies:** `PARKACC-FR-001`, `PARKACC-FR-002`, `PARKACC-FR-003`
**Layer:** unit

**Given** the plugin.
**When** the test reflects the settings schema of `SettingsModel`.
**Then** the schema holds `username` (free text, user scope, required) and `password`
(secret, user scope, required), and no other field.

## `PARKACC-SC-002`

**Verifies:** `PARKACC-FR-004`
**Layer:** unit

**Given** no `session` secret.
**When** user A reads `GET /connections/self`.
**Then** the answer is 200 with `state: not_connected`. The fake service records no call.

## `PARKACC-SC-003`

**Verifies:** `PARKACC-FR-005`
**Layer:** unit

**Given** a `session` secret, and the fake service answers the person check with
`firstName: Nino`, `lastName: Beridze`, `balanceAmount: 4.52`, `dailyFreeParkingLeft: 4`.
**When** user A reads `GET /connections/self`.
**Then** the answer is 200 with `state: connected`, the names `Nino` and `Beridze`, the
balance `{"amount": 4.52, "currency": "GEL"}`, and `free_parkings_left: 4`.

## `PARKACC-SC-004`

**Verifies:** `PARKACC-FR-006`
**Layer:** unit

**Given** a `session` secret, and the fake service answers the person check with 503.
**When** user A reads `GET /connections/self`.
**Then** the answer is 502 `/problems/dev.maroid.parking/service-unavailable`. The `session` secret stays.

## `PARKACC-SC-005`

**Verifies:** `PARKACC-FR-007`
**Layer:** unit

**Given** the settings of user A are absent.
**When** user A reads `GET /connections/self`, then calls `POST /pending-connections`.
**Then** the read answers `credentials_stored: false`. The `POST` answers 422
`/problems/dev.maroid.parking/credentials-absent`. The fake service records no call.

## `PARKACC-SC-006`

**Verifies:** `PARKACC-FR-008`, `PARKACC-FR-009`, `PARKACC-FR-010`, `PARKACC-FR-011`
**Layer:** unit

**Given** stored credentials, and the fake service answers the sign in with `REQUIRES_TWO_FA`
and the send with `phone: 5******27`, `delay: "60"`.
**When** user A calls `POST /pending-connections` at time T.
**Then** the fake service records the sign in with the stored credentials and no
`Authorization` header, then the send with the identifier of the sign in. The answer is
`phone: 5******27`, `resend_at: T + 60 s`, `expires_at: T + 5 min`. A `pending`
secret exists with that expiry. The answer is 201, and `Location` ends with
`/pending-connections/self`.

## `PARKACC-SC-007`

**Verifies:** `PARKACC-FR-012`
**Layer:** unit

**Given** stored credentials, and the fake service answers the sign in with 401.
**When** user A calls `POST /pending-connections`.
**Then** the answer is 422 `/problems/dev.maroid.parking/credentials-rejected`. No `pending` secret exists.

## `PARKACC-SC-008`

**Verifies:** `PARKACC-FR-013`
**Layer:** unit

**Given** the fake service answers the sign in with `TWO_FA.PHONE_VERIFICATION: false`.
**When** user A calls `POST /pending-connections`.
**Then** the answer is 422 `/problems/dev.maroid.parking/verification-unsupported`. The fake service records no send.

## `PARKACC-SC-009`

**Verifies:** `PARKACC-FR-014`
**Layer:** unit

**Given** a `pending` secret, and the fake service answers the submit with `SUCCESS` and the header `session-token: T1`.
**When** user A posts `{"code":"0118"}` to `/pending-connections/self/verifications`.
**Then** the answer is 200 with `state: connected`. The `session` secret holds `T1`. No `pending` secret exists.

## `PARKACC-SC-010`

**Verifies:** `PARKACC-FR-014`, `PARKACC-INV-001`
**Layer:** unit

**Given** a `session` secret `T0` and a `pending` secret.
**When** `/balance` runs, then the verification succeeds with `T2`, then `/balance` runs again.
**Then** the first balance call carries `T0`, the second carries `T2`. One `session` secret exists.

## `PARKACC-SC-011`

**Verifies:** `PARKACC-FR-015`
**Layer:** unit

**Given** a `pending` secret, and the fake service rejects the code `1111` and accepts `0118`.
**When** user A posts `1111`, then `0118`.
**Then** the first answer is 422 `/problems/dev.maroid.parking/code-rejected` and the
`pending` secret stays. The second answer is 200 with `state: connected`.

## `PARKACC-SC-012`

**Verifies:** `PARKACC-FR-016`
**Layer:** unit

**Given** the fake `SecretProvider` reports `pending` absent, as after its expiry.
**When** user A posts a code to `/pending-connections/self/verifications`.
**Then** the answer is 404 `/problems/http/not-found`. The fake service records no submit.

## `PARKACC-SC-013`

**Verifies:** `PARKACC-FR-017`
**Layer:** unit

**Given** a `pending` secret with `resend_at` at T, and the fake service answers the send with `delay: "60"`.
**When** user A posts to `/pending-connections/self/code-requests` at T + 1 s.
**Then** the answer is 200 with `resend_at: T + 61 s`. The fake service records a send with the same identifier.

## `PARKACC-SC-014`

**Verifies:** `PARKACC-FR-017`
**Layer:** unit

**Given** a `pending` secret with `resend_at` 30 seconds ahead.
**When** user A posts to `/pending-connections/self/code-requests`.
**Then** the answer is 429 `/problems/dev.maroid.parking/code-wait` with `Retry-After: 30`. The fake service records no call.

## `PARKACC-SC-015`

**Verifies:** `PARKACC-FR-018`
**Layer:** unit

**Given** a `session` secret and stored credentials.
**When** user A calls `DELETE /connections/self`.
**Then** the answer is 204. No `session` secret exists. The settings are unchanged.

## `PARKACC-SC-016`

**Verifies:** `PARKACC-FR-019`
**Layer:** unit

**Given** a `session` secret, and the fake service answers the active session with 401.
**When** `/status` runs.
**Then** no `session` secret exists, and the bot answers "not connected" with the link.

## `PARKACC-SC-017`

**Verifies:** `PARKACC-FR-020`
**Layer:** unit

**Given** a `session` secret.
**When** the settings of user A change the password, then `/balance` runs.
**Then** the balance call carries the stored token, and the bot answers the balance.

## `PARKACC-SC-018`

**Verifies:** `PARKACC-FR-021`
**Layer:** unit

**Given** no `session` secret, and the fake `PageLinker` answers `https://deck.example/w/W/plugins/dev.maroid.parking/account`.
**When** each of `/start`, `/balance`, `/status`, and `/stop` runs.
**Then** each answer holds "not connected" and that link. The fake service records no call.

## `PARKACC-SC-019`

**Verifies:** `PARKACC-FR-021`
**Layer:** unit

**Given** no `session` secret, and the fake `PageLinker` answers `ErrPageAddressAbsent`.
**When** `/balance` runs.
**Then** the answer holds "not connected", names the Account page, and holds no link.

## `PARKACC-SC-020`

**Verifies:** `PARKACC-FR-022`
**Layer:** unit

**Given** a `pageLinker` with `deck_url` `https://deck.example`, and workspace W as the acting workspace.
**When** a plugin asks for the path `/account` of `dev.maroid.parking`.
**Then** the address is `https://deck.example/w/W/plugins/dev.maroid.parking/account`.

## `PARKACC-SC-021`

**Verifies:** `PARKACC-FR-022`
**Layer:** unit

**Given** a `pageLinker` with an empty `deck_url`, and a second one with a context that carries no acting workspace.
**When** a plugin asks each one for a page address.
**Then** each answers `ErrPageAddressAbsent`.

## `PARKACC-SC-022`

**Verifies:** `PARKACC-FR-023`
**Layer:** unit

**Given** the plugin.
**When** the test reads `Permissions` and `Routes`.
**Then** `account.connect` has the lowest role `viewer`, and each of the five routes names it.

## `PARKACC-SC-023`

**Verifies:** `PARKACC-FR-024`, `PARKACC-FR-025`
**Layer:** unit

**Given** the fake service answers the vehicles with one vehicle, id 584842.
**When** the member runs `/start` and chooses a lot and a type.
**Then** the fake service records two reads of the vehicles. The conversation enters
`confirm` with no vehicle question, and the start call carries `vehicleId: 584842`.

## `PARKACC-SC-024`

**Verifies:** `PARKACC-FR-026`, `PARKACC-FR-028`
**Layer:** unit

**Given** the fake service answers the vehicles with `Chevy (AB000EO)` id 1 and `Fiat (CD111EF)` id 2.
**When** the member chooses a type, then presses `Fiat (CD111EF)`, then confirms.
**Then** `select_vehicle` shows the two buttons and Cancel. The start call carries `vehicleId: 2`.
A callback `vehicle:3` keeps the step and starts nothing.

## `PARKACC-SC-025`

**Verifies:** `PARKACC-FR-027`
**Layer:** unit

**Given** the fake service answers the vehicles with an empty list.
**When** the member runs `/start`.
**Then** the bot refuses and names the reason. The conversation does not start.

## `PARKACC-SC-026`

**Verifies:** `PARKACC-INV-002`
**Layer:** unit

**Given** the fake `SecretProvider` holds `session` `TA` for user A only.
**When** user B runs `/balance`.
**Then** the bot answers "not connected". The fake service records no call with `TA`.

## `PARKACC-SC-027`

**Verifies:** `PARKACC-INV-003`
**Layer:** unit

**Given** every route of the connect flow runs to success and to each failure of section 4.5,
with the password `p4ss`, the code `0118`, the identifier `uuid-1`, and the token `T1`.
**When** the test reads every answer body and every log line.
**Then** none holds `p4ss`, `0118`, `uuid-1`, or `T1`.

## `PARKACC-SC-028`

**Verifies:** `PARKACC-FR-012`, `PARKACC-FR-015`, `PARKACC-FR-017`
**Layer:** manual

**Given** the owner's parking account and the hub with the plugin.
**When** the owner connects with a wrong password, then with the correct one, types a
wrong code, asks for a second code after 60 seconds, and types the second code.
**Then** the page reports each state of section 4.5. The owner records the status and the
body of each answer of the parking service in `PARKACC-DD-006` and `PARKACC-DD-007`.

## `PARKACC-SC-029`

**Verifies:** `PARKACC-FR-021`
**Layer:** unit

**Given** a call that fails because the parking service answered 503.
**When** the notice builds the answer of the bot.
**Then** the answer says that the parking service is unavailable. Any other failure gets
no answer of the notice and goes back to the engine.

## `PARKACC-SC-030`

**Verifies:** `PARKACC-FR-014`
**Layer:** unit

**Given** a `pending` secret, the fake service accepts the code with the header
`session-token: T1`, and answers the person check with 503.
**When** user A posts the code to `/pending-connections/self/verifications`.
**Then** the answer is 200 with `state: connected` and no `account`. The `session` secret holds `T1`.

## Retired identifiers

This file has no retired identifier.
