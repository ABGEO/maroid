---
id: CFG
title: Configuration and secrets
type: guideline
status: active
created: 2026-09-11
updated: 2026-09-11
scope: [apps/hub/internal/config/, libs/pluginconfig/, config.yaml, chart/values.yaml]
related: [ARC, PLG, OWN]
---

# Configuration and secrets

## CFG-001

Configuration uses Viper.
The hub reads `config.yaml` from the working directory or from `~/.maroid/`.

## CFG-002

An environment variable with the prefix `MAROID_` overrides a value in the file.

## CFG-003

A configuration struct declares the default value in a `default` tag.
A configuration struct declares the validation in a `validate` tag.
The hub validates the configuration at the start and stops on an error.

**Why:** A wrong configuration must fail at the start, not at the first request.

## CFG-004

A secret does not go into the repository.
`config.yaml`, `.env`, and `.keys/` stay in `.gitignore`.
An example file with no secret value ships in the repository.

## CFG-005

A plugin reads its own configuration from the map that the constructor receives.
A plugin does not read a file and does not read an environment variable itself.

**Why:** The hub owns the configuration. One source. One validation point.

## CFG-006

A plugin decodes the configuration map with `pluginconfig.DecodeAndValidateConfig`
into a struct of its own. The struct declares the defaults and the validation
with the same tags that `CFG-003` gives.

**Why:** A plugin gets the same failure behavior as the hub. A wrong value stops
the load, not the first job.

## CFG-007

Configuration holds a value of the installation. A value that belongs to one person
lives in the database, in a scoped record. See `OWN-004`.

| Value                                    | Place         |
| ---------------------------------------- | ------------- |
| The address of an external service       | Configuration |
| A token that the installation shares     | Configuration |
| The credential of one person             | The database  |
| The account or the vehicle of one person | The database  |

A plugin that keeps a per-user value in its configuration serves one person.
Moving that value is the step that makes the plugin serve everybody, and a plugin
moves one value at a time.

**Why:** A configuration file has one copy for the process. A per-user value in it
caps the plugin at one user.

## Retired identifiers

This file has no retired identifier.
