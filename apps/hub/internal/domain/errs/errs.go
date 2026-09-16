// Package errs defines common error variables used across the application.
package errs

import (
	"errors"
)

var (
	// ErrPluginAlreadyRegistered indicates that a plugin with the same ID
	// has already been loaded and registered.
	ErrPluginAlreadyRegistered = errors.New("plugin: already registered")
	// ErrPluginCapabilityNotSupported indicates that a plugin capability is not supported.
	ErrPluginCapabilityNotSupported = errors.New("plugin: capability not supported")
	// ErrInvalidPluginID indicates that a plugin configuration is missing its required ID, or it is not valid.
	ErrInvalidPluginID = errors.New("plugin: ID is missing or invalid")
	// ErrUnexpectedPluginSymbolType indicates that a plugin symbol has an unexpected type
	// (e.g., constructor symbol does not match the expected type).
	ErrUnexpectedPluginSymbolType = errors.New("plugin: symbol has unexpected type")
	// ErrIncompatiblePluginAPIVersion indicates that a plugin was built for a different
	// API version than the one expected by the host.
	ErrIncompatiblePluginAPIVersion = errors.New("plugin: incompatible API version")
	// ErrUnknownMigrationTarget is returned when a migration target is not recognized.
	ErrUnknownMigrationTarget = errors.New("unknown migration target")
	// ErrCommandAlreadyRegistered indicates that a command has already been registered.
	ErrCommandAlreadyRegistered = errors.New("command: already registered")
	// ErrMigrationSourceAlreadyRegistered indicates that a migration source has already been registered.
	ErrMigrationSourceAlreadyRegistered = errors.New("migration: source already registered")
	// ErrTelegramCommandAlreadyRegistered indicates that a telegram command has already been registered.
	ErrTelegramCommandAlreadyRegistered = errors.New("telegram command: already registered")
	// ErrCronAlreadyRegistered indicates that a cron job has already been registered.
	ErrCronAlreadyRegistered = errors.New("cron: already registered")
	// ErrTelegramConversationNotFound indicates that a telegram conversation was not found.
	ErrTelegramConversationNotFound = errors.New("telegram conversation: not found")
	// ErrTelegramConversationStepNotFound indicates that a step within a telegram conversation was not found.
	ErrTelegramConversationStepNotFound = errors.New("telegram conversation: step not found")
	// ErrMQTTSubscriberAlreadyRegistered indicates that an MQTT subscriber has already been registered for a topic.
	ErrMQTTSubscriberAlreadyRegistered = errors.New("mqtt subscriber: already registered for topic")
	// ErrInvalidMQTTTopic indicates that an MQTT subscriber topic is invalid.
	ErrInvalidMQTTTopic = errors.New("mqtt subscriber: invalid topic")
	// ErrMQTTBrokerNotConfigured indicates that MQTT subscribers are registered but no broker is configured.
	ErrMQTTBrokerNotConfigured = errors.New("mqtt: broker not configured")
	// ErrUnknownWorkerType indicates that a requested worker type is not registered.
	ErrUnknownWorkerType = errors.New("worker: unknown type")
	// ErrUnknownStatus indicates that a user record carries a status that OWN-002 does not allow.
	ErrUnknownStatus = errors.New("user: unknown status")
	// ErrUserNotFound indicates that no active user record holds the given identity.
	ErrUserNotFound = errors.New("user: not found")
	// ErrUnknownIntent indicates that an authorization flow carries an intent that
	// EXTID-DD-004 does not name.
	ErrUnknownIntent = errors.New("auth flow: unknown intent")
	// ErrIdentityTaken indicates that the external account already holds an identity
	// that names another user record. See EXTID-FR-005.
	ErrIdentityTaken = errors.New("identity: the external account belongs to another user")
	// ErrIdentityNotFound indicates that no identity of the user record names the
	// given provider.
	ErrIdentityNotFound = errors.New("identity: not found")
	// ErrLastIdentity indicates that the identity is the last one of its user record,
	// and EXTID-INV-002 keeps it.
	ErrLastIdentity = errors.New("identity: the last external account cannot be detached")
	// ErrInvitationNotValid indicates that the invitation is absent, consumed, or
	// expired. The three cases answer the same way.
	ErrInvitationNotValid = errors.New("invitation: not valid")
	// ErrAuthFlowNotFound indicates that no unconsumed authorization flow holds the
	// given state.
	ErrAuthFlowNotFound = errors.New("auth flow: not found")
	// ErrAuthFlowBindingMismatch indicates that the browser that finished the
	// authorization flow is not the browser that started it. A forged callback
	// reaches this, and so does a browser that lost the cookie.
	ErrAuthFlowBindingMismatch = errors.New("auth flow: the browser does not hold the binding")
	// ErrAuthFlowExpired indicates that the authorization flow outlived
	// auth.flow_ttl. The caller still holds the row, so it can report the failure
	// at the target that the row names.
	ErrAuthFlowExpired = errors.New("auth flow: expired")
	// ErrInvalidSettingsModel indicates that a plugin declared a settings model that
	// the hub cannot reflect into a schema.
	ErrInvalidSettingsModel = errors.New("settings model: invalid")
	// ErrSettingsSchemaAlreadyRegistered indicates that a settings schema has already
	// been registered for a plugin.
	ErrSettingsSchemaAlreadyRegistered = errors.New("settings schema: already registered")
	// ErrSettingsSchemaNotFound indicates that no loaded plugin holds a settings
	// schema under the given identifier.
	ErrSettingsSchemaNotFound = errors.New("settings schema: not found")
	// ErrProtectionUnavailable indicates that the service that protects a secret is
	// unreachable, or that it answered with no usable data.
	ErrProtectionUnavailable = errors.New("protection: unavailable")
	// ErrUnsupportedFieldsSource indicates that the driver returned a type that the
	// settings fields cannot decode.
	ErrUnsupportedFieldsSource = errors.New("plugin settings: unsupported fields source")
)
