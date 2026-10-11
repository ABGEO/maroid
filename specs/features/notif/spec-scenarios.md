---
id: NOTIF
title: The scenarios of the notification providers and the notification routes
type: spec
status: approved
created: 2026-10-11
updated: 2026-10-11
approved_by: Temuri
approved_on: 2026-10-11
constrained_by: [TST, OWN, TRC, NTF]
requirements: features/notif/requirements.md
---

# Specification: The scenarios of the notification providers and the notification routes

`spec.md` holds the design. This file holds the scenarios. See `SPC-001`.

Every integration scenario runs against the container that `testdb.Start` gives, with
every hub migration applied. Workspace A has user A as its manager and user V as a viewer.
Workspace B has user B as its manager. Each workspace enables the probe plugin, which
declares the topics `reminder` and `report`. The cipher is the transit stub of `PSET`.
The kind registry holds `telegram.Kind` with an HTTP client that points at a stub of the
Bot API. The stub answers `ok: true` unless a scenario says otherwise, and records each call.

## `NOTIF-SC-001`

**Verifies:** `NOTIF-FR-001`
**Layer:** unit

**Given** the probe plugin declares `reminder` with the name "Reminder".
**When** the notification topic registrar registers the plugin.
**Then** `NotificationTopicRegistry.Of` answers `reminder`, and the capability registry
records `notification_topics` with `["reminder"]` for the plugin.

## `NOTIF-SC-002`

**Verifies:** `NOTIF-FR-002`
**Layer:** unit

Each case loads a plugin that declares the topics in the table. The load fails, the error
names the plugin, and the registry holds no topic of it.

| Case           | Topics                                         |
| -------------- | ---------------------------------------------- |
| Repeated id    | `reminder`, `reminder`                         |
| Bad identifier | `Reminder-1`                                   |
| Long name      | `reminder` with a name of 65 characters        |
| Empty name     | `reminder` with the name "   "                 |

## `NOTIF-SC-003`

**Verifies:** `NOTIF-FR-003`, `NOTIF-FR-029`
**Layer:** unit

**Given** a `PluginNotifier` bound to the probe plugin, and a context that carries workspace A.
**When** the plugin calls `Notify(ctx, "invoice", msg)`.
**Then** the call returns an error that wraps `pluginapi.ErrNotificationTopicUndeclared`,
and the Bot API stub records no call.

## `NOTIF-SC-004`

**Verifies:** `NOTIF-FR-004`, `NOTIF-FR-005`, `NOTIF-FR-034`
**Layer:** integration

**Given** workspace A holds "Family bot" with a stored token.
**When** user V reads `GET /workspaces/A/notification-providers`.
**Then** the answer is 200. The item holds the name, the kind `telegram`, `enabled: true`,
and `fields.botToken` equal to `******`. No byte of the token or of its ciphertext appears.

## `NOTIF-SC-005`

**Verifies:** `NOTIF-FR-006`
**Layer:** integration

| Case   | Name                                        | Answer | Rows |
| ------ | ------------------------------------------- | ------ | ---- |
| Normal | "Family bot"                                | 201    | 1    |
| Limit  | 64 characters with a space at each end      | 201    | 1    |
| Over   | 65 characters                               | 422, `/name` | 0 |

**When** user A posts each body with a token to `/workspaces/A/notification-providers`.
**Then** the table gives the answer and the count of rows of workspace A.

## `NOTIF-SC-006`

**Verifies:** `NOTIF-FR-007`, `NOTIF-FR-008`
**Layer:** integration

**Given** "Family bot" with the token `T1`.
**When** user A sends `PATCH` with `name: "Home bot"` and `fields.botToken: "******"`.
**Then** the answer is 200, the name is "Home bot", the dispatcher decrypts `T1`, and the
Bot API stub records no `getMe`.

## `NOTIF-SC-007`

**Verifies:** `NOTIF-FR-009`
**Layer:** integration

**Given** "Family bot" carries routes for `reminder` and `report`.
**When** user A deletes the provider.
**Then** the answer is 204. Workspace A holds no route, and `GET /notification-topics`
marks both topics unrouted.
**When** user B deletes the same identifier in workspace B.
**Then** the answer is 404.

## `NOTIF-SC-008`

**Verifies:** `NOTIF-FR-010`
**Layer:** integration

**Given** "Family bot" carries an enabled route for `reminder`.
**When** user A patches `enabled: false`, and the plugin sends `reminder`.
**Then** the Bot API stub records no call, and the route still exists.
**When** user A patches `enabled: true`, and the plugin sends `reminder`.
**Then** the stub records one `sendMessage`.

## `NOTIF-SC-009`

**Verifies:** `NOTIF-FR-011`, `NOTIF-FR-012`
**Layer:** integration

| Case                 | Body                                      | Pointers                   |
| -------------------- | ----------------------------------------- | -------------------------- |
| Empty token          | `fields: {}`                              | `/fields/botToken`         |
| Unknown field        | `fields: {botToken: "x", proxy: "y"}`     | `/fields/proxy`            |
| Chat id not a number | route with `destination: {chatId: "abc"}` | `/destination/chatId`      |

**Then** each answer is 422 `settings-invalid`, each names the pointers of its row, and
no row changes.

## `NOTIF-SC-010`

**Verifies:** `NOTIF-FR-013`
**Layer:** integration

| Case        | Stub answer to `getMe`              | Answer                        |
| ----------- | ----------------------------------- | ----------------------------- |
| Accepted    | `ok: true`                          | 201                           |
| Refused     | `ok: false, description: "Unauthorized"` | 422 `delivery-refused`, detail "Unauthorized" |
| No answer   | no answer within 10 seconds         | 502 `messaging-unreachable`   |

**Then** only the accepted case stores a row.

## `NOTIF-SC-011`

**Verifies:** `NOTIF-FR-014`, `NOTIF-FR-015`
**Layer:** integration

**Given** `reminder` has one enabled route to chat `-100` thread `2749`, and `report` has none.
**When** user V reads `GET /workspaces/A/notification-topics`.
**Then** `reminder` holds the route with its provider and `destination: {chatId: "-100",
threadId: "2749"}`, and `unrouted: false`. `report` holds no route, and `unrouted: true`.

## `NOTIF-SC-012`

**Verifies:** `NOTIF-FR-016`
**Layer:** integration

| Case                  | Body                                                    | Answer                    |
| --------------------- | ------------------------------------------------------- | ------------------------- |
| Normal                | `reminder`, "Family bot", chat `-100`                   | 201                       |
| Second route          | `reminder`, "Family bot", chat `-200`                   | 201                       |
| Plugin not enabled    | a topic of a plugin that workspace A does not enable    | 422, `/plugin_id`         |
| Provider of B         | the identifier of a provider of workspace B             | 422, `/provider_id`       |

## `NOTIF-SC-013`

**Verifies:** `NOTIF-FR-017`
**Layer:** integration

**Given** a route of `reminder` through "Family bot" to `{chatId: "-100", threadId: "2749"}`.
**When** user A posts the same route with the members in the order `threadId`, `chatId`.
**Then** the answer is 409 `notification-route-exists`, and workspace A holds one route.

## `NOTIF-SC-014`

**Verifies:** `NOTIF-FR-018`, `NOTIF-FR-019`
**Layer:** integration

**Given** a route of `reminder` to chat `-100`.
**When** user A patches `destination.chatId` to `-200`, and the plugin sends `reminder`.
**Then** the stub records `chat_id: -200`.
**When** user A deletes the route.
**Then** the answer is 204, and `reminder` is unrouted.

## `NOTIF-SC-015`

**Verifies:** `NOTIF-FR-020`
**Layer:** integration

**Given** two routes of `reminder`.
**When** user A patches one with `enabled: false`, and the plugin sends `reminder`.
**Then** the stub records one call, to the enabled route.

## `NOTIF-SC-016`

**Verifies:** `NOTIF-FR-021`
**Layer:** integration

| Case     | Stub answer to `sendMessage`                                   | Answer                         |
| -------- | -------------------------------------------------------------- | ------------------------------ |
| Accepted | `ok: true`                                                     | 204                            |
| Refused  | `ok: false, description: "Bad Request: chat not found"`        | 422 `delivery-refused`, that detail |

**When** user A posts to `/notification-routes/{id}/test-messages` of a disabled route.
**Then** the table gives the answer, and the stub records one `sendMessage`.
**When** user V posts to `/notification-routes/{id}/test-messages`.
**Then** the answer is 403 `permission-denied`, and the stub records no call.

## `NOTIF-SC-017`

**Verifies:** `NOTIF-FR-022`
**Layer:** integration

**Given** a route of `reminder`.
**When** user A disables the probe plugin in workspace A.
**Then** `GET /notification-topics` lists no topic of the probe plugin, and the route row exists.
**When** user A enables the plugin again.
**Then** the topic list shows the route, and a send of `reminder` reaches the stub.

## `NOTIF-SC-018`

**Verifies:** `NOTIF-FR-023`
**Layer:** integration

**Given** a route of the topic `legacy`, written by SQL, and the probe plugin does not declare `legacy`.
**When** user V reads `GET /notification-topics`.
**Then** no item names `legacy`, and the row exists.

## `NOTIF-SC-019`

**Verifies:** `NOTIF-FR-024`
**Layer:** integration

**Given** `reminder` routes to chat `-100` through "Family bot", and to chat `-200` through
"Work bot". Workspace B routes `reminder` to chat `-900`.
**When** the plugin sends `reminder` with a context that carries workspace A.
**Then** the stub records `-100` and `-200`, in the order of the route identifiers, and not `-900`.

## `NOTIF-SC-020`

**Verifies:** `NOTIF-FR-025`, `NOTIF-FR-026`, `NOTIF-FR-031`
**Layer:** integration

| Case        | Route 1 | Route 2 | `Notify` returns | Error logs |
| ----------- | ------- | ------- | ---------------- | ---------- |
| One fails   | refused | ok      | nil              | 1          |
| Both fail   | refused | refused | an error         | 2          |

**Then** the stub records a call for route 2 in both cases. Each error log carries
`workspace_id`, `plugin`, `topic`, `route_id`, and the description of Telegram.

## `NOTIF-SC-021`

**Verifies:** `NOTIF-FR-027`, `NOTIF-FR-030`
**Layer:** integration

**Given** `report` is unrouted in workspace A.
**When** the plugin sends `report` with a context that carries workspace A.
**Then** `Notify` returns nil, the stub records no call, and one `INFO` log "notification
unrouted" carries `workspace_id`, `plugin`, and `topic`.

## `NOTIF-SC-022`

**Verifies:** `NOTIF-FR-028`, `NOTIF-FR-030`
**Layer:** unit

**Given** a context that carries user A and no workspace.
**When** the plugin sends `reminder`.
**Then** `Notify` returns nil, the dispatcher opens no transaction, and one `WARN` log
"notification has no workspace" carries `plugin` and `topic`.

## `NOTIF-SC-023`

**Verifies:** `NOTIF-FR-032`
**Layer:** unit

| Message                      | Stub records                                      |
| ---------------------------- | ------------------------------------------------- |
| No attachment                | `sendMessage` with the bold title and the body    |
| One PNG                      | `sendPhoto` with the caption                      |
| One PDF                      | `sendDocument` with the caption                   |
| Two PNG files and one PDF    | `sendMessage`, then two `sendMediaGroup` calls     |

**Then** each call carries `message_thread_id` when the destination holds `threadId`.

## `NOTIF-SC-024`

**Verifies:** `NOTIF-FR-033`, `NOTIF-INV-003`
**Layer:** integration

**Given** user A stores "Family bot" with the token `T1`.
**When** the test reads `fields` of the row with SQL.
**Then** the entry `botToken` holds the ciphertext that the stub cipher returned, and
does not hold `T1`. The stub cipher records the key `maroid-workspace-<A>`.

## `NOTIF-SC-025`

**Verifies:** `NOTIF-FR-035`
**Layer:** integration

**Given** a log handler that records every line at `DEBUG`.
**When** user A creates, updates, and tests "Family bot" with the token `T1`, and a send fails.
**Then** no line holds `T1`.

## `NOTIF-SC-026`

**Verifies:** `NOTIF-NFR-001`
**Layer:** integration

**Given** a route of `reminder` to chat `-100`.
**When** user A patches the chat to `-200`, waits 1 second after the response, and the plugin sends.
**Then** the stub records `-200`. The test repeats 20 times, and every run records `-200`.

## `NOTIF-SC-027`

**Verifies:** `NOTIF-NFR-002`
**Layer:** integration

**Given** workspace A holds 10 providers and 50 routes, of which 3 serve `reminder`, and
the cipher runs against the OpenBao container.
**When** the plugin sends `reminder` 200 times.
**Then** the 95th percentile of the time from the call of `Notify` to the first request
at the stub is no more than 50 milliseconds.

## `NOTIF-SC-028`

**Verifies:** `NOTIF-INV-001`, `NOTIF-INV-002`
**Layer:** integration

**Given** a provider of workspace B.
**When** an insert in the scope of workspace A names that provider in `notification_routes`.
**Then** the insert fails with a foreign key violation.
**When** a read in the scope of workspace A selects every row of both tables.
**Then** it answers no row of workspace B.

## `NOTIF-SC-029`

**Verifies:** `NOTIF-INV-004`
**Layer:** unit

**Given** a configuration file that holds the `notifier` section of the old form.
**When** the hub loads the configuration.
**Then** the load succeeds, and `config.Config` holds no field for it.

## Retired identifiers

This file has no retired identifier.
