---
id: WIDGET
title: The scenarios of the widgets of the home page
type: spec
status: approved
created: 2026-10-12
updated: 2026-10-12
approved_by: Temuri
approved_on: 2026-10-12
constrained_by: [TST, TRC]
requirements: features/widget/requirements.md
---

# Specification: The scenarios of the widgets of the home page

Plugin P is a test plugin that declares the permissions `things.read` (viewer) and
`things.write` (editor). Workspace W enables P. User A is a viewer of W unless the
scenario says otherwise. A manual scenario runs against `pnpm hub:serve` and
`pnpm deck:dev`, in a workspace that enables parking and jasmine.

## `WIDGET-SC-001`

**Verifies:** `WIDGET-FR-001`, `WIDGET-FR-016`
**Layer:** unit

**Given** `NewWidget` with a meta and a load function.
**When** the test calls `Meta`, `Load`, and each answer helper.
**Then** `Meta` answers the meta, and `Load` answers what the function answers.
`StatAnswer` sets `ready` and `Stat` alone. `ListAnswer` sets `ready` and `List` alone.
`EmptyAnswer` sets `empty` and `Message`. `NotSetUpAnswer` sets `not_set_up`, `Message`, and `Link`.

## `WIDGET-SC-002`

**Verifies:** `WIDGET-FR-002`, `WIDGET-FR-003`, `WIDGET-FR-004`, `WIDGET-FR-005`
**Layer:** unit

**Given** P declares one widget, and the test changes one member of its meta for each case.
**When** `WidgetRegistrar.Register` runs.
**Then** each case answers:

| Case | Result |
| ---- | ------ |
| `balance`, "Balance", `stat`, `s`, `things.read` | Registered. |
| Title "  Balance  " | Registered with the title "Balance". |
| Key of 64 characters | Registered. |
| Key "" or of 65 characters, or `Balance`, or `a_b` | `ErrInvalidWidget`, ending in `key`. |
| Title "", or "   ", or 65 characters | `ErrInvalidWidget`, ending in `title`. |
| Kind `chart` | `ErrInvalidWidget`, ending in `kind`. |
| Size `xl` | `ErrInvalidWidget`, ending in `size`. |
| No permission | `ErrPermissionMissing`. |
| Permission `things.delete` | `ErrPermissionUndeclared`. |

Each error names P and the key. The registry holds nothing after a refusal.

## `WIDGET-SC-003`

**Verifies:** `WIDGET-FR-005`, `WIDGET-FR-006`
**Layer:** unit

**Given** three cases: P declares two widgets with the key `balance`. P and plugin Q each
declare one with the key `summary`. P declares a `list` at `xs`.
**When** `WidgetRegistrar.Register` runs for each plugin.
**Then** the first answers `ErrWidgetAlreadyRegistered`. The second registers both. The third registers.

## `WIDGET-SC-004`

**Verifies:** `WIDGET-FR-007`
**Layer:** unit

**Given** P declares `b` then `a`.
**When** the registrar runs.
**Then** `CapabilityRegistry.Of(P)` holds `widgets` with two `WidgetItem` values, `b`
then `a`, each with its title, kind, size, and the permission `dev.maroid.p:things.read`.

## `WIDGET-SC-005`

**Verifies:** `WIDGET-FR-008`
**Layer:** integration

**Given** P and Q each declare one widget. W enables P alone.
**When** A reads `GET /workspaces/W/widgets`.
**Then** the page holds the widget of P alone. A read of the widget of Q answers 404 `not-found`.

## `WIDGET-SC-006`

**Verifies:** `WIDGET-FR-009`
**Layer:** integration

**Given** P declares `open` under `things.read` and `closed` under `things.write`.
**When** A, a viewer, reads the list and then each widget.
**Then** the list holds `open` alone. `open` answers 200. `closed` answers 403
`permission-denied`. An editor reads both in the list.

## `WIDGET-SC-007`

**Verifies:** `WIDGET-FR-010`
**Layer:** integration

**Given** W enables `dev.maroid.zeta` with `z1` and `dev.maroid.alpha` with `a2` then `a1`.
**When** A reads the list ten times.
**Then** each answer orders `a2`, `a1`, `z1`.

## `WIDGET-SC-008`

**Verifies:** `WIDGET-FR-012`, `WIDGET-INV-002`
**Layer:** unit

**Given** a widget whose `Load` counts its calls and answers the count as the value.
**When** `Reader.Read` runs twice.
**Then** `Load` ran twice, and the second answer carries the value 2.

## `WIDGET-SC-009`

**Verifies:** `WIDGET-FR-013`, `WIDGET-INV-001`
**Layer:** integration

**Given** a widget whose `Load` answers `ActingUserFromContext` and
`ActingWorkspaceFromContext` as its value. A is a member of W and of V.
**When** A reads the widget in W, then in V. B, a member of V alone, reads it in W.
**Then** the first two answer A with W, then A with V. B reads 404 `not-found`.

## `WIDGET-SC-010`

**Verifies:** `WIDGET-FR-016`
**Layer:** unit

**Given** a `stat` widget and a `list` widget.
**When** `Load` answers each case below.
**Then** `Reader.Read` answers `ErrInvalidWidgetAnswer`:

| Case |
| ---- |
| State `broken`. |
| `ready` with no `Stat` for `stat`, or with `Stat.Value` empty. |
| `ready` with `List` for `stat`, or no `List` for `list`. |
| `ready` of a `list` with an item whose label is empty. |
| `empty` with no message. `not_set_up` with no link. |

A `ready` `list` with no item, and each answer of the helpers with its members set, pass.

## `WIDGET-SC-011`

**Verifies:** `WIDGET-FR-020`
**Layer:** integration

**Given** one widget whose `Load` answers an error, and one whose `Load` panics.
**When** A reads each.
**Then** each answers 500 `/problems/http/internal`. The hub keeps serving. A log record names P, the key, and the cause.

## `WIDGET-SC-012`

**Verifies:** `WIDGET-FR-022`
**Layer:** unit

**Given** a `stat` widget, and the test sets `Stat.Link` for each case.
**When** `Reader.Read` runs.
**Then** `/account`, `/plants/0199c5e2`, and an empty link pass. `account`,
`https://example.com`, `//example.com`, `/\example.com`, and `/plants/../../x` answer
`ErrInvalidWidgetAnswer`. The same check applies to `Link`, `List.More`, and each item link.

## `WIDGET-SC-013`

**Verifies:** `WIDGET-NFR-001`
**Layer:** unit

**Given** `widget.Deadline`, and a `Load` that sleeps 10 seconds and ignores its context.
**When** `Reader.Read` runs with the deadline set to 50 milliseconds.
**Then** `widget.Deadline` is 5 seconds. The read answers `ErrWidgetTimeout` within
100 milliseconds of its start. The handler maps it to 504 `/problems/hub/widget-timeout`.

## `WIDGET-SC-014`

**Verifies:** `WIDGET-FR-030`
**Layer:** unit

**Given** a fake `ConnectionService` that answers `connected` with the balance 4.5 and 4 free parkings, then 1 free parking.
**When** the balance widget loads twice.
**Then** the first answers `ready` with `4.50`, `GEL`, "4 free parkings left today", and
`/account`. The second carries "1 free parking left today". The meta holds `balance`,
`stat`, `s`, and `sessions.read`.

## `WIDGET-SC-015`

**Verifies:** `WIDGET-FR-031`, `WIDGET-FR-020`
**Layer:** unit

**Given** the fake answers `not_connected`, then an error.
**When** the balance widget loads twice.
**Then** the first answers `not_set_up` with "Connect your parking account." and
`/account`. The second answers the error.

## `WIDGET-SC-016`

**Verifies:** `WIDGET-FR-032`, `WIDGET-FR-033`, `WIDGET-FR-034`, `WIDGET-FR-035`
**Layer:** integration

**Given** jasmine in W with P001 "Fern", watering every 7 days, latest 11 days ago, and
P002 "Gardenia", watering every 7 days, latest 7 days ago, in one environment.
**When** the attention widget loads as A in W.
**Then** it answers `ready` with two items: "P001 Fern", "watering, 11 days, every 7
days, overdue", `/plants/<P001 id>`, then "P002 Gardenia" with `due`. `More` is `/attention`.

## `WIDGET-SC-017`

**Verifies:** `WIDGET-FR-036`
**Layer:** integration

**Given** jasmine in W with every care interval on schedule.
**When** the attention widget loads.
**Then** it answers `empty` with "No plant needs care."

## `WIDGET-SC-018`

**Verifies:** `WIDGET-FR-007`
**Layer:** manual

**Given** the hub with parking and jasmine.
**When** the person opens the plugin list.
**Then** each plugin shows the chip "Widgets".

## `WIDGET-SC-019`

**Verifies:** `WIDGET-FR-008`
**Layer:** manual

**Given** the home page shows the two widgets.
**When** a manager disables parking and opens the home page again.
**Then** the home page shows the jasmine widget alone.

## `WIDGET-SC-020`

**Verifies:** `WIDGET-FR-011`
**Layer:** manual

**Given** a workspace that enables no plugin with a widget.
**When** the person opens the home page.
**Then** the page shows the links of today and no empty area above them.

## `WIDGET-SC-021`

**Verifies:** `WIDGET-FR-014`, `WIDGET-FR-015`
**Layer:** manual

**Given** the browser blocks the request of the parking widget in the developer tools.
**When** the person opens the home page.
**Then** the jasmine widget shows its data. The parking card keeps its placeholder of size `s`, then shows the error when the request fails.

## `WIDGET-SC-022`

**Verifies:** `WIDGET-FR-017`, `WIDGET-FR-018`, `WIDGET-FR-019`, `WIDGET-FR-023`, `WIDGET-FR-024`
**Layer:** manual

**Given** a connected parking account, and plants with due care.
**When** the person opens the home page, then disconnects parking and records every care.
**Then** first the balance and the list show, and each link opens its page in the
workspace. Then parking shows "Connect your parking account." with a link to the
connection page, and jasmine shows "No plant needs care."

## `WIDGET-SC-023`

**Verifies:** `WIDGET-FR-020`, `WIDGET-FR-021`
**Layer:** manual

**Given** a plant named `<b>Fern</b>` with due care, and the parking service down.
**When** the person opens the home page.
**Then** the jasmine item reads the characters `<b>Fern</b>`. The parking card shows
"Parking balance" and "This widget did not load."

## `WIDGET-SC-024`

**Verifies:** `WIDGET-FR-025`, `WIDGET-FR-026`, `WIDGET-FR-027`
**Layer:** manual

**Given** 8 due care intervals, then 5.
**When** the person opens the home page each time.
**Then** first 5 items, "3 more", and "See all". Then 5 items, no count, and "See all".

## `WIDGET-SC-025`

**Verifies:** `WIDGET-FR-028`, `WIDGET-FR-029`
**Layer:** manual

**Given** the home page with the two widgets.
**When** the person views it at a width of 1280 pixels, then of 375 pixels.
**Then** at 1280 the balance takes two of four columns and one row, and the list two
columns and two rows. At 375 each takes the full width, the balance one row and the list two.

## Retired identifiers

This file has no retired identifier.
