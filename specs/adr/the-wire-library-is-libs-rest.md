---
id: ADR-0007
title: The wire library is libs/rest
type: adr
status: accepted
created: 2026-09-24
updated: 2026-09-25
decided: 2026-09-24
changes: [ERR-003, ERR-007, BLD-004]
supersedes:
superseded_by:
---

# The wire library is libs/rest

## Context

`ADR-0005` created `libs/problem`. It holds the `Problem` type, the constructor of
each registered failure, the writer, and the middleware that gives a request its
identifier. `ERR-007` names the module and gives the reason: a plugin fills
`instance` too, and `PLG-007` denies a plugin the import of `apps/hub`.

`ADR-0006` bound the Zalando guidelines and added `RES`. Two of its rules need a
type that no module holds today:

- `RES-005` gives the page object that every collection answers.
- `RES-006` gives the cursor that a page carries.

Both reach a plugin. `plugins/jasmine` answers four collections, and every plugin
that follows will answer its own. So the page and the cursor take the same
constraint that the problem took: a plugin must import them, and `PLG-007` denies
it `apps/hub`. `ARC-005` gives `libs/*` the code that a plugin and the hub both
import.

Three modules could hold them:

| Module            | Holds today                                      |
| ----------------- | ------------------------------------------------ |
| `libs/problem`    | The failure half of the wire contract.           |
| `libs/pluginapi`  | What a plugin declares to the hub.               |
| A new `libs/page` | Nothing. It would be the tenth Go module.        |

`APIFMT-DD-001` selected the first and recorded that this decision must ratify it.

`ADR-0006` also moved two things that `libs/problem` holds. `ERR-006` replaced the
request identifier with the flow identifier, which reads `X-Flow-ID` and answers
`/flows/<value>`. `ERR-003` added `cursor-stale` to the registry, which is a
failure about a cursor and not about a problem.

## Decision

`libs/problem` becomes `libs/rest`, with the module path
`github.com/abgeo/maroid/libs/rest`. It holds the problem, the page, the cursor,
the writer, and every middleware that shapes an HTTP answer.

## Rationale

The module already carries the wire contract that the hub and every plugin share.
`Problem` is one of the things it carries, and after `ADR-0006` it is not even the
largest: the page reaches every collection route, and the middleware set grows to
the flow identifier, the concurrency check, the idempotency store and the cache
period.

A module named for one of its types tells a reader the wrong thing about the other
four. `rest` names the guideline that governs all of them, so `RES-005` and
`libs/rest` read as one subject.

`libs/pluginapi` was the alternative that costs no new module. It defines what a
plugin declares to the hub: a capability, a route, a command, a migration. A page
is what a plugin answers to a client, and the hub is not the reader. Putting the
wire contract there would make every plugin that imports one import both.

A tenth module holding one generic type is the weakest option. It splits the wire
contract across two imports with no rule to say which half a new type joins.

## Consequences

### What changes

| Item                        | Change                                                          |
| --------------------------- | ----------------------------------------------------------------- |
| `libs/problem/go.mod`       | The module path becomes `github.com/abgeo/maroid/libs/rest`.    |
| `go.work`                   | `./libs/problem` becomes `./libs/rest`.                         |
| 17 Go files                 | The import path and the `problem.` qualifier become `rest.`.    |
| `ERR-003`, `ERR-007`        | Every sentence that names `libs/problem` names `libs/rest`.     |
| `ERR-007`                   | It names the page and the cursor as the other residents, and cites `ARC-005`. |
| `errors.md` frontmatter     | The `scope` list names `libs/rest/`.                            |
| `BLD-004`                   | It reaches any module of `libs/` that every plugin imports.     |
| `ADR-0005`                  | No change. It records what was true when it was decided.        |

The out-of-tree `asg` plugin of `go.work` imports nothing from this module, so it
takes no change.

`ERR-007` also holds a defect that this decision corrects. It cites `ARC-004` for
the rule that gives `libs/` the shared code. `ARC-004` says the database is
PostgreSQL, and `ARC-005` holds the rule. `TRC-006` makes a reference to the wrong
rule a defect.

### What does not change

`Problem`, `FieldFailure`, `Fill` and `Write` keep the names they have. Each one
names its subject already, or takes a `Problem` that a reader sees at the call
site. The package clause becomes `package rest`, so a call site reads
`rest.Write` and `rest.NewNotFound`.

One identifier takes the word that the package name used to carry. `MediaType`
becomes `ProblemMediaType`, because a bare constant offers a reader no type to
read the subject from, and this module answers `application/json` for a page as
well. `Instance` and the four names of the request identifier have the same
shape, and `ERR-006` renames them to the flow identifier, so they take their new
names there.

The media type stays `application/problem+json`, and every registered type keeps
its identifier. `libs/api-client` and `@maroid/plugin-sdk` read a body and not a
Go module, so neither changes for this decision.

### Migration

One commit renames the directory, the module path and every import. No commit in
between compiles, because a Go module path is not aliasable, so the rename cannot
be staged. Every plugin then rebuilds, because each one imports the module.

`BLD-004` names `libs/pluginapi` alone, and it is the rule that asks for that
rebuild. This decision widens it to any module of `libs/` that every plugin
imports. `libs/rest` is the second one.

`APIFMT` step 1 of the build plan in `specs/features/apifmt/spec.md` is this
commit, and it lands before any other step of that feature.

### Result

- Positive: one module holds the wire contract, under the name of the guideline
  that governs it.
- Positive: a page and a cursor reach a plugin with no new module and no import
  of the hub.
- Negative: 17 files change for a rename that adds no behavior.
- Negative: an ADR, a guideline and a spec all name a module that did not exist
  yesterday, so a reader of the history meets two names for one thing.
