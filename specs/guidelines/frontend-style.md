---
id: TS
title: Frontend code style
type: guideline
status: active
created: 2026-09-11
updated: 2026-09-11
scope: [apps/deck/, plugins/*/ui/, libs/plugin-sdk/, libs/api-client/]
related: [UI, GO, TST, LNG]
---

# Frontend code style

`UI` holds the contract between the shell and a plugin.
This guideline holds the code style of the same files.

## TS-001

Every TypeScript package sets `"strict": true`.
A package does not disable a strict check for one file.

## TS-002

A Svelte component uses the Svelte 5 runes: `$state`, `$derived`, `$props`, `$effect`.
A component does not use the legacy syntax `export let`.

## TS-003

ESLint and Prettier check the deck:

```
pnpm --filter @maroid/deck lint
```

A package under `plugins/*/ui` has no lint script today.
Its build reports a type error. See `BLD-001`.

## TS-004

A type check must report no error:

```
pnpm --filter @maroid/deck check
pnpm --filter @maroid/plugin-sdk check
pnpm --filter @maroid/api-client check
```

Run the check for the package that the change touches. See `BLD-001`.

## TS-005

Tailwind CSS 4 and daisyUI give the style.
Do not add a different CSS framework. Do not write a global stylesheet for a component.

## TS-006

A comment obeys `LNG`. Write one only when the code cannot explain itself (`LNG-011`).
`LNG-012` lists the comments that are correct.

## Retired identifiers

This file has no retired identifier.
