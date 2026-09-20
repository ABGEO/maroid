---
id: MCPHUB
title: The hub as a Model Context Protocol server
type: requirements
status: approved
created: 2026-09-17
updated: 2026-09-18
approved_by: Temuri
approved_on: 2026-09-18
constrained_by: [SEC, OWN, API, ARC, PLG, LOG, DAT]
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

The second iteration adds the plugins. The hub answers three tools of its own,
and every one of them reports about the hub. A plugin holds the plants, the
parking, and the bills, and an agent reaches none of it. The owner still reads
each of those in the deck or in a bot reply, and types the answer back to the
agent by hand.

## 2. Users

- The owner: connects an agent of their choice to Maroid, and asks it to read
  or to change something that only their own account can reach.
- A Maroid user who is not the owner: connects their own agent the same way,
  and reaches only their own data.
- The author of a plugin: declares the functions of that plugin once, and they
  reach every agent that any Maroid user connects.

## 3. Out of scope

- The exact identifier and the redirect rule of the IdP client that an MCP
  client authenticates against. Provisioning it is a deployment concern.
- A rate limit or a quota on an MCP client.
- Revoking a token before its expiry.
- A tool call that streams more than one response. Every tool call answers with
  one JSON response.
- A tool that a plugin declares after the hub loaded that plugin. The set of the
  tools of a plugin is fixed at the load.
- A tool list that differs by acting user. Every acting user reads one list.
- A prompt and a resource of the Model Context Protocol. This feature exposes a
  tool and nothing else.
- A confirmation step before a tool changes data. The MCP client of the person
  owns that step.

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

### `MCPHUB-FR-007`

The hub must expose a tool that a plugin declares, beside the tools that the
hub itself declares.

**Why:** A plugin holds the knowledge of its own domain. An agent reaches it
only when the plugin offers its own function.

**Examples:**

- Normal case: a loaded plugin declares two tools, and an MCP client reads all
  of them in the tool list.
- Limit case: no loaded plugin declares a tool. The list holds the tools of the
  hub, and no error.
- Unwanted case: a plugin declares a tool that the hub cannot read. The hub
  refuses to load that plugin, and it names the plugin in the failure.

### `MCPHUB-FR-008`

The hub must give the tool of one plugin a name that the tool of another plugin
cannot take.

**Why:** An MCP client calls a tool by name, in one flat list. Two plugins that
pick one word would otherwise block each other, and a person would reach the
wrong plugin with no sign of it.

**Examples:**

- Normal case: two plugins each declare a tool named "list", and an MCP client
  reaches both.
- Limit case: one plugin declares two tools under one name. The hub refuses to
  load that plugin.
- Unwanted case: a plugin declares a name that a tool of the hub already holds.
  The hub refuses to load that plugin.

### `MCPHUB-FR-009`

The hub must name the plugin that declares a tool, in the report that an MCP
client reads.

**Why:** An agent reports to a person what it did. That person needs to know
which part of Maroid answered, and where to change it.

### `MCPHUB-FR-010`

The hub must give the acting user of the call to the tool of a plugin.

**Why:** A tool that reaches a scoped table with no acting user reads no row.
An agent then reports an empty result, which reads like an answer and is not
one. See `OWN-003`, `OWN-006`.

**Examples:**

- Normal case: two people each call the same tool of the same plugin, and each
  reads only their own rows.
- Unwanted case: the tool reaches a scoped table with no acting user. The hub
  does not reach this state, because `MCPHUB-INV-001` holds at the entry point.

### `MCPHUB-FR-011`

The hub must let a tool of a plugin change data that belongs to the acting user.

**Why:** The owner connects an agent to act, not only to read.

**Examples:**

- Normal case: an agent calls a tool that records an action, and the row names
  the acting user.
- Limit case: the change names a row of another person. It changes no row.
- Unwanted case: a tool changes data and names no acting user. The change
  reaches no row.

### `MCPHUB-FR-012`

The hub must report whether a tool changes data.

**Why:** An agent asks a person before it changes something. It needs to know
which call changes something to ask at the right moment.

**Examples:**

- Normal case: a tool that only reports carries the mark that it changes
  nothing, and an agent calls it with no question.
- Unwanted case: a plugin marks a tool that changes data as one that changes
  nothing. The hub reports the mark that the plugin declared, and verifies
  nothing.

### `MCPHUB-FR-013`

The hub must answer a tool call with a failure that names the plugin and the
place to fill its settings, when the acting user holds no complete settings for
that plugin.

**Why:** A plugin that reads an external account cannot act before that person
fills its settings. A bare failure tells the person nothing they can act on.

**Examples:**

- Normal case: the acting user filled every required field, and the call runs.
- Unwanted case: the acting user filled none of them. The failure names the
  plugin and the place to fill them.
- Limit case: the plugin declares no settings at all. The call runs, and this
  failure never happens.

### `MCPHUB-FR-014`

The hub must list a tool of a plugin whether or not the acting user filled the
settings of that plugin.

**Why:** One tool list serves every person. A list that changes by person costs
a read of every setting on every listing, and an agent that keeps the list holds
a stale one.

**Examples:**

- Normal case: two people read the tool list, and both read the same tools.
- Limit case: the acting user filled no setting of any plugin. The list still
  holds every tool, and `MCPHUB-FR-013` answers the call.

## 5. Non-functional requirements

### `MCPHUB-NFR-001`

Over 1000 consecutive tool calls that carry a valid token, no more than one
waits for the IdP. Measure at the hub.

**Why:** A verification that calls the IdP on every tool call makes the IdP a
dependency of every read. See `EXTID-NFR-001`.

### `MCPHUB-NFR-002`

A report of the tool list reads no user record and no setting of a plugin.
Measure at the hub: zero database statements while the hub answers a request
for the tool list.

**Why:** `MCPHUB-FR-014` gives one list to every person. A read for each plugin
on each listing would buy nothing, and an agent lists the tools on every new
connection.

## 6. Invariants

### `MCPHUB-INV-001`

One MCP tool call resolves to exactly one acting user.

### `MCPHUB-INV-002`

Two loaded plugins never expose one tool name.

## 7. Constraints from the guidelines

| Rule      | Guideline        | Effect on this feature                                                                                  |
| --------- | ----------------- | --------------------------------------------------------------------------------------------------------- |
| `SEC-002` | Security          | A verified token's audience names the MCP client's own identifier of the IdP, distinct from the hub's. `ADR-0003` widens the rule. |
| `SEC-004` | Security          | `MCPHUB-FR-003` enforces it at the new entry point.                                                     |
| `OWN-003` | Record ownership  | An MCP tool call becomes a third entry point that resolves an acting user. `ADR-0003` adds the row.     |
| `API-003` | HTTP API          | The MCP route and the discovery route take a place in the fixed list of prefixes.                       |
| `API-006` | HTTP API          | A response body is JSON. This feature answers a tool call with one JSON response, and needs no exception to the rule. |
| `ARC-008` | Architecture      | The plugin list tool reports every loaded plugin through the existing registry. It names no plugin by hand. |
| `PLG-006` | Plugin model      | This iteration adds the interface and the registrar. The registry already exists.                       |
| `PLG-007` | Plugin model      | A plugin declares a tool through the plugin interface, and imports no package of the hub.                |
| `PLG-011` | Plugin model      | `MCPHUB-INV-002` is the outcome. A second tool under one name is refused at the load.                    |
| `OWN-005` | Record ownership  | A change that a tool makes names no user. The policy sets it.                                           |
| `OWN-006` | Record ownership  | A tool that reaches a scoped table with no acting user reads no row. `MCPHUB-FR-010` prevents that state. |
| `OWN-007` | Record ownership  | `MCPHUB-FR-010` carries the acting user into the transaction of the plugin.                             |
| `DAT-004` | Data              | A plugin reaches the database on one path. A tool adds no other.                                        |
| `LOG-003` | Logging           | A log line of the MCP server carries its own component attribute.                                       |

`ADR-0003` changes `SEC-002` and `OWN-003`. The owner accepted that decision on
2026-09-17.

## 8. Open questions

| #   | Question                                                                                                          | Owner  | Answer |
| --- | --------------------------------------------------------------------------------------------------------------- | ------ | ------ |
| 1   | Does a tool call ever stream more than one response (progress, a partial result), or does every call answer with one JSON response? Streaming needs an exception to `API-006`. | Temuri | No streaming this iteration. Every tool call answers with one JSON response. |
| 2   | Does the plugin list tool report whether a plugin declares settings, the way `GET /plugins` does, or only the identifier and the version that `MCPHUB-FR-005` gives? | Temuri | The same shape `GET /plugins` gives: the identifier, the version, the settings flag, and the user interface manifest. |
| 3   | May a tool of a plugin change data, or does this iteration keep every tool a report? | Temuri | A tool may change data. `MCPHUB-FR-011` and `MCPHUB-FR-012` state it. |
| 4   | Does a tool of a plugin disappear from the list when the acting user filled no setting for that plugin, or does it stay and fail at the call? | Temuri | It stays. `MCPHUB-FR-013` and `MCPHUB-FR-014` state it. |

Answer every question before the approval. An open question blocks stage 2.

## Retired identifiers

This file has no retired identifier.
