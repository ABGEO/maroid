package repository_test

import (
	"crypto/sha256"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/apps/hub/internal/domain/errs"
	"github.com/abgeo/maroid/apps/hub/internal/model"
	"github.com/abgeo/maroid/apps/hub/internal/repository"
)

const flowTTL = 10 * time.Minute

func signInFlow(state string) model.AuthFlow {
	// EXTID-DD-001: The row carries the digest of the secret that the browser
	// holds, so no flow exists without one.
	digest := sha256.Sum256([]byte("binding-" + state))

	return model.AuthFlow{
		State:       state,
		Intent:      model.IntentSignIn,
		BindingHash: digest[:],
		Nonce:       "nonce-" + state,
		Verifier:    "verifier-" + state,
		Redirect:    "http://maroid.localhost",
		ExpiresAt:   time.Now().Add(flowTTL),
	}
}

// EXTID-FR-006: The state spends one time, so a replay of the callback finds
// nothing and cannot attach a second account.
func TestAuthFlowConsumesOneTime(t *testing.T) {
	t.Parallel()

	instance := startWithCoreMigrations(t)
	flowRepo := repository.NewAuthFlow(instance.DB)
	ctx := t.Context()

	created, err := flowRepo.Create(ctx, signInFlow("state-one"))
	require.NoError(t, err)
	require.NotEmpty(t, created.ID)
	require.Nil(t, created.ConsumedAt)

	consumed, err := flowRepo.ConsumeByState(ctx, "state-one")
	require.NoError(t, err)
	require.Equal(t, created.ID, consumed.ID)
	require.NotEmpty(t, consumed.BindingHash, "the row returns the binding digest")
	require.Equal(t, model.IntentSignIn, consumed.Intent)
	require.NotNil(t, consumed.ConsumedAt)

	_, err = flowRepo.ConsumeByState(ctx, "state-one")
	require.ErrorIs(t, err, errs.ErrAuthFlowNotFound, "the second callback finds nothing")

	_, err = flowRepo.ConsumeByState(ctx, "state-that-nobody-wrote")
	require.ErrorIs(t, err, errs.ErrAuthFlowNotFound)
}

// EXTID-FR-006: An attach names its user record, and no other intent does.
// The database refuses every other shape, so no handler can write one.
func TestAuthFlowIntentNamesItsTarget(t *testing.T) {
	t.Parallel()

	instance := startWithCoreMigrations(t)
	flowRepo := repository.NewAuthFlow(instance.DB)
	ctx := t.Context()

	userID := insertUser(t, instance, nameOfA)

	attachFlow := signInFlow("state-attach")
	attachFlow.Intent = model.IntentAttach
	attachFlow.UserID = &userID

	stored, err := flowRepo.Create(ctx, attachFlow)
	require.NoError(t, err)
	require.Equal(t, userID, *stored.UserID)

	t.Run("an attach with no user record is refused", func(t *testing.T) {
		t.Parallel()

		flow := signInFlow("state-attach-orphan")
		flow.Intent = model.IntentAttach

		_, err := flowRepo.Create(t.Context(), flow)
		require.ErrorContains(t, err, "auth_flows_intent_target_check")
	})

	t.Run("a sign in that names a user record is refused", func(t *testing.T) {
		t.Parallel()

		flow := signInFlow("state-signin-with-user")
		flow.UserID = &userID

		_, err := flowRepo.Create(t.Context(), flow)
		require.ErrorContains(t, err, "auth_flows_intent_target_check")
	})

	t.Run("a redemption with no invitation is refused", func(t *testing.T) {
		t.Parallel()

		flow := signInFlow("state-redeem-orphan")
		flow.Intent = model.IntentRedeem

		_, err := flowRepo.Create(t.Context(), flow)
		require.ErrorContains(t, err, "auth_flows_intent_target_check")
	})

	t.Run("an unknown intent is refused", func(t *testing.T) {
		t.Parallel()

		flow := signInFlow("state-unknown-intent")
		flow.Intent = "elevate"

		_, err := flowRepo.Create(t.Context(), flow)
		require.ErrorContains(t, err, "auth_flows_intent_check")
	})
}

// EXTID-FR-006: Two flows never share a state, because the callback reads the row
// by that value alone.
func TestAuthFlowStateIsUnique(t *testing.T) {
	t.Parallel()

	instance := startWithCoreMigrations(t)
	flowRepo := repository.NewAuthFlow(instance.DB)
	ctx := t.Context()

	_, err := flowRepo.Create(ctx, signInFlow("state-twice"))
	require.NoError(t, err)

	_, err = flowRepo.Create(ctx, signInFlow("state-twice"))
	require.ErrorContains(t, err, "auth_flows_state_key")
}
