package auth

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"net/url"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/abgeo/maroid/apps/hub/internal/database"
	"github.com/abgeo/maroid/apps/hub/internal/domain/errs"
	"github.com/abgeo/maroid/apps/hub/internal/model"
	"github.com/abgeo/maroid/apps/hub/internal/repository"
)

// ErrNoDeckURL indicates that the configuration names no address for the deck, so
// no invitation can point a person at it.
var ErrNoDeckURL = errors.New("auth: auth.deck_url is empty")

// Service holds the operations that change the identities of a user record.
type Service struct {
	db             *sqlx.DB
	userRepo       repository.UserRepository
	identityRepo   repository.IdentityRepository
	invitationRepo repository.InvitationRepository
}

// NewService creates a new Service instance.
func NewService(
	db *sqlx.DB,
	userRepo repository.UserRepository,
	identityRepo repository.IdentityRepository,
	invitationRepo repository.InvitationRepository,
) *Service {
	return &Service{
		db:             db,
		userRepo:       userRepo,
		identityRepo:   identityRepo,
		invitationRepo: invitationRepo,
	}
}

// Attach binds the external account to the user record.
func (s *Service) Attach(
	ctx context.Context,
	userID string,
	provider string,
	providerUserID string,
	profile model.Profile,
) error {
	err := database.WithTx(ctx, s.db, func(tx *sqlx.Tx) error {
		return s.identityRepo.Attach(ctx, tx, userID, provider, providerUserID, profile)
	})

	if errors.Is(err, errs.ErrIdentityTaken) {
		return s.reportConflict(ctx, userID, provider, providerUserID, err)
	}

	if err != nil {
		return fmt.Errorf("attaching the external account: %w", err)
	}

	return nil
}

// Detach removes the identity of the provider from the user record.
func (s *Service) Detach(ctx context.Context, userID string, provider string) error {
	if err := s.identityRepo.Detach(ctx, userID, provider); err != nil {
		return fmt.Errorf("detaching the external account: %w", err)
	}

	return nil
}

// InviteRequest names what the owner wants an invitation for. An empty UserID
// asks for a new user record, and the two names then apply to it.
type InviteRequest struct {
	UserID    string
	FirstName string
	LastName  string
}

// InviteResult holds the record that the invitation names and the token that
// redeems it. The token appears here one time and never again.
type InviteResult struct {
	UserID string
	Token  string
}

// Invite writes the invitation, and the user record too when the request names
// none. Both go in one transaction.
func (s *Service) Invite(
	ctx context.Context,
	request InviteRequest,
	ttl time.Duration,
) (*InviteResult, error) {
	const tokenBytes = 32

	token, err := generateRandomString(tokenBytes)
	if err != nil {
		return nil, fmt.Errorf("generating the invitation token: %w", ErrRandomGeneration)
	}

	digest := sha256.Sum256([]byte(token))
	result := &InviteResult{UserID: request.UserID, Token: token}

	err = database.WithTx(ctx, s.db, func(tx *sqlx.Tx) error {
		if result.UserID == "" {
			user, createErr := s.userRepo.Create(ctx, tx, request.FirstName, request.LastName)
			if createErr != nil {
				return fmt.Errorf("creating the user record: %w", createErr)
			}

			result.UserID = user.ID
		}

		_, createErr := s.invitationRepo.Create(
			ctx, tx, result.UserID, digest[:], time.Now().Add(ttl),
		)
		if createErr != nil {
			return fmt.Errorf("writing the invitation: %w", createErr)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("issuing the invitation: %w", err)
	}

	return result, nil
}

// Redeem spends the invitation and binds the first identity of its record.
func (s *Service) Redeem(
	ctx context.Context,
	invitationID string,
	provider string,
	providerUserID string,
	profile model.Profile,
) (*model.User, error) {
	var user *model.User

	err := database.WithTx(ctx, s.db, func(tx *sqlx.Tx) error {
		invitation, consumeErr := s.invitationRepo.Consume(ctx, tx, invitationID)
		if consumeErr != nil {
			return fmt.Errorf("spending the invitation: %w", consumeErr)
		}

		attachErr := s.identityRepo.Attach(
			ctx, tx, invitation.UserID, provider, providerUserID, profile,
		)
		if attachErr != nil {
			return fmt.Errorf("binding the first identity: %w", attachErr)
		}

		user = &model.User{ID: invitation.UserID}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("redeeming the invitation: %w", err)
	}

	return user, nil
}

// reportConflict tells an account that this record already holds apart from one
// that names another record.
func (s *Service) reportConflict(
	ctx context.Context,
	userID string,
	provider string,
	providerUserID string,
	conflict error,
) error {
	owner, err := s.identityRepo.GetActiveUserByProvider(ctx, provider, providerUserID)
	if err == nil && owner.ID == userID {
		return nil
	}

	return fmt.Errorf("attaching the external account: %w", conflict)
}

// InviteAddress builds the address that the owner hands the person.
func InviteAddress(deckURL string, token string) (string, error) {
	if deckURL == "" {
		return "", ErrNoDeckURL
	}

	parsed, err := url.Parse(deckURL)
	if err != nil {
		return "", fmt.Errorf("reading the address of the deck: %w", err)
	}

	parsed.Path = "/invite"
	parsed.RawQuery = url.Values{"token": {token}}.Encode()

	return parsed.String(), nil
}
