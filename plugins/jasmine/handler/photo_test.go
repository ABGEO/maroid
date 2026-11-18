package handler_test

import (
	"fmt"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// insertFile stands in for an upload: it gives the workspace a ready plant photo
// and answers it.
func insertFile(t *testing.T, fixture *garden, workspace string) string {
	t.Helper()

	return insertFileOf(t, fixture, workspace, pluginID, "plant-photo", "ready")
}

// insertFileOf stands in for an upload of a file of the plugin, the kind, and the
// state.
func insertFileOf(t *testing.T, fixture *garden, workspace, plugin, kind, state string) string {
	t.Helper()

	var id string

	require.NoError(t, fixture.in(t, workspace, func(tx *sqlx.Tx) error {
		if err := tx.Get(
			&id,
			`INSERT INTO public.workspace_files (workspace_id, plugin_id, kind, state)
			VALUES ($1, $2, $3, $4) RETURNING id`,
			workspace,
			plugin,
			kind,
			state,
		); err != nil {
			return fmt.Errorf("inserting a file: %w", err)
		}

		return nil
	}))

	return id
}

// insertMoment gives the plant a moment of today that holds one photo of each file,
// and answers the moment and its photos.
func insertMoment(
	t *testing.T,
	fixture *garden,
	plantID string,
	files ...string,
) (string, []string) {
	t.Helper()

	var (
		moment string
		photos []string
	)

	require.NoError(t, fixture.inHome(t, func(tx *sqlx.Tx) error {
		if err := tx.Get(&moment,
			`INSERT INTO moments (plant_id, taken_on) VALUES ($1, current_date) RETURNING id`,
			plantID,
		); err != nil {
			return fmt.Errorf("inserting a moment: %w", err)
		}

		for _, file := range files {
			var photo string
			if err := tx.Get(
				&photo,
				`INSERT INTO photos (plant_id, moment_id, file_id) VALUES ($1, $2, $3) RETURNING id`,
				plantID,
				moment,
				file,
			); err != nil {
				return fmt.Errorf("inserting a photo: %w", err)
			}

			photos = append(photos, photo)
		}

		return nil
	}))

	return moment, photos
}

// GARDEN-SC-068: The database refuses a photo in a moment of another plant, a
// profile picture of another plant, and a moment of a plant of another workspace.
func TestThePhotosKeepTheirPlantAndTheirWorkspace(t *testing.T) {
	t.Parallel()

	fixture := gardenUnderTest(t)
	router := fixture.router(workspaceH)

	environment := makeEnvironment(t, router, "Office")
	first := makePlant(t, router, "Fern", environment.ID)
	second := makePlant(t, router, "Monstera", environment.ID)
	moment, _ := insertMoment(t, fixture, first.ID, insertFile(t, fixture, workspaceH))
	_, foreign := insertMoment(t, fixture, second.ID, insertFile(t, fixture, workspaceH))
	loose := insertFile(t, fixture, workspaceH)

	attempts := []struct {
		workspace  string
		statement  string
		args       []any
		constraint string
	}{
		{
			workspace:  workspaceH,
			statement:  `INSERT INTO photos (plant_id, moment_id, file_id) VALUES ($1, $2, $3)`,
			args:       []any{second.ID, moment, loose},
			constraint: "photos_moment_fkey",
		},
		{
			workspace:  workspaceH,
			statement:  `UPDATE plants SET profile_photo_id = $1 WHERE id = $2`,
			args:       []any{foreign[0], first.ID},
			constraint: "plants_profile_photo_fkey",
		},
		{
			workspace:  workspaceG,
			statement:  `INSERT INTO moments (plant_id, taken_on) VALUES ($1, current_date)`,
			args:       []any{first.ID},
			constraint: "moments_plant_fkey",
		},
	}

	for _, attempt := range attempts {
		err := fixture.in(t, attempt.workspace, func(tx *sqlx.Tx) error {
			if _, execErr := tx.Exec(attempt.statement, attempt.args...); execErr != nil {
				return fmt.Errorf("running the statement: %w", execErr)
			}

			return nil
		})
		require.Error(t, err, attempt.constraint)
		assert.Contains(t, err.Error(), "SQLSTATE 23503")
		assert.Contains(t, err.Error(), attempt.constraint)
	}
}
