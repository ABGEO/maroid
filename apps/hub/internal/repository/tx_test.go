package repository_test

import (
	"fmt"
	"testing"

	"github.com/jmoiron/sqlx"

	"github.com/abgeo/maroid/apps/hub/internal/database"
	"github.com/abgeo/maroid/libs/testdb"
)

// fetch runs load in a transaction of its own that commits when load succeeds, as a
// service runs a repository, and answers what load answers.
func fetch[T any](
	t *testing.T,
	instance *testdb.Instance,
	load func(*sqlx.Tx) (T, error),
) (T, error) {
	t.Helper()

	result, err := database.FetchTx(t.Context(), instance.DB, load)
	if err != nil {
		return result, fmt.Errorf("fetching in the test: %w", err)
	}

	return result, nil
}

// exec runs write in a transaction of its own that commits when write succeeds.
func exec(t *testing.T, instance *testdb.Instance, write func(*sqlx.Tx) error) error {
	t.Helper()

	if err := database.WithTx(t.Context(), instance.DB, write); err != nil {
		return fmt.Errorf("writing in the test: %w", err)
	}

	return nil
}
