package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/libs/rest/page"
	"github.com/abgeo/maroid/plugins/jasmine/errs"
	"github.com/abgeo/maroid/plugins/jasmine/model"
	"github.com/abgeo/maroid/plugins/jasmine/repository"
)

func newFile(t *testing.T, tx *sqlx.Tx) string {
	t.Helper()

	var id string

	require.NoError(
		t,
		tx.Get(&id, `INSERT INTO public.workspace_files DEFAULT VALUES RETURNING id;`),
	)

	return id
}

// addMoment gives the plant a moment of the day with one photo of a new file.
func addMoment(t *testing.T, tx *sqlx.Tx, plantID, day string) model.Moment {
	t.Helper()

	takenOn, err := time.Parse(time.DateOnly, day)
	require.NoError(t, err)

	moment := model.Moment{PlantID: plantID, TakenOn: takenOn}
	require.NoError(t, repository.NewMoment(tx).Insert(context.Background(), &moment))

	photos, err := repository.NewPhoto(tx).
		InsertMany(context.Background(), plantID, moment.ID, []string{newFile(t, tx)})
	require.NoError(t, err)

	moment.Photos = photos

	return moment
}

func daysOf(moments []model.Moment) []string {
	days := make([]string, len(moments))
	for i, moment := range moments {
		days[i] = moment.TakenOn.Format(time.DateOnly)
	}

	return days
}

// GARDEN-DD-020: The keyset reads the moments by day and then by identifier, the
// latest first, and a backward read answers the moments before its boundary in the
// same order.
func TestTheMomentsReadTheLatestDayFirst(t *testing.T) {
	t.Parallel()

	database, plants := plantsUnderTest(t, 1)
	tx := begin(t, database)
	ctx := context.Background()

	addMoment(t, tx, plants[0], "2026-03-12")
	first := addMoment(t, tx, plants[0], "2026-06-01")
	addMoment(t, tx, plants[0], "2026-06-02")
	second := addMoment(t, tx, plants[0], "2026-06-01")

	moments := repository.NewMoment(tx)

	head, err := moments.List(ctx, plants[0], page.Seek{Direction: page.DirectionForward, Limit: 2})
	require.NoError(t, err)
	require.Equal(t, []string{"2026-06-02", "2026-06-01"}, daysOf(head))
	require.Equal(t, second.ID, head[1].ID)
	require.Len(t, head[1].Photos, 1)

	tail, err := moments.List(ctx, plants[0], page.Seek{
		Direction: page.DirectionForward, Boundary: head[1].ID, Limit: 2,
	})
	require.NoError(t, err)
	require.Equal(t, []string{"2026-06-01", "2026-03-12"}, daysOf(tail))
	require.Equal(t, first.ID, tail[0].ID)

	back, err := moments.List(ctx, plants[0], page.Seek{
		Direction: page.DirectionBackward, Boundary: tail[0].ID, Limit: 2,
	})
	require.NoError(t, err)
	require.Equal(t, []string{head[0].ID, head[1].ID}, []string{back[0].ID, back[1].ID})
}

// GARDEN-INV-005: A file that is already a photo answers errs.ErrFileTaken.
func TestAFileIsThePhotoOfOneMoment(t *testing.T) {
	t.Parallel()

	database, plants := plantsUnderTest(t, 1)
	tx := begin(t, database)
	taken := addMoment(t, tx, plants[0], "2026-06-01")
	other := addMoment(t, tx, plants[0], "2026-06-02")

	_, err := repository.NewPhoto(tx).InsertMany(
		context.Background(), plants[0], other.ID, []string{taken.Photos[0].FileID},
	)
	require.ErrorIs(t, err, errs.ErrFileTaken)
}

// GARDEN-FR-063: The fill sets a profile picture only on a plant that has none, and
// the plant reads the file of its profile picture. GARDEN-INV-006: a photo of another
// plant answers errs.ErrPhotoForeign.
func TestTheProfilePictureIsAPhotoOfThePlant(t *testing.T) {
	t.Parallel()

	database, plants := plantsUnderTest(t, 2)
	tx := begin(t, database)
	ctx := context.Background()
	repo := repository.NewPlant(tx)

	first := addMoment(t, tx, plants[0], "2026-06-01")
	later := addMoment(t, tx, plants[0], "2026-06-02")
	foreign := addMoment(t, tx, plants[1], "2026-06-02")

	require.NoError(t, repo.FillProfilePhoto(ctx, plants[0], first.Photos[0].ID))
	require.NoError(t, repo.FillProfilePhoto(ctx, plants[0], later.Photos[0].ID))

	plant, err := repo.GetByID(ctx, plants[0])
	require.NoError(t, err)
	require.Equal(t, first.Photos[0].ID, *plant.ProfilePhotoID)
	require.Equal(t, first.Photos[0].FileID, *plant.ProfileFileID)

	plant, err = repo.SetProfilePhoto(ctx, plants[0], later.Photos[0].ID)
	require.NoError(t, err)
	require.Equal(t, later.Photos[0].FileID, *plant.ProfileFileID)

	_, err = repo.SetProfilePhoto(ctx, plants[0], foreign.Photos[0].ID)
	require.ErrorIs(t, err, errs.ErrPhotoForeign)
}
