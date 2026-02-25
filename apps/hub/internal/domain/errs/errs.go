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
)
