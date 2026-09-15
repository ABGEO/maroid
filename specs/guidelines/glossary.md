---
id: GLO
title: Glossary
type: glossary
status: active
created: 2026-09-11
updated: 2026-09-15
related: [LNG]
---

# Glossary

Each term has one meaning in Maroid.
Use the term from this list. Do not use a synonym. See `LNG-002`.

Add a term when a requirements document introduces a domain noun.
A term identifier has the form `GLO-<term>`. The identifier is permanent.

| ID                  | Term              | Meaning                                                                                                           |
| ------------------- | ----------------- | ----------------------------------------------------------------------------------------------------------------- |
| `GLO-hub`           | hub               | The host process in `apps/hub`. It loads the plugins and owns the configuration.                                  |
| `GLO-deck`          | deck              | The web shell in `apps/deck`. It renders the user interface of the hub and of the plugins.                        |
| `GLO-plugin`        | plugin            | A Go shared object in `build/plugins/`. The hub loads it at runtime. It adds a capability.                        |
| `GLO-plugin-id`     | plugin identifier | The permanent name of a plugin, with the form `dev.maroid.<name>`.                                                |
| `GLO-capability`    | capability        | A function that a plugin adds to the hub. The plugin gets it when it implements an interface in `libs/pluginapi`. |
| `GLO-host`          | host              | The `pluginapi.Host` interface. It is the only path from a plugin to an external resource.                        |
| `GLO-registrar`     | registrar         | A component in the hub. It finds one capability in a plugin and puts it into a registry.                          |
| `GLO-registry`      | registry          | A component in the hub. It holds the items of one capability for every plugin.                                    |
| `GLO-schema`        | schema            | The PostgreSQL schema that one plugin owns. The name comes from the plugin identifier.                            |
| `GLO-remote`        | remote            | A Module Federation bundle that holds the user interface of one plugin.                                           |
| `GLO-route-mounter` | route mounter     | A function in a remote. It renders one page of a plugin into an element of the deck.                              |
| `GLO-notification`  | notification      | A message that Maroid sends to a person, through Telegram or through the deck.                                    |
| `GLO-owner`         | owner             | The maintainer of the repository. The owner approves each stage.                                                  |
| `GLO-user`          | user              | A person that Maroid serves. One row in `public.users`. An identity binds it to an external account.              |
| `GLO-identity`      | identity          | The binding of one external account to one user record. One row in `public.identities`.                          |
| `GLO-provider`      | provider          | One account system that Dex federates. Telegram is one.                                                          |
| `GLO-external-account` | external account | The account that one person holds at one provider.                                                            |
| `GLO-invitation`    | invitation        | A grant that the owner issues. It lets one sign in create the first identity of one user record.                  |
| `GLO-attach`        | attach            | The operation that creates an identity.                                                                          |
| `GLO-detach`        | detach            | The operation that removes an identity.                                                                          |
| `GLO-acting-user`   | acting user       | The user that one unit of work runs for. Every request, update, and job run has exactly one.                      |
| `GLO-scoped-table`  | scoped table      | A table whose every row belongs to one user. It carries `user_id` and row level security.                         |
| `GLO-shared-table`  | shared table      | A table whose rows belong to no user. It carries no `user_id`.                                                    |
| `GLO-scoped-record` | scoped record     | A record that belongs to exactly one user.                                                                        |
| `GLO-shared-record` | shared record     | A record that belongs to no user. Every user reads it.                                                            |
| `GLO-setting`       | setting           | One value that one user stores for one field of one plugin.                                                       |
| `GLO-settings-schema` | settings schema | The list of the fields that one plugin declares.                                                                  |
| `GLO-secret-field`  | secret field      | A field whose value Maroid returns to nobody after the user stores it.                                            |
| `GLO-secret`        | secret            | The value of a secret field.                                                                                      |
| `GLO-protection`    | protection        | The means that makes a secret unreadable to a reader of the database.                                             |

## Retired identifiers

This file has no retired identifier.
