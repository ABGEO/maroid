---
id: PLGBIND
title: The host services bound to one plugin
type: requirements
status: approved
created: 2026-10-15
updated: 2026-10-15
approved_by: Temuri
approved_on: 2026-10-15
constrained_by: [PLG, DAT, LOG, ARC, BLD, OWN]
---

# Requirements: The host services bound to one plugin

## 1. Problem

Every plugin receives the same host. A service of the host asks the caller for a plugin
identifier, and the plugin passes its own on each call: to the database, to the plugin
secrets, to the settings, to the notifications, and to the page addresses. A plugin that
passes a wrong identifier, through a copied line or a typing error, reads or writes the
schema, the plugin secrets, or the settings of a different plugin. Nothing in the hub
detects it.

The host also hands every plugin the database pool of the hub. `DAT-004` forbids a plugin
to use it, but no check enforces the rule.

## 2. Users

- A plugin author: receives each service already bound to the plugin, and passes no
  identifier.
- The owner: loads a plugin and knows that a mistake in it reaches its own data alone.
- A household member: keeps the data that one plugin holds out of every other plugin.

## 3. Out of scope

- A defense against a plugin that attacks the hub on purpose. A loaded plugin is
  trusted code. It shares the memory of the hub process.
- A database account for each plugin.
- A plugin that runs outside the hub process.
- A plugin that changes the acting user or the acting workspace inside its own data
  operation.
- A change to the Telegram bot or to the conversation engine that the host gives.
- A change to the shape of a data operation. A plugin still runs its statements inside
  one transaction.

## 4. Functional requirements

### `PLGBIND-FR-001`

The hub must read the declared metadata of a plugin before it builds the plugin.

**Why:** The hub binds the host to the plugin identifier, so the identifier must exist before the host.

### `PLGBIND-FR-002`

The hub must refuse to load a plugin that exports no declared metadata. The error names
the file of the plugin.

### `PLGBIND-FR-003`

The hub must refuse to build a plugin whose declared metadata breaks `PLG-003`,
`PLG-004`, or `PLG-012`. No code of the constructor runs.

**Why:** A constructor of a refused plugin can still change a state of the process.

**Examples:**

- Normal case: the declared metadata names `dev.maroid.gwp`, the name `GWP`, and the
  current API version. The hub builds the plugin.
- Unwanted case: the declared metadata names `Dev.Maroid.GWP`. The hub refuses it.
- Unwanted case: the declared metadata names a previous API version. The hub refuses it,
  and the constructor does not run.

### `PLGBIND-FR-004`

The hub must build one bound host for each plugin that it loads.

### `PLGBIND-FR-005`

The hub must use the declared metadata that it read at the load, for every capability
and for the life of the process.

**Why:** The plugin can change the value that it exports after the load.

**Examples:**

- Normal case: the migrations of `dev.maroid.gwp` run in the schema `dev_maroid_gwp`.
- Unwanted case: the plugin changes the identifier in its exported metadata after the
  load. Each registered capability keeps the identifier that the hub read.

### `PLGBIND-FR-006`

A data operation through a bound host must resolve an unqualified table name in the
schema of the bound plugin.

### `PLGBIND-FR-007`

A plugin secret that a plugin reads, writes, or deletes through a bound host must belong
to the bound plugin.

**Examples:**

- Normal case: `dev.maroid.gwp` writes `token` and reads it back.
- Unwanted case: `dev.maroid.telasi` reads `token` in the same scope. It gets an absent
  plugin secret, not the value of `dev.maroid.gwp`.

### `PLGBIND-FR-008`

A read of the settings through a bound host must return the settings of the bound
plugin.

### `PLGBIND-FR-009`

A notification that a plugin sends through a bound host must carry the bound plugin
as its sender. The hub checks the topic against the topics of that plugin.

**Examples:**

- Normal case: `dev.maroid.gwp` sends its topic `new-reading`, and the routes of that topic deliver it.
- Unwanted case: `dev.maroid.gwp` sends the topic `new-bill` of `dev.maroid.telasi`.
  The hub reports an undeclared topic, and delivers nothing.

### `PLGBIND-FR-010`

A page address that a plugin builds through a bound host must lead to a page of the
bound plugin.

### `PLGBIND-FR-011`

Every record that a plugin logs through a bound host must name the identifier, the
version, and the API version of the declared metadata.

**Why:** A record that the plugin names itself can name a different plugin.

### `PLGBIND-FR-012`

A bound host must offer no service that takes a plugin identifier from the plugin.

**Why:** A service that takes the identifier from the caller brings back the problem of section 1.

### `PLGBIND-FR-013`

A bound host must offer no access to the database other than the data operation of
`PLGBIND-FR-006`.

**Why:** The pool of the hub reaches every schema and every table of the hub.

## 5. Non-functional requirements

### `PLGBIND-NFR-001`

Each service of a bound host, except the transaction of a data operation, must take and
return values alone. A value holds no reference to an object that the hub owns.
Measured by review of the contract that `libs/pluginapi` declares.

**Why:** A later plugin that runs outside the hub process receives a copy of each value.
A service of this shape moves there with no change to its signature.

## 6. Invariants

### `PLGBIND-INV-001`

A bound host acts for one plugin identifier from its creation to the end of the process.
No call of its contract changes the identifier.

### `PLGBIND-INV-002`

No call through the bound host of one plugin reads or writes the schema, a plugin secret,
a setting, or a notification topic of a different plugin.

**Why:** This is the aim of the feature. It holds for a plugin that does not attack the
hub. Section 3 excludes an attack.

## 7. Constraints from the guidelines

| Rule      | Guideline    | Effect on this feature                                                            |
| --------- | ------------ | --------------------------------------------------------------------------------- |
| `PLG-002` | Plugin model | The plugin exports the declared metadata as a symbol.                             |
| `PLG-004` | Plugin model | `PLGBIND-FR-003` checks the API version before the constructor.                   |
| `PLG-012` | Plugin model | `PLGBIND-FR-003` checks the name and the description before the constructor.      |
| `PLG-013` | Plugin model | A loaded plugin is trusted. Section 3 excludes an attack.                         |
| `PLG-003` | Plugin model | The form that `PLGBIND-FR-003` checks.                                            |
| `PLG-007` | Plugin model | The host stays the only path from a plugin to a resource.                         |
| `PLG-011` | Plugin model | A second plugin with a loaded identifier stays refused.                           |
| `DAT-001` | Data         | The schema name comes from the declared metadata.                                 |
| `DAT-003` | Data         | The bound data operation keeps a plugin in its schema.                            |
| `DAT-004` | Data         | The host gives the bound database, and no pool.                                   |
| `LOG-004` | Logging      | The hub adds the three attributes of the plugin logger.                           |
| `ARC-008` | Architecture | The loader binds each plugin the same way, and names no plugin.                   |
| `BLD-004` | Build        | A change to `libs/pluginapi` rebuilds every plugin.                               |
| `OWN-005` | Ownership    | The data operation keeps the acting user and the acting workspace of the context. |

## 8. Open questions

This file has no open question.

## Retired identifiers

This file has no retired identifier.
