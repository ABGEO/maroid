// Package appctx provides a centralized structure to hold
// application-wide dependencies.  It serves as a shared context
// for managing and accessing core services and plugin-related components
// throughout the application.
package appctx

import (
	"github.com/abgeo/maroid/apps/hub/internal/depresolver"
)

// AppContext holds application-wide dependencies.
type AppContext struct {
	DepResolver depresolver.Resolver
}
