---
id: RES
title: The REST conventions
type: guideline
status: active
created: 2026-09-22
updated: 2026-09-24
scope: [apps/hub/internal/handler/, apps/hub/internal/middleware/, plugins/, libs/api-client/, libs/plugin-sdk/, apps/deck/, specs/features/*/api.yaml, specs/templates/api.yaml, specs/api/, build/openapi/]
related: [API, ERR, SPC, DAT, EXT, SEC, OWN, UI, TG, MQT, CFG, BLD, CLI]
---

# The REST conventions

## RES-001

The Zalando RESTful API Guidelines bind every HTTP API that Maroid designs.

| Field    | Value                                                  |
| -------- | ------------------------------------------------------ |
| Source   | `github.com/zalando/restful-api-guidelines`            |
| Revision | `41b74fa6`                                             |
| Date     | 2026-07-07                                             |

Write a Zalando rule as `Z-<number>`: `Z-118`, `Z-248`. A rule of this file wins
over a Zalando rule only when `RES-002` names it.

A member name, a path, and a description use U.S. English. `Z-103`.

A point in time travels as UTC, in the `date-time` format of `Z-169`, with the
`Z` suffix and never a numeric offset. The hub reads no zone of a person and
renders no local time. A client converts for the reader that it serves.

The event chapter does not bind. It governs an event that a service publishes,
and Maroid publishes none. `MQT-001` gives a plugin a subscriber, and a device
outside Maroid publishes the message. The chapter binds on the day that Maroid
publishes its first event.

## RES-002

This table holds every deviation, except the derived address that `RES-004`
registers. Every rule that neither table names binds in full.

| Rule           | Deviation                                                                                                                               |
| -------------- | ---------------------------------------------------------------------------------------------------------------------------------------- |
| `Z-223`, `Z-224` | A hostname comes from the deployment. `Z-224` asks for one under `zalandoapis.com`, and no name resolves for every reader of Maroid. The rule binds every document at the level of a must, because `RES-007` gives each one the audience `external-public`. |
| `Z-192`        | `RES-008` writes each document into the build tree, and a release carries it. The rule asks for the `/zalando-apis` directory of a deployment, which feeds a portal that Maroid has no equal of. It binds every document, because it exempts a component-internal API only. |
| `Z-129`, `Z-134` | `RES-004` exempts the one address that a client derives from a standard. `.well-known` opens with a stop, which `Z-129` forbids, and `oauth-protected-resource` is singular, which `Z-134` forbids. |
| `Z-134`        | `callback`, `schema`, `api`, `ui`, `auth`, `telegram`, and `webhook` are singular, and Maroid picks each name. Each one names one thing, and none is a collection. |
| `Z-148`, `Z-149` | `GET /auth/callback` writes, and a `GET` reads. RFC 6749 section 4.1.2 makes the redirection endpoint a `GET` that carries `code` and `state` as query parameters, and the browser arrives there by redirect. Maroid picks the address of that route, and it does not pick the method. |
| `Z-150`        | `ERR-003` answers 422 for a body that fails validation and for settings that fail their schema. `Z-150` marks 422 do-not-use and prefers 400. A client of Maroid branches on the problem type, and the two conditions differ from a body that does not parse, which already answers 400. |
| `Z-150`, `Z-251` | `GET /auth/callback` answers 302. `Z-150` marks 302 do-not-use and `Z-251` advises against a redirection code. The browser arrives at that route by redirect and leaves by one, which RFC 6749 section 4.1.2 fixes. |
| `Z-217`        | The first sentence of `Z-217` is a must, and `RES-005` meets it with an absolute address. The deviation is from the second, a should: it asks for the scheme, the host, and the port of the request. A proxy sets the `Host` header, and a caller sets it behind a proxy that passes it, so a link built from it points where the caller chose. |
| `Z-110`        | `GET /plugins/{id}/settings` answers a map at the top level, which `Z-110` forbids because a map grows no member. The settings of a user hold that shape, and an envelope changes the route and the deck together. `Z-110` binds a response body, so the request body of the `PUT` is not in scope, and that `PUT` answers 204 with no body at all. |
| `Z-143`        | `Z-143` asks that every intermediate path resolve. `/plugins/{id}`, `/auth`, `/.well-known`, the mount prefix of `API-004`, and the asset prefix of `SEC-006` answer nothing. `/plugins/{id}` would repeat one row of the list that `/plugins` answers, and the other four name no resource of their own. |
| `Z-144`        | `DAT-009` gives every row a UUID version 7, and `Z-144` discourages a UUID as the primary key of configuration data. Version 7 sorts by time, so the primary key answers the keyset page of `RES-006` at no index cost, which is the case that the hint of `Z-144` concedes. |
| `Z-186`        | The rule asks every external partner to consent to the window before it uses the API. `RES-007` gives the audience `external-public`, so no deployment can tell whether a consumer is a partner. The condition cannot be discharged, rather than being unmet. `RES-011` publishes the window, which answers the first sentence of the rule and not the second. |
| `Z-185`        | The rule asks for the consent of every client before a route goes. `RES-007` gives the audience `external-public`, so a deployment cannot enumerate its clients. `RES-011` publishes a window in place of the consent. |
| `Z-188`, `Z-193` | The rules ask a producer to monitor the usage of an API and of a route that a sunset date names. Maroid collects no usage of its own, and a deployment reads its own log. |
| `Z-135`, `Z-142` | `/plugins/{id}/api` and `/plugins/{id}/ui` carry the two most general segments in the tree. `API-004` makes the first the base path of the API of a plugin, which is where `Z-135` reaches it. `SEC-006` gives the second. `API-004` builds the prefix from the plugin identifier so that two plugins cannot collide, and the segment says which surface answers. |
| `Z-156`        | No middleware of `API-002` compresses an answer. The route that would gain most serves the bundle of a plugin, and a deployment puts a proxy in front of the hub that compresses for it. |
| `Z-102`        | Maroid publishes no user manual, so no document carries `externalDocs`. The repository holds the documentation that a reader needs. |
| `Z-183`        | `X-Telegram-Bot-Api-Secret-Token` is not on the list of proprietary headers that the rule allows. Telegram names the header, `TG-001` records it, and `SEC-010` reads it. `X-Flow-ID` is on the list, and `ERR-006` uses it. |
| `Z-105`        | The rule gives the pseudo permission `uid` to every endpoint that needs no permission of its own. `uid` names a user behind the call. Telegram calls `/telegram/webhook` under the secret of `SEC-010`, and a route that declares an empty requirement carries no scheme to hang a permission on, so neither carries `uid`. |
| `Z-104`        | The rule names the `http` bearer scheme and the `oauth2` scheme. `hub.yaml` guards a route with the session cookie of `SEC-005`, which `ADR-0004` chose, and `telegram.yaml` with the header of `SEC-010`. OpenAPI describes each as `apiKey`. `mcp.yaml` uses bearer and meets the rule. Separately, these routes carry no credential, and each one must: the sign in, the sign out, the redemption of an invitation, `GET /auth/callback`, the two discovery routes, and the assets of a plugin that `SEC-006` publishes. A person holds no session before a sign in, and RFC 9728 asks a client that holds no account to read discovery. |

A rule whose condition Maroid does not meet is not a deviation, and takes no row.
A rule that offers an optional capability waits for a client that asks for it,
and `Z-157` and `Z-158` are two. `Z-114` asks for a version in the media type
when a version is unavoidable,
and Maroid versions no route today, so the rule waits. The first change that `Z-106`
forbids starts a version, and `RES-011` gives the window that precedes it.

## RES-003

The standard governs the API that Maroid designs: every route of `API-003` and
every route that a plugin declares under `API-004`.

It does not reach:

| Surface                                                    | Governed by                    |
| ---------------------------------------------------------- | ------------------------------ |
| The request and the answer of a third-party service        | `EXT`                          |
| The payload of an MQTT message that a device publishes     | `MQT`                          |
| The transport route of MCP, `/mcp` today, and every payload it carries | The MCP schema     |
| The payload of a Telegram update, and the answer to it     | `TG`                           |
| A keyword of JSON Schema, `maxLength` among them           | JSON Schema                    |
| A parameter that OAuth 2.0 or OpenID Connect names         | RFC 6749, OpenID Connect Core  |
| The key of a JSON map, a stored settings field among them  | `Z-118` only. `Z-216`.         |

A DTO in `plugins/*/dto/api-client.go` describes the answer of a third-party
service. It keeps the names that the service sends.

A transport route carries JSON-RPC or the update of a bot. It answers no
resource of Maroid, so no rule about a resource reaches it: not the name of a
path, not the shape of a body, not the case of a member. The result of an MCP
tool travels inside that body, so a member of that result keeps the case that the
MCP schema gives it. The configuration holds the address of the route.

A rule about the specification still reaches it. `mcp.yaml` of `RES-007`
describes `/mcp`, and it carries the meta block, the scheme of `RES-009`, and
the problem body of a transport failure that `ERR-001` gives.

`Z-216` puts the key of a map outside `Z-118`, and asks the schema of that map
to record it in the description. A map declares `additionalProperties` with one
value schema and no property of its own. An object that declares a property is
not a map, whatever its keys look like, and `Z-118` reaches every one of them.

**Why:** A rule cannot bind a shape that Maroid does not choose. A name that
Maroid rewrites on the way in costs one mapping and hides the source.

## RES-004

A route keeps an address that breaks `Z-129` or `Z-134` when a client derives
that address from a standard. The table holds every one.

| Route                                      | Derived by                                          |
| ------------------------------------------ | --------------------------------------------------- |
| `/.well-known/oauth-protected-resource*`   | RFC 9728, and the IANA registry of well known URIs. |

A registered address is not a derived one. Maroid picks the address of a redirect
and of a webhook, and gives each to the IdP and to Telegram, so either one
renames and neither takes a row.

**Why:** A client that derives an address never reads it from a configuration,
so a rename breaks it.

## RES-005

A collection answers a page object. `Z-110` and `Z-248` give the shape, and
`Z-165` allows a link to carry a bare address in place of the object of `Z-164`.

Maroid writes `items`, `self`, `first`, `next`, and `prev`. `items`, `self`, and
`first` are required, and `items` holds an array that may be empty. `Z-248` marks
`items` required and names no other. Every page of Maroid carries `self` and
`first`, so the schema requires them too.

An absent link carries a meaning, and a page never omits one for another reason.
`next` is absent on the last page and `prev` on the first, which is what `Z-248`
asks for. A client that finds no `next` reads no further page.

`last` stays out. A keyset cursor reaches the last page only through a count or a
descending scan of the whole collection, and `Z-254` advises against the count. Add `last` when a
client asks to jump to the end. A page carries no total either. `Z-254`.

`query` stays out. `Z-248` returns the applied filters so that a client can put
them in the body of the next request, which a `GET` with a body needs. Every
route of Maroid takes its filters in the query string, so the links already
carry them. `Z-248` permits the omission for a plain `GET`.

```json
{
  "items": [],
  "self": "https://hub.example.com/plugins?limit=20",
  "first": "https://hub.example.com/plugins?limit=20",
  "next": "https://hub.example.com/plugins?limit=20&cursor=b3..."
}
```

A link is an absolute address that the hub builds from the external address of
the deployment, and never from the `Host` header. The feature that adds the page
declares the configuration key that holds it. `RES-002` holds the deviation
from `Z-217` that the second sentence needs.

`limit` gives the count of items that a client asks for, and `Z-137` gives that
name. This guideline sets the default at 20 and the cap at 100. No Zalando rule
fixes either number.

A collection that a deployment bounds answers every item in one page, and neither
number reaches it. `Z-159` asks for a page when a collection can pass a few
hundred entries, and one of these cannot.

A route that answers the rows of the acting user alone records that in its
description. `OWN-006` filters every scoped collection in the database, and
`Z-226` asks the specification to say so.

Each route declares in its operation the set of fields that `sort` takes, and it
sorts by `id` when it declares none. A field may carry `+` or `-`, and a field with no prefix ascends. A `sort` that
names a field outside that set answers `/problems/http/request-invalid`, and the
`detail` names the field. `Z-137`, `Z-154`.

## RES-006

A cursor is opaque. A client passes back the value that it received, and it
constructs none. `Z-137`.

The hub encodes a cursor as `base64url` over a JSON object with five members: the
sort, the direction, the filters, the value of each sort field at the boundary
row, and the identifier of that row. `next` carries the last row of the page with
the direction `forward`, and `prev` carries the first row with `backward`.

Every link of `RES-005` repeats the `sort` and the filters of the request. A
request that carries a cursor and no `sort` takes the sort of the cursor.

A cursor that Maroid did not produce answers `/problems/http/request-invalid`. A cursor whose sort or filters differ
from the request is stale, and answers `/problems/http/cursor-stale`.

The page reads with a keyset, never with an offset. `DAT-009` gives every row a
UUID version 7, which sorts by time, so the primary key answers the page and the
query reads no row that it skips. `Z-160`.

**Why:** An offset scans the rows that it discards, and it skips a row when a
write lands between two pages.

## RES-007

Maroid serves three APIs. Each one carries its own meta block. `Z-218`, `Z-116`,
`Z-215`, `Z-219`.

| Document        | Holds                                       | `x-api-id`                             | `x-audience`       |
| --------------- | ------------------------------------------- | -------------------------------------- | ------------------ |
| `hub.yaml`      | Every route of `API-003` that the other two do not hold | `01a0cae5-eb36-777a-824e-6e7e28d7a6b1` | `external-public`  |
| `mcp.yaml`      | `/mcp` and `/.well-known/oauth-protected-resource*` | `01a0cae5-eb36-777a-824e-71285629ec17` | `external-public`  |
| `telegram.yaml` | `/telegram/webhook`                         | `01a0cb39-8680-779f-bb60-05116c522d22` | `external-public`  |

```yaml
info:
  title: The Maroid hub
  description: The HTTP API of the hub and of every plugin that it loads.
  version: 1.0.0
  x-api-id: 01a0cae5-eb36-777a-824e-6e7e28d7a6b1
  x-audience: external-public
  contact:
    name: The owner of Maroid
    url: https://github.com/abgeo/maroid
    email: <the address that the owner publishes>
```

Every document declares `external-public`. A person who deploys Maroid writes a
client against the instance that they run, so the consumer set has no boundary.
`Z-219` allows one audience for one document, and a smaller group sits inside a
wider one, so the widest value answers all three.

The three stay apart for a reason beyond the audience. Each describes a surface
that moves on its own schedule: `mcp.yaml` with the revision of MCP that the hub
implements, `telegram.yaml` with the Bot API, and `hub.yaml` with Maroid. One
document could not carry one version.

`Z-218` asks the contact block for a name, a URL, and an email. The deployment
publishes an address that it is willing to show, because every document reaches
any reader.

`x-api-id` is permanent, and no second document repeats one. `version` follows
semantic versioning and describes the document, not the build of the hub.
`Z-106` binds from the day that a version reaches a reader.

## RES-008

`SPC-002` gives the fragment, the merge, and the document that each fragment
belongs to. This rule gives the tools.

Every operation takes the `X-Flow-ID` parameter and every answer the header of
the same name, both from `specs/api/components.yaml`. `Z-233` asks a
specification to declare the restriction that `ERR-006` applies.

`pnpm api:build` merges the fragments and writes each document to
`build/openapi/`. It groups the fragments by `x-maroid-document`, adds the
components of `specs/api/components.yaml`, and writes the meta block of
`RES-007` on the result. `telegram.yaml`, which no fragment feeds, takes the same treatment: the build resolves
every reference to `components.yaml` into the result and adds the components it
used. That document carries its own meta block. `Z-234` asks each published
document to stand alone. `components.yaml` feeds every merge and is no document
of its own. The build runs
this, and the hub does not: `CLI-001` keeps a build step out of the command tree.

`pnpm api:lint` lints each document, and the build fails on an error. Only a rule
of that severity blocks, and the recommended set carries more warnings than
errors, so a warning reports and does not stop the build:

```
vacuum lint -r specs/api/vacuum-ruleset.yaml -n error build/openapi/hub.yaml
vacuum lint -r specs/api/vacuum-ruleset-transport.yaml -n error \
  build/openapi/mcp.yaml build/openapi/telegram.yaml
```

Two rulesets, because `RES-003` drops every rule about a resource for a transport
route, and both transport routes sit in a document of their own. A ruleset also disables a rule of the linter
that contradicts a rule that `RES-001` binds, and the comment names both. A
ruleset reaches a whole document, so an exemption that one route of a document
must not receive takes a rule of its own and never a disable.
Every other exception lands after a real run reports the rule, and each one cites
the row of `RES-002` or the surface of `RES-003` that registers the reason.

Tranche 3 of `ADR-0006` adds both scripts. `BLD-001` carries no command for
this artifact until then.

**Why:** A fragment holds the routes of one feature next to the statements that
demand them. `Z-101` asks the designer for one self-contained document, and a
client never reads a fragment.

## RES-009

Each document declares the scheme that guards it. `Z-104`.

| Document        | Scheme                                                        | Rule      |
| --------------- | ------------------------------------------------------------- | --------- |
| `hub.yaml`      | `apiKey` in the session cookie that `SEC-005` gives           | `SEC-005` |
| `mcp.yaml`      | `http` bearer, a token that the IdP issued                    | `SEC-002` |
| `telegram.yaml` | `apiKey` in the header that `TG-001` names                    | `SEC-010` |

Maroid grants no scope, and a verification reads the audience of a token and
never a scope. `SEC-002`.

An operation whose caller is the acting user declares the pseudo permission
`uid`, whatever the access column of `API-003` says. `DELETE /auth/sessions/self` is public in `API-003`
and its caller is the acting user, so it declares `uid` too. `/telegram/webhook`
resolves a user from the update and its caller is Telegram, so it declares none.
`Z-105` gives it to an operation whose authorization sits at the level of one
object, and `OWN-006` puts it there. The policy of the database answers each
row, so no permission of the route can.

A route that carries no credential declares an empty requirement. `RES-002` holds
its deviation from `Z-104`.

A route that takes no credential and still resolves an acting user declares both
alternatives, the empty requirement first, and the permission attaches to the
second. `DELETE /auth/sessions/self` is the one route of `API-003` that does.

The day that Maroid grants a scope, the name follows `Z-225`:
`maroid-hub.<access-mode>`, with `read` or `write` as the mode.

## RES-010

`RES-001` binds the API that Maroid designs. These rules bind a shipped client
instead.

| Rule    | Level  | Asks a client to                                                                    |
| ------- | ------ | ------------------------------------------------------------------------------------ |
| `Z-108` | Must   | Ignore a member it does not know, keep it in a body that it sends back, accept a new value of an open enumeration, handle a status that no operation declares, and follow a 301. |
| `Z-190` | Should | Read the `Deprecation` and the `Sunset` header, and report what it finds.           |
| `Z-191` | Must   | Start no call to a route that a document already marks deprecated.                 |
| `Z-189` | Should | Stop calling a route before the `Sunset` date that its answer names. `Z-189` binds the producer that sends the date, and this row binds the client that reads it. |

**Why:** `RES-005` lets a collection grow a member without a new version. That
holds only while every client of Maroid ignores the member it does not know.

## RES-011

A route that goes carries `deprecated: true` in its API document. It carries a
`Sunset` date at least 90 days after the release that first publishes that flag.
Until that date it answers the `Deprecation` and the `Sunset` header of `Z-189`,
and it keeps its behavior.

The description of the route says why it goes, what replaces it, and how a client
moves. `Z-187` asks for all three.

No window binds a route that goes before the first version of a document reaches
a reader. `ADR-0006` names the routes that go under this sentence.

**Why:** `Z-185` asks the producer for the consent of every client before a route
goes, and `RES-007` gives the audience `external-public`, so the consumer set has
no boundary and no producer can enumerate it. A published window replaces the
consent that nobody can collect. `RES-002` holds `Z-185`, `Z-186`, `Z-188`, and
`Z-193`, which this window does not satisfy.

## Retired identifiers

This file has no retired identifier.
