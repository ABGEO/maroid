---
id: ADR-0017
title: The service package of a plugin holds its domain services
type: adr
status: accepted
created: 2026-10-13
updated: 2026-10-13
decided: 2026-10-13
changes: [PKG-001]
supersedes:
superseded_by:
---

# The service package of a plugin holds its domain services

## Context

`PKG-001` gives `service/` the clients of an external system alone. The parking plugin
and `GWPACC` keep a connection service there, and `GWPACC` a collector, because no
other directory of the table fits a type that joins a client, the settings, the plugin
secrets, and the database. `REP-003` keeps the transaction out of `repository/`, so the
type that opens a transaction for a handler, a widget, or a job also needs a home.

## Decision

`service/` holds the clients of an external system and the domain services of the
plugin. A domain service may hold `pluginapi.PluginDB`, and it creates each repository
inside `PluginDB.WithTx`.

## Rationale

A handler, a job, and a widget call the same operations, so one type owns each
transaction. A consumer declares the interface it needs, and a test replaces the
service with a fake.

## Alternatives

| Alternative | Why we did not select it |
| ----------- | ------------------------ |
| A new directory `store/` | A second home for one kind of type, and the table grows for each plugin. |
| A transaction in each handler and widget | Each caller repeats the transaction, and a unit test needs a database. |
| A store in `repository/` | `REP-003` forbids a repository that holds the pool. |

## Consequences

`PKG-001` reads, in the row of `service/`: "The clients of an external system
(`EXT-001`), and the domain services that use them."

- Positive: the parking plugin and `GWPACC` follow the table.
- Negative: `service/` mixes two kinds of type, and its file names tell them apart.
- Work that follows: `GWPACC` moves its store from `repository/` to `service/`.
