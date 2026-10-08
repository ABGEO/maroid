package handler

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/jmoiron/sqlx"

	"github.com/abgeo/maroid/apps/hub/internal/auth"
	"github.com/abgeo/maroid/apps/hub/internal/database"
	"github.com/abgeo/maroid/apps/hub/internal/domain/errs"
	"github.com/abgeo/maroid/apps/hub/internal/domain/problems"
	"github.com/abgeo/maroid/apps/hub/internal/model"
	"github.com/abgeo/maroid/apps/hub/internal/provider"
	"github.com/abgeo/maroid/apps/hub/internal/repository"
	"github.com/abgeo/maroid/libs/rest/address"
	"github.com/abgeo/maroid/libs/rest/problem"
)

const identityProviderParam = "provider"

// userIdentityBody is one identity of a user record. A local account carries its email
// address as the username.
type userIdentityBody struct {
	Provider    string    `json:"provider"`
	Username    *string   `json:"username,omitempty"`
	DisplayName *string   `json:"display_name,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

// localAccountInput is the body of a new local account. The password never reaches an
// answer or a log.
type localAccountInput struct {
	Provider *string `json:"provider"`
	Email    *string `json:"email"`
	Password *string `json:"password"`
}

// passwordInput is the merge patch of a local account.
type passwordInput struct {
	Password *string `json:"password"`
}

func toUserIdentityBody(identity *model.Identity) userIdentityBody {
	return userIdentityBody{
		Provider:    identity.Provider,
		Username:    identity.Username,
		DisplayName: identity.DisplayName,
		CreatedAt:   identity.CreatedAt.UTC(),
	}
}

// Identities answers every identity of a user record, oldest first.
func (h *User) Identities(w http.ResponseWriter, r *http.Request) error {
	record, err := h.service.Get(r.Context(), chi.URLParam(r, userIDParam))
	if err != nil {
		return h.fail(w, r, err, "reading the user record")
	}

	identities, err := h.identitiesOf(r, record.ID)
	if err != nil {
		problem.Write(w, r, problem.NewInternal())

		return fmt.Errorf("listing the identities of the record: %w", err)
	}

	bodies := make([]userIdentityBody, 0, len(identities))
	for i := range identities {
		bodies = append(bodies, toUserIdentityBody(&identities[i]))
	}

	return answerPage(w, r, bodies)
}

// GiveLocalAccount gives a user record a local account.
func (h *User) GiveLocalAccount(w http.ResponseWriter, r *http.Request) error {
	var input localAccountInput

	if failure := decodeObject(r, &input); failure != nil {
		problem.Write(w, r, failure)

		return nil
	}

	if input.Provider == nil || *input.Provider != auth.ProviderLocal {
		problem.Write(
			w,
			r,
			fieldFailure("/provider", "an administrator gives a local account alone"),
		)

		return nil
	}

	record, err := h.service.Get(r.Context(), chi.URLParam(r, userIDParam))
	if err != nil {
		return h.fail(w, r, err, "reading the user record")
	}

	err = h.accounts.Give(
		r.Context(),
		record.ID,
		valueOf(input.Email),
		[]byte(valueOf(input.Password)),
	)
	if err != nil {
		return failLocalAccount(w, r, err, "giving a local account")
	}

	identities, err := h.identitiesOf(r, record.ID)
	if err != nil {
		problem.Write(w, r, problem.NewInternal())

		return fmt.Errorf("reading the new local account: %w", err)
	}

	for i := range identities {
		if identities[i].Provider == auth.ProviderLocal {
			w.Header().Set("Location", address.BaseFromContext(r.Context())+
				"/users/"+record.ID+"/identities/"+auth.ProviderLocal)
			render.Status(r, http.StatusCreated)
			render.JSON(w, r, toUserIdentityBody(&identities[i]))

			return nil
		}
	}

	problem.Write(w, r, problem.NewInternal())

	return fmt.Errorf("reading the new local account: %w", errs.ErrIdentityNotFound)
}

// ResetLocalAccount sets a new password for the local account of a user record.
func (h *User) ResetLocalAccount(w http.ResponseWriter, r *http.Request) error {
	if chi.URLParam(r, identityProviderParam) != auth.ProviderLocal {
		problem.Write(w, r, problem.NewNotFound())

		return nil
	}

	var input passwordInput

	if failure := decodeObject(r, &input); failure != nil {
		problem.Write(w, r, failure)

		return nil
	}

	err := h.accounts.Reset(
		r.Context(),
		chi.URLParam(r, userIDParam),
		[]byte(valueOf(input.Password)),
	)
	if err != nil {
		return failLocalAccount(w, r, err, "resetting a local account")
	}

	render.NoContent(w, r)

	return nil
}

// RemoveLocalAccount removes the local account of a user record.
func (h *User) RemoveLocalAccount(w http.ResponseWriter, r *http.Request) error {
	if chi.URLParam(r, identityProviderParam) != auth.ProviderLocal {
		problem.Write(w, r, problem.NewNotFound())

		return nil
	}

	if err := h.accounts.Remove(r.Context(), chi.URLParam(r, userIDParam)); err != nil {
		return failLocalAccount(w, r, err, "removing a local account")
	}

	render.NoContent(w, r)

	return nil
}

// identitiesOf reads every identity of the record, oldest first.
func (h *User) identitiesOf(r *http.Request, userID string) ([]model.Identity, error) {
	ctx := r.Context()

	identities, err := database.FetchTx(ctx, h.db, func(tx *sqlx.Tx) ([]model.Identity, error) {
		return repository.NewIdentity(tx).ListByUser(ctx, userID)
	})
	if err != nil {
		return nil, fmt.Errorf("reading the identities of the record: %w", err)
	}

	return identities, nil
}

// failLocalAccount answers the problem that a failure of a local account carries.
func failLocalAccount(w http.ResponseWriter, r *http.Request, err error, doing string) error {
	var field *provider.FieldError

	switch {
	case errors.As(err, &field):
		problem.Write(w, r, fieldFailure(field.Pointer, field.Detail))
	case errors.Is(err, errs.ErrLocalAccountExists):
		problem.Write(w, r, problems.NewLocalAccountExists())
	case errors.Is(err, errs.ErrLocalProviderAbsent):
		problem.Write(w, r, problems.NewLocalProviderAbsent())
	case errors.Is(err, errs.ErrLastIdentity):
		problem.Write(w, r, problems.NewIdentityLast())
	case errors.Is(err, errs.ErrLocalAccountNotFound), errors.Is(err, errs.ErrIdentityNotFound),
		errors.Is(err, errs.ErrUserNotFound):
		problem.Write(w, r, problem.NewNotFound())
	case errors.Is(err, errs.ErrIDPUnavailable):
		problem.Write(w, r, problems.NewNotReady(dependencyDex))

		return fmt.Errorf("%s: %w", doing, err)
	default:
		problem.Write(w, r, problem.NewInternal())

		return fmt.Errorf("%s: %w", doing, err)
	}

	return nil
}

func fieldFailure(pointer string, detail string) *problem.ValidationProblem {
	return problem.NewValidationFailed(problem.FieldFailure{Detail: detail, Pointer: pointer})
}
