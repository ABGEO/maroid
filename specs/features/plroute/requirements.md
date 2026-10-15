---
id: PLROUTE
title: The pages of a plugin, with the parameters of their address
type: requirements
status: approved
created: 2026-10-11
updated: 2026-10-11
approved_by: Temuri
approved_on: 2026-10-11
constrained_by: [UI, ARC, TS]
---

# Requirements: The pages of a plugin, with the parameters of their address

## 1. Problem

A plugin page has a fixed address. A page of one record, one plant for example, has
no address of its own, so the plugin shows it inside the page of the list:

- A reload, or the back button of the browser, leaves the record and shows the list.
- A person cannot keep or share a link to one record.
- The trail of the deck ends at the list, so the page needs its own control to go back.

## 2. Users

| Person                  | Needs                                                                    |
| ----------------------- | ------------------------------------------------------------------------ |
| A member of a workspace | An address for each record page, which a reload and a link keep.         |
| The author of a plugin  | To declare a page whose address carries a value, and to read that value. |

## 3. Out of scope

- The query of an address. A page reads the path only.
- A parameter that spans several segments, an optional parameter, and a parameter
  with a type or a pattern.
- The title of the browser tab.
- A parameter in an address of the deck itself.

## 4. Definitions

| Term              | Meaning                                                                                      |
| ----------------- | -------------------------------------------------------------------------------------------- |
| page route        | A path that a remote declares, and the page that the deck opens for it.                      |
| parameter segment | A segment of a page route that takes any one value, and gives the page that value by a name. |
| navigation entry  | A page route that the deck lists under the plugin in the navigation of the workspace.        |
| trail             | The line of links above a page that names each level from the hub to the page.               |
| page title        | The name that a page gives itself for the last link of the trail.                            |

## 5. Functional requirements

### Matching

### `PLROUTE-FR-001`

A plugin must declare a page route that holds one or more parameter segments.

**Examples:**

- Normal case: `/plants/:id` opens for `/plants/0199c5e2-8f3a-7c21-9b4e-2f6a1d0c5e77`.
- Normal case: `/environments/:environment/plants/:plant` opens for an address of four segments.
- Unwanted case: `/plants/:id` does not open for `/plants` or for `/plants/a/b`.

### `PLROUTE-FR-002`

A parameter segment must match exactly one segment of the address, and the segment must not be empty.

**Examples:**

- Normal case: `/plants/:id` matches `/plants/P003-x`, and the value is `P003-x`.
- Unwanted case: `/plants/:id` does not match `/plants/`.

### `PLROUTE-FR-003`

When two page routes match one address, the deck must open the route whose first differing segment is fixed rather than a parameter.

**Why:** A plugin declares `/plants/new` beside `/plants/:id`, and the fixed page must win.

**Examples:**

- Normal case: `/plants/new` and `/plants/:id` both match `/plants/new`. The deck opens `/plants/new`.
- Normal case: `/plants/:id` alone matches `/plants/0199c5e2-8f3a-7c21-9b4e-2f6a1d0c5e77`.

### `PLROUTE-FR-004`

The deck must give the page the value of each parameter segment, by its name, decoded from the address.

**Examples:**

- Normal case: `/plants/:id` at `/plants/0199c5e2` gives `id` the value `0199c5e2`.
- Limit case: the segment `Office%20north` gives the value `Office north`.

### `PLROUTE-FR-005`

When the address changes to another value of the same page route, the page must show the record of the new value.

**Examples:**

- Normal case: a link from `/plants/A` to `/plants/B` shows plant B.
- Normal case: the back button from `/plants/B` shows plant A again.

### `PLROUTE-FR-006`

An address of a page route with a parameter must open the same page after a reload, and from a link in a new browser tab.

### `PLROUTE-FR-007`

The deck must show a not-found page for an address of a plugin that matches no page route of the plugin. The page links to the first navigation entry of the plugin.

**Why:** A mistyped or an outdated link shows a way back instead of a failure.

**Examples:**

- Unwanted case: `/plants/a/b` matches no route. The deck shows the not-found page with a link to the first navigation entry.
- Limit case: the plugin declares no navigation entry. The not-found page holds no link to the plugin.

### Navigation and trail

### `PLROUTE-FR-008`

Maroid must refuse to load a plugin whose navigation entry holds a parameter segment.

**Why:** An entry of the navigation needs one address that a person can open with no value.

### `PLROUTE-FR-009`

The deck must mark as active the navigation entry whose path is the longest leading part of the address, in whole segments.

**Examples:**

- Normal case: at `/plants/0199c5e2`, the entry `/plants` is active.
- Unwanted case: at `/plantsbed`, the entry `/plants` is not active.

### `PLROUTE-FR-010`

The trail of a page must link each leading part of the address that a navigation entry declares.

**Example:** at `/plants/0199c5e2`, the trail links "Plants" to `/plants`.

### `PLROUTE-FR-011`

The last link of the trail must show the page title that the page gives.

**Examples:**

- Normal case: the plant page gives "P003 Gardenia", and the trail ends with "P003 Gardenia".
- Limit case: the page gives a second title after a rename. The trail shows the second title.

### `PLROUTE-FR-012`

Until a page with a parameter gives a page title, the last link of the trail must show a placeholder.

**Why:** The value of a parameter is an internal identifier, which tells a person nothing.

### The plant page of jasmine

### `PLROUTE-FR-013`

The page of one plant must have the address `/plants/<identifier>`, where the identifier is the internal identifier of the plant.

**Why:** The display identifier is a second value, and the internal identifier names every reference.

### `PLROUTE-FR-014`

The plant page must give the display identifier and the common name of the plant as its page title.

### `PLROUTE-FR-015`

A plant address that names no plant of the acting workspace must show that the plant does not exist, with a link to the plant list.

**Examples:**

- Unwanted case: the plant was deleted. The page says so and links to `/plants`.
- Unwanted case: the identifier names a plant of another workspace. The page shows the same answer.

### `PLROUTE-FR-016`

After the delete of a plant from its page, the deck must show the plant list.

## 6. Invariants

### `PLROUTE-INV-001`

One address of a plugin opens at most one page route.

**Why:** Two routes for one address would make the page depend on the order of the declarations.

## 7. Constraints from the guidelines

| Rule      | Guideline      | Effect on this feature                                                                                 |
| --------- | -------------- | ------------------------------------------------------------------------------------------------------ |
| `UI-005`  | Web UI         | Every path of a remote is a navigation entry today. A parameter route is not one, so the rule changes. |
| `UI-008`  | Web UI         | The page gives its page title through the host that the deck gives it, so the host gains a member.     |
| `UI-006`  | Web UI         | A plugin reads its parameters from the host, never from a module of the deck.                          |
| `ARC-008` | Architecture   | The deck and the hub match every plugin the same way. No rule names jasmine.                           |
| `TS-002`  | Frontend style | The page reads its parameters as runes props.                                                          |

## 8. Open questions

| #   | Question                                                   | Owner  | Answer                                          |
| --- | ---------------------------------------------------------- | ------ | ----------------------------------------------- |
| 1   | The key of the feature.                                    | Temuri | `PLROUTE`.                                      |
| 2   | What an address that matches no route shows.               | Temuri | A not-found page of the deck. `PLROUTE-FR-007`. |
| 3   | Whether a page reads the query of its address.             | Temuri | No. Out of scope.                               |
| 4   | Whether the plant page of jasmine is part of this feature. | Temuri | Yes. `PLROUTE-FR-013` to `PLROUTE-FR-016`.      |

## Retired identifiers

This file has no retired identifier.
