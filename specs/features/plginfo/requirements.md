---
id: PLGINFO
title: The detail page of a plugin
type: requirements
status: approved
created: 2026-10-16
updated: 2026-10-16
approved_by: Temuri
approved_on: 2026-10-16
constrained_by: [UI, ARC, SEC, OWN]
---

# Requirements: The detail page of a plugin

## 1. Problem

The hub reports each capability of a plugin and each item it holds: the method,
the path, and the permission of a route, the schedule of a job, the text of a bot
command. The page that lists the plugins of a workspace shows one chip for each
capability, with a count. "Scheduled jobs: 2" tells a person that a plugin runs on
a schedule. It does not tell them when.

A person who decides whether to turn a plugin on, or who wonders why a plugin
sent a message at six in the morning, must read the report itself. No page in
the deck shows it.

## 2. Users

- A manager: reads what a plugin does before they turn it on in a workspace, and
  turns it on or off from the same page.
- A member: reads which commands, routes, and jobs a plugin in use holds, and the
  permission each one needs.
- The author of a plugin: confirms that the hub loaded each capability that the
  plugin declares.

## 3. Out of scope

- A fact that the report of a plugin does not carry. The page reads the report
  and adds nothing to it.
- A plain text reading of a schedule, and the time of the next run of a job.
- The run history of a job, and the traffic of a route.
- A detail page under the administration of the instance. The administrator
  reaches the page through a workspace.
- A call to a route, a command, or a tool from the page.

## 4. Definitions

`GLO-plugin-detail-page` defines the plugin detail page.

## 5. Functional requirements

### `PLGINFO-FR-001`

The deck must open the plugin detail page when a person selects a plugin on the
page that lists the plugins of a workspace.

**Examples:**

- Normal case: a member selects "Jasmine" under "In use". The deck opens the
  plugin detail page of Jasmine in that workspace.
- Normal case: a manager selects "Tbilisi Energy" under "To turn on". The deck
  opens its plugin detail page.

### `PLGINFO-FR-002`

The deck must show the plugin detail page to a person for a plugin that the
workspace enables, or for a plugin that the person may enable in the workspace.

**Why:** A manager decides whether to turn a plugin on. The page that helps them
decide must exist before the plugin is on.

**Examples:**

- Normal case: a viewer opens the page of a plugin that the workspace enables.
  The deck shows it.
- Normal case: a manager opens the page of a plugin on their plugin allowlist
  that the workspace does not enable. The deck shows it.
- Unwanted case: a viewer opens the address of a plugin that the workspace does
  not enable. The deck shows the not-found page.
- Unwanted case: a person opens the address of an identifier that no plugin
  holds. The deck shows the not-found page.

### `PLGINFO-FR-003`

The plugin detail page must show the name, the identifier, the version, and the
description of the plugin.

**Examples:**

- Normal case: Jasmine declares a description. The page shows all four.
- Limit case: Telasi declares no description. The page shows the other three,
  with no empty space for the description.

### `PLGINFO-FR-004`

The plugin detail page must show every capability that the report names for the
plugin, in the order that the page that lists the plugins gives them.

**Why:** A person who moves from the list to the page reads the same set in the
same order, and does not read a change that did not happen.

**Examples:**

- Normal case: a plugin declares settings, routes, and two jobs. The page shows
  the three capabilities.
- Limit case: a plugin declares no capability. The page states that the plugin
  declares none.

### `PLGINFO-FR-005`

The plugin detail page must show every item of a capability, with every field
that the report carries for the item, in the order of the report.

**Why:** The count on the list answers "how many". The fields answer "which" and
"when", and that is the question the person brings to this page.

**Examples:**

- Normal case: a route shows its method, its path, and its permission.
- Normal case: a job shows its identifier and its schedule.
- Normal case: a permission shows its name and the lowest workspace role that
  holds it.
- Limit case: the settings capability and the migrations capability hold no
  item. The page shows each one as present, with no empty list.

### `PLGINFO-FR-006`

The plugin detail page must show the schedule of a job and the path of a route
as the report states them.

**Why:** The expression is what the author wrote and what the hub runs. A second
form on the page is a second statement that can disagree with the first.

**Examples:**

- Normal case: a job reports `0 6 * * *`. The page shows `0 6 * * *`.
- Normal case: a route reports a path that holds `{workspaceId}`. The page shows
  `{workspaceId}`, not the identifier of the current workspace.

### `PLGINFO-FR-007`

The plugin detail page must state whether the workspace uses the plugin.

### `PLGINFO-FR-008`

The plugin detail page must let a person who may enable plugins in the workspace
turn the plugin on and off, with the outcome that the page that lists the plugins
gives.

**Examples:**

- Normal case: a manager turns a plugin on. The page states that the workspace
  uses it, and the list shows it under "In use".
- Unwanted case: the hub refuses the change. The page states the reason, and the
  plugin stays as it was.
- Unwanted case: a viewer opens the page. The page offers no switch.

### `PLGINFO-FR-009`

The plugin detail page must link to the settings of the plugin when the plugin
declares settings, the workspace uses the plugin, and the person reads the
settings of the workspace.

**Examples:**

- Normal case: a member opens the page of Jasmine in use. The page links to its
  settings.
- Unwanted case: an administrator opens the page in the workspace of another
  person. The page offers no link, as the list offers none.
- Unwanted case: the workspace does not use the plugin. The page offers no link.

### `PLGINFO-FR-010`

The plugin detail page must link to each navigation entry of the plugin when the
workspace uses the plugin.

**Examples:**

- Normal case: Jasmine declares four navigation entries. The page links to the
  four pages, by their labels.
- Unwanted case: the workspace does not use the plugin. The page shows the
  labels and no link.

### `PLGINFO-FR-011`

The plugin detail page must show the identifier of a plugin that the workspace
enables and the hub did not load, and must state that the hub did not load it.

**Why:** The person must still turn the plugin off. A not-found page hides the
plugin that still occupies the workspace.

**Examples:**

- Normal case: the workspace enables `dev.maroid.gwp`, and the hub started
  without it. The page shows the identifier, the state, and the switch.

### `PLGINFO-FR-012`

Maroid must refuse to load a plugin that declares a page route at the root of
the address of the plugin.

**Why:** The root of the address of a plugin in a workspace opens the plugin
detail page. A page route there would never open, and the author would learn it
from a person, not from the load.

**Examples:**

- Normal case: Jasmine declares `/plants` and `/environments`. Maroid loads it.
- Normal case: a plugin declares `/overview`. Maroid loads it, and the page opens
  below the root.
- Unwanted case: a plugin declares `/`. Maroid does not load it, and the error
  names the plugin and the page route.

## 6. Non-functional requirements

### `PLGINFO-NFR-001`

The plugin detail page asks the hub for no item of a capability alone. Measure at
the hub: the requests that one opening of the page causes do not grow with the
count of capabilities or items of the plugin.

**Why:** The report already carries every item. `PCAP-FR-004` gives the reason
that a request for each capability costs ten requests for one page.

## 7. Invariants

### `PLGINFO-INV-001`

The plugin detail page names no capability and no item that the report of the
plugin does not carry.

**Why:** The page describes what the hub loaded. A capability that the page
invents is a capability that a person expects and does not get.

## 8. Constraints from the guidelines

| Rule      | Guideline          | Effect on this feature                                                                        |
| --------- | ------------------ | --------------------------------------------------------------------------------------------- |
| `UI-001`  | Web user interface | The page lives in the deck.                                                                   |
| `UI-005`  | Web user interface | A plugin maps a page route to a page. `PLGINFO-FR-012` takes the root from that set.          |
| `ARC-008` | Architecture       | The page holds no rule that names one plugin. Every plugin passes one rendering.              |
| `SEC-012` | Security           | The workspace decides whether a plugin serves it. `PLGINFO-FR-007` shows that state.          |
| `SEC-013` | Security           | Each route, command, and tool names its permission. `PLGINFO-FR-005` shows it.                |
| `OWN-011` | Ownership          | A permission names the lowest workspace role that holds it. `PLGINFO-FR-005` shows that role. |

`PCAP-FR-009` already gives the name of a plugin on every page of the deck, the
trail of this page among them.

## 9. Open questions

| #   | Question                                                   | Owner  | Answer                                                                                                  |
| --- | ---------------------------------------------------------- | ------ | ------------------------------------------------------------------------------------------------------- |
| 1   | Where does the page live?                                  | Temuri | In a workspace only.                                                                                    |
| 2   | Does a plugin that the workspace does not use open a page? | Temuri | Yes, for a person who may enable it.                                                                    |
| 3   | Which actions does the page offer?                         | Temuri | The switch, the settings, the plugin pages.                                                             |
| 4   | Does the page explain a schedule in plain text?            | Temuri | No. It shows the report verbatim.                                                                       |
| 5   | Which address does the plugin detail page take?            | Temuri | The root of the address of the plugin. Every other page route stays open to a plugin. `PLGINFO-FR-012`. |

Answer every question before the approval. An open question blocks stage 2.

## Retired identifiers

This file has no retired identifier.
