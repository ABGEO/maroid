package provider

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/jmoiron/sqlx"
	"golang.org/x/crypto/bcrypt"

	"github.com/abgeo/maroid/apps/hub/internal/database"
	"github.com/abgeo/maroid/apps/hub/internal/dex"
	"github.com/abgeo/maroid/apps/hub/internal/domain/errs"
	"github.com/abgeo/maroid/apps/hub/internal/model"
	"github.com/abgeo/maroid/apps/hub/internal/repository"
)

const (
	minPasswordRunes = 12
	// maxPasswordBytes is the length that bcrypt reads. A longer password differs from
	// the one that the person typed.
	maxPasswordBytes = 72
	// hashCost is the cost that Dex recommends, inside the range of 10 to 16 it accepts.
	hashCost = 12
)

// LocalAccounts gives, resets, and removes the local account of a user record.
type LocalAccounts interface {
	Give(ctx context.Context, userID string, email string, password []byte) error
	Reset(ctx context.Context, userID string, password []byte) error
	Remove(ctx context.Context, userID string) error
	EnsureProvider(ctx context.Context) error
	HasAccount(ctx context.Context, userID string) (bool, error)
}

// Accounts is the LocalAccounts that Dex and the identities back. The user_id of a
// password in Dex is the identifier of the record, so the identity of a local account
// is (local, the identifier of the record).
type Accounts struct {
	client    dex.Client
	db        *sqlx.DB
	providers *Manager
}

var _ LocalAccounts = (*Accounts)(nil)

// NewAccounts creates an Accounts.
func NewAccounts(client dex.Client, db *sqlx.DB, providers *Manager) *Accounts {
	return &Accounts{client: client, db: db, providers: providers}
}

// Give writes the identity and the password of a new local account. Dex answers inside
// the transaction of the identity, so a failure of Dex leaves no identity.
func (a *Accounts) Give(ctx context.Context, userID string, email string, password []byte) error {
	address, err := checkEmail(email)
	if err != nil {
		return err
	}

	hash, err := hashPassword(password)
	if err != nil {
		return err
	}

	if err = a.requireProvider(ctx); err != nil {
		return err
	}

	stored, err := a.storedPassword(ctx, userID, address)
	if err != nil {
		return err
	}

	err = database.WithTx(ctx, a.db, func(tx *sqlx.Tx) error {
		err := repository.NewIdentity(tx).Attach(ctx, userID, localID, userID, model.Profile{
			Username: address,
		})
		if errors.Is(err, errs.ErrIdentityTaken) {
			return fmt.Errorf("giving a local account: %w", errs.ErrLocalAccountExists)
		}

		if err != nil {
			return fmt.Errorf("attaching the local account: %w", err)
		}

		if stored {
			return nil
		}

		return a.client.CreatePassword(ctx, dex.Password{
			Email: address, Hash: hash, Name: address, UserID: userID,
		})
	})
	if err != nil {
		return fmt.Errorf("giving a local account: %w", err)
	}

	return nil
}

// Reset replaces the password of the local account of the record.
func (a *Accounts) Reset(ctx context.Context, userID string, password []byte) error {
	hash, err := hashPassword(password)
	if err != nil {
		return err
	}

	account, err := a.passwordOf(ctx, userID)
	if err != nil {
		return err
	}

	if err = a.client.UpdatePassword(ctx, account.Email, hash); err != nil {
		return fmt.Errorf("resetting a local account: %w", err)
	}

	return nil
}

// Remove deletes the identity and the password of the local account. The identity goes
// first, inside the transaction, so the check of the last identity refuses before Dex
// loses the password, and a failure of Dex keeps the identity.
func (a *Accounts) Remove(ctx context.Context, userID string) error {
	err := database.WithTx(ctx, a.db, func(tx *sqlx.Tx) error {
		if err := repository.NewIdentity(tx).Detach(ctx, userID, localID); err != nil {
			return fmt.Errorf("detaching the local account: %w", err)
		}

		account, err := a.passwordOf(ctx, userID)
		if errors.Is(err, errs.ErrLocalAccountNotFound) {
			return nil
		}

		if err != nil {
			return err
		}

		return a.client.DeletePassword(ctx, account.Email)
	})
	if err != nil {
		return fmt.Errorf("removing a local account: %w", err)
	}

	return nil
}

// HasAccount reports whether the record holds a local account.
func (a *Accounts) HasAccount(ctx context.Context, userID string) (bool, error) {
	identities, err := database.FetchTx(ctx, a.db, func(tx *sqlx.Tx) ([]model.Identity, error) {
		return repository.NewIdentity(tx).ListByUser(ctx, userID)
	})
	if err != nil {
		return false, fmt.Errorf("reading the identities of the record: %w", err)
	}

	return slices.ContainsFunc(identities, func(identity model.Identity) bool {
		return identity.Provider == localID
	}), nil
}

// EnsureProvider adds the local provider when Dex holds none.
func (a *Accounts) EnsureProvider(ctx context.Context) error {
	_, err := a.providers.Get(ctx, localID)
	if err == nil {
		return nil
	}

	if !errors.Is(err, errs.ErrProviderNotFound) {
		return err
	}

	_, err = a.providers.Create(ctx, Input{Preset: PresetLocal})
	if err != nil && !errors.Is(err, errs.ErrProviderExists) {
		return fmt.Errorf("adding the local provider: %w", err)
	}

	return nil
}

func (a *Accounts) requireProvider(ctx context.Context) error {
	_, err := a.providers.Get(ctx, localID)
	if errors.Is(err, errs.ErrProviderNotFound) {
		return fmt.Errorf("giving a local account: %w", errs.ErrLocalProviderAbsent)
	}

	return err
}

// storedPassword reports whether Dex already holds the password of this record for the
// address, which a call that passed its deadline leaves. Another account of the record,
// or of the address, refuses the local account.
func (a *Accounts) storedPassword(
	ctx context.Context,
	userID string,
	address string,
) (bool, error) {
	passwords, err := a.client.ListPasswords(ctx)
	if err != nil {
		return false, fmt.Errorf("listing the local accounts: %w", err)
	}

	for _, one := range passwords {
		ofRecord, ofAddress := one.UserID == userID, one.Email == address

		switch {
		case ofRecord && ofAddress:
			return true, nil
		case ofRecord, ofAddress:
			return false, fmt.Errorf("giving a local account: %w", errs.ErrLocalAccountExists)
		}
	}

	return false, nil
}

func (a *Accounts) passwordOf(ctx context.Context, userID string) (*dex.Password, error) {
	passwords, err := a.client.ListPasswords(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing the local accounts: %w", err)
	}

	for i := range passwords {
		if passwords[i].UserID == userID {
			return &passwords[i], nil
		}
	}

	return nil, fmt.Errorf("reading the local account: %w", errs.ErrLocalAccountNotFound)
}

// checkEmail answers the address in lower case, because Dex compares an address without
// regard to case.
func checkEmail(email string) (string, error) {
	trimmed := strings.TrimSpace(email)

	parsed, err := mail.ParseAddress(trimmed)
	if err != nil || parsed.Address != trimmed {
		return "", fieldError("/email", "the value is not an email address")
	}

	return strings.ToLower(trimmed), nil
}

func hashPassword(password []byte) ([]byte, error) {
	if utf8.RuneCount(password) < minPasswordRunes {
		return nil, fieldError("/password", "the password holds fewer than 12 characters")
	}

	if len(password) > maxPasswordBytes {
		return nil, fieldError("/password", "the password holds more than 72 bytes")
	}

	hash, err := bcrypt.GenerateFromPassword(password, hashCost)
	if err != nil {
		return nil, fmt.Errorf("hashing the password: %w", err)
	}

	return hash, nil
}
