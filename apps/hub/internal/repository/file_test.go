package repository_test

import (
	"context"
	"fmt"
	"testing"
	"testing/fstest"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/apps/hub/internal/database"
	"github.com/abgeo/maroid/apps/hub/internal/model"
	"github.com/abgeo/maroid/apps/hub/internal/repository"
	"github.com/abgeo/maroid/libs/pluginapi"
)

const (
	catName     = "cat.jpg"
	jpegType    = "image/jpeg"
	catSize     = 3 << 20
	photoKey    = "photo"
	otherPlugin = "dev.maroid.other"
)

func catDraft() model.FileDraft {
	return model.FileDraft{
		PluginID:  probePlugin,
		Kind:      photoKey,
		Name:      catName,
		MediaType: jpegType,
		Size:      catSize,
	}
}

// withFiles runs fn with a file repository in a transaction of the scope of the context.
func withFiles(ctx context.Context, t *testing.T, db *sqlx.DB, fn func(*repository.File) error) {
	t.Helper()

	require.NoError(t, database.WithScopeTx(ctx, db, func(tx *sqlx.Tx) error {
		return fn(repository.NewFile(tx))
	}))
}

func createFile(
	ctx context.Context,
	t *testing.T,
	db *sqlx.DB,
	scope pluginapi.FileScope,
) *model.File {
	t.Helper()

	var created *model.File

	withFiles(ctx, t, db, func(files *repository.File) error {
		var err error

		created, err = files.Create(ctx, scope, catDraft())
		require.NoError(t, err)

		return nil
	})

	return created
}

func listFiles(
	ctx context.Context,
	t *testing.T,
	db *sqlx.DB,
	pluginID string,
	ids ...string,
) []model.File {
	t.Helper()

	var listed []model.File

	withFiles(ctx, t, db, func(files *repository.File) error {
		var err error

		listed, err = files.List(ctx, pluginID, ids)
		require.NoError(t, err)

		return nil
	})

	return listed
}

// referenceFile writes a row of the probe that references the workspace file, as a
// table of a plugin does under DAT-003.
func (o *fileOwners) referenceFile(t *testing.T, file string) {
	t.Helper()

	o.instance.Migrate(t, "dev_maroid_probe", fstest.MapFS{
		"1_photos.up.sql": {Data: []byte(`CREATE TABLE dev_maroid_probe.photos (
			id UUID PRIMARY KEY DEFAULT uuidv7(),
			file_id UUID NOT NULL REFERENCES public.workspace_files (id) ON DELETE RESTRICT);`)},
		"1_photos.down.sql": {Data: []byte(`DROP TABLE dev_maroid_probe.photos;`)},
	})

	_, err := o.instance.DB.Exec(`INSERT INTO dev_maroid_probe.photos (file_id) VALUES ($1);`, file)
	require.NoError(t, err)
}

// FILES-SC-021: A file holds its state, its name, its media type, its size, the time
// of the start, and its uploader. The session gives the owner and the uploader.
func TestAFileHoldsTheFactsOfItsStart(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("skipping the integration test, it needs Docker")
	}

	owners := newFileOwners(t)
	ctx := actingIn(context.Background(), owners.userA, owners.workspaceA)

	for scope, owner := range map[pluginapi.FileScope]string{
		pluginapi.FileScopeWorkspace: owners.workspaceA,
		pluginapi.FileScopeUser:      owners.userA,
	} {
		created := createFile(ctx, t, owners.instance.DB, scope)

		listed := listFiles(ctx, t, owners.instance.DB, probePlugin, created.ID)
		require.Len(t, listed, 1)

		file := listed[0]
		require.Equal(t, scope, file.Scope)
		require.Equal(t, owner, file.OwnerID)
		require.Equal(t, owners.userA, file.UploadedBy)
		require.Equal(t, pluginapi.FileStatePending, file.State)
		require.Nil(t, file.Failure)
		require.Equal(t, catName, file.Name)
		require.Equal(t, jpegType, file.MediaType)
		require.Equal(t, int64(catSize), file.Size)
		require.False(t, file.CreatedAt.IsZero())
		require.Nil(t, file.ClaimedAt)
	}
}

// FILES-SC-041: A plugin reads, claims, and deletes no file that another plugin started.
func TestAPluginReachesNoFileOfAnotherPlugin(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("skipping the integration test, it needs Docker")
	}

	owners := newFileOwners(t)
	ctx := actingIn(context.Background(), owners.userA, owners.workspaceA)
	created := createFile(ctx, t, owners.instance.DB, pluginapi.FileScopeWorkspace)

	require.Empty(t, listFiles(ctx, t, owners.instance.DB, otherPlugin, created.ID))

	withFiles(ctx, t, owners.instance.DB, func(files *repository.File) error {
		claimed, err := files.Claim(ctx, otherPlugin, created.ID)
		require.NoError(t, err)
		require.False(t, claimed)

		deleted, err := files.Delete(ctx, otherPlugin, created.ID)
		require.NoError(t, err)
		require.Nil(t, deleted)

		return nil
	})

	require.Len(t, listFiles(ctx, t, owners.instance.DB, probePlugin, created.ID), 1)
}

// FILES-SC-028: A file of one workspace reaches no member acting in another workspace.
func TestAWorkspaceFileReachesNoOtherWorkspace(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("skipping the integration test, it needs Docker")
	}

	owners := newFileOwners(t)
	created := createFile(actingIn(context.Background(), owners.userB, owners.workspaceB), t,
		owners.instance.DB, pluginapi.FileScopeWorkspace)

	ctx := actingIn(context.Background(), owners.userA, owners.workspaceA)
	require.Empty(t, listFiles(ctx, t, owners.instance.DB, probePlugin, created.ID))
}

// FILES-SC-024: A file in pending, in processing, and in failed each takes a claim,
// and the expiry leaves each claimed file.
func TestAFileInAnyStateTakesAClaim(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("skipping the integration test, it needs Docker")
	}

	owners := newFileOwners(t)
	ctx := actingIn(context.Background(), owners.userA, owners.workspaceA)
	db := owners.instance.DB
	scope := pluginapi.FileScopeWorkspace

	pending := createFile(ctx, t, db, scope)
	processing := createFile(ctx, t, db, scope)
	failed := createFile(ctx, t, db, scope)

	withFiles(ctx, t, db, func(files *repository.File) error {
		_, err := files.StartProcessing(ctx, scope, processing.ID)
		require.NoError(t, err)

		_, err = files.MarkFailed(ctx, scope, failed.ID, pluginapi.FileFailureDamaged, "")
		require.NoError(t, err)

		for _, file := range []*model.File{pending, processing, failed} {
			claimed, claimErr := files.Claim(ctx, probePlugin, file.ID)
			require.NoError(t, claimErr)
			require.True(t, claimed)

			expired, expireErr := files.Expire(ctx, scope, file.ID)
			require.NoError(t, expireErr)
			require.False(t, expired)
		}

		return nil
	})

	require.Len(t, listFiles(ctx, t, db, probePlugin, pending.ID, processing.ID, failed.ID), 3)
}

// FILES-SC-022: The expiry removes a file that no consumer claimed and no row references.
func TestTheExpiryRemovesAnUnclaimedFile(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("skipping the integration test, it needs Docker")
	}

	owners := newFileOwners(t)
	ctx := actingIn(context.Background(), owners.userA, owners.workspaceA)
	created := createFile(ctx, t, owners.instance.DB, pluginapi.FileScopeUser)

	withFiles(ctx, t, owners.instance.DB, func(files *repository.File) error {
		expired, err := files.Expire(ctx, pluginapi.FileScopeUser, created.ID)
		require.NoError(t, err)
		require.True(t, expired)

		found, err := files.Find(ctx, pluginapi.FileScopeUser, created.ID)
		require.NoError(t, err)
		require.Nil(t, found)

		return nil
	})
}

// FILES-SC-023: The expiry of an unclaimed file that a row references claims the file
// in its place.
func TestTheExpiryClaimsAReferencedFile(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("skipping the integration test, it needs Docker")
	}

	owners := newFileOwners(t)
	ctx := actingIn(context.Background(), owners.userA, owners.workspaceA)
	scope := pluginapi.FileScopeWorkspace
	created := createFile(ctx, t, owners.instance.DB, scope)
	owners.referenceFile(t, created.ID)

	withFiles(ctx, t, owners.instance.DB, func(files *repository.File) error {
		expired, err := files.Expire(ctx, scope, created.ID)
		require.NoError(t, err)
		require.False(t, expired)

		found, err := files.Find(ctx, scope, created.ID)
		require.NoError(t, err)
		require.NotNil(t, found)
		require.NotNil(t, found.ClaimedAt)

		return nil
	})
}

// FILES-SC-026: The delete of a file that a row references fails with
// ErrFileReferenced, and the file stays.
func TestTheDeleteOfAReferencedFileFails(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("skipping the integration test, it needs Docker")
	}

	owners := newFileOwners(t)
	ctx := actingIn(context.Background(), owners.userA, owners.workspaceA)
	created := createFile(ctx, t, owners.instance.DB, pluginapi.FileScopeWorkspace)
	owners.referenceFile(t, created.ID)

	err := database.WithScopeTx(ctx, owners.instance.DB, func(tx *sqlx.Tx) error {
		_, deleteErr := repository.NewFile(tx).Delete(ctx, probePlugin, created.ID)

		return fmt.Errorf("deleting the file: %w", deleteErr)
	})
	require.ErrorIs(t, err, pluginapi.ErrFileReferenced)

	require.Len(t, listFiles(ctx, t, owners.instance.DB, probePlugin, created.ID), 1)
}

// FILES-SC-025: The delete of a claimed file that no row references removes it, and
// answers the row for the purge of its bytes.
func TestTheDeleteRemovesAClaimedFile(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("skipping the integration test, it needs Docker")
	}

	owners := newFileOwners(t)
	ctx := actingIn(context.Background(), owners.userA, owners.workspaceA)
	created := createFile(ctx, t, owners.instance.DB, pluginapi.FileScopeUser)

	withFiles(ctx, t, owners.instance.DB, func(files *repository.File) error {
		_, err := files.Claim(ctx, probePlugin, created.ID)
		require.NoError(t, err)

		deleted, err := files.Delete(ctx, probePlugin, created.ID)
		require.NoError(t, err)
		require.NotNil(t, deleted)
		require.Equal(t, pluginapi.FileScopeUser, deleted.Scope)
		require.Equal(t, owners.userA, deleted.OwnerID)

		return nil
	})

	require.Empty(t, listFiles(ctx, t, owners.instance.DB, probePlugin, created.ID))
}

// FILES-SC-012: A file whose content the kind does not accept fails with media_type,
// keeps the detected type, and never turns ready.
func TestAFailedFileKeepsTheDetectedType(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("skipping the integration test, it needs Docker")
	}

	owners := newFileOwners(t)
	ctx := actingIn(context.Background(), owners.userA, owners.workspaceA)
	scope := pluginapi.FileScopeWorkspace
	created := createFile(ctx, t, owners.instance.DB, scope)

	withFiles(ctx, t, owners.instance.DB, func(files *repository.File) error {
		moved, err := files.StartProcessing(ctx, scope, created.ID)
		require.NoError(t, err)
		require.True(t, moved)

		moved, err = files.MarkFailed(
			ctx,
			scope,
			created.ID,
			pluginapi.FileFailureMediaType,
			"application/pdf",
		)
		require.NoError(t, err)
		require.True(t, moved)

		moved, err = files.MarkReady(ctx, scope, created.ID, model.FileReady{MediaType: jpegType})
		require.NoError(t, err)
		require.False(t, moved)

		found, err := files.Find(ctx, scope, created.ID)
		require.NoError(t, err)
		require.Equal(t, pluginapi.FileStateFailed, found.State)
		require.Equal(t, pluginapi.FileFailureMediaType, *found.Failure)
		require.Equal(t, "application/pdf", found.MediaType)

		return nil
	})
}

// FILES-SC-017: A processed image turns ready with the detected type and the width and
// the height after the orientation. A second start of the processing moves nothing.
func TestAProcessedImageTurnsReadyWithItsSize(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("skipping the integration test, it needs Docker")
	}

	owners := newFileOwners(t)
	ctx := actingIn(context.Background(), owners.userA, owners.workspaceA)
	scope := pluginapi.FileScopeWorkspace
	created := createFile(ctx, t, owners.instance.DB, scope)
	width, height := 6048, 8064

	withFiles(ctx, t, owners.instance.DB, func(files *repository.File) error {
		moved, err := files.StartProcessing(ctx, scope, created.ID)
		require.NoError(t, err)
		require.True(t, moved)

		moved, err = files.StartProcessing(ctx, scope, created.ID)
		require.NoError(t, err)
		require.False(t, moved)

		moved, err = files.MarkReady(ctx, scope, created.ID,
			model.FileReady{MediaType: "image/heic", Width: &width, Height: &height})
		require.NoError(t, err)
		require.True(t, moved)

		found, err := files.Find(ctx, scope, created.ID)
		require.NoError(t, err)
		require.Equal(t, pluginapi.FileStateReady, found.State)
		require.Equal(t, "image/heic", found.MediaType)
		require.Equal(t, width, *found.Width)
		require.Equal(t, height, *found.Height)

		return nil
	})
}
