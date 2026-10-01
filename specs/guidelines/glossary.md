---
id: GLO
title: Glossary
type: glossary
status: active
created: 2026-09-11
updated: 2026-10-01
related: [LNG, ERR, RES, SPC]
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
| `GLO-idp`           | IdP               | The external identity provider that authenticates every person for Maroid. Dex is the IdP today.                  |
| `GLO-provider`      | provider          | One account system that Dex federates. Telegram is one.                                                          |
| `GLO-external-account` | external account | The account that one person holds at one provider.                                                            |
| `GLO-invitation`    | invitation        | A grant that an administrator issues. It lets one sign in create the first identity of one user record.          |
| `GLO-attach`        | attach            | The operation that creates an identity.                                                                          |
| `GLO-detach`        | detach            | The operation that removes an identity.                                                                          |
| `GLO-session`       | session           | The period from one sign in of a person at the deck to the end of it.                                            |
| `GLO-sign-in`       | sign in           | The operation that starts a session.                                                                             |
| `GLO-sign-out`      | sign out          | The operation that ends a session.                                                                               |
| `GLO-session-cookie` | session cookie   | The cookie that carries the credential of a session.                                                             |
| `GLO-access-token`  | access token      | The value that the IdP issues for a call to Maroid.                                                              |
| `GLO-identity-token` | identity token   | The value that the IdP issues to describe a person to the client that signed them in.                            |
| `GLO-acting-user`   | acting user       | The user that one unit of work runs for. A request, an update, a tool call, and the run of a job for each user each have exactly one. Any other cron run has none. |
| `GLO-workspace` | workspace | The place that owns the records of a plugin. Its members share them. `OWN-010` gives it. |
| `GLO-member` | member | A user who belongs to a workspace, with one workspace role. `OWN-011` gives it. |
| `GLO-workspace-role` | workspace role | The access of a member: manager, editor, or viewer, in that order. |
| `GLO-acting-workspace` | acting workspace | The workspace that one unit of work runs in. |
| `GLO-administrator` | administrator | A user who manages the users, the plugin allowlists, and every workspace of the instance. |
| `GLO-plugin-allowlist` | plugin allowlist | The plugins that one user may enable in a workspace that they manage. |
| `GLO-enablement` | enablement | The fact that one plugin serves one workspace. |
| `GLO-permission` | permission | One action that the hub or a plugin checks, with the lowest workspace role that holds it. |
| `GLO-scoped-table`  | scoped table      | A table whose every row belongs to one workspace or to one user. It carries the scope column and row level security. |
| `GLO-shared-table`  | shared table      | A table whose rows belong to no workspace and to no user. It carries no scope column. |
| `GLO-scoped-record` | scoped record     | A record that belongs to exactly one workspace or exactly one user. |
| `GLO-shared-record` | shared record     | A record that belongs to nobody. Every user reads it. |
| `GLO-setting`       | setting           | One value that a user or a workspace stores for one field of one plugin. The plugin declares which. |
| `GLO-settings-schema` | settings schema | The list of the fields that one plugin declares.                                                                  |
| `GLO-secret-field`  | secret field      | A field whose value Maroid returns to nobody after the user stores it.                                            |
| `GLO-secret`        | secret            | The value of a secret field.                                                                                      |
| `GLO-protection`    | protection        | The means that makes a secret unreadable to a reader of the database.                                             |
| `GLO-problem`       | problem           | The body of an error response. It obeys RFC 9457 and `ERR-001`. A problem type names one failure.                |
| `GLO-mcp-tool`      | MCP tool          | A function that Maroid exposes over the Model Context Protocol. An MCP client calls it by name and gets a result.  |
| `GLO-mcp-client`    | MCP client        | The application that connects to the MCP server of the hub and calls an MCP tool.                                  |
| `GLO-page`          | page              | One answer of a collection. It holds the items and the links that `RES-005` gives.                                |
| `GLO-cursor`        | cursor            | The opaque value that names the position of a page. `RES-006` gives its content.                                  |
| `GLO-flow-id`       | flow identifier   | The value that joins every record of one request. `ERR-006` gives its source and its bound.                       |
| `GLO-binding`       | binding           | The short lived value that joins one hand off to the IdP to the browser that started it. It lives in a cookie and in no table. |
| `GLO-handoff`       | hand off          | The step that sends a person from Maroid to the IdP and back. Three routes of `API-003` start one.                |
| `GLO-transport-route` | transport route | A route that carries the protocol of another system and answers no resource of Maroid. `RES-003` gives which rules reach it. |
| `GLO-api-document`  | API document      | One of the three OpenAPI files that `RES-007` names. `RES-008` builds each one.                                                       |
| `GLO-api-fragment`  | API fragment      | The `api.yaml` of one feature. `SPC-002` merges it into an API document.                                          |
| `GLO-shipped-client` | shipped client   | One of `libs/api-client`, `apps/deck`, and `libs/plugin-sdk`. `RES-010` binds each one.                           |
| `GLO-theme`         | theme             | The set of colors, radii, and fonts that gives a page its Maroid visual identity. Maroid keeps one definition of it. |
| `GLO-theme-variant` | theme variant     | One of the two forms of the theme that the deck offers: light and dark.                                           |
| `GLO-component-library` | component library | The one shared set of components and styling that the deck and every remote import.                       |
| `GLO-component`     | component         | One reusable control of the component library. It receives every value it shows from its caller.                  |
| `GLO-catalog`       | catalog           | A page that shows each component alone, with sample values, outside the deck.                                     |
| `GLO-orchestrator`  | orchestrator      | The system that starts, stops, and routes traffic to a hub process.                                               |
| `GLO-liveness`      | liveness          | The state of a hub process that still answers a request.                                                          |
| `GLO-readiness`     | readiness         | The state of a hub process that can serve a request of a person.                                                  |
| `GLO-dependency`    | dependency        | A service outside the hub that every request path reads: the database, the IdP, and the secret store.            |
| `GLO-secret-store`  | secret store      | The service that gives each secret its protection. OpenBao is the secret store today.                             |
| `GLO-drain-period`  | drain period      | The time from the start of a shutdown to the close of the listener.                                               |

## Retired identifiers

This file has no retired identifier.
