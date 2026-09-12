---
id: EXT
title: External services
type: guideline
status: active
created: 2026-09-11
updated: 2026-09-11
scope: [plugins/*/service/, plugins/*/dto/]
related: [PKG, GO, LOG, CFG]
---

# External services

A plugin that reads a bill, a balance, or a meter calls a third-party API.

## EXT-001

The client of an external system lives in `service/`.
The file name is `api-client.go`.

## EXT-002

The HTTP client is resty. The base URL comes from the configuration of the plugin.
The client sets `Accept: application/json` once, at construction.

## EXT-003

Declare the interface, implement it, and assert the implementation at compile
time. See `GO-007`.

```go
type APIClientService interface { ... }
type APIClient struct { client *resty.Client }
var _ APIClientService = (*APIClient)(nil)
```

**Why:** The interface is the seam. A test replaces the client without a network.

## EXT-004

Every request carries the context of the caller.

## EXT-005

Check the status code. A status that is not a success is an error.
Never treat it as an empty result.

**Why:** A silent empty result becomes a missing bill, not a visible failure.

## EXT-006

A request shape and a response shape live in `dto/`.
A DTO converts to a `model` entity. A `model` entity never carries a JSON tag
of a third party.

**Why:** The third party changes its field names. The domain does not.

## EXT-007

A sentinel error names the failure of the service. See `GO-008`.
Wrap it with the status code and the body of the response.

## Retired identifiers

This file has no retired identifier.
