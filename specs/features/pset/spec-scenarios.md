---
id: PSET
title: The scenarios of the settings of a plugin
type: spec
status: approved
created: 2026-09-14
updated: 2026-10-01
approved_by: Temuri
approved_on: 2026-09-14
constrained_by: [TST, OWN, TRC]
requirements: features/pset/requirements.md
---

# Specification: The scenarios of the settings of a plugin

`spec.md` holds the design. This file holds the scenarios, because the two together
pass the size that `LNG-013` gives. See `SPC-001`.

Every integration scenario runs against the container that `testdb.Start` gives, with
the migrations of `public.plugin_workspace_settings` and `public.plugin_user_settings`
applied. Three records exist in `public.users`: user A, user B, and user C. Two records
exist in `public.workspaces`. Workspace A has user A as its manager and user C as an
editor. Workspace B has user B as its manager. Each workspace enables the probe plugin,
which declares this model:

```go
type Settings struct {
    Email         string `json:"email"          jsonschema:"title=Email,required"`
    Password      string `json:"password"       jsonschema:"title=Password,format=password,writeOnly=true,required"`
    AccountNumber string `json:"accountNumber" jsonschema:"title=Account number"`
    Period        string `json:"period"         jsonschema:"title=Period,enum=month,enum=year"`
    Notify        bool   `json:"notify"         jsonschema:"title=Notify me"`
    Pin           string `json:"pin"            jsonschema:"title=PIN,format=password,writeOnly=true" jsonschema_extras:"x-maroid-scope=user"`
}
```

Every field except `pin` belongs to the workspace.

## `PSET-SC-001`

**Verifies:** `PSET-FR-001`
**Layer:** unit

**Given** a plugin whose `SettingsModel` returns a string and not a struct.
**When** the settings registrar registers the plugin.
**Then** the registration returns an error that names the plugin, and the registry
holds no entry for it.

## `PSET-SC-002`

**Verifies:** `PSET-FR-002`
**Layer:** unit

**Given** the model of the probe plugin.
**When** the settings registrar infers the document.
**Then** the property `password` carries `"format": "password"` and
`"writeOnly": true`, the property `period` carries the enum `month` and `year`, the
property `notify` carries the type `boolean`, the `required` array holds `email` and
`password`, and `additionalProperties` is false.

## `PSET-SC-003`

**Verifies:** `PSET-FR-003`
**Layer:** integration

**Given** workspace A holds no settings record for the probe plugin.
**When** user A saves `email`, `password`, `period`, and `notify` in workspace A.
**Then** one row exists in `public.plugin_workspace_settings`, its `fields` column
holds four entries, and its `workspace_id` holds the identifier of workspace A.

## `PSET-SC-004`

**Verifies:** `PSET-FR-004`, `PSET-FR-005`
**Layer:** integration

**Given** workspace A stored an email and a password.
**When** user C reads the settings of the probe plugin in workspace A.
**Then** the answer holds the email, it holds `******` for the password, it holds no
member for `accountNumber`, and it holds no value of the password in any form.

## `PSET-SC-005`

**Verifies:** `PSET-FR-006`
**Layer:** integration

**Given** workspace A stored an email and a password.
**When** user A saves a new email in workspace A and names no value for the password.
**Then** the stored entry of the password does not change, and the read of the password
returns the value that workspace A stored first.

## `PSET-SC-006`

**Verifies:** `PSET-FR-007`
**Layer:** integration

**Given** workspace A stored the optional field `accountNumber`.
**When** user A saves that field with the value `null` in workspace A.
**Then** the `fields` column holds no entry for `accountNumber`.

## `PSET-SC-007`

**Verifies:** `PSET-FR-008`, `PSET-FR-009`
**Layer:** unit

**Given** the schema of the probe plugin.
**When** a save names the unknown field `nickname`, then leaves `password` empty, then
gives `period` the value `week`, then gives `email` a value of 4097 characters, then
gives `notify` the value `"yes"`.
**Then** each save is rejected, no value reaches the database, and each answer names
exactly the field that caused the rejection.

## `PSET-SC-008`

**Verifies:** `PSET-FR-010`
**Layer:** integration

**Given** the row of workspace A holds an entry for `legacy`, and the current schema
declares no field with that key.
**When** user A reads the settings in workspace A, and a run of the probe plugin in
workspace A reads the settings.
**Then** neither answer holds `legacy`.

## `PSET-SC-009`

**Verifies:** `PSET-FR-011`
**Layer:** integration

**Given** the row of `PSET-SC-008`.
**When** user A saves the declared fields in workspace A.
**Then** the `fields` column holds no entry for `legacy`.

## `PSET-SC-010`

**Verifies:** `PSET-FR-012`, `PSET-INV-001`
**Layer:** integration

**Given** workspace A and workspace B each stored a settings record for the probe
plugin, with a different email in each.
**When** a run with workspace A as the acting workspace reads the settings.
**Then** the answer holds the email of workspace A, and the count of the values of
workspace B in the answer is zero.

## `PSET-SC-011`

**Verifies:** `PSET-FR-013`
**Layer:** integration

**Given** the schema declares `password` as required, and the row of workspace A holds
no entry for it.
**When** a run with workspace A as the acting workspace reads the settings.
**Then** the read returns `pluginapi.ErrSettingsAbsent` and no map.

## `PSET-SC-012`

**Verifies:** `PSET-FR-014`
**Layer:** unit

**Given** a cron job with the scope `per-workspace`, two workspaces that enable its
plugin, and a job that returns `pluginapi.ErrSettingsAbsent` for the first workspace.
**When** the cron worker runs the job.
**Then** the log holds no record at the error level for the first workspace, and the
worker runs the job for the second workspace.

## `PSET-SC-013`

**Verifies:** `PSET-FR-015`, `PSET-INV-002`
**Layer:** integration

**Given** the schema declares `password` as a secret.
**When** user A saves the password `hunter2` in workspace A.
**Then** a statement that reads the `fields` column with no decryption returns a value
that starts with `vault:v` and holds no `hunter2`.

## `PSET-SC-014`

**Verifies:** `PSET-FR-016`
**Layer:** integration

**Given** workspace A stored a password, and an administrator created one key for each
workspace.
**When** a decrypt of the stored ciphertext of workspace A runs against the key of
workspace B.
**Then** the call fails, and it returns no value.

## `PSET-SC-015`

**Verifies:** `PSET-FR-017`
**Layer:** integration

**Given** workspace A stored the password `hunter2` under version 1 of its key.
**When** an administrator rotates that key to version 2, and a run reads the settings.
**Then** the read returns `hunter2`.

## `PSET-SC-016`

**Verifies:** `PSET-FR-018`
**Layer:** manual

**Given** the configuration names an address that answers nothing.
**When** the operator starts the hub.
**Then** the process stops before it loads a plugin, the log names the address, and the
port serves no request.

## `PSET-SC-017`

**Verifies:** `PSET-FR-019`
**Layer:** integration

**Given** a statement wrote `vault:v1:not-a-ciphertext` into the password entry of
workspace A.
**When** a run with workspace A as the acting workspace reads the settings.
**Then** the read returns an error and no map, and the error names the field `password`
and holds no ciphertext.

## `PSET-SC-018`

**Verifies:** `PSET-FR-020`
**Layer:** unit

**Given** a `model.PluginSettings` whose password entry holds `hunter2`.
**When** a log statement passes it as an attribute at the debug level.
**Then** the record holds the plugin identifier and the key `password`, and it holds no
`hunter2` and no ciphertext.

## `PSET-SC-019`

**Verifies:** `PSET-NFR-001`
**Layer:** integration

**Given** workspace A stored the password `old`, and a run already read it one time.
**When** user A saves the password `new` in workspace A, and a run starts 1 second
after the save returns.
**Then** that run reads `new`.

## `PSET-SC-020`

**Verifies:** `PSET-NFR-002`
**Layer:** integration

**Given** the probe plugin and a stored record for workspace A with one secret field
and three fields that are not secret.
**When** a run reads the settings 100 times.
**Then** the 95th percentile of the time from the call of the plugin to the return of
the values is under 200 milliseconds.

## `PSET-SC-021`

**Verifies:** `PSET-FR-006`
**Layer:** integration

**Given** workspace A stored an email and a password.
**When** user A saves a new email and the mask `******` for the password in workspace A.
**Then** the stored entry of the password does not change, and the read of the password
returns the value that workspace A stored first.

## `PSET-SC-022`

**Verifies:** `PSET-FR-007`
**Layer:** integration

**Given** workspace A stored the optional field `accountNumber`.
**When** user A saves that field with the empty string in workspace A.
**Then** the `fields` column holds no entry for `accountNumber`.

## `PSET-SC-023`

**Verifies:** `PSET-FR-021`, `PSET-INV-001`
**Layer:** integration

**Given** user A saved the PIN `1234` in workspace A, beside the fields of the workspace.
**When** user A reads the settings in workspace A, then user C reads them, then a cron
run in workspace A reads them.
**Then** `public.plugin_user_settings` holds the PIN in one row whose `user_id` holds
the identifier of user A, and `public.plugin_workspace_settings` holds no entry for
`pin`. User A reads `******` for the PIN, user C reads the empty string for it, and
the run reads no value for it.

## Retired identifiers

This file has no retired identifier.
