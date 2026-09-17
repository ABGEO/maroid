---
id: MCPHUB
title: The hub as a Model Context Protocol server
type: requirements
status: approved
created: 2026-09-17
updated: 2026-09-17
approved_by: Temuri
approved_on: 2026-09-17
constrained_by: [SEC, OWN, API, ARC, PLG, LOG]
---

# Requirements: The hub as a Model Context Protocol server

## 1. Problem

An agent that acts for the owner reads Maroid through no direct path today. The
owner copies a fact from the deck or from a bot reply by hand. The agent
changes nothing in Maroid on its own, and it holds no way to prove which
person it acts for.

The hub holds no route that lets an agent authenticate as the owner and call
one function of the hub over the Model Context Protocol. No plugin and no hub
function reaches an agent this way.

## 2. Users

- The owner: connects an agent of their choice to Maroid, and asks it to read
  or to change something that only their own account can reach.
- A Maroid user who is not the owner: connects their own agent the same way,
  and reaches only their own data.

## 3. Out of scope

- A plugin that exposes its own tool over the Model Context Protocol. A future
  iteration adds the capability, the registry, and the registrar that
  `PLG-006` demands.
- The exact identifier and the redirect rule of the IdP client that an MCP
  client authenticates against. Provisioning it is a deployment concern.
- A tool that changes data. Every tool of this iteration reads and reports.
- A rate limit or a quota on an MCP client.
- Revoking a token before its expiry.
- A tool call that streams more than one response. Every tool call of this
  iteration answers with one JSON response.

## 4. Functional requirements

### `MCPHUB-FR-001`

The hub must let an MCP client discover the authorization server of the hub
before that client holds a token.

**Why:** An MCP client holds no token on its first connection, and needs a
path to the IdP that names no client secret.

**Examples:**

- Normal case: an MCP client with no token asks the hub where to sign in, and
  the hub names the IdP.
- Unwanted case: an MCP client calls a tool with no discovery step first, and
  reaches the same rejection that `MCPHUB-FR-002` gives.

### `MCPHUB-FR-002`

The hub must reject an MCP tool call that carries no valid token from the IdP.

**Why:** `SEC-002` verifies a token before the hub does anything else with it.

**Examples:**

- Normal case: an MCP client presents a token that the IdP issued for it, and
  the call proceeds.
- Limit case: the token expired one second before the call. The hub rejects
  it.
- Unwanted case: the call carries no token. The hub rejects it and names where
  to get one.

### `MCPHUB-FR-003`

The hub must resolve an MCP tool call to the user record of the identity
behind the token. The hub must reject the call when that identity names no
active user record.

**Why:** A tool that reaches a scoped table needs one acting user, and the
user record is the allowlist for every entry point. See `OWN-003`, `SEC-004`.

**Examples:**

- Normal case: the identity behind the token names an active user record. The
  hub resolves the call to that user, and the tool runs.
- Unwanted case: a person authenticates through the IdP with an account that
  holds no identity in Maroid. The hub rejects the call.
- Limit case: the identity names a blocked user record. The hub rejects the
  call.

### `MCPHUB-FR-004`

The hub must report the name and the provider of the acting user when an MCP
client calls the identity tool.

**Why:** An agent that acts for a person needs to confirm which person it acts
for.

**Examples:**

- Normal case: an MCP client calls the identity tool, and reads the name and
  the provider that signed the acting user in.
- Limit case: the acting user attached more than one provider. The report
  names the provider of the current sign in, not every one.

### `MCPHUB-FR-005`

The hub must report the identifier, the version, whether the plugin declares a
settings schema, and the user interface manifest of each loaded plugin, when
an MCP client calls the plugin list tool.

**Why:** An agent that helps configure or diagnose Maroid needs to know which
plugin runs. The report matches the one report of a loaded plugin that Maroid
already gives, so the same fact does not gain a second shape.

**Examples:**

- Normal case: three plugins are loaded, and the report names all three, with
  the settings flag and the user interface manifest of each.
- Limit case: no plugin is loaded. The report is an empty list, not an error.
- Limit case: a plugin declares no user interface. The report names the
  plugin, with no manifest.

### `MCPHUB-FR-006`

The hub must confirm that it is reachable when an MCP client calls the
connectivity tool.

**Why:** An MCP client tests a new connection before it trusts one.

## 5. Non-functional requirements

### `MCPHUB-NFR-001`

Over 1000 consecutive tool calls that carry a valid token, no more than one
waits for the IdP. Measure at the hub.

**Why:** A verification that calls the IdP on every tool call makes the IdP a
dependency of every read. See `EXTID-NFR-001`.

## 6. Invariants

### `MCPHUB-INV-001`

One MCP tool call resolves to exactly one acting user.

## 7. Constraints from the guidelines

| Rule      | Guideline        | Effect on this feature                                                                                  |
| --------- | ----------------- | --------------------------------------------------------------------------------------------------------- |
| `SEC-002` | Security          | A verified token's audience names the MCP client's own identifier of the IdP, distinct from the hub's. `ADR-0003` widens the rule. |
| `SEC-004` | Security          | `MCPHUB-FR-003` enforces it at the new entry point.                                                     |
| `OWN-003` | Record ownership  | An MCP tool call becomes a third entry point that resolves an acting user. `ADR-0003` adds the row.     |
| `API-003` | HTTP API          | The MCP route and the discovery route take a place in the fixed list of prefixes.                       |
| `API-006` | HTTP API          | A response body is JSON. This feature answers a tool call with one JSON response, and needs no exception to the rule. |
| `ARC-008` | Architecture      | The plugin list tool reports every loaded plugin through the existing registry. It names no plugin by hand. |
| `PLG-006` | Plugin model      | A plugin's own tool needs the interface, the registry, and the registrar this rule demands. Out of scope for this iteration. |
| `LOG-003` | Logging           | A log line of the MCP server carries its own component attribute.                                       |

`ADR-0003` changes `SEC-002` and `OWN-003`. The owner accepted that decision on
2026-09-17.

## 8. Open questions

| #   | Question                                                                                                          | Owner  | Answer |
| --- | --------------------------------------------------------------------------------------------------------------- | ------ | ------ |
| 1   | Does a tool call ever stream more than one response (progress, a partial result), or does every call answer with one JSON response? Streaming needs an exception to `API-006`. | Temuri | No streaming this iteration. Every tool call answers with one JSON response. |
| 2   | Does the plugin list tool report whether a plugin declares settings, the way `GET /plugins` does, or only the identifier and the version that `MCPHUB-FR-005` gives? | Temuri | The same shape `GET /plugins` gives: the identifier, the version, the settings flag, and the user interface manifest. |

Answer every question before the approval. An open question blocks stage 2.

## Retired identifiers

This file has no retired identifier.
