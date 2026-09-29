---
id: HEALTH
title: The health of the hub
type: requirements
status: approved
created: 2026-09-29
updated: 2026-09-29
approved_by: Temuri
approved_on: 2026-09-29
constrained_by: [API, ERR, RES, LOG, LIF, CFG]
---

# Requirements: The health of the hub

## 1. Problem

An orchestrator cannot tell whether a hub process runs or whether it can serve a
request. `APIFMT-FR-012` removed the route that answered a fixed word, and left
the design of its replacement to a later decision. This document is that decision.

Without a readiness report, the orchestrator sends traffic to a hub whose database
is down, and a request fails instead of reaching a second instance. Without a
liveness report, the orchestrator cannot restart a process that has stopped
answering.

## 2. Users

| Person                      | Needs                                                                |
| --------------------------- | -------------------------------------------------------------------- |
| An orchestrator             | One answer that says restart, and one that says send no traffic.     |
| A person who deploys Maroid | The name of the dependency that failed, without a search of the log. |

## 3. Out of scope

- The health of the worker process. The worker holds the connection to the MQTT
  broker, and the hub HTTP server does not use the broker.
- The health of a plugin. A plugin declares no check today.
- A history or a metric of past answers.

## 4. Definitions

`GLO-orchestrator`, `GLO-liveness`, `GLO-readiness`, `GLO-dependency`,
`GLO-secret-store`, and `GLO-drain-period` give the terms of this feature.

## 5. Functional requirements

### `HEALTH-FR-001`

The hub must answer a liveness request with a success while the process answers HTTP.

**Why:** A restart cures a process that answers nothing. It does not cure a
database that is down.

**Examples:**

- Normal case: every dependency answers. The liveness request succeeds.
- Limit case: the database is down. The liveness request still succeeds.
- Unwanted case: the orchestrator restarts the hub because the IdP is down.

### `HEALTH-FR-002`

The hub must read no dependency when it answers a liveness request.

**Why:** A liveness answer that reads a dependency restarts every instance at
the same time when that dependency fails.

### `HEALTH-FR-003`

The hub must answer a readiness request with a success when every dependency answers.

**Examples:**

- Normal case: the database, the IdP, and the secret store answer. The request succeeds.
- Unwanted case: the database answers, the IdP does not, and the request succeeds.

### `HEALTH-FR-004`

The hub must answer a readiness request with the status 503 when one dependency does not answer.

**Examples:**

- Normal case: the database refuses the connection. The answer is 503.
- Limit case: the IdP answers after the time limit of `HEALTH-NFR-001`. The answer is 503.

### `HEALTH-FR-005`

The hub must count the IdP as failed when its discovery document does not answer with the status 200.

**Why:** A proxy in front of the IdP answers 401, 403, or 404, and a sign in
fails in each case.

**Examples:**

- Normal case: the discovery document answers 200. The IdP counts as up.
- Unwanted case: the address of the IdP answers 404, and the IdP counts as up.

### `HEALTH-FR-006`

The hub must count the secret store as failed when the store is sealed or not initialized.

**Why:** A sealed store answers the network, and it protects no secret.

### `HEALTH-FR-007`

A failed readiness answer must be a problem that names each failed dependency.

**Why:** `ERR-001` gives every failure the shape of a problem. A person who
deploys Maroid reads the name and knows which service to fix.

**Examples:**

- Normal case: the secret store is sealed. The problem names the secret store.
- Limit case: the database and the IdP both fail. The problem names both.

### `HEALTH-FR-008`

An answer must not hold the error text of a dependency.

**Why:** The probe routes take no credential. An error text holds a host, a
port, or an address inside the deployment. The log holds the text for the person
who deploys Maroid.

**Examples:**

- Normal case: the database refuses the connection. The answer names the
  database. The log holds the refused address.
- Unwanted case: the answer holds the text `dial tcp 10.0.3.7:5432: connect: connection refused`.

### `HEALTH-FR-009`

The hub must answer a liveness request and a readiness request with no credential.

**Why:** An orchestrator holds no session and no token.

### `HEALTH-FR-010`

The hub must answer every readiness request with the status 503 from the start of a shutdown.

**Why:** The orchestrator stops the traffic before the listener closes, so no
request of a person reaches a closed socket.

**Examples:**

- Normal case: the hub receives the stop signal. The next readiness request answers 503.
- Limit case: every dependency answers during the shutdown. The answer is still 503.

### `HEALTH-FR-011`

The hub must serve every request during the drain period.

**Why:** The orchestrator reads the 503 of `HEALTH-FR-010` on its next probe.
A request that arrives before that probe must still land.

### `HEALTH-FR-012`

The hub must write no access record for a probe request that succeeds.

**Why:** A probe every 10 seconds from each orchestrator writes 8,640 records a
day that say nothing. A failed probe still writes its record.

## 6. Non-functional requirements

### `HEALTH-NFR-001`

The hub must answer a readiness request in less than 3 seconds when a dependency
does not answer. Measure from the arrival of the request at the hub to the last byte
of the answer.

**Why:** An orchestrator waits a fixed time for a probe. A probe that times out
counts as a failure with no name of the dependency.

### `HEALTH-NFR-002`

The hub must answer a liveness request in less than 100 milliseconds. Measure at
the hub, from the arrival of the request to the last byte of the answer.

### `HEALTH-NFR-003`

The drain period must be 5 seconds when the deployment configures none. The deployment
can configure it.

**Why:** Two probe periods of an orchestrator can pass before it reads the 503.
The value stays inside the shutdown timeout of `LIF-006`.

## 7. Invariants

### `HEALTH-INV-001`

The liveness answer depends on no dependency.

## 8. Constraints from the guidelines

| Rule      | Guideline        | Effect on this feature                                                        |
| --------- | ---------------- | ----------------------------------------------------------------------------- |
| `API-003` | HTTP API         | The two probe routes join the route table as public routes. The table changes directly. |
| `API-007` | HTTP API         | Every answer carries `X-Flow-ID` and `Cache-Control`.                          |
| `ERR-001` | Errors           | A failed readiness answer is a problem.                                       |
| `ERR-003` | Errors           | The type of the failed readiness answer takes a row in a table of types.      |
| `RES-002` | REST             | A probe route whose name departs from a Zalando rule takes a row.             |
| `RES-007` | REST             | `hub.yaml` describes the two probe routes.                                    |
| `LOG-008` | Logging          | The log of a failed dependency holds no secret of a connection string.        |
| `LOG-010` | Logging          | The rule takes the exception of `HEALTH-FR-012` directly.                     |
| `LIF-005` | Lifecycle        | The drain period comes before the HTTP server stops.                          |
| `LIF-006` | Lifecycle        | The drain period fits inside the 10 seconds of a shutdown step.               |
| `CFG-003` | Configuration    | The key of the drain period carries a default and a validation.               |

## Retired identifiers

This file has no retired identifier.
