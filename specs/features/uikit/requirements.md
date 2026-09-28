---
id: UIKIT
title: The shared component library
type: requirements
status: approved
created: 2026-09-28
updated: 2026-09-28
approved_by: Temuri
approved_on: 2026-09-28
constrained_by: [ARC, UI, TS, BLD, RES]
---

# Requirements: The shared component library

## 1. Problem

The deck carries the theme. A remote does not. The jasmine remote writes its own
colors, with fixed fallback values, and they do not follow the dark variant that a
person selects in the deck. A plugin page looks different from the deck page next
to it.

Each remote also builds its own copy of a common control. Jasmine holds its own
pagination. The next plugin with a collection writes a second one.

## 2. Users

- A person using the deck: moves between a deck page and a plugin page and sees one
  product. The theme variant they select applies to both.
- A plugin author: builds a plugin page from ready components and the theme, and
  writes no color and no common control.
- The owner: corrects a component once, and every remote gets the correction at its
  next build.

## 3. Out of scope

- The catalog itself. `UIKIT-INV-003` keeps it possible. A later feature builds it.
- A page number, a total count, or a jump to the first page. `RES-005` gives a page
  no total, and the pagination moves one page at a time.
- A translation of the labels.
- A change to the theme itself. `THEMING` owns its definition.
- The deck pages. No deck page shows a collection in pages today.

## 4. Definitions

| Term                        | Meaning                                                                                           |
| --------------------------- | ------------------------------------------------------------------------------------------------- |
| `GLO-component-library`     | The one shared set of components and styling that the deck and every remote import.              |
| `GLO-component`             | One reusable control of the component library. It receives every value it shows from its caller. |
| `GLO-theme-variant`         | One of the two forms of the theme that the deck offers: light and dark.                          |
| `GLO-catalog`               | A page that shows each component alone, with sample values, outside the deck.                    |

## 5. Functional requirements

### Styling

#### `UIKIT-FR-001`

A plugin page must show the colors, the fonts, and the radii of the theme.

**Why:** The person sees one product, not a deck with foreign pages inside it.

**Examples:**

- Normal case: the jasmine plant list shows the primary color of the theme on its
  controls, the same color as a deck button.
- Unwanted case: a plugin page shows a color that the theme does not define.

#### `UIKIT-FR-002`

A plugin page must show the theme variant that the person selected in the deck.
It must change to the other variant when the person switches, with no reload.

**Examples:**

- Normal case: the person selects the dark variant. The open plant list turns dark.
- Limit case: the person opens a plugin page for the first time with the dark variant
  already selected. The page shows the dark variant from its first display.
- Unwanted case: the deck turns dark and the plugin page stays light.

#### `UIKIT-FR-003`

A plugin author must be able to use every style rule that the deck uses, and the
rule must take effect on the plugin page.

**Why:** Today a style rule that only a plugin page names produces no visible effect,
because nothing generates it. The author falls back to a hand-written color.

**Examples:**

- Normal case: a plugin page names the error button style. The button shows the
  error color of the theme.
- Unwanted case: a plugin page names a style rule that no deck page names, and the
  element renders with no style.

#### `UIKIT-FR-004`

A change to a theme value must reach every plugin page after the rebuild of the deck
alone.

**Why:** `THEMING-FR-002` makes a theme change one edit. A rebuild of each plugin
would make it one edit and six releases.

**Examples:**

- Normal case: the owner changes the primary color and rebuilds the deck. The jasmine
  page shows the new color, with the old jasmine build.

### Pagination

#### `UIKIT-FR-005`

The pagination must show one control that opens the previous page and one control
that opens the next page of a collection.

#### `UIKIT-FR-006`

The pagination must disable the previous control when the page has no previous
page. It must disable the next control when the page has no next page.

**Examples:**

- Normal case: the first page of three. Previous is disabled. Next is enabled.
- Limit case: the last page. Next is disabled. Previous is enabled.

#### `UIKIT-FR-007`

The pagination must show no control when the collection fits on one page.

**Examples:**

- Normal case: a collection of 4 items with a page limit of 20. No control shows.
- Limit case: an empty collection. No control shows.

#### `UIKIT-FR-008`

The pagination must open a page only through a link that the current page carries.

**Why:** `RES-006` makes a cursor opaque. A client that builds one breaks when the
hub changes the encoding.

#### `UIKIT-FR-009`

The pagination must disable both controls while a page loads.

**Why:** A second click during a load sends a second request, and the slower answer
wins.

**Examples:**

- Unwanted case: the person double-clicks next on page 1 and lands on page 3.

#### `UIKIT-FR-010`

The component library must give a plugin page the state of the current page:
loading, shown, or failed.

#### `UIKIT-FR-011`

After a failed load, the component library must let the plugin page request the same
page again.

**Examples:**

- Normal case: page 2 fails. The person selects retry. Page 2 loads, not page 1.

### Adoption

#### `UIKIT-FR-012`

The jasmine plugin pages must use the pagination of the component library, and hold
no pagination of their own.

**Why:** One remote on the component library proves the path for every later plugin.

#### `UIKIT-FR-013`

The jasmine plugin pages must take every color from the theme.

## 6. Non-functional requirements

### `UIKIT-NFR-001`

A plugin page must show the new theme variant less than 100 milliseconds after the
person switches it, measured from the click on the switch to the last changed color
on the plugin page.

### `UIKIT-NFR-002`

The component library must add less than 15 kilobytes, compressed with gzip, to the
build output of one remote that uses the pagination, measured on the jasmine build
output before and after the adoption.

**Why:** `UIKIT-INV-002` puts a copy in each remote. The copy must stay small.

## 7. Invariants

### `UIKIT-INV-001`

No remote holds a copy of a theme value.

**Why:** A copy goes stale at the next theme change. `THEMING-FR-002`.

### `UIKIT-INV-002`

A built remote shows the components of the version it was built with. A rebuild of
the deck changes no component behavior on a plugin page.

**Why:** The deck can change. A plugin does not break (`UI-006`). No contract between
a deck version and a remote version is needed.

### `UIKIT-INV-003`

Every component renders and works with no deck, no hub, and no signed-in user. It
receives every value it needs from its caller.

**Why:** A catalog can show each component alone. A test can render it alone.

## 8. Constraints from the guidelines

| Rule      | Guideline         | Effect on this feature                                                                  |
| --------- | ----------------- | --------------------------------------------------------------------------------------- |
| `ARC-003` | Architecture      | The components use TypeScript and Svelte 5.                                             |
| `ARC-005` | Architecture      | The component library is a shared module, the existing kind, in a package of its own.   |
| `ARC-007` | Architecture      | A new TypeScript package joins `pnpm-workspace.yaml`.                                   |
| `UI-002`  | Web UI            | A remote builds on its own. The styling of `UIKIT-FR-003` must reach that build.        |
| `UI-006`  | Web UI            | The component library lives outside `apps/deck`, so a remote can import it.            |
| `TS-001`  | Frontend style    | The package sets strict type checks.                                                    |
| `TS-002`  | Frontend style    | Every component uses runes.                                                             |
| `TS-005`  | Frontend style    | The styling is Tailwind CSS 4 and daisyUI. No other framework.                          |
| `BLD-001` | Build and release | The component library gets a row: a lint command and a check command.                   |
| `BLD-003` | Build and release | A change to the component library rebuilds each remote before its shared object.        |
| `RES-005` | REST              | The pagination reads the `next` and `prev` links of a page.                             |
| `RES-006` | REST              | The pagination builds no cursor.                                                        |

## 9. Open questions

| #   | Question | Owner | Answer |
| --- | -------- | ----- | ------ |
| 1   | `UIKIT-FR-004` makes a theme change reach an old plugin build. The alternative is that a plugin shows the theme it was built with, like its components (`UIKIT-INV-002`). Which? | Temuri | A theme change reaches an old plugin build. `UIKIT-FR-004` stands. |
| 2   | `UIKIT-NFR-002` sets 15 kilobytes. Is the figure acceptable? | Temuri | Yes. |
| 3   | Does the component library become its own package, or join `@maroid/theme` or `@maroid/plugin-sdk`? | Temuri | A package of its own. |

## Retired identifiers

This file has no retired identifier.
