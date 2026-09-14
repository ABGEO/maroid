---
id: PSET
title: The scenarios of the settings of a user for a plugin
type: spec
status: approved
created: 2026-09-14
updated: 2026-09-14
approved_by: Temuri
approved_on: 2026-09-14
constrained_by: [TST, OWN, TRC]
requirements: features/pset/requirements.md
---

# Specification: The scenarios of the settings of a user for a plugin

`spec.md` holds the design. This file holds the scenarios, because the two together
pass the size that `LNG-013` gives. See `SPC-001`.

Every integration scenario runs against the container that `testdb.Start` gives, with
the migration of `public.plugin_settings` applied. Two records exist in `public.users`,
user A and user B. The probe plugin declares this model:

```go
type UserSettings struct {
    Email         string `json:"email"          jsonschema:"title=Email,required"`
    Password      string `json:"password"       jsonschema:"title=Password,format=password,writeOnly=true,required"`
    AccountNumber string `json:"account_number" jsonschema:"title=Account number"`
    Period        string `json:"period"         jsonschema:"title=Period,enum=month,enum=year"`
    Notify        bool   `json:"notify"         jsonschema:"title=Notify me"`
}
```

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

**Given** user A holds no settings record for the probe plugin.
**When** user A saves `email`, `password`, `period`, and `notify`.
**Then** one row exists, its `fields` column holds four entries, and its `user_id`
holds the identifier of user A.

## `PSET-SC-004`

**Verifies:** `PSET-FR-004`, `PSET-FR-005`
**Layer:** integration

**Given** user A stored an email and a password.
**When** user A reads the settings of the probe plugin.
**Then** the answer holds the email, it holds `{"set": true}` for the password, and it
holds no value of the password in any form.

## `PSET-SC-005`

**Verifies:** `PSET-FR-006`
**Layer:** integration

**Given** user A stored an email and a password.
**When** user A saves a new email and names no value for the password.
**Then** the stored entry of the password does not change, and the read of the password
returns the value that user A stored first.

## `PSET-SC-006`

**Verifies:** `PSET-FR-007`
**Layer:** integration

**Given** user A stored the optional field `account_number`.
**When** user A saves that field with the value `null`.
**Then** the `fields` column holds no entry for `account_number`.

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

**Given** the row of user A holds an entry for `legacy`, and the current schema declares
no field with that key.
**When** user A reads the settings, and a run of the probe plugin reads the settings.
**Then** neither answer holds `legacy`.

## `PSET-SC-009`

**Verifies:** `PSET-FR-011`
**Layer:** integration

**Given** the row of `PSET-SC-008`.
**When** user A saves the declared fields.
**Then** the `fields` column holds no entry for `legacy`.

## `PSET-SC-010`

**Verifies:** `PSET-FR-012`, `PSET-INV-001`
**Layer:** integration

**Given** user A and user B each stored a settings record for the probe plugin, with a
different email in each.
**When** a run with user A as the acting user reads the settings.
**Then** the answer holds the email of user A, and the count of the values of user B in
the answer is zero.

## `PSET-SC-011`

**Verifies:** `PSET-FR-013`
**Layer:** integration

**Given** the schema declares `password` as required, and the row of user A holds no
entry for it.
**When** a run with user A as the acting user reads the settings.
**Then** the read returns `pluginapi.ErrSettingsAbsent` and no map.

## `PSET-SC-012`

**Verifies:** `PSET-FR-014`
**Layer:** unit

**Given** a cron job with the scope `per-user`, two active users, and a job that returns
`pluginapi.ErrSettingsAbsent` for the first user.
**When** the cron worker runs the job.
**Then** the log holds no record at the error level for the first user, and the worker
runs the job for the second user.

## `PSET-SC-013`

**Verifies:** `PSET-FR-015`, `PSET-INV-002`
**Layer:** integration

**Given** the schema declares `password` as a secret.
**When** user A saves the password `hunter2`.
**Then** a statement that reads the `fields` column with no decryption returns a value
that starts with `vault:v` and holds no `hunter2`.

## `PSET-SC-014`

**Verifies:** `PSET-FR-016`
**Layer:** integration

**Given** user A stored a password, and the owner created one key for each user.
**When** a decrypt of the stored ciphertext of user A runs against the key of user B.
**Then** the call fails, and it returns no value.

## `PSET-SC-015`

**Verifies:** `PSET-FR-017`
**Layer:** integration

**Given** user A stored the password `hunter2` under version 1 of their key.
**When** the owner rotates that key to version 2, and a run reads the settings.
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
user A.
**When** a run with user A as the acting user reads the settings.
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

**Given** user A stored the password `old`, and a run already read it one time.
**When** user A saves the password `new`, and a run starts 1 second after the save
returns.
**Then** that run reads `new`.

## `PSET-SC-020`

**Verifies:** `PSET-NFR-002`
**Layer:** integration

**Given** the probe plugin and a stored record for user A with one secret field and
three fields that are not secret.
**When** a run reads the settings 100 times.
**Then** the 95th percentile of the time from the call of the plugin to the return of
the values is under 200 milliseconds.

## Retired identifiers

This file has no retired identifier.
