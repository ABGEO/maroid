package handler

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"

	"github.com/abgeo/maroid/apps/hub/internal/auth"
	"github.com/abgeo/maroid/apps/hub/internal/authz"
	"github.com/abgeo/maroid/apps/hub/internal/domain/errs"
	"github.com/abgeo/maroid/apps/hub/internal/domain/problems"
	"github.com/abgeo/maroid/apps/hub/internal/model"
	"github.com/abgeo/maroid/apps/hub/internal/registry"
	"github.com/abgeo/maroid/apps/hub/internal/workspace"
	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/libs/rest/problem"
)

// The settings routes of a plugin already name this segment, and chi refuses two
// names for one segment of one router.
const (
	widgetPluginParam = "id"
	widgetKeyParam    = "key"
)

// EnabledPlugins lists the plugins that the acting workspace enables, in the order
// of their identifiers.
type EnabledPlugins interface {
	Enabled(ctx context.Context) ([]model.Enablement, error)
}

// WidgetReader answers the data of one widget for the acting user.
type WidgetReader interface {
	Read(ctx context.Context, target registry.Widget) (pluginapi.WidgetAnswer, error)
}

// Widget is the handler of the widgets of the home page.
type Widget struct {
	logger   *slog.Logger
	verifier auth.TokenVerifier
	resolver auth.IdentityResolver
	access   WorkspaceAccess
	enabled  EnabledPlugins
	widgets  *registry.WidgetRegistry
	reader   WidgetReader
}

var _ Handler = (*Widget)(nil)

// NewWidget creates a new Widget handler.
func NewWidget(
	logger *slog.Logger,
	verifier auth.TokenVerifier,
	resolver auth.IdentityResolver,
	access WorkspaceAccess,
	enabled EnabledPlugins,
	widgets *registry.WidgetRegistry,
	reader WidgetReader,
) *Widget {
	return &Widget{
		logger: logger.With(
			slog.String("component", "handler"),
			slog.String("handler", "widget"),
		),
		verifier: verifier,
		resolver: resolver,
		access:   access,
		enabled:  enabled,
		widgets:  widgets,
		reader:   reader,
	}
}

// Register registers the routes of the widgets. Each takes a route of its own, as the
// notification routes do, because a mount of /workspaces/{workspaceId} would shadow
// the routes of the workspace handler. Neither admits an administrator who is no
// member, because a widget reads the records of the workspace.
func (h *Widget) Register(router chi.Router) {
	h.logger.Debug("registering routes")

	base := "/workspaces/{" + workspace.PathParam + "}"
	enabled := workspace.RequireEnabledOf(h.logger, h.access.Enablements,
		func(r *http.Request) string { return chi.URLParam(r, widgetPluginParam) },
	)

	router.Group(func(router chi.Router) {
		router.Use(auth.Middleware(h.logger, h.verifier, h.resolver))
		router.Use(workspace.Middleware(h.logger, h.access.DB))

		read := workspace.Require(h.logger, h.access.Authorizer, authz.PermissionWorkspaceRead)

		router.With(read).Get(base+"/widgets", Wrap(h.logger, h.List))
		router.With(enabled, read).Get(
			base+"/plugins/{"+widgetPluginParam+"}/widgets",
			Wrap(h.logger, h.ListOfPlugin),
		)
		router.With(enabled).Get(
			base+"/plugins/{"+widgetPluginParam+"}/widgets/{"+widgetKeyParam+"}",
			Wrap(h.logger, h.Read),
		)
	})
}

// List answers every widget of the enabled plugins whose permission the role of the
// acting user holds.
func (h *Widget) List(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()

	enabled, err := h.enabled.Enabled(ctx)
	if err != nil {
		problem.Write(w, r, problem.NewInternal())

		return fmt.Errorf("listing the widgets: %w", err)
	}

	bodies := make([]registry.WidgetItem, 0, len(enabled))

	for _, enablement := range enabled {
		items, allowErr := h.allowedOf(r, enablement.PluginID)
		if allowErr != nil {
			problem.Write(w, r, problem.NewInternal())

			return allowErr
		}

		bodies = append(bodies, items...)
	}

	return answerPage(w, r, bodies)
}

// ListOfPlugin answers the widgets of one enabled plugin whose permission the role of
// the acting user holds.
func (h *Widget) ListOfPlugin(w http.ResponseWriter, r *http.Request) error {
	bodies, err := h.allowedOf(r, chi.URLParam(r, widgetPluginParam))
	if err != nil {
		problem.Write(w, r, problem.NewInternal())

		return err
	}

	return answerPage(w, r, bodies)
}

// Read answers the data of one widget, after the hub checks its permission.
func (h *Widget) Read(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	pluginID := chi.URLParam(r, widgetPluginParam)

	held, found := h.widgets.Get(pluginID, chi.URLParam(r, widgetKeyParam))
	if !found {
		problem.Write(w, r, problem.NewNotFound())

		return nil
	}

	allowed, lowest, err := h.access.Authorizer.Allowed(
		workspace.RoleFromContext(ctx),
		held.Permission,
	)
	if err != nil {
		problem.Write(w, r, problem.NewInternal())

		return fmt.Errorf("checking the permission %s of a widget: %w", held.Permission, err)
	}

	if !allowed {
		problem.Write(w, r, problem.NewPermissionDenied(held.Permission, string(lowest)))

		return nil
	}

	answer, err := h.reader.Read(ctx, held)

	switch {
	case errors.Is(err, errs.ErrWidgetTimeout):
		problem.Write(w, r, problems.NewWidgetTimeout())

		return fmt.Errorf("reading a widget: %w", err)
	case err != nil:
		problem.Write(w, r, problem.NewInternal())

		return fmt.Errorf("reading a widget: %w", err)
	}

	render.JSON(w, r, answerBodyOf(answer))

	return nil
}

func (h *Widget) allowedOf(r *http.Request, pluginID string) ([]registry.WidgetItem, error) {
	role := workspace.RoleFromContext(r.Context())
	held := h.widgets.Of(pluginID)
	items := make([]registry.WidgetItem, 0, len(held))

	for _, one := range held {
		allowed, _, err := h.access.Authorizer.Allowed(role, one.Permission)
		if err != nil {
			return nil, fmt.Errorf(
				"checking the permission %s of a widget: %w",
				one.Permission,
				err,
			)
		}

		if allowed {
			items = append(items, itemOf(one))
		}
	}

	return items, nil
}

func itemOf(held registry.Widget) registry.WidgetItem {
	return registry.WidgetItem{
		PluginID:   held.PluginID,
		Key:        held.Meta.Key,
		Title:      held.Meta.Title,
		Kind:       string(held.Meta.Kind),
		Size:       string(held.Meta.Size),
		Permission: held.Permission,
	}
}

type widgetAnswerBody struct {
	State   string          `json:"state"`
	Message string          `json:"message,omitempty"`
	Link    string          `json:"link,omitempty"`
	Stat    *widgetStatBody `json:"stat,omitempty"`
	List    *widgetListBody `json:"list,omitempty"`
}

type widgetStatBody struct {
	Value   string `json:"value"`
	Unit    string `json:"unit,omitempty"`
	Caption string `json:"caption,omitempty"`
	Link    string `json:"link,omitempty"`
}

type widgetListBody struct {
	Items []widgetListItemBody `json:"items"`
	More  string               `json:"more,omitempty"`
}

type widgetListItemBody struct {
	Label string `json:"label"`
	Meta  string `json:"meta,omitempty"`
	Link  string `json:"link,omitempty"`
}

func answerBodyOf(answer pluginapi.WidgetAnswer) widgetAnswerBody {
	body := widgetAnswerBody{
		State:   string(answer.State),
		Message: answer.Message,
		Link:    answer.Link,
	}

	if answer.Stat != nil {
		body.Stat = &widgetStatBody{
			Value:   answer.Stat.Value,
			Unit:    answer.Stat.Unit,
			Caption: answer.Stat.Caption,
			Link:    answer.Stat.Link,
		}
	}

	if answer.List != nil {
		items := make([]widgetListItemBody, 0, len(answer.List.Items))
		for _, item := range answer.List.Items {
			items = append(items, widgetListItemBody(item))
		}

		body.List = &widgetListBody{Items: items, More: answer.List.More}
	}

	return body
}
