---
id: ERR
title: Errors
type: guideline
status: active
created: 2026-09-21
updated: 2026-09-25
scope: [apps/hub/internal/handler/, apps/hub/internal/middleware/, libs/rest/, libs/api-client/, plugins/]
related: [API, DAT, LOG, OWN, PLG, RES, UI]
---

# Errors

## ERR-001

An error response carries the media type `application/problem+json` and a body that
obeys RFC 9457. The body holds these members:

| Member     | Required | Holds                                                          |
| ---------- | -------- | -------------------------------------------------------------- |
| `type`     | Yes      | The type of the problem. `ERR-002` gives the grammar.          |
| `title`    | Yes      | A short sentence in English. It is constant for the type.      |
| `status`   | Yes      | The HTTP status of the response. The same number.              |
| `detail`   | No       | One sentence about this occurrence. `ERR-005` bounds it.       |
| `instance` | No       | A URI reference that names it. `ERR-006` gives the form.       |

```json
{
  "type": "/problems/http/request-invalid",
  "title": "The request is not valid.",
  "status": 400,
  "detail": "The redirect parameter is absent.",
  "instance": "/flows/0199c5e2-8f3a-7c21-9b4e-2f6a1d0c5e77"
}
```

A problem always names a type, and never `about:blank`. The hub does not negotiate:
a failure answers this media type whatever the request accepts.

Every failure that the hub writes obeys this rule: every route of `API-003`, the
`NotFound`, `MethodNotAllowed` and panic answers of the router, a transport failure
of `/mcp`, the Telegram webhook, and the asset route of a plugin. `API-007` holds
the headers that a status adds. A JSON-RPC error inside an MCP tool call is not an
HTTP failure, and keeps its own shape.

**Why:** A client reads one member to learn what happened. It parses no sentence.

## ERR-002

`Z-<number>` names a rule of the standard that `RES-001` binds.

A problem type is a relative reference with the form `/problems/<owner>/<slug>`.

The owner names who defines the failure, not who answers it. The slug holds
lowercase letters, digits, and a hyphen.

| Owner       | Defines                                                          |
| ------------- | ------------------------------------------------------------------ |
| `http`      | A failure of the protocol. Any component answers one. `ERR-003`. |
| `hub`       | A failure of a domain of the hub. `ERR-003`.                     |
| A plugin id | A failure of the domain of that plugin.                          |

```
/problems/http/not-found
/problems/hub/settings-invalid
/problems/dev.maroid.jasmine/watering-overlap
```

A slug is permanent. Never reuse a slug for a second meaning, and never change the
meaning of a slug that a deployment answers with. A plugin owns every slug under its
identifier, as `DAT-001` gives it one schema.

**Why:** `Z-176` proposes a relative reference and discourages an absolute one.
A type never resolves, so no deployment must serve a page for it, and a client
compares one constant.

## ERR-003

A type names a condition that HTTP itself defines, or a mechanism that every
component runs, or a mechanism of one domain of the hub. The first two live in
`libs/rest`, and the third in the hub. Which component answers it today
decides nothing.

A cursor is a mechanism that every component runs: `RES-005` gives a page to
every collection, and a plugin answers one. `PLG-007` denies a plugin the hub, so
the type of a stale cursor lives in the library.

`libs/rest` holds these, under `/problems/http/`.

| Slug                     | Status | Title                                               |
| ------------------------ | ------ | ----------------------------------------------------- |
| `request-invalid`        | 400    | The request is not valid.                           |
| `body-invalid`           | 400    | The body is not a JSON object.                      |
| `member-unknown`         | 400    | The body carries a member that the schema does not declare. |
| `cursor-stale`           | 400    | The cursor does not match this request.             |
| `access-denied`          | 401    | The request carries no active user record.          |
| `not-found`              | 404    | The resource does not exist.                        |
| `method-not-allowed`     | 405    | The method does not reach this resource.            |
| `precondition-failed`    | 412    | The record changed after the client read it.        |
| `validation-failed`      | 422    | The body failed validation.                         |
| `internal`               | 500    | The request failed.                                 |

The title of `access-denied` says how Maroid reads the condition. The condition is
the one that HTTP gives every 401, so the type is of the first kind.

A record that is absent and one that belongs to another user both answer
`not-found`, because the policy of `OWN-006` hides the second from the query.

`cursor-stale` and `request-invalid` share a status. 409 names a conflict with
the state of a target resource, and `Z-150` applies it to a method that changes
something. A cursor arrives on a `GET`, so `Z-220` leaves 400 as the most
specific code. A client reads the type and not the status, which `ERR-001` gives
as the reason the registry exists.

`apps/hub/internal/domain/problems` holds these, under `/problems/hub/`.
Each names a mechanism that Maroid runs: an allowlist, a settings schema, an
identity, the secret of the bot.

| Slug                     | Status | Title                                               |
| ------------------------ | ------ | ----------------------------------------------------- |
| `network-not-allowed`    | 403    | The caller is not on the network allowlist.         |
| `settings-absent`        | 404    | The plugin declares no settings.                    |
| `identity-last`          | 409    | The last external account cannot be detached.       |
| `webhook-secret-invalid` | 401    | The update carries no valid secret of the bot.      |
| `settings-invalid`       | 422    | The settings do not match the schema.               |

A body that carries a member which the schema does not declare answers
`member-unknown`. `Z-109` and `Z-111` ask a document to say how a route handles an
unknown member, and this is the answer for every route that takes a body. A
schema declares no `additionalProperties: false`, which `Z-111` forbids, so the
description of the body records the rejection and the handler performs it.

A plugin answers with a type of the first table for a failure that it names, and
declares a type of its own only for a failure of its own domain. The `api.yaml` of
its feature holds that declaration.

## ERR-004

A problem that reports a failed field carries `errors`. Each item holds `detail`
and `pointer`. `pointer` is a JSON Pointer in the fragment form of RFC 6901, so it
starts with `#` and it reaches a field at any depth.

```json
{
  "type": "/problems/http/validation-failed",
  "title": "The body failed validation.",
  "status": 422,
  "errors": [{ "detail": "is required", "pointer": "#/address/city" }]
}
```

`errors` extends the five members of `ERR-001`, in the shape that RFC 9457 gives in
its own example. A client ignores an extension member that it does not know.

## ERR-005

`detail` holds a sentence that a person reads. It never holds the text of a Go
error, a SQL statement, a stack, a path of the file system, a token, or a secret.
A problem of the type `internal` carries no `detail`. The log holds the cause:
`LOG-005` puts it in an attribute, and `ERR-006` joins the record to the response.

**Why:** A body reaches the browser, and it travels further than a log.

## ERR-006

Each request carries a flow identifier. The first middleware reads it from the
`X-Flow-ID` header, and generates a UUID version 7 when the request carries
none. `API-002` gives the order, and `Z-233` requires the header.

The middleware bounds the value that a caller sends. It removes every character
outside `[a-zA-Z0-9/+_=-]`, cuts the result to 128 characters, and generates a
UUID version 7 when nothing remains. `Z-233` gives the character set and the
length.

The hub sets `X-Flow-ID` on every response. A problem holds the value in
`instance` as `/flows/<value>`, which `Z-176` shapes like a type. The header and
the `flow_id` attribute of `LOG-009` hold the bare value.

`middleware.RequestID` of chi does not serve this rule. It builds its own value
from the hostname of the process, and this rule needs a UUID version 7.

A caller that sends a flow identifier picks the value that every log record of
its request carries, and that value reaches the body of a problem. `Z-233` asks
a client to supply it, so the character set and the length bound the risk and do
not remove it. A trace that crosses a service carries `traceparent`, which W3C
Trace Context defines and OpenTelemetry reads.

**Why:** A person reports the value that the deck shows. One search over the log
then reaches the line that holds the cause.

## ERR-007

`libs/rest` holds the type, one constructor for each type of the first table of
`ERR-003`, the writer, and the middleware of `ERR-006`. The identifier lives there
because a plugin fills `instance` too, and `PLG-007` denies it the hub.

The second table lives in the hub, because `ARC-005` gives `libs/` the code that a
plugin and the hub both import.

Every caller writes a problem with `Write`, which reports nothing, as `http.Error`
of the standard library reports nothing. It encodes before it writes the status,
so a body that does not encode still leaves the answer well formed. No caller
builds a body of its own, and none answers a failure with `render.JSON`, which
writes `application/json`.

`RES-005` and `RES-006` put the page and the cursor in the same module, for the
same reason. `ADR-0007` gives the name.

A plugin imports `libs/rest` and never `apps/hub`. `PLG-007`.

`libs/api-client` parses the media type and puts the problem on `ApiError`.
`@maroid/plugin-sdk` re-exports it for a plugin user interface. `UI-006`.

## Retired identifiers

This file has no retired identifier.
