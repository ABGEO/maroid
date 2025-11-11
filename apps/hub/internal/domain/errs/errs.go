// Package errs defines common error variables used across the application.
package errs

import (
	"errors"
)

var (
	// ErrPluginAlreadyRegistered indicates that a plugin with the same ID
	// has already been loaded and registered.
	ErrPluginAlreadyRegistered = errors.New("plugin: already registered")
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
)
