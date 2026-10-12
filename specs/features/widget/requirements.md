---
id: WIDGET
title: The widgets of the home page
type: requirements
status: approved
created: 2026-10-12
updated: 2026-10-12
approved_by: Temuri
approved_on: 2026-10-12
constrained_by: [PLG, ARC, SEC, API, UI, OWN]
---

# Requirements: The widgets of the home page

## 1. Problem

The home page of a workspace lists four links. A member who wants the balance of
their parking account opens the connection page. A member who wants the plants that
need care opens the attention list. Each value needs its own page, and the person
opens them one by one.

A plugin has no way to put its one important value in front of the person.

## 2. Users

- A member of a workspace: reads the current value of each plugin on the home page,
  and opens the page of the plugin from it.
- The author of a plugin: declares a widget and answers its data, with no code in the deck.

## 3. Out of scope

- A copy of the data of a widget that Maroid keeps between two reads.
- A widget that a plugin draws with its own component.
- A layout that a person arranges: hide, reorder, or choose a widget.
- A size that a person chooses.
- An action on a widget, for example "mark watered" or "start parking".
- A refresh on a timer, or when the browser tab regains the focus.
- A widget outside the deck, for example in a bot command or an MCP tool.
- A widget of any plugin other than parking and jasmine.
- One page that shows the widgets of several workspaces.

## 4. Definitions

| Term         | Meaning |
| ------------ | ------- |
| home page    | The page of a workspace that the deck opens first for that workspace. |
| widget       | One card on the home page. A plugin declares it and answers its data. |
| widget key   | The name of a widget, unique within its plugin. |
| widget kind  | The shape of the data of a widget: `stat` or `list`. |
| widget size  | The space of a widget on the home page: `xs`, `s`, `m`, or `l`. |
| widget state | What one answer of a widget says: `ready`, `empty`, or `not set up`. |

## 5. Functional requirements

### Declaration

### `WIDGET-FR-001`

A plugin must declare zero or more widgets.

**Why:** A plugin with several values, a bill and a reading for example, declares one widget for each.

**Examples:**

- Normal case: jasmine declares one widget, parking declares one.
- Limit case: a plugin with no page of its own declares a widget. Maroid shows it.

### `WIDGET-FR-002`

A widget must declare a key, a title, a kind, a size, and one permission of its plugin.

### `WIDGET-FR-003`

A widget key must hold 1 to 64 characters: lowercase letters, digits, and the hyphen.

**Why:** The key names the widget in an address.

### `WIDGET-FR-004`

A widget title must hold 1 to 64 characters after Maroid trims the white space at each end.

### `WIDGET-FR-005`

Maroid must refuse to load a plugin when one of its widgets breaks a rule of `WIDGET-FR-002`, `WIDGET-FR-003`, or `WIDGET-FR-004`, or repeats the key of another widget of the plugin.
The error must name the plugin, the widget key, and the rule.

**Why:** Every other declaration of a plugin fails at the load. The author reads the error at the first start.

**Examples:**

- Unwanted case: two widgets of one plugin carry the key `balance`. The plugin does not load.
- Unwanted case: a widget declares the size `xl`. The plugin does not load.
- Unwanted case: a widget names a permission that the plugin does not declare. The plugin does not load.
- Limit case: two plugins each declare a widget with the key `summary`. Both load.

### `WIDGET-FR-006`

Maroid must accept every pair of a widget kind and a widget size.

**Why:** The owner keeps the check of the size to the known values. A `list` at `xs` is legal.

### `WIDGET-FR-007`

The report of a loaded plugin must name each widget of the plugin with its key, title, kind, and size.

**Why:** The plugin list shows what each plugin can do, as `PCAP` gives. A widget is one more capability.

### Which widgets appear

### `WIDGET-FR-008`

The home page must show each widget of each plugin that the acting workspace enables.

**Examples:**

- Normal case: the workspace enables jasmine and parking. The home page shows two widgets.
- Unwanted case: the workspace disables parking. The balance widget is absent.

### `WIDGET-FR-009`

The home page must omit a widget whose permission the role of the person in the acting workspace does not hold.

**Examples:**

- Normal case: a viewer holds `plants.read` of jasmine. The plants widget shows.
- Unwanted case: a plugin declares a widget with a permission whose lowest role is editor. A viewer does not see it.

### `WIDGET-FR-010`

The home page must order the widgets by the order of their plugins in the navigation of the workspace, then by the order of the declaration within one plugin.

**Why:** Every person of a workspace sees the same order, and it does not change between two loads.

### `WIDGET-FR-037`

The Overview entry of the navigation must open the home page of the acting workspace, and show as active on it.

**Why:** The widgets make the home page the overview of a workspace. The entry pointed at the root of the deck, which redirects, so it never showed as active.

**Examples:**

- Normal case: the person opens the home page of workspace H. Overview shows as active.
- Limit case: the person belongs to no workspace. Overview opens the root of the deck.
- Unwanted case: the person opens a plugin page of H. Overview does not show as active.

### Reading a widget

### `WIDGET-FR-012`

Maroid must ask the plugin for the data of a widget at each load of the home page.

**Why:** The balance changes after each parking session. A kept copy shows a value that is no longer true.

### `WIDGET-FR-013`

The plugin must answer the data of a widget for the acting user in the acting workspace.

**Examples:**

- Normal case: two members each connect a parking account. Each member reads their own balance.
- Unwanted case: a member of workspace A never reads a plant of workspace B.

### `WIDGET-FR-014`

The home page must read each widget on its own, so a widget that fails or waits leaves every other widget shown.

**Examples:**

- Normal case: the parking service takes 4 seconds. The plants widget shows before it.
- Unwanted case: the parking service fails. The plants widget still shows.

### `WIDGET-FR-015`

A widget must show a placeholder of its size while its data loads.

**Why:** The page keeps its shape, so the content does not jump when the data arrives.

### `WIDGET-FR-016`

The plugin must answer one widget state with each answer.

### `WIDGET-FR-017`

A widget in the state `ready` must show the data of its kind.

### `WIDGET-FR-018`

A widget in the state `empty` must show the message that the plugin answers.

**Examples:**

- Normal case: no plant needs care. The widget shows "No plant needs care."

### `WIDGET-FR-019`

A widget in the state `not set up` must show the message that the plugin answers, and a link to a page of the plugin.

**Why:** The person learns what to do and reaches the page that does it in one click.

**Examples:**

- Normal case: the member has no parking connection. The widget shows "Connect your parking account." and a link to the connection page.

### `WIDGET-FR-020`

A widget must show an error, with its title, when the plugin fails to answer.

**Examples:**

- Normal case: the parking service answers status 503. The balance widget shows the error. The plants widget shows its data.
- Unwanted case: a widget shows an empty card with no reason.

### `WIDGET-FR-021`

A widget must show every text of the plugin as plain text.

**Examples:**

- Unwanted case: the title of a plant holds `<b>Fern</b>`. The widget shows the characters, not bold text.

### `WIDGET-FR-022`

A link of a widget must open a page of its own plugin, in the acting workspace.

**Why:** A widget sends the person into the plugin, never to another site.

**Examples:**

- Normal case: a link to `/plants/0199c5e2` opens the plant page of jasmine.
- Unwanted case: a plugin answers a link to `https://example.com`. The widget shows the error of `WIDGET-FR-020`.

### Kinds

### `WIDGET-FR-023`

A `stat` widget must show one value, and the unit, the caption, and the link to a page of its plugin that the plugin answers with it.

**Examples:**

- Normal case: "4.52" with the unit "GEL", the caption "4 free parkings left today", and a link to the connection page.
- Limit case: a value alone, with no unit, caption, or link.

### `WIDGET-FR-024`

A `list` widget must show its items in the order of the answer. Each item shows a label, and the meta line and the link to a page of its plugin that the plugin answers with it.

### `WIDGET-FR-025`

A `list` widget must show at most 2 items at a size of one row, and at most 5 items at a size of two rows.

### `WIDGET-FR-026`

A `list` widget with more items than it shows must show the count of the items it hides.

**Examples:**

- Normal case: 8 items at size `m`. The widget shows 5 items and "3 more".
- Limit case: 5 items at size `m`. The widget shows 5 items and no count.

### `WIDGET-FR-027`

A `list` widget must show the link to see all its items when the plugin answers one, whether or not the widget hides an item.

### Sizes

### `WIDGET-FR-028`

On a screen wide enough for four columns, a widget must take the space of its size:

| Size | Columns of four | Rows |
| ---- | --------------- | ---- |
| `xs` | 1               | 1    |
| `s`  | 2               | 1    |
| `m`  | 2               | 2    |
| `l`  | 4               | 2    |

**Why:** The owner chose four presets. A plugin picks one, and every widget of a size has the same space.

### `WIDGET-FR-029`

On a screen narrower than two columns, every widget must take the full width and keep its rows.

### The first widgets

### `WIDGET-FR-030`

Parking must declare a `stat` widget of size `s`, under the permission that reads the balance.
It shows the balance of the parking account of the acting user, with its currency, the count of the free parkings left today, and a link to the connection page.

### `WIDGET-FR-031`

The parking widget must answer `not set up` when the acting user has no connection, with a link to the connection page.

### `WIDGET-FR-032`

Jasmine must declare a `list` widget of size `m`, under the permission that reads the plants.
It shows each care interval of the attention list, in the order of the attention list.

### `WIDGET-FR-033`

Each item of the jasmine widget must show the plant, the maintenance type, the days since the latest entry, and the care state.

### `WIDGET-FR-034`

Each item of the jasmine widget must link to the plant page of its plant.

### `WIDGET-FR-035`

The jasmine widget must link to the attention list to see all its items.

### `WIDGET-FR-036`

The jasmine widget must answer `empty` when no care interval is due or overdue.

### The navigation of a workspace

### `WIDGET-FR-038`

The home page must show no link to Members, Plugins, Notifications, or Settings of the workspace.

**Why:** A person opens the home page for the widgets. The navigation holds Members, Plugins, Notifications, and Settings.

### `WIDGET-FR-039`

The home page must show an empty state with a link to the plugins page of the workspace when no widget applies.

**Examples:**

- Normal case: the workspace enables no plugin with a widget. The page shows "No plugin of this workspace shows a widget yet." and a link to the plugins page.
- Unwanted case: the role of the person holds no permission of any widget. The page shows the same empty state, not an empty grid.

### `WIDGET-FR-040`

The navigation must hold a Workspace group with the entries Members, Plugins, Notifications, and Settings, each opening its page of the acting workspace.

### `WIDGET-FR-041`

The Workspace group must be closed unless one of its pages is open, and the entry of the open page must show as active.

**Examples:**

- Normal case: the person opens Members. The group opens, and Members shows as active.
- Limit case: the person opens the notification settings. The group opens, and Notifications shows as active, not Settings.
- Unwanted case: the person opens the home page. The group is closed.

### `WIDGET-FR-042`

The Plugins group of the navigation must list only the navigation entries of the enabled plugins, and must be absent when no enabled plugin holds one.

**Why:** Workspace > Plugins enables and disables a plugin. A second entry to the same page repeats it.

## 6. Non-functional requirements

### `WIDGET-NFR-001`

A widget whose plugin has not answered 5 seconds after the hub asks it must show the error of `WIDGET-FR-020`. The hub measures the time from its call to the plugin.

**Why:** A slow external service must not hold a card in its placeholder with no end.

## 7. Invariants

### `WIDGET-INV-001`

A widget shows only records of the acting workspace.

### `WIDGET-INV-002`

Maroid keeps no answer of a widget after it sends it.

**Why:** Out of scope. A later feature adds a copy with a time to live.

## 8. Constraints from the guidelines

| Rule      | Guideline    | Effect on this feature |
| --------- | ------------ | ---------------------- |
| `PLG-005`, `PLG-006` | Plugin model | A widget is a capability: an interface, a registry, a registrar. |
| `PLG-011` | Plugin model | A second widget with one key is refused. |
| `ARC-008` | Architecture | The hub treats every widget alike and names no plugin. |
| `SEC-012` | Security     | A plugin disabled in the workspace shows no widget. |
| `SEC-013` | Security     | A widget acts in a workspace, so it declares one permission. |
| `OWN-005` | Ownership    | The plugin reads through the policy, which limits the rows to the workspace. |
| `API-003` | HTTP API     | The hub adds routes under `/workspaces*`. |
| `UI-001`  | Web UI       | The deck draws every widget. A plugin serves no part of the home page. |

## 9. Open questions

| #   | Question | Owner | Answer |
| --- | -------- | ----- | ------ |
| 1   | Is a cache in scope? | Temuri | No. Each load asks the plugin. |
| 2   | How long does the hub wait for a plugin? | Temuri | 5 seconds, the same for every widget. |
| 3   | Does the deck read a widget again on focus? | Temuri | No. Once, when the page opens. |
| 4   | Does the registrar check a pair of kind and size? | Temuri | No. Only that the size is known. |

## Retired identifiers

| ID              | Retired    | Reason |
| --------------- | ---------- | ------ |
| WIDGET-FR-011   | 2026-10-12 | The links of the home page move to the Workspace group of the navigation. `WIDGET-FR-038` to `WIDGET-FR-040` replace it. |
