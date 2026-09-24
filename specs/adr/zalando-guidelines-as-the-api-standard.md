---
id: ADR-0006
title: The Zalando guidelines as the standard of the HTTP API
type: adr
status: accepted
created: 2026-09-22
updated: 2026-09-24
decided: 2026-09-24
changes: [API-001, API-002, API-003, API-006, API-007, BLD-001, CLI-001, DAT-010, DAT-011, ERR-001, ERR-002, ERR-003, ERR-004, ERR-006, GLO-api-document, GLO-api-fragment, GLO-binding, GLO-cursor, GLO-flow-id, GLO-handoff, GLO-page, GLO-shipped-client, GLO-transport-route, LNG-013, LOG-009, LOG-010, RES-001, RES-002, RES-003, RES-004, RES-005, RES-006, RES-007, RES-008, RES-009, RES-010, RES-011, SEC-008, SEC-010, SPC-001, SPC-002, TG-001, TRC-001]
supersedes:
superseded_by:
---

# The Zalando guidelines as the standard of the HTTP API

## Context

`ADR-0005` gave every failure one shape. A success carries one sentence:

> `API-006`: A response body is JSON. An error response obeys `ERR-001`.

No rule names the case of a property, the shape of a collection, the format of a
date, or the members of a page. Each handler decides, and the decisions differ.

| Fact                                                                          | Sites                                             |
| ----------------------------------------------------------------------------- | ------------------------------------------------- |
| A list answers a bare JSON array, so no member can join it later.             | `/plugins`, `/auth/identities`, two of `jasmine`  |
| No route takes a page parameter. No contract names an order.                  | Every list route                                  |
| Every list query sorts by `id`, and no document says so.                      | 4 repositories                                    |
| A property name is camel case. `PCAP-DD-002` decided it.                      | 11 tags, and 3 capability names                   |
| A timestamp reads `updatedAt`, and three plugins store it without a zone.     | `jasmine`, `telasi`, `pensions`, `tbilisi-energy` |
| Five routes name an action, and `/auth/me` names a pronoun.                   | `apps/hub/internal/handler/`                      |
| Three fragments declare a bearer for a route that a cookie guards.            | `pcap`, `pset`, `templates/api.yaml`              |
| No specification carries an identifier or an audience. Each carries `0.1.0`.  | The five `api.yaml`                               |
| Five `api.yaml` fragments exist, and no document merges them.                 | `specs/features/`                                 |

Five questions have no answer today: the shape of a successful response, the
page and the sort of a collection, the format of a date and its zone, the
merged OpenAPI document, and the health check. One missing standard causes all
five.

The Zalando RESTful API and Event Guidelines hold 144 rules. The last
change landed on 2026-07-07. Zalando runs them against a linter, and the rules
carry the rationale that produced them.

## Decision

The Zalando RESTful API Guidelines bind every HTTP API that Maroid designs, at
the revision that `RES-001` names. A new guideline, `RES`, holds that revision,
the register of every deviation, and the choices that Zalando leaves to the owner
of an API.

## Rationale

A standard that 144 rules already decide costs less than a house style. The
repository holds one API, one first-party client, and one reviewer, so the price
of a rule is the price of following it, and never the price of agreeing on it.
The rules arrive with a rationale, a linter reads them, and a reader who knows
them needs no second document.

Full adoption costs less to hold than a subset. A subset must give a reason for
each rule that it drops, and a rule that it never names stays undecided. A
register that holds the exceptions answers the same question in fewer lines.

The naming rule reaches less code than it looks. The repository holds 109 camel
case JSON tags. 11 of them sit on the API surface and change, and 4 test
fixtures change with the production tag that each one asserts.

| Surface                                        | Tags | Changes                                |
| ---------------------------------------------- | ---- | -------------------------------------- |
| `plugins/*/dto/api-client.go`                  | 90   | No. A third-party service sends the name, and `EXT` governs it. |
| `apps/hub/internal/handler/auth.go`            | 5    | Yes                                    |
| `plugins/jasmine/dto/`                         | 6    | Yes                                    |
| `plugins/parking/config/config.go`             | 2    | No. `SettingsValues` declares `additionalProperties` and no property, so it is a map of `Z-216` and its keys carry their own format. No migration rewrites `public.plugin_settings.fields`. |
| `apps/hub/internal/mcpserver/tools/`           | 2    | No. `RES-003` puts the payload of an MCP tool outside the reach of the standard. |
| Test fixtures in `apps/hub/internal/`          | 4    | With the tag that each one asserts.      |

The deck and a plugin user interface read those names in 15 places across 5
files. `libs/plugin-sdk` reads none of them.

Five of those reads already declare the snake case name. `apps/deck` types
`first_name`, `last_name`, `display_name`, `picture_url`, and `attached_at`
against a hub that answers camel case, so `displayName()` falls through to a
default and the profile page renders no display name. Tranche 1 repairs a defect
that runs today.

The cursor fits the data that `DAT-009` already gives. Every row identifier is a
UUID version 7, which sorts by time, and every list query already ends with
`ORDER BY id`. A keyset page needs no new index and reads no row that it skips.

The event chapter has no subject. `MQT-001` gives a plugin a subscriber, and a
device publishes the message. Maroid publishes no event to any bus, so 24 rules
describe work that does not exist. `RES-001` holds the condition that starts them.

## Alternatives

| Alternative                                                      | Why we did not select it                                                                                                                                       |
| ---------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------- |
| A house standard assembled from the RFCs.                        | Every rule then needs a decision and a reason. The repository holds no second opinion to test a decision against, and the reasons cost more than the rules.   |
| Google AIP.                                                      | Its JSON mapping emits camel case, so `PCAP-DD-002` survives, and its page is lighter. It assumes protocol buffers, resource names, and long running operations, and it ships 150 documents and no linter for this stack. |
| The Microsoft REST guidelines.                                   | Camel case and an OData collection. A member that starts with `@` reads badly in TypeScript, and the standard carries the same size as Zalando with less rationale. |
| JSON:API 1.1.                                                    | It rewrites every DTO, both clients, and the settings form, to serve one first-party consumer.                                                                |
| A curated subset of Zalando, with the rest cited as a source.    | A rule that the subset does not name stays undecided, and the next feature decides it again. The register of `RES-002` holds fewer lines than the subset.     |
| Adopt Zalando and keep camel case.                               | It deviates from the rule that Zalando argues at the greatest length, on the surface that a reader meets first. 11 tags and 15 reads buy the whole standard.  |
| Adopt the event chapter now.                                     | Maroid publishes no event. A rule with no subject cannot be checked, and it goes stale before it gets one.                                                    |

## Consequences

### Rules that change

`RES` is new. `specs/guidelines/rest.md` holds `RES-001` through `RES-011`. They
give the standard and its revision, the deviation register, the reach of the
standard, the derived address, the page, the cursor, the three documents and
their meta blocks, the merge and the linter, the security scheme with the
permission, the rules that bind a client, and the window before a route goes.

`API-002` renames the first middleware. It reads the flow identifier.

`API-003` holds the new path table. `RES-004` holds the exemption.

| New                                | Old                  | Answers                                      |
| ---------------------------------- | -------------------- | -------------------------------------------- |
| `POST /auth/sessions`              | `GET /auth`          | 202 with the address of the IdP.             |
| `GET /auth/sessions/self`       | `GET /auth/me`       | The acting user and the provider.            |
| `DELETE /auth/sessions/self`    | `POST /auth/logout`  | 204. Public, and idempotent.                 |
| `POST /auth/identities`            | `GET /auth/link`     | 202 with the address of the IdP.             |
| `POST /auth/invitation-redemptions`| `GET /auth/invite`   | 202 with the address of the IdP.             |
| None                               | `GET /ping`          | The route goes. Nothing calls it.            |

The three routes that hand off to the IdP take `POST`, and each one answers 202.
`GET /auth` writes today: it mints a binding and sets a cookie, which `Z-149`
forbids a `GET` to do. The call creates no session, because the person reaches
the IdP after it returns and `GET /auth/callback` finishes the work. 202 names
that state, and 201 would name a row that does not exist. `Z-220`, `Z-253`.

The body carries the address of the IdP, and the deck performs the navigation.
`Z-253` pairs its 202 with a `Location` header, and this answer carries none.
RFC 9110 gives `Location` on a 202 the resource that the call created or a
monitor of its progress, and the address of the IdP is neither. A member of the
body says what it is, and the document declares it.

```
POST /auth/sessions
{ "redirect": "https://maroid.abgeo.cloud/auth/callback" }

202 Accepted
Set-Cookie: __Host-maroid_binding=...; SameSite=Lax
{ "authorization_url": "https://auth.example.com/authorize?..." }
```

`SEC-008` gains the `SameSite` row, with the value `Lax`. The code sets it
today, and the table omits it. The attribute becomes load bearing with this
change: a browser stores a `Lax` cookie from a `POST` only when the request is
same site, and `SEC-008` already puts the hub and the deck on one domain.

The sign in reaches the hub through a client that declares no `onUnauthorized`.
`redirectToAuth` of `apps/deck/src/lib/api/client.ts` is that handler, so a
handler that posts through its own client re-enters itself.

`/ping` answers `{"message": "pong"}`. No probe of the chart, of the compose
file, or of a Dockerfile reads it, and no client calls it. The route goes with
this decision. Maroid then answers no health check, and a later decision designs
one and gives it a path.

The MCP tool `ping` is a different thing. `MCPHUB-FR-006` gives it, and it
stays.

`GET /auth/callback`, `/mcp`, `/telegram/webhook`, and
`/.well-known/oauth-protected-resource` keep their paths, and each one keeps it
for a different reason.

| Route                                    | Why it keeps the path                                                                                     |
| ---------------------------------------- | ----------------------------------------------------------------------------------------------------------- |
| `/.well-known/oauth-protected-resource*` | A client derives the address from RFC 9728. `RES-004`.                                                    |
| `/mcp`, `/telegram/webhook`              | A transport route, outside the reach of `RES-003`. The configuration holds the address.                   |
| `GET /auth/callback`                     | `RES-002` holds the deviation. The address is ours, and RFC 6749 section 4.1.2 makes the method a `GET` that writes. |

Only the first address is fixed. Maroid registers the second pair with Dex and
with Telegram, and it renames either one by changing the configuration and the
registration. A registered address constrains a deployment, not a design.

`API-006` points at the new guideline:

> A response body is JSON. A success response obeys `RES-001`, and an error
> response obeys `ERR-001`.

`ERR-002` replaces the URN with a relative reference, which `Z-176` proposes
and which the owner chose.
The owner segment stays, so a plugin still owns its namespace.

```
/problems/http/not-found
/problems/hub/settings-invalid
/problems/dev.maroid.jasmine/watering-overlap
```

`ERR-003` repeats the prefix in two tables, and each one changes with `ERR-002`.
The member table of `ERR-001` and the example of `ERR-004` name the old form, and
both take the new one.

`ERR-006` replaces `X-Request-Id` with `X-Flow-ID`. The hub reads the header of
the request when it holds one, and it generates a UUID version 7 when it does
not. It echoes the value, and `instance` carries `/flows/<value>`.

The rule bounds the value that a caller sends, with the character set that `Z-233`
gives. The hub removes every character outside `[a-zA-Z0-9/+_=-]` and cuts
the value to 128 characters. It generates a UUID version 7 when nothing remains.

The bound limits the risk that `ERR-006` named and does not remove it. A caller
still picks the value that every log record of its request carries, and it still
reaches the body of a problem. `Z-233` requires that a caller supplies the
value, and the two cannot both hold.

`LOG-009` renames the attribute from `request` to `flow_id`. The name matches the
header that `ERR-006` reads, in the snake case that `Z-118` gives every member.

`LOG-010` held a record of `GET /ping` as its example. The example names a route
that the hub serves.

Every document declares the audience `external-public`. Maroid is software that a
person deploys, and a person who deploys it writes a user interface, a command
line tool, or any other client against the instance that they run, so the
consumer set has no boundary. A cloud offer of Maroid needs no second value,
because this one already holds every reader.

The audience decides more than one field. It raises `Z-223` to the level of a
must in `RES-002`, it denies the component-internal exemption of `Z-192`, and it
gives `Z-106` and the deprecation rules a subject that they have not had.

`RES-007` holds three documents, and the audience is the same for each one. The
version of `mcp.yaml` follows the revision of MCP that the hub implements, the
version of `telegram.yaml` follows the Bot API, and the version of `hub.yaml`
follows Maroid, so one document could not carry one version. `/telegram/webhook`
takes the third document because `SEC-010` guards it with a scheme of its own,
and no fragment declares that route today.

`SEC-010` is new. A route that a third-party service calls verifies a secret that
the hub registered with that service. `handler.go` already does this for the
Telegram webhook, and no document held the guard. `TG-001` names the header that
carries it, because `TG` owns Telegram and a rule of `SEC` names no one service.

The rule gives `telegram.yaml` a scheme, which `RES-002` registers against
`Z-104` beside the cookie of `hub.yaml`. `Z-104` names bearer and `oauth2` only.

`ERR-003` gains `member-unknown`. `Z-109` and `Z-111` ask a document to say how a
route handles a member that the schema does not declare, and `Z-111` forbids the
`additionalProperties: false` that would answer it in the schema, so the registry
answers it.

`ERR-003` gains `cursor-stale`. `RES-006` refuses a cursor whose sort or filters
differ from the request, and no type answered that condition, so a client could
not tell a cursor it must retire from one it built wrong. Both answer 400,
because a client reads the type and not the status.

`ERR-003` gains `webhook-secret-invalid`. `ERR-001` binds the Telegram webhook to
answer a problem, and the only 401 in the registry says that the request carries
no active user record, which is false for this failure. The handler writes a bare
status today, with no body at all, and `ERR-001` forbids that.

No `CFG` rule changes. `RES-005` needs one absolute address of the deployment,
and a key is an implementation detail that a feature declares, not a rule of the
guideline. The feature that adds the page declares it.

That feature also corrects two defects of the address that the hub holds today.
`telegram/handler.go` registers the webhook as `Server.Hostname` joined to the
path, with no scheme at all. `handler/mcp.go` holds `https://` as a constant and
joins the same bare name, so a deployment on another scheme reports an address
that no client reaches.

`BLD-001` gains a row for the merge and the lint of `RES-008`. Its table names no
artifact under `specs/` or `build/` today. The row reads `None today` until
tranche 3 adds `pnpm api:build` and `pnpm api:lint`.
`CLI-001` gains the sentence that keeps a build step out of the command tree.

`SPC-001` keeps `api.yaml` in the feature directory. `SPC-002` gains the merge:
the fragments of every feature build the document of the API that holds their
routes, and each document carries the meta block of `RES-007`. A fragment
declares which of the three it belongs to.

`DAT-010` is new. A column that holds a point in time is `TIMESTAMPTZ`, and it
stores UTC.

The column of the last write stays `updated_at`. `Z-235` governs that name, it
asks a date property to end in `_at`, and this one does. `Z-174` names `id`,
`xyz_id`, and `e_tag` only, and carries `modified_at` in an example of its own,
which compels nothing. `Z-118` still turns the member of the body from
`updatedAt` into `updated_at`, so one name reaches the column, the field of the
model, and the wire.

`DAT-011` is new. The core migrations run before the migrations of any plugin.

`GLO-page`, `GLO-cursor`, `GLO-flow-id`, `GLO-binding`, `GLO-handoff`,
`GLO-transport-route`, `GLO-api-document`, `GLO-api-fragment`, and
`GLO-shipped-client` are new terms.

`LNG-013` gains the sentence that a guideline does not divide, the three remedies
that replace a divide, and the row that targets a decision record at 400 lines. `TRC-002` binds one prefix to one file, so no
convention could divide a guideline. `rest.md` passes the target that `LNG-013`
holds. This decision asks the owner for that exception until a later pass cuts it.

| Term      | Meaning                                                                    |
| --------- | -------------------------------------------------------------------------- |
| page      | One answer of a collection. It holds the items and the links of `RES-005`. |
| cursor    | The opaque value that names the position of a page. `RES-006`.             |
| flow identifier | The value that joins every record of one request. `ERR-006`.         |

The prefix register in `traceability.md` gains the row for `RES`.

`specs/README.md` divides a specification at 750 lines, which is the target of
`LNG-013` plus half. It read 500, which was the target itself.

### Specifications to examine

| Document                                  | What the examination found                                                                                      |
| ----------------------------------------- | ---------------------------------------------------------------------------------------------------------------- |
| `features/pcap/spec.md`, `PCAP-DD-002`    | The decision reverses. `Z-216` exempts the key of a map, and `Capabilities` is not a map: it declares ten properties, each with its own value schema. An object that declares a property is not a map, whatever its keys look like. Every key is a property name, so `telegramCommands`, `telegramConversations`, and `mcpTools` become snake case. |
| `apps/hub/internal/registry/capability.go` | The ten keys are Go constants, and `registry.PluginEntries` feeds both the HTTP answer and the `list_plugins` MCP tool. Renaming three for `Z-118` renames them in the MCP payload too. `RES-003` exempts that payload from the rule, and it does not forbid the change, so one name serves both. |
| `features/pcap/api.yaml`                  | The three camel case keys change. `additionalProperties: false` on `Capabilities` breaks `Z-111`, which forbids a schema to close itself, and it goes. |
| `features/extid/api.yaml`                 | `nullable: true` on `firstName` and `lastName` is a keyword of OpenAPI 3.0 that 3.1 ignores, so each takes `type: ["string", "null"]`. `ProviderState` in the same file already does. Three renamed paths, and five camel case members of `CurrentUser` and `ProviderState`: `firstName`, `lastName`, `displayName`, `pictureUrl`, `attachedAt` become snake case, in the schema and in each example. |
| `features/websess/api.yaml`               | `/auth/logout` becomes `DELETE /auth/sessions/self`. |
| `features/extid/spec.md`, `spec-scenarios.md` | `GET /auth/link` becomes `POST /auth/identities`, and the deck reads the address of the IdP from a body in place of a redirect. Every scenario that names the old route changes. |
| `apps/deck/src/lib/api/client.ts` and five callers | `buildAuthUrl`, `buildInviteUrl`, and `buildLinkUrl` stop composing an address. Each one posts and returns the address of the body. The five call sites await it before they assign `window.location.href`. |
| `features/websess/spec.md`                | `GET /auth`, `GET /auth/me`, and `POST /auth/logout` change. The rule that keeps the sign out public stands.      |
| `features/ident/spec.md`                  | `GET /auth/invite` becomes `POST /auth/invitation-redemptions`.                                                  |
| `features/pset/spec.md`, `api.yaml`       | `SettingsValues` and `SettingsInput` declare `additionalProperties` and no property, so `Z-216` puts their keys outside `Z-118`. `authToken` and `vehicleId` stand, and no migration rewrites the stored rows. `SettingsValues` is a map at the top level of a response, which `Z-110` forbids, and `RES-002` registers that until an envelope lands. The `PUT` answers 204 with no body, and `Z-110` binds no request body. |
| The `Problem` schema of every fragment    | `type` and `instance` declare `format: uri`, and `ERR-002` and `ERR-006` make each one a relative reference. `Z-238` gives `uri-reference` for that. Fifteen literal `urn:maroid:problem:` strings across the five fragments take the new form, in a description and in an example value. |
| `templates/api.yaml`, the resource schema | `format: uuid` on an identifier. `Z-144` asks a UUID identifier to carry no format, and the template propagates it into every new feature. |
| The five `api.yaml` with no `default`     | `Z-151` covers a standard failure with one `default` response that names the problem body. None declares it, so 500 is undocumented everywhere. |
| `features/mcphub/spec.md`, `api.yaml`     | `/mcp` keeps its path and its payload. `RES-003` puts the fields of the MCP protocol outside the reach of rule 118. |
| The five `api.yaml`                       | Each names the document of `RES-007` that holds its routes, and gains the `uid` permission on each operation that needs a caller, and a format on each number. A fragment carries no meta block, and `SPC-002` gives the merged document that one. |
| The security scheme of every fragment     | Three fragments answer one question three ways. `extid` declares `apiKey`, which `ADR-0004` made correct. `pcap`, `pset`, and `templates/api.yaml` still declare a bearer, which no web route uses. `mcphub` declares `http` bearer, which `RES-009` confirms for `mcp.yaml`. `websess` declares none. `RES-009` gives the one answer for each document. |
| `templates/api.yaml`                      | It gains the page object, the cookie scheme, the `uid` permission, and the snake case example. `<Resource>` and `<Resource>Input` collapse into one schema that carries `readOnly` and `writeOnly`, which `Z-252` asks for and which the template already uses on `id`. `pset` splits the same way.                     |

### Migration

Five tranches. Each one lints, builds, and ships on its own.

| # | Tranche              | Holds                                                                                                          |
| - | -------------------- | -------------------------------------------------------------------------------------------------------------- |
| 0 | The database         | The core migration that replaces the body of `set_updated_at`; the deterministic order of `buildMigrationPlan`; and the four columns of `telasi`, `pensions`, and `tbilisi-energy` that hold a bare `TIMESTAMP`, each altered `USING <column> AT TIME ZONE 'UTC'` because the producing code wrote UTC. `DAT-010`, `DAT-011`. It ships first, because `DAT-011` orders every migration that follows. |
| 1 | The payload          | `.golangci.yaml` sets `tagliatelle` to `json: camel` under `default: all`, so the linter refuses this tranche until the rule flips and every exempt tag of `RES-003` gains an exception. Snake case (118), a number format (171), the null rules (122 to 124), an enum in upper snake case (240), the date and the zone (169, 238), the map through `Z-216`. No migration touches `plugin_settings.fields`, and 15 readers change. |
| 2 | The collection       | An object at the top level (110) and the page object (248) together, because a list route keeps its bare array until the page replaces it. The cursor (160), the links as absolute addresses (161, 217), the conventional parameter (137), the sort with `+` and `-` (137, 154).                                     |
| 3 | The meta and the lint | The meta block of each document (218, 116, 215, 219), the permission on each operation (105, 225), the split and the merge (101, 219), `vacuum` in the build.                                                             |
| 4 | The header           | `ETag` with `If-Match` (182), `Idempotency-Key` (230), the cacheable route (227). `X-Flow-ID` ships with tranche 1.                                                                                       |

The route table of `API-003` lands with tranche 1, because the deck reads a body
in place of a redirect and the two changes reach the same files. `ERR-002`,
`ERR-006` and `LOG-009` land with it too, so no rule of this decision binds code
that contradicts it for longer than one tranche.

`specs/templates/api.yaml` lands with tranche 1, so that a feature started before
tranche 3 turns on the linter produces a fragment that passes it.

This migration needs no deprecation window, and the next one will. Every
consumer of the API sits in this repository today, and one commit reaches all of
them. No document reaches a reader today, and no identifier of an API has been
published, so no reader outside the repository holds a contract to break.

That ends when this decision lands. `RES-007` declares the audience
`external-public`, so `Z-106` binds from the first version that reaches a
reader, and the deprecation rules gain a subject that they do not have today.
`RES-011` gives the window, `RES-010` binds the three clients, and `RES-002`
holds `Z-185`, `Z-188`, and `Z-193`. The route table above is the last free
rename.

Every document ships at `1.0.0`, not at `0.1.0`. `Z-116` reads `0.y.z` as a
design that still moves, and these routes stop moving here.

#### The zone of `set_updated_at`

No column renames. One core migration replaces the body of the function:

```sql
-- old: NEW.updated_at = (now() AT TIME ZONE 'utc');
NEW.updated_at = now();
```

`CREATE OR REPLACE FUNCTION` carries the change, so no table alters, no trigger
recreates, and the five tables that call it need no migration of their own.

`now() AT TIME ZONE 'utc'` answers a `TIMESTAMP` without a zone. The column is
`TIMESTAMPTZ`, so PostgreSQL reads that value in the zone of the session and
shifts it by the offset of that zone. No code sets the zone of the session. The
`DEFAULT NOW()` of `created_at` carries no such conversion, so an insert and an
update write two different instants on any server that does not run in UTC.

`DAT-011` does not gate this change, and it corrects a defect of its own.
`buildMigrationPlan` in
`apps/hub/internal/migrator/migrator.go` builds the order from
`slices.Collect(maps.Keys(migrations))`, and `core` sits in that map beside
every plugin. Go randomizes the iteration of a map, so `migrate up --target all`
orders the components differently on each run. On a fresh database a plugin can
create a trigger against `set_updated_at()` before the core migration that
creates that function.

### Result

- Positive: 144 rules answer the five open questions of the context. 24 of them
  wait for the first event that Maroid publishes.
- Positive: a collection can grow a member without a breaking change.
- Positive: a linter checks the contract, and no reviewer holds the rules.
- Positive: a person who deploys Maroid writes a client against a published
  document, and `Z-106` keeps that client working.
- Negative: `PCAP-DD-002` reverses, and three capability names change with it.
  The other seven are already one lowercase word.
- Negative: the wire carries two cases. A member is snake case under `Z-118`,
  and the key of a map keeps its own under `Z-216`, so a stored settings field
  answers `authToken` beside a `created_at` next to it.
- Negative: two rules of `ADR-0005` change, one day after the owner accepted it.
  `ERR-006` refused an identifier that a caller supplies, and `Z-233` requires
  one. The character set bounds the value. It does not remove the risk.
- Negative: six routes change, and the deck changes with them.
- Negative: the audience `external-public` ends the free rename. `Z-106` binds
  the next change, and `RES-011` puts a window of 90 days and two headers before
  a route goes. This decision is the last one that moves a path at no cost.
- Work that follows: the five tranches. The guideline edits land with this
  decision.
