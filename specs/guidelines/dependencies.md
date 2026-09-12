---
id: DEP
title: Dependency resolution
type: guideline
status: active
created: 2026-09-11
updated: 2026-09-11
scope: [apps/hub/internal/depresolver/]
related: [LIF, ARC, PLG, CLI]
---

# Dependency resolution

`depresolver` is the only place that builds a shared dependency.

## DEP-001

`Container` holds every shared dependency and implements `Resolver`.
A component receives what it needs as a constructor argument.
A component never holds the container and never calls it at run time.

**Why:** A constructor argument states the real dependencies of a component.
A container hides them.

## DEP-002

`Resolver` lists every provider. Assert the implementation. See `GO-007`.

## DEP-003

A dependency is one anonymous struct field holding `once`, `instance`, and `mu`.
The field name is the name of the provider in lower camel case.
Add `mu` only when the build can fail.

```go
database struct { mu sync.Mutex; once sync.Once; instance *sqlx.DB }
```

## DEP-004

A build that cannot fail returns the instance alone, uses `once` without a mutex,
and declares no error in the signature.

## DEP-005

A build that can fail returns the instance and an error.
Hold `mu` for the whole method. Build inside `once.Do`.
**Reset the `once` of the same dependency when the build fails.**

```go
func (c *Container) Database() (*sqlx.DB, error) {
    c.database.mu.Lock()
    defer c.database.mu.Unlock()

    var err error

    c.database.once.Do(func() {
        c.database.instance, err = database.New(c.Config())
    })

    if err != nil {
        c.database.once = sync.Once{}

        return nil, fmt.Errorf("initializing database: %w", err)
    }

    return c.database.instance, nil
}
```

**Why:** `sync.Once` runs one time, even after a failure. Without the reset, a
database that is not ready at the first call stays unavailable for the life of the
process, and every later call returns a nil instance and no error.
The mutex exists because the reset writes the `once`.

Reset the `once` that this method owns. A reset of a different dependency leaves
this one poisoned and writes a field that this mutex does not guard.

## DEP-006

A provider resolves what it needs by calling the other providers inside `once.Do`.
It copies the error to the outer variable and returns at once.

## DEP-007

Move a build that needs more than four dependencies into a private `build<Name>`
method, and call it from `once.Do`.

## DEP-008

A dependency that holds a resource gets `Close<Name>`.
It returns nil when the instance is nil. `Close` joins every step. See `LIF-008`.

## Retired identifiers

This file has no retired identifier.
