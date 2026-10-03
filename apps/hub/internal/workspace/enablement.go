package workspace

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"slices"

	"github.com/abgeo/maroid/apps/hub/internal/auth"
	"github.com/abgeo/maroid/apps/hub/internal/domain/errs"
	"github.com/abgeo/maroid/apps/hub/internal/model"
	"github.com/abgeo/maroid/apps/hub/internal/repository"
	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/libs/rest/problem"
)

// EnablementService enables and disables the plugins of the acting workspace, and
// answers whether a workspace enables a plugin.
type EnablementService interface {
	EnablementChecker
	Enabled(ctx context.Context) ([]model.Enablement, error)
	Enablement(ctx context.Context, pluginID string) (*model.Enablement, error)
	Enable(ctx context.Context, pluginID string) (*model.Enablement, bool, error)
	Disable(ctx context.Context, pluginID string) error
	WorkspacesEnabling(ctx context.Context, pluginID string) ([]string, error)
}

// EnablementChecker answers whether a workspace enables a plugin. An entry point of a
// plugin reads it.
type EnablementChecker interface {
	IsEnabled(ctx context.Context, workspaceID string, pluginID string) (bool, error)
}

// PluginCatalog lists the plugins that the hub loaded. registry.PluginRegistry
// satisfies it.
type PluginCatalog interface {
	All() []pluginapi.Plugin
}

// Enablements is the implementation of EnablementService over the repositories of the hub.
type Enablements struct {
	enablements repository.WorkspacePluginRepository
	allowed     repository.AllowedPluginRepository
	plugins     PluginCatalog
}

var _ EnablementService = (*Enablements)(nil)

// NewEnablements creates a new Enablements.
func NewEnablements(
	enablements repository.WorkspacePluginRepository,
	allowed repository.AllowedPluginRepository,
	plugins PluginCatalog,
) *Enablements {
	return &Enablements{enablements: enablements, allowed: allowed, plugins: plugins}
}

// IsEnabled reports whether the workspace enables the plugin.
func (e *Enablements) IsEnabled(
	ctx context.Context,
	workspaceID string,
	pluginID string,
) (bool, error) {
	_, err := e.enablements.Get(ctx, workspaceID, pluginID)
	if errors.Is(err, errs.ErrEnablementNotFound) {
		return false, nil
	}

	if err != nil {
		return false, fmt.Errorf("reading the enablement: %w", err)
	}

	return true, nil
}

// Enabled lists the plugins that the acting workspace enables.
func (e *Enablements) Enabled(ctx context.Context) ([]model.Enablement, error) {
	enabled, err := e.enablements.List(ctx, pluginapi.ActingWorkspaceFromContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("listing the plugins of the acting workspace: %w", err)
	}

	return enabled, nil
}

// Enablement reads one plugin that the acting workspace enables.
func (e *Enablements) Enablement(ctx context.Context, pluginID string) (*model.Enablement, error) {
	enabled, err := e.enablements.Get(ctx, pluginapi.ActingWorkspaceFromContext(ctx), pluginID)
	if err != nil {
		return nil, fmt.Errorf("reading a plugin of the acting workspace: %w", err)
	}

	return enabled, nil
}

// Enable enables a loaded plugin in the acting workspace. A manager enables a plugin
// of their allowlist, and an administrator any loaded plugin. The second answer is
// true when the call wrote the row.
func (e *Enablements) Enable(
	ctx context.Context,
	pluginID string,
) (*model.Enablement, bool, error) {
	if !e.loaded(pluginID) {
		return nil, false, fmt.Errorf("%w: %s", errs.ErrPluginNotLoaded, pluginID)
	}

	if !auth.IsAdministratorFromContext(ctx) {
		allowed, err := e.allowed.List(ctx, pluginapi.ActingUserFromContext(ctx))
		if err != nil {
			return nil, false, fmt.Errorf("reading the allowlist: %w", err)
		}

		if !slices.ContainsFunc(allowed, func(one model.AllowedPlugin) bool {
			return one.PluginID == pluginID
		}) {
			return nil, false, fmt.Errorf("%w: %s", errs.ErrPluginNotAllowed, pluginID)
		}
	}

	enabled, created, err := e.enablements.Add(
		ctx,
		pluginapi.ActingWorkspaceFromContext(ctx),
		pluginID,
	)
	if err != nil {
		return nil, false, fmt.Errorf("enabling the plugin: %w", err)
	}

	return enabled, created, nil
}

// Disable disables a plugin in the acting workspace. Its records and its settings stay.
func (e *Enablements) Disable(ctx context.Context, pluginID string) error {
	if err := e.enablements.Remove(
		ctx,
		pluginapi.ActingWorkspaceFromContext(ctx),
		pluginID,
	); err != nil {
		return fmt.Errorf("disabling the plugin: %w", err)
	}

	return nil
}

// WorkspacesEnabling lists every workspace that enables the plugin.
func (e *Enablements) WorkspacesEnabling(ctx context.Context, pluginID string) ([]string, error) {
	workspaces, err := e.enablements.WorkspacesEnabling(ctx, pluginID)
	if err != nil {
		return nil, fmt.Errorf("listing the workspaces that enable the plugin: %w", err)
	}

	return workspaces, nil
}

func (e *Enablements) loaded(pluginID string) bool {
	return slices.ContainsFunc(e.plugins.All(), func(plugin pluginapi.Plugin) bool {
		return plugin.Meta().ID.String() == pluginID
	})
}

// RequireEnabled answers not-found unless the acting workspace enables the plugin, as
// for a route that does not exist. It runs after Middleware and before Require.
func RequireEnabled(
	logger *slog.Logger,
	enablements EnablementChecker,
	pluginID string,
) func(http.Handler) http.Handler {
	return RequireEnabledOf(logger, enablements, func(*http.Request) string { return pluginID })
}

// RequireEnabledOf checks the plugin that pick names for each request, for a route
// whose plugin is a segment of its path.
func RequireEnabledOf(
	logger *slog.Logger,
	enablements EnablementChecker,
	pick func(*http.Request) string,
) func(http.Handler) http.Handler {
	logger = logger.With(
		slog.String("component", "middleware"),
		slog.String("middleware", "enablement"),
	)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			enabled, err := enablements.IsEnabled(
				ctx,
				pluginapi.ActingWorkspaceFromContext(ctx),
				pick(r),
			)
			if err != nil {
				logger.ErrorContext(ctx, "reading the enablement failed", slog.Any("error", err))
				problem.Write(w, r, problem.NewInternal())

				return
			}

			if !enabled {
				problem.Write(w, r, problem.NewNotFound())

				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
