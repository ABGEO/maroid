---
id: NTF
title: Notifications
type: guideline
status: active
created: 2026-09-11
updated: 2026-09-11
scope: [libs/notifier/, libs/notifierapi/]
related: [PLG, TG, CFG]
---

# Notifications

## NTF-001

A plugin sends a notification through `Host.Notifier()`, which gives a
`notifierapi.Dispatcher`.

## NTF-002

A plugin names a channel, not a transport: `Send(ctx, channelName, msg)`.

**Why:** The owner changes where a notification goes by editing the configuration.
The plugin does not change.

## NTF-003

A message holds a title, a body, and attachments.
An attachment holds a file name, the content, and a MIME type.

## NTF-004

A channel lists primary transports and fallback transports.
The dispatcher tries every primary transport. It uses the fallback transports only
when every primary transport fails.
The call returns an error only when every transport fails.

## NTF-005

A URL defines a transport. The scheme of the URL selects the factory.

## NTF-006

A new transport registers a factory for its scheme in the registry.
The registry rejects a scheme that it already holds, and rejects a nil factory.

## Retired identifiers

This file has no retired identifier.
