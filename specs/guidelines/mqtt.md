---
id: MQT
title: MQTT
type: guideline
status: active
created: 2026-09-11
updated: 2026-09-11
scope: [apps/hub/internal/worker/mqtt.go, apps/hub/internal/plugin/registrar/mqtt_subscriber.go, plugins/*/mqtt/]
related: [PLG, JOB, CFG]
---

# MQTT

The client library is paho.

## MQT-001

A plugin declares a subscriber as `pluginapi.MQTTSubscriber`.
The metadata gives an identifier, a topic, and a quality of service of 0, 1, or 2.

## MQT-002

The topic in the metadata is relative.
The hub prefixes it with the namespace of the plugin.
The namespace is the plugin identifier with `/` in place of each dot.

```
plugin    dev.maroid.jasmine
relative  measurement/+/+
effective dev/maroid/jasmine/measurement/+/+
```

**Why:** A plugin cannot subscribe to the topics of another plugin.
The namespace comes from the identifier, so the hub needs no configuration for it.

## MQT-003

A relative topic is not empty. It does not start with `/`. It does not start with `$`.
The wildcards `+` and `#` are allowed.

## MQT-004

`Handle` receives the relative topic. The hub removes the namespace first.

**Why:** A plugin parses the part of the topic that it owns. It never sees its own prefix.

## MQT-005

The hub subscribes through a shared subscription, `$share/<group>/<topic>`,
when `mqtt.shared_group` holds a value.

**Why:** Two hub instances then divide the messages instead of both handling each one.

## MQT-006

The hub handles each message in a separate goroutine.
A subscriber must be safe for concurrent use.
A slow subscriber does not block the receive loop and does not delay another subscriber.

## MQT-007

The broker must be configured when a plugin registers a subscriber.
The worker reports the error at the prepare step, before it starts.

**Why:** A missing broker must stop the start, not the first message.

## Retired identifiers

This file has no retired identifier.
