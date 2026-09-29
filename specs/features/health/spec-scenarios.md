---
id: HEALTH
title: The scenarios of the health of the hub
type: spec
status: approved
created: 2026-09-29
updated: 2026-09-29
approved_by: Temuri
approved_on: 2026-09-29
constrained_by: [TST, TRC]
requirements: features/health/requirements.md
---

# Specification: The scenarios of the health of the hub

## `HEALTH-SC-001`

**Verifies:** `HEALTH-FR-001`
**Layer:** unit

**Given** a `Checker` whose readiness is `Unavailable` with `database` failed.
**When** a client sends `GET /livez`.
**Then** the answer is 200, and the body holds `"status": "OK"`.

## `HEALTH-SC-002`

**Verifies:** `HEALTH-FR-002`, `HEALTH-INV-001`
**Layer:** unit

**Given** a `Service` whose IdP is a test server that counts its requests.
**When** the test calls `Liveness` ten times.
**Then** each result is `OK`, and the test server counts no request.

## `HEALTH-SC-003`

**Verifies:** `HEALTH-FR-003`
**Layer:** unit

**Given** a `Checker` whose readiness is `OK` with the time `2026-09-29T14:30:00+04:00`.
**When** a client sends `GET /readyz`.
**Then** the answer is 200 with the media type `application/json`. The body holds
`"status": "OK"`, `"component": {"name": "maroid-hub", ...}`, and
`"timestamp": "2026-09-29T10:30:00Z"`.

## `HEALTH-SC-004`

**Verifies:** `HEALTH-FR-004`
**Layer:** unit

**Given** a `Checker` whose readiness is `Unavailable` with `database` failed.
**When** a client sends `GET /readyz`.
**Then** the answer is 503 with the media type `application/problem+json`.

## `HEALTH-SC-005`

**Verifies:** `HEALTH-FR-004`, `HEALTH-NFR-001`
**Layer:** unit

**Given** a `Service` whose IdP is a test server that answers after 10 seconds,
under `GOMAXPROCS=1`.
**When** the test calls `Readiness`.
**Then** the call returns in less than 3 seconds. The status is `Unavailable`, and
`Failures` holds `idp`.

## `HEALTH-SC-006`

**Verifies:** `HEALTH-FR-005`
**Layer:** unit

**Given** a test server as the IdP.
**When** the discovery document answers 200, then 404, then 503.
**Then** the check `idp` passes on 200, and fails on 404 and on 503.

## `HEALTH-SC-007`

**Verifies:** `HEALTH-FR-006`
**Layer:** unit

**Given** a test server that answers `sys/health` for an OpenBao client.
**When** the body holds `sealed: true`, then `initialized: false`, then neither.
**Then** the check `secret-store` fails twice, then passes.

## `HEALTH-SC-008`

**Verifies:** `HEALTH-FR-007`
**Layer:** unit

**Given** a `Checker` whose readiness is `Unavailable` with `idp` and `database` failed.
**When** a client sends `GET /readyz`.
**Then** the body holds `"type": "/problems/hub/not-ready"`, `"status": 503`, an
`instance`, and `"dependencies": ["database", "idp"]`.

## `HEALTH-SC-009`

**Verifies:** `HEALTH-FR-008`
**Layer:** unit

**Given** a `Checker` whose readiness fails `database` with the text
`dial tcp 10.0.3.7:5432: connect: connection refused`.
**When** a client sends `GET /readyz`.
**Then** no part of the answer holds `10.0.3.7`. One `WARN` record holds the message
`dependency check failed`, `dependency` set to `database`, `error` set to the
text, and the `flow_id` of the answer.

## `HEALTH-SC-010`

**Verifies:** `HEALTH-FR-009`
**Layer:** unit

**Given** the router of `server.NewHTTPRouter` with every handler of the hub.
**When** a client sends `GET /livez` and `GET /readyz` with no cookie and no header.
**Then** neither answer is 401.

## `HEALTH-SC-011`

**Verifies:** `HEALTH-FR-010`
**Layer:** unit

**Given** a `Service` whose three dependencies answer, and a call to `Drain`.
**When** a client sends `GET /readyz`.
**Then** the answer is 503 with `not-ready`, an empty `dependencies`, and the
`detail` of section 4.5. No dependency receives a request.

## `HEALTH-SC-012`

**Verifies:** `HEALTH-FR-011`, `HEALTH-NFR-003`
**Layer:** manual

**Given** `pnpm hub:serve` runs with no `server.drain_period`.
**When** the person sends SIGTERM, then `GET /readyz` and `GET /plugins` after 4 seconds.
**Then** `/readyz` answers 503, `/plugins` answers as before the signal, and the
process exits between 5 and 6 seconds after the signal.

## `HEALTH-SC-013`

**Verifies:** `HEALTH-FR-012`
**Layer:** unit

**Given** the router of `server.NewHTTPRouter` with a logger that records every entry.
**When** `GET /livez` answers 200, `GET /readyz` answers 503, and `GET /plugins` answers 200.
**Then** the log holds an access record for `/readyz` and for `/plugins`, and none for `/livez`.

## `HEALTH-SC-014`

**Verifies:** `HEALTH-NFR-002`
**Layer:** unit

**Given** a `Service` whose three dependencies do not answer.
**When** a client sends `GET /livez` 100 times.
**Then** each answer arrives in less than 100 milliseconds.

## `HEALTH-SC-015`

**Verifies:** `HEALTH-NFR-003`
**Layer:** unit

**Given** a configuration with no `server.drain_period`, and one with `11s`.
**When** the hub loads each one.
**Then** the first holds 5 seconds. The second fails the validation.

## `HEALTH-SC-016`

**Verifies:** `HEALTH-FR-007`
**Layer:** unit

**Given** `problem.NewValidationFailed(...)` with one failed field, and
`problems.NewNotReady(nil)`.
**When** `problem.Write` answers each one.
**Then** the first body holds `errors` with one item at the top level. The second
holds `"dependencies": []`. Neither body nests a member named `Problem`.

## `HEALTH-SC-018`

**Verifies:** `HEALTH-FR-008`
**Layer:** unit

**Given** a `Checker` whose readiness fails `idp` with the text
`Timeout during health check`, and `database` with the text
`pinging the database: connection refused`.
**When** a client sends `GET /readyz`.
**Then** the log holds two `WARN` records with the message `dependency check failed`:
one with `dependency` set to `idp` and `error` set to `Timeout during health check`,
and one for `database` with its text. The drain case of `HEALTH-SC-011` writes none.

## `HEALTH-SC-019`

**Verifies:** `HEALTH-FR-007`
**Layer:** unit

**Given** a `ValidationProblem` of the type `internal` with a `detail`, and a request
with a flow identifier.
**When** `problem.Write` answers it.
**Then** the body holds the `instance` of the flow, holds no `detail`, and holds
`errors`. The status is the status of the embedded `Problem`.

## Retired identifiers

| ID               | Retired    | Reason                                                              |
| ---------------- | ---------- | ------------------------------------------------------------------- |
| `HEALTH-SC-017`  | 2026-09-29 | `HEALTH-DD-010` embeds `Problem`, so no type encodes to a non-object. |
