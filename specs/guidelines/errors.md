---
id: ERR
title: Errors
type: guideline
status: active
created: 2026-09-21
updated: 2026-09-22
scope: [apps/hub/internal/handler/, apps/hub/internal/middleware/, libs/problem/, libs/api-client/, plugins/]
related: [API, DAT, LOG, OWN, PLG, UI]
---

# Errors

## ERR-001

An error response carries the media type `application/problem+json` and a body that
obeys RFC 9457. The body holds these members:

| Member     | Required | Holds                                                          |
| ---------- | -------- | -------------------------------------------------------------- |
| `type`     | Yes      | The URN of the problem type. `ERR-002` gives the grammar.      |
| `title`    | Yes      | A short sentence in English. It is constant for the type.      |
| `status`   | Yes      | The HTTP status of the response. The same number.              |
| `detail`   | No       | One sentence about this occurrence. `ERR-005` bounds it.       |
| `instance` | No       | A URI that names this occurrence. `ERR-006` gives its form.    |

```json
{
  "type": "urn:maroid:problem:http:request-invalid",
  "title": "The request is not valid.",
  "status": 400,
  "detail": "The redirect parameter is absent.",
  "instance": "urn:maroid:request:0199c5e2-8f3a-7c21-9b4e-2f6a1d0c5e77"
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

A problem type is a URN with the form `urn:maroid:problem:<owner>:<slug>`.

The owner names who defines the failure, not who answers it. The slug holds
lowercase letters, digits, and a hyphen.

| Owner       | Defines                                                          |
| ------------- | ------------------------------------------------------------------ |
| `http`      | A failure of the protocol. Any component answers one. `ERR-003`. |
| `hub`       | A failure of a domain of the hub. `ERR-003`.                     |
| A plugin id | A failure of the domain of that plugin.                          |

```
urn:maroid:problem:http:not-found
urn:maroid:problem:hub:settings-invalid
urn:maroid:problem:dev.maroid.jasmine:watering-overlap
```

A slug is permanent. Never reuse a slug for a second meaning, and never change the
meaning of a slug that a deployment answers with. A plugin owns every slug under its
identifier, as `DAT-001` gives it one schema.

**Why:** The deployment chooses the domain, so an address resolves to a page that
nobody serves. A URN reads the same everywhere, so a client compares one constant.

## ERR-003

A type names a condition that HTTP itself defines, or it names a mechanism that
Maroid invented. The first kind lives in `libs/problem`, the second in the hub.
Which component answers it today decides nothing.

`libs/problem` holds these, under `urn:maroid:problem:http:`.

| Slug                     | Status | Title                                               |
| ------------------------ | ------ | ----------------------------------------------------- |
| `request-invalid`        | 400    | The request is not valid.                           |
| `body-invalid`           | 400    | The body is not a JSON object.                      |
| `access-denied`          | 401    | The request carries no active user record.          |
| `not-found`              | 404    | The resource does not exist.                        |
| `method-not-allowed`     | 405    | The method does not reach this resource.            |
| `validation-failed`      | 422    | The body failed validation.                         |
| `internal`               | 500    | The request failed.                                 |

The title of `access-denied` says how Maroid reads the condition. The condition is
the one that HTTP gives every 401, so the type is of the first kind.

A record that is absent and one that belongs to another user both answer
`not-found`, because the policy of `OWN-006` hides the second from the query.

`apps/hub/internal/domain/problems` holds these, under `urn:maroid:problem:hub:`.
Each names a mechanism of Maroid: an allowlist, a settings schema, an identity.

| Slug                     | Status | Title                                               |
| ------------------------ | ------ | ----------------------------------------------------- |
| `network-not-allowed`    | 403    | The caller is not on the network allowlist.         |
| `settings-absent`        | 404    | The plugin declares no settings.                    |
| `identity-last`          | 409    | The last external account cannot be detached.       |
| `settings-invalid`       | 422    | The settings do not match the schema.               |

A plugin answers with a type of the first table for a failure that it names, and
declares a type of its own only for a failure of its own domain. The `api.yaml` of
its feature holds that declaration.

## ERR-004

A problem that reports a failed field carries `errors`. Each item holds `detail`
and `pointer`. `pointer` is a JSON Pointer in the fragment form of RFC 6901, so it
starts with `#` and it reaches a field at any depth.

```json
{
  "type": "urn:maroid:problem:http:validation-failed",
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

The hub gives each request an identifier, a UUID version 7 that the first
middleware generates. `API-002` gives the order.

The hub ignores an `X-Request-Id` header that a request carries, and sets that
header on every response. RFC 9457 asks for a URI in `instance`, so a problem holds
the identifier there as `urn:maroid:request:<uuid>`. The header and the `request`
attribute of `LOG-009` hold the bare UUID.

`middleware.RequestID` of chi does not serve this rule: it reuses the header of the
request, so a caller picks the value that every log record holds.

Read this rule again when a proxy of a deployment sets the header, and then trust
it only from a network that `SEC-009` holds. None sets one today. A trace that
crosses a service carries `traceparent`, which W3C Trace Context defines and
OpenTelemetry reads, so that header answers correlation and this one does not.

**Why:** A person reports the value that the deck shows. One search over the log
then reaches the line that holds the cause. A value that any caller sets reaches
every record of the request, so an attacker writes the log of an investigation.

## ERR-007

`libs/problem` holds the type, one constructor for each type of the first table of
`ERR-003`, the writer, and the middleware of `ERR-006`. The identifier lives there
because a plugin fills `instance` too, and `PLG-007` denies it the hub.

The second table lives in the hub, because `ARC-004` gives `libs/` the code that a
plugin and the hub both import.

Every caller writes a problem with `Write`, which reports nothing, as `http.Error`
of the standard library reports nothing. It encodes before it writes the status,
so a body that does not encode still leaves the answer well formed. No caller
builds a body of its own, and none answers a failure with `render.JSON`, which
writes `application/json`.

A plugin imports `libs/problem` and never `apps/hub`. `PLG-007`.

`libs/api-client` parses the media type and puts the problem on `ApiError`.
`@maroid/plugin-sdk` re-exports it for a plugin user interface. `UI-006`.

## Retired identifiers

This file has no retired identifier.
