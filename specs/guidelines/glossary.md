---
id: GLO
title: Glossary
type: glossary
status: active
created: 2026-09-11
updated: 2026-10-12
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
| `GLO-plugin-name`   | plugin name       | The name that a person reads for a plugin. The plugin declares it in its metadata. See `PLG-012`.                 |
| `GLO-plugin-description` | plugin description | One or two sentences that tell a person what a plugin does. It is optional. See `PLG-012`.               |
| `GLO-capability`    | capability        | A function that a plugin adds to the hub. The plugin gets it when it implements an interface in `libs/pluginapi`. |
| `GLO-host`          | host              | The `pluginapi.Host` interface. It is the only path from a plugin to an external resource.                        |
| `GLO-registrar`     | registrar         | A component in the hub. It finds one capability in a plugin and puts it into a registry.                          |
| `GLO-registry`      | registry          | A component in the hub. It holds the items of one capability for every plugin.                                    |
| `GLO-schema`        | schema            | The PostgreSQL schema that one plugin owns. The name comes from the plugin identifier.                            |
| `GLO-remote`        | remote            | A Module Federation bundle that holds the user interface of one plugin.                                           |
| `GLO-route-mounter` | route mounter     | A function in a remote. It renders one page of a plugin into an element of the deck.                              |
| `GLO-notification`  | notification      | A message that a plugin sends to a workspace through the notification routes of one notification topic.           |
| `GLO-notification-topic` | notification topic | One kind of notification that a plugin declares and sends, with an identifier, a name, and a description. |
| `GLO-notification-provider` | notification provider | An account at an external messaging service that a workspace stores. Maroid delivers a notification through it. Never write "provider" alone. |
| `GLO-notification-provider-kind` | notification provider kind | The external messaging service of a notification provider. It fixes the fields of the notification provider and of a destination. |
| `GLO-destination`   | destination       | The place at the external messaging service that receives a notification. For Telegram: a chat, and a thread of a forum. |
| `GLO-notification-route` | notification route | The rule of a workspace that sends one notification topic through one notification provider to one destination. |
| `GLO-unrouted`      | unrouted          | The state of a notification topic in a workspace with no enabled notification route.                              |
| `GLO-owner`         | owner             | The maintainer of the repository. The owner approves each stage.                                                  |
| `GLO-user`          | user              | A person that Maroid serves. One row in `public.users`. An identity binds it to an external account.              |
| `GLO-identity`      | identity          | The binding of one external account to one user record. One row in `public.identities`.                          |
| `GLO-idp`           | IdP               | The external identity provider that authenticates every person for Maroid. Dex is the IdP today.                  |
| `GLO-provider`      | provider          | One account system that Dex federates. Telegram is one. An administrator manages it, or the file of Dex holds it. |
| `GLO-provider-preset` | provider preset | The kind of a provider that the hub offers: local, Telegram, or generic OIDC. It fixes the fields that decide the user identifier. |
| `GLO-static-provider` | static provider | A provider that the file of Dex holds. The hub lists it and changes nothing on it. |
| `GLO-local-provider` | local provider   | The provider whose accounts sign in with an email address and a password that Dex holds. |
| `GLO-local-account` | local account    | The email address and the password of one person at the local provider. |
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
| `GLO-selected-workspace` | selected workspace | The workspace that one Telegram chat acts in. `OWN-003` reads it. |
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
| `GLO-plugin-secret` | plugin secret | A value that a plugin writes and reads under a name, in the scope of a user or of a workspace. No person reads it. |
| `GLO-expiry`        | expiry            | The time after which Maroid reports a plugin secret as absent.                                                    |
| `GLO-parking-service` | parking service | The external service of the Tbilisi municipality that holds the parking accounts. |
| `GLO-parking-account` | parking account | The account of one person at the parking service. |
| `GLO-connection` | connection | The session token of an external service that Maroid holds for the account of one user or of one workspace. |
| `GLO-pending-connection` | pending connection | The state between the request of a code and the acceptance of that code. |
| `GLO-code` | code | The number that the parking service sends by SMS to the phone of the parking account. |
| `GLO-connection-page` | connection page | The page of a plugin that shows a connection, and starts and ends it. |
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
| `GLO-store-access`  | store access      | The permission that the secret store grants to Maroid at a login. The store sets its end.                         |
| `GLO-page-route`    | page route        | A path that a remote declares, and the page that the deck opens for it.                                           |
| `GLO-parameter-segment` | parameter segment | A segment of a page route that takes any one value, and gives the page that value by a name.               |
| `GLO-navigation-entry` | navigation entry | A page route that the deck lists under the plugin in the navigation of the workspace.                          |
| `GLO-trail`         | trail             | The line of links above a page that names each level from the hub to the page.                                    |
| `GLO-page-title`    | page title        | The name that a page gives itself for the last link of the trail.                                                 |
| `GLO-home-page` | home page | The page of a workspace that the deck opens first for that workspace. |
| `GLO-widget` | widget | One card on the home page. A plugin declares it and answers its data. |
| `GLO-widget-key` | widget key | The name of a widget, unique within its plugin. |
| `GLO-widget-kind` | widget kind | The shape of the data of a widget: `stat` or `list`. |
| `GLO-widget-size` | widget size | The space of a widget on the home page: `xs`, `s`, `m`, or `l`. |
| `GLO-widget-state` | widget state | What one answer of a widget says: `ready`, `empty`, or `not set up`. |
| `GLO-water-service` | water service | The external service of Georgian Water and Power that holds the water accounts. |
| `GLO-customer` | customer | One supply point of the water service, at one address. One water account holds one or more customers. |
| `GLO-assigned-customer` | assigned customer | The customer of the water account that an editor assigns to a workspace. The workspace collects its readings. |
| `GLO-meter` | meter | One water meter of a customer, named by its serial number. |
| `GLO-reading` | reading | The value of one meter on one date, in cubic metres, with the previous value that the water service reports for that meter. |
| `GLO-usage` | usage | The value of a reading minus its previous value. |

## Retired identifiers

This file has no retired identifier.
