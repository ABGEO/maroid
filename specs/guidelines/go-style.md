---
id: GO
title: Go code style
type: guideline
status: active
created: 2026-09-11
updated: 2026-09-11
scope: [apps/hub/, libs/, plugins/]
related: [TS, TST, BLD, LNG, PKG, REP, EXT]
---

# Go code style

## GO-001

`golangci-lint` checks every Go module. The configuration is `.golangci.yaml`
at the repository root.

Run the check in the module that the change touches:

```
cd <module> && golangci-lint run ./...
```

Each Go module in `go.work` is a separate module, so the check runs per module.
`BLD-001` gives the command for each artifact.

## GO-002

The import order is: the standard library, then the external package,
then the package with the prefix `github.com/abgeo/maroid`.
`gci` enforces the order.

## GO-003

Every exported identifier has a documentation comment.
Every package has a package comment.

## GO-004

An error gets a context with `fmt.Errorf` and the verb `%w`.
The context text is lowercase and describes the action:

```go
return fmt.Errorf("opening the plugin: %w", err)
```

## GO-005

A sentinel error lives in `apps/hub/internal/domain/errs` for the hub,
or in an `errs` package inside the plugin.

## GO-006

A comment obeys `LNG`. Write one only when the code cannot explain itself (`LNG-011`).
`LNG-012` lists the comments that are correct.
`GO-003` still requires a documentation comment on every exported identifier.

## GO-007

Assert at compile time that a type implements the interface it claims.

```go
var _ Registrar = (*UIRegistrar)(nil)
```

**Why:** The build fails at the definition, not at the call site far away.

## GO-008

A sentinel error is a package-level variable named `ErrXxx`.
The message is lowercase with the form `"<subject>: <problem>"`.

```go
ErrInvalidMQTTTopic = errors.New("mqtt subscriber: invalid topic")
```

## GO-009

A constructor is `New` when the package builds one type, and `NewXxx` otherwise.
It returns the concrete type, not an interface.

**Why:** The caller decides which interface it needs. The `ireturn` linter enforces this.

## GO-010

Two external documents are the standard of Go code in this repository:

| Standard            | Source                                                | Governs                          |
| ------------------- | ----------------------------------------------------- | -------------------------------- |
| Effective Go        | https://go.dev/doc/effective_go                       | The idiom of the language.       |
| Uber Go Style Guide | https://github.com/uber-go/guide/blob/master/style.md | The style of a large code base.  |

Write the code that these documents describe. A change that contradicts one of
them is a defect.

The order when two of the three disagree:

1. A `GO` rule in this file.
2. The Uber Go Style Guide.
3. Effective Go.

**Why:** Effective Go gives the idiom of the language. The Uber guide decides the
case that Effective Go leaves open. A `GO` rule holds the choice that Maroid makes
for itself, so it wins over both.

## Retired identifiers

This file has no retired identifier.
