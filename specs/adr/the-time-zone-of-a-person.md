---
id: ADR-0024
title: The time zone of a person
type: adr
status: accepted
created: 2026-11-22
updated: 2026-11-22
decided: 2026-11-22
changes: [RES-001, DAT-010, GLO-time-zone]
supersedes:
superseded_by:
---

# The time zone of a person

## Context

`RES-001` says: "The hub reads no zone of a person and renders no local time. A
client converts for the reader that it serves." `GARDEN-FR-024` applies it: a view
shows each moment in the time zone of the browser.

That holds while a browser reads the data. Three cases have no browser:

- A job that acts for a person. A plugin that asks a person a question at 09:00 of
  their day must know which 09:00.
- A Telegram update. The bot answers a person whose zone it does not know.
  Telegram sends no time zone with an update.
- A count of the days of a person. "Today" in UTC is not "today" for a person at
  UTC+4 between 00:00 and 04:00 of their time.

A browser also gives the wrong zone when a person reads the data from a device set
to another zone, at work or while they travel.

## Decision

The user record holds an optional time zone from the IANA time zone database, which
the person sets. The hub and a plugin compute a local hour or a local day of the
acting user in it, and the web shell shows each moment in it. A record with no time
zone computes in UTC, and its web shell uses the zone of the browser.

A moment stays UTC in every column and on the wire.

## Rationale

The person is the one fact that a job, an update, and a request share. One value on
the record serves all three. Each plugin reads it, so no plugin asks the person a
second time.

The zone changes no stored value. It changes the hour that a job picks and the text
that a view shows, so a change of the zone needs no migration of data.

UTC as the fallback of a computation keeps a job deterministic for a person who set
nothing. The browser as the fallback of a view keeps the shell as it is today for
that person.

## Alternatives

| Alternative                                              | Why we did not select it                                                                                                                |
| -------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------- |
| A setting of each plugin that needs a zone               | Each plugin asks the same question again. The values drift apart.                                                                       |
| A time zone of the workspace                             | A workspace holds people in different zones. A person in two workspaces would get two zones.                                            |
| The browser sends its zone with each request             | A job and a Telegram update carry no request.                                                                                           |
| The zone that Telegram gives                             | Telegram gives none. A shared location gives coordinates, and a lookup from coordinates to a zone needs a data set or a service.        |
| A required zone, with UTC as the default of every record | The shell cannot tell "the person chose UTC" from "the person chose nothing", so it cannot fall back to the browser for the second one. |

## Consequences

`RES-001` replaces the paragraph that starts "A point in time travels as UTC" with:

> A point in time travels as UTC, in the `date-time` format of `Z-169`, with the
> `Z` suffix and never a numeric offset. The hub renders no local time in a body. A
> client converts for the reader that it serves: in the time zone of the person
> (`GLO-time-zone`) when the record holds one, else in the zone of the browser. The
> hub reads the time zone of the acting user only to compute a local hour or a local
> day. `ADR-0024`.

`DAT-010` gains a paragraph after the first one:

> A column that holds an hour of the day that a person names, the hour of a
> schedule for example, is `TIME`. It is no point in time. The time zone of the
> person gives its instant on each day. A column that holds a time zone holds the
> name from the IANA time zone database as `TEXT`. `ADR-0024`.

The glossary gains:

| ID              | Term      | Definition                                                                                                                       |
| --------------- | --------- | -------------------------------------------------------------------------------------------------------------------------------- |
| `GLO-time-zone` | time zone | The zone from the IANA time zone database that a person sets on their own record. It gives their local hour and their local day. |

- Positive: a job, an update, and a view of one person agree on the hour and the day.
- Positive: a person who reads the shell from another device sees their own time.
- Negative: every view that shows a moment reads the zone of the record. A view that
  forgets it shows the browser zone, and no test catches it.
- Negative: the contract between the hub and a plugin grows, so every plugin builds
  again (`BLD-004`).
- Work that follows:
  - `PROFILE` changes its requirements and its specification: the field, the route,
    the web shell, and the read of a plugin.
  - `GARDEN-FR-024` retires. A statement that follows the time zone of the person
    replaces it.
  - `EXTID`: the person of the current session carries the time zone, so the shell
    reads it at the start.
  - A core migration adds the column to `public.users`.
