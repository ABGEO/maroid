---
id: ADR-0005
title: Problem details as the one error format
type: adr
status: accepted
created: 2026-09-21
updated: 2026-09-22
decided: 2026-09-22
changes: [ERR-001, ERR-002, ERR-003, ERR-004, ERR-005, ERR-006, ERR-007, API-002, API-006, LOG-009, GLO-problem]
supersedes:
superseded_by:
---

# Problem details as the one error format

## Context

`API-006` gives the whole rule for a failure:

> A response body is JSON.
> An error response carries the HTTP status and a JSON body that names the reason.

The rule names no member, so each site chose one. The code writes five JSON shapes
today, one plain text body, and one empty body.

| Site                                     | Body                                                 |
| ---------------------------------------- | ---------------------------------------------------- |
| `auth/middleware.go:115`                 | `{"error": "access denied"}`                         |
| `handler/auth.go:304,309,569`            | `{"error": "..."}`                                   |
| `middleware/allowed_networks.go:60`      | `{"message": "Forbidden"}`                           |
| `handler/plugin.go:152,174,177,184`      | `{"reason": "...", "fields": [...]}`                 |
| `handler/mcp.go:158`                     | `{"reason": "..."}`                                  |
| `plugins/jasmine/dto/error.go`           | `{"error": "...", "fields": [{"field", "message"}]}` |
| chi `NotFound`, `MethodNotAllowed`       | `text/plain`                                         |
| chi `Recoverer`                          | An empty body                                        |

Three member names carry one idea. Two shapes carry a field failure, and they differ.
`specs/templates/api.yaml` declares `Error { reason }`, so every new feature copies
one of the five at random.

A client cannot branch on a failure. `libs/api-client` throws `ApiError` with the
status and an untyped body. `SettingsForm.svelte` reads `body.reason` and gives up
when the member is absent, because the member name depends on the handler that
answered. The deck holds no translation table, and no value in a body is stable
enough to key one.

The hub sets no request identifier. A failure that a person reports carries nothing
that finds the log line which holds the cause.

RFC 9457 defines `application/problem+json`. It replaced RFC 7807 in July 2023 with
the same media type and the same members.

## Decision

Every error response of the hub carries the media type `application/problem+json`
and a body that obeys RFC 9457. A new guideline, `ERR`, holds the shape, the grammar
of a problem type, and the registry of the types that the hub owns.

## Rationale

The registry carries the value, not the media type. A failure gets a permanent
identifier the day it exists, so a client compares one constant and a translation
table gets a key. Three member names collapse into one shape that a person can read
without opening the handler that wrote it.

A URN holds that identifier. Maroid runs on the domain of whoever deploys it, so an
absolute address resolves to a page that nobody serves, and a relative reference
resolves against the request and gives one type a different value on each
deployment. `urn:maroid:problem:<owner>:<slug>` reads the same everywhere. The owner
segment repeats the rule that `DAT-001` gives a schema: a plugin owns the namespace
that carries its identifier, so two plugins cannot collide.

The standard adds no dependency. A problem is a struct with five members, and the
writer sets the header itself, because `render.JSON` writes `application/json`. The
writer lives in `libs/problem`, because `PLG-007` denies a plugin the import of
`apps/hub`.

The request identifier answers the question that the current log cannot. A person
reads the value in the deck, and one search over the log reaches the line that holds
the wrapped Go error. The body then needs to carry no internal text, which `ERR-005`
forbids, and the two halves still join.

The hub generates that identifier and does not take `middleware.RequestID` of chi.
That middleware reuses the `X-Request-Id` header of the request, so a caller picks
the value that every log record of the request then holds, and it builds its own
value from the hostname of the process. A UUID version 7 carries neither problem,
and it sorts by time. `DAT-009` gives a row the same shape.

The namespace identifier `maroid` is not registered with IANA. RFC 9457 asks for a
URI and not for a registered namespace, and nothing resolves this value.

A JSON Pointer reaches a field at any depth. The validator holds the path of each
leaf failure in `ValidationError.InstanceLocation`, and `record` in
`settings/validate.go` drops it for the property name alone, so a nested field
reports as a bare word today. The deck renders the settings form with `@sjsf/form`,
which reads the same schema, so a pointer names a field that the form knows.

The change reaches every consumer in one commit, because every consumer sits in this
repository. `libs/api-client` matches `application/json` as a substring and would
read a problem body as text, so the client changes with the hub.

## Alternatives

| Alternative                                                             | Why we did not select it                                                                                                                   |
| ----------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------- |
| Name one member in `API-006` and keep `application/json`.               | It fixes the member name and nothing else. No registry, no permanent identifier for a failure, and a client still branches on a sentence.  |
| Cite RFC 7807.                                                          | RFC 9457 replaced it. The media type and the members are the same, so the older citation buys nothing and dates the document.              |
| Use an absolute `https` address as the type.                            | RFC 9457 prefers an address a person can open. Maroid answers at the domain of a deployment, so the page reaches nobody who self hosts.    |
| Use a relative reference as the type.                                   | It resolves against the request address. One type then holds a different absolute value on each deployment, and a client matches nothing.  |
| Allow `about:blank` when the status says enough.                        | It sends the client back to the status code, which is what the four shapes already do. A type costs one constant.                          |
| Negotiate on `Accept`, and answer `application/json` when asked.        | Two shapes for one failure, and a branch in every writer. No client of Maroid asks for the other one.                                      |
| Keep the field failure as `fields: [{field, message}]`.                 | RFC 9457 gives `errors` with `detail` and `pointer` in its own example, and the item reuses the member names of the problem itself.        |
| Use `invalid-params` with `name` and `reason`.                          | That is the example of RFC 7807, which RFC 9457 replaced. A bare name reaches no nested field, and the spelling needs a camel case deviation. |
| Put the rules in `API`.                                                 | `http-api.md` holds 71 lines, and these rules add roughly 140. The registry grows with every plugin. `LNG-013` targets 150 lines.          |
| Omit `instance` and add no request identifier.                          | Nothing joins a failure that a person reports to the log line that holds its cause. One middleware and one attribute buy that join.        |
| Take `middleware.RequestID` of chi for the identifier.                  | It reuses the header of the request, so a caller picks the value that every log record holds. Its own value carries the hostname of the process. |
| Cover the API routes only, and leave the webhook and the assets bare.   | Two rules and a branch, to save a body that no reader parses. One rule with no exception costs less to hold and less to check.             |

## Consequences

### Rules that change

`ERR` is new. `specs/guidelines/errors.md` holds `ERR-001` through `ERR-007`: the
media type and the shape, the grammar of a type URN, the registry of the types that
the hub owns, the `errors` extension, the bound on `detail`, `instance` and the
request identifier, and the place of the code that writes and reads a problem.

`API-002` gains one middleware, the request identifier, and it runs first. The rule
holds the order of the rest.

`API-006` loses its second sentence and points at the new guideline:

> A response body is JSON. An error response obeys `ERR-001`.

`LOG-009` is new. Every log record of a request carries the `request` attribute. It
holds the bare UUID, and `instance` carries the same UUID as a URN, because RFC 9457
asks for a URI there.

`GLO-problem` is new. A problem is the body of an error response.

The prefix register in `traceability.md` gains the row for `ERR`. That register
asks for the row before the file exists.

### Specifications to examine

| Document                                       | What the examination found                                                            |
| ------------------------------------------------ | --------------------------------------------------------------------------------------- |
| The five `api.yaml` of `extid`, `mcphub`, `pcap`, `pset`, `websess` | Each declared its own error schema, under three member names. Each now declares `Problem`, and each response names the type that `ERR-003` registers for it. |
| `features/pset/api.yaml`                       | `ValidationError` held a map of field to message. The `errors` member of `ERR-004` replaces it. |
| `features/mcphub/api.yaml`                     | Three 405 answers carried no body. They share one `MethodNotAllowed` response now. The 401 keeps `WWW-Authenticate`. |
| `features/mcphub/spec.md`                      | Section 4.5 put the reason from the verifier in the answer, which `ERR-005` forbids. The reason reaches the log, and the three 401 conditions answer with one type. |
| `features/pset/spec.md`                        | `InvalidError` held `map[string]string`, keyed by a bare property name, so a nested field lost its path. It holds a sorted `[]FieldFailure` with the pointer. |
| `features/mcphub/spec-settings-tools.md`       | `MCPHUB-DD-020` sorted the fields by map key. It follows the order of the slice now, and section 4.5 names a pointer in place of a key. |
| The 4.5 of `extid`, `ident`, `pset`, `websess`, `mcphub` | Each listed the body text of a failure, under the old member names. Each names the type now. `extid` and `ident` also list a redirect, which carries its reason in a parameter and writes no problem. |
| `features/pcap/spec.md`, `features/theming/spec.md`, `features/mcphub/spec-plugin-tools.md` | Each 4.5 lists a failure of a load, of a build, or of a tool call. None writes an HTTP body, so none changes. |
| `features/pset/requirements.md`                | `PSET-FR-009` asks Maroid to name each field that caused a rejection. A pointer names it, so the statement stands unchanged. No requirements document changes. |

### Migration

No migration of the database. The code changes at every site in the table above, in
one commit, with `libs/problem` and `libs/api-client` in the same commit. A JSON-RPC
error inside an MCP tool call keeps the MCP shape, because it is not an HTTP failure.

`specs/templates/api.yaml` changed with this decision. It declares the `Problem`
schema, so a new feature copies the format and not the `Error` shape.

### Result

- Positive: one shape for every failure, with one member name for each idea.
- Positive: a permanent identifier for each failure, which a client matches and a
  translation table keys.
- Positive: a person reports an identifier that finds the log line.
- Negative: a new guideline, a new library, and a new prefix to hold.
- Negative: every handler that writes a failure changes, and so does each test that
  asserts on the old body.
- Work that follows: the five `api.yaml` files, then the code.
