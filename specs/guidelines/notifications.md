---
id: NTF
title: Notifications
type: guideline
status: active
created: 2026-09-11
updated: 2026-10-11
scope: [apps/hub/internal/notification/, libs/pluginapi/]
related: [PLG, PSET, OWN, CFG]
---

# Notifications

## NTF-001

A plugin declares each notification topic that it sends, with an identifier, a name,
and an optional description. The declaration is a capability, as `PLG-006` gives.
A plugin sends a notification through `Host.Notifier()`, which gives a `pluginapi.Notifier`.

## NTF-002

A plugin names a notification topic, never a notification provider or a destination:
`Notify(ctx, topicID, msg)`. The hub returns an error for a notification topic that the
plugin does not declare.

**Why:** The manager changes where a notification goes in the deck. The plugin does not change.

## NTF-003

A message holds a title, a body, and attachments.
An attachment holds a file name, the content, and a MIME type.

## NTF-004

The hub delivers a notification through each enabled notification route of its
notification topic in the acting workspace, when the notification provider is enabled.
A failure of one notification route does not stop the others. The call returns an
error only when every notification route that the hub tried failed.

The hub logs and drops a notification of an unrouted notification topic, or of a unit
of work with no acting workspace. The call returns no error.

## NTF-006

A new notification provider kind registers in the kind registry of the hub.
The registry rejects an identifier that it already holds, and rejects a nil kind.
A plugin adds no notification provider kind.

## NTF-007

A notification provider kind declares two settings schemas: the fields of a
notification provider and the fields of a destination. The field kinds are those of
`PSET-FR-002`. The kind validates a credential against its external service before
the hub stores it.

## NTF-008

A notification provider and a notification route are scoped records of a workspace.
A secret field of a notification provider takes the protection of its workspace.
The configuration holds none of them. See `CFG-007`.

## Retired identifiers

| ID        | Retired    | Reason                                                             |
| --------- | ---------- | ------------------------------------------------------------------ |
| `NTF-005` | 2026-10-11 | A URL defines no transport. `NTF-007` replaces it. See `ADR-0013`. |
