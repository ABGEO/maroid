---
id: PCAP
title: The capabilities that a plugin declares
type: requirements
status: approved
created: 2026-09-21
updated: 2026-09-21
approved_by: Temuri
approved_on: 2026-09-21
constrained_by: [PLG, ARC, API, UI]
---

# Requirements: The capabilities that a plugin declares

## 1. Problem

A plugin adds a capability to the hub, and the hub loads ten kinds of them
today. The report of a loaded plugin names two: it carries a flag for the
settings and the manifest of the user interface. The other eight reach nobody.

A person who opens the plugin list reads a name, a version, and one chip. They
cannot see that a plugin answers a bot command, serves a route, runs on a
schedule, or offers a function to an agent. They install a plugin and then guess
what it does.

An agent reads the same report and has the same gap. It learns that a plugin is
loaded and cannot tell what the plugin is for.

The report also states each of the two facts its own way, one as a flag and one
as a nested manifest. A ninth capability would add a ninth member with a ninth
shape, and every client would learn a ninth rule.

## 2. Users

- The owner: opens the plugin list and reads what each loaded plugin can do,
  before they configure it.
- An agent that acts for a person: reads the same report and learns which plugin
  answers which kind of request, and which commands and routes it holds.
- The author of a plugin: adds a capability to a plugin, and it appears in the
  report with no change to the hub.

## 3. Out of scope

- A capability that a plugin gains after the hub loaded it. The set is fixed at
  the load.
- A capability of the hub itself. This report describes a plugin.
- The fields of the settings of a plugin. The settings capability names itself,
  and the surface that already serves the schema holds the fields.
- The content of a migration. The migrations capability names itself.
- A permission or a scope on one capability. Every capability of a loaded plugin
  reaches every person who may read the plugin list.
- Turning a capability off for one plugin from the configuration.

## 4. Functional requirements

### `PCAP-FR-001`

The hub must report every capability that a loaded plugin declares.

**Why:** A client decides what to offer for a plugin. Today it detects two of
the ten, so it offers nothing for the other eight.

**Examples:**

- Normal case: a plugin declares settings, a bot command, and a function for an
  agent. The report names all three.
- Limit case: a plugin declares no capability. The report holds an empty set of
  capabilities, not an absent member.
- Unwanted case: a plugin declares a capability that this hub does not load. The
  report does not name it.

### `PCAP-FR-002`

The hub must report the capabilities of a plugin in one shape, to the browser of
a person and to an agent.

**Why:** Two shapes for one fact drift apart, and the person and the agent then
read different answers about one plugin.

### `PCAP-FR-003`

The hub must name each capability with an identifier that survives a release.

**Why:** A client keys its behavior off the name. A name that changes breaks that
client with no failure that anyone sees.

**Examples:**

- Normal case: the identifier of the capability that serves a user interface
  reads the same before and after a release that changes nothing else.
- Unwanted case: the identifier follows the internal name of a component of the
  hub, and a rename of that component silently changes the report.

### `PCAP-FR-004`

The hub must report the items that a capability holds, beside the name of that
capability.

**Why:** The name answers what a plugin can do. The items answer what a person
or an agent can call, and a client that must make a second request for each
capability makes ten requests to render one page.

**Examples:**

- Normal case: a plugin serves four routes. The report names the capability and
  the method and the path of each route.
- Normal case: a plugin answers two bot commands. The report names the
  capability, and each command with the text that describes it.
- Limit case: a plugin declares settings. The report names the capability and no
  item, because the surface that serves the schema holds the fields.
- Unwanted case: the report names a capability and omits items that the plugin
  holds. A client then shows a capability that looks empty.

### `PCAP-FR-005`

The hub must report the items of one capability in one order, for one set of
items.

**Why:** A person reads the same list in the same place each time, and a client
compares one report with the next. An order that changes between two requests
reads as a change to the plugin.

**Examples:**

- Normal case: two requests for one loaded plugin give its routes in one order.
- Unwanted case: the order follows the order that the hub happened to load
  things in, and the list rearranges itself between two requests.

### `PCAP-FR-006`

The deck must show every capability of a plugin, on the page that lists the
plugins.

**Why:** The person who decides whether to configure a plugin is looking at that
page. A capability that the report carries and the page hides helps nobody.

**Examples:**

- Normal case: a plugin with four capabilities shows all four.
- Limit case: a plugin with no capability shows the plugin and no capability,
  with no empty frame around nothing.

### `PCAP-FR-007`

The hub must let a client tell a capability that holds no item from a capability
that the plugin does not declare.

**Why:** The settings capability and the migrations capability hold no item, and
they are the two that a client acts on most. A report that states "present with
nothing inside" the same way it states "absent" makes a client hide the
configure action of every plugin that has one.

**Examples:**

- Normal case: a plugin declares settings and no bot command. A client reads the
  settings capability as present and the bot command capability as absent.
- Normal case: a plugin declares migrations. A client reads the capability as
  present, and asks for no item.
- Unwanted case: a capability that holds no item reads as an empty answer, and a
  client that tests the capability for a value treats it as absent.

## 5. Non-functional requirements

### `PCAP-NFR-001`

Reporting the capabilities of every loaded plugin reads no database. Measure at
the hub: zero database statements while the hub answers one request for the
plugin list.

**Why:** The deck reads this list on every visit to the page, and an agent reads
it on every new connection. `MCPHUB-NFR-002` holds the tool list to the same
limit, and this report travels beside it.

## 6. Invariants

### `PCAP-INV-001`

One fact about a plugin appears in one place in the report.

### `PCAP-INV-002`

The capabilities that the report names for a plugin are the capabilities that
the hub loaded for that plugin, and no other.

## 7. Constraints from the guidelines

| Rule      | Guideline          | Effect on this feature                                                                                   |
| --------- | ------------------ | ---------------------------------------------------------------------------------------------------------- |
| `PLG-006` | Plugin model       | A capability is an interface in the plugin contract. The report names the capabilities that rule creates. |
| `ARC-008` | Architecture       | The report holds no rule that names one plugin. Every plugin passes one detection.                        |
| `API-004` | HTTP API           | The route of a plugin carries the prefix that the hub gives it. The report states the path a client calls. |
| `API-006` | HTTP API           | The report travels as JSON.                                                                               |
| `UI-001`  | Web user interface | `apps/deck` is the only web shell, so `PCAP-FR-006` lands there.                                          |

`MCPHUB-FR-005` already gives an agent the report of a loaded plugin. The
capabilities travel in that report, so this feature adds no statement for the
agent beyond `PCAP-FR-002`.

## 8. Open questions

| #   | Question                                                                          | Owner  | Answer                                                                                      |
| --- | --------------------------------------------------------------------------------- | ------ | --------------------------------------------------------------------------------------------- |
| 1   | Does the report keep a separate flag for the settings beside the capabilities?     | Temuri | No. The capability names itself, and the flag goes.                                          |
| 2   | Does a capability carry its items?                                                | Temuri | Yes. Every capability that holds items reports them. |
| 4   | How does the report state the settings capability and the migrations capability, which hold no item? | Temuri | As present. `PCAP-FR-007` states the behavior, and a client must not read them as absent. |
| 3   | Does the report hold a list of names beside the data, or one member for both?      | Temuri | One member. A capability names itself and carries its items in one place, so the two cannot disagree. |

Answer every question before the approval. An open question blocks stage 2.

## Retired identifiers

This file has no retired identifier.
