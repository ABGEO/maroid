---
id: THEMING
title: The shared theme and the identity provider's pages
type: requirements
status: approved
created: 2026-09-17
updated: 2026-09-17
approved_by: Temuri
approved_on: 2026-09-17
constrained_by: [ARC, TS, BLD]
---

# Requirements: The shared theme and the identity provider's pages

## 1. Problem

The identity provider serves every sign-in page with its own stock theme. A person
who moves from the shell to a sign-in page sees a sudden change: different colors,
different type, no Maroid brand mark. The shell already holds one visual identity,
the theme, defined once. Nothing outside the shell reads that definition.

## 2. Users

- A person signing in: reaches the identity provider's pages from the shell, or
  directly, and must recognize one product, not two.
- The owner: changes the visual identity once, in one place, and every page reflects it.

## 3. Out of scope

- A dark theme at the identity provider. The shell keeps its own toggle. A later
  feature decides how an anonymous visitor at the identity provider gets one.
- A distinct icon for one provider. The identity provider keeps its current icon,
  keyed by the protocol.
- The wording or the content of a page. Only the visual identity changes.
- The sign-in flow itself: the tokens, the storage, the providers it federates.
  `EXTID`, `IDENT`, and `PSET` own that.
- Whether the identity provider's session feature is turned on. The signed-in
  account page and the sign-out page carry the theme whenever a deployment reaches
  them; this feature does not decide when that is.
- Which container image or which infrastructure serves the identity provider today.
  This feature changes what a page looks like, not what runs it.

## 4. Definitions

| Term         | Meaning                                                                                     |
| ------------ | --------------------------------------------------------------------------------------------- |
| `GLO-theme`      | The set of colors, radii, and fonts that gives a page its Maroid visual identity. One definition. Every page that carries it reads the same definition. |
| `GLO-brand-mark` | The logo and the product name, shown together, identifying Maroid on a page.               |

## 5. Functional requirements

### `THEMING-FR-001`

The identity provider must show the theme on every page it can show to a person:
the sign-in chooser, the credential form, the consent screen, the device code
entry, the device confirmation, the error page, the out-of-band code page, the
two-factor code entry, the security-key screen, the sign-out page, and the
signed-in account page.

**Why:** A page that keeps the identity provider's stock theme breaks the recognition
that the other pages built.

**Examples:**

- Normal case: a person follows a sign-in link from the shell and reaches the
  credential form. Its colors and its font match the shell.
- Limit case: the identity provider shows an error page. The error page carries the
  theme too.
- Unwanted case: a page renders with the identity provider's own stock colors.

### `THEMING-FR-002`

Maroid must keep one definition of the theme. The shell and the identity provider
must both read that definition, and neither holds its own copy.

**Why:** A color, a radius, or a font changes once. Every page that carries the theme
changes with it, with no second edit.

**Examples:**

- Normal case: the owner changes the primary color in the one definition. The shell
  and the identity provider both show the new color once each rebuilds.
- Unwanted case: the identity provider still shows the old color after the shell
  already shows the new one.

### `THEMING-FR-003`

The identity provider must show the brand mark on every page it can show to a
person, in the position and the style that the shell uses for it.

### `THEMING-FR-004`

The identity provider's pages must use the shell's own page structure: a header that
carries the brand mark, and a centered panel that carries the page's content.

**Why:** The person, not only the color, must feel the same design.

## 6. Invariants

### `THEMING-INV-001`

The identity provider shows the same theme regardless of a visitor's device
color-scheme preference.

**Why:** The theme carries one visual identity. A device setting must not silently
replace it with a different one.

## 7. Constraints from the guidelines

| Rule      | Guideline    | Effect on this feature                                                                           |
| --------- | ------------ | -------------------------------------------------------------------------------------------------- |
| `ARC-003` | Architecture | The shared theme ships in a form the shell's TypeScript toolchain imports with no separate step.  |
| `ARC-005` | Architecture | The shared theme is a shared library, the existing kind. The identity provider's own build lives under `apps/`, alongside the host and the shell. See open question 2. |
| `ARC-007` | Architecture | Each new TypeScript package this feature adds joins `pnpm-workspace.yaml`.                         |
| `ARC-009` | Architecture | However the themed pages ship, the chart in `chart/` deploys them as a container image, the same as every other component. |
| `TS-005`  | Frontend style | The theme uses Tailwind CSS 4 and daisyUI. No other CSS framework.                               |
| `BLD-001` | Build and release | Each new artifact this feature adds gets its own row: a lint command and a build command.    |

## 8. Open questions

| #   | Question | Owner | Answer |
| --- | -------- | ----- | ------ |
| 1   | Who produces the final, correctly sized logo and favicon for the identity provider, and by when? A prototype used the shell's own image, unresized. | Temuri | The favicon stays the identity provider's own stock favicon. The logo is the shell's own image, unresized. Both are temporary, and later work replaces them. |
| 2   | `ARC-005` names four kinds of module: the host, a shared library, a plugin, the shell. Does the shared theme become a shared library, an existing kind? Where does the identity provider's themed build live? | Temuri | The shared theme is a shared library, the existing kind. The identity provider's own build lives under `apps/gate`, with its own `package.json`, the same shape as every other buildable package in the workspace. This needs no ADR: `THEMING`'s specification places it, the way a specification places any other file. |

## Retired identifiers

This file has no retired identifier.
