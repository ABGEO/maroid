package handler_test

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jmoiron/sqlx"

	"github.com/abgeo/maroid/libs/pluginapi"
)

const (
	filesURL    = "https://files.test/"
	photoWidth  = 4032
	photoHeight = 3024

	foreignKeyViolation = "23503"
)

// fakeFiles stands in for the file host of the hub over the stand-in tables. It
// reads, claims, and deletes the files of jasmine in the acting workspace, and
// records a purge for each delete through the transaction of the plugin.
type fakeFiles struct {
	db pluginapi.PluginDB
}

var _ pluginapi.PluginFiles = (*fakeFiles)(nil)

type fileRow struct {
	ID      string `db:"id"`
	Kind    string `db:"kind"`
	State   string `db:"state"`
	Claimed bool   `db:"claimed"`
}

// Get answers the files of jasmine in the acting workspace, with an address of each
// variant of a ready file.
func (f *fakeFiles) Get(ctx context.Context, ids ...string) ([]pluginapi.File, error) {
	if len(ids) > pluginapi.MaxFilesPerGet {
		return nil, pluginapi.ErrTooManyFiles
	}

	var rows []fileRow

	err := f.db.WithTx(ctx, func(tx *sqlx.Tx) error {
		query, args, err := sqlx.In(`SELECT id, kind, state, claimed FROM public.workspace_files
			WHERE id IN (?) AND workspace_id = ? AND plugin_id = ?`,
			ids, pluginapi.ActingWorkspaceFromContext(ctx), pluginID)
		if err != nil {
			return fmt.Errorf("binding the files: %w", err)
		}

		return tx.SelectContext(ctx, &rows, tx.Rebind(query), args...)
	})
	if err != nil {
		return nil, fmt.Errorf("reading the files: %w", err)
	}

	files := make([]pluginapi.File, 0, len(rows))
	for _, row := range rows {
		files = append(files, fileOf(row))
	}

	return files, nil
}

// Claim marks the file of jasmine in the acting workspace as used.
func (f *fakeFiles) Claim(ctx context.Context, id string) error {
	err := f.db.WithTx(ctx, func(tx *sqlx.Tx) error {
		if _, err := tx.ExecContext(ctx,
			`UPDATE public.workspace_files SET claimed = TRUE WHERE id = $1 AND workspace_id = $2`,
			id, pluginapi.ActingWorkspaceFromContext(ctx)); err != nil {
			return fmt.Errorf("claiming the file: %w", err)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("claiming in the stand-in: %w", err)
	}

	return nil
}

// Delete removes the files and records their purges through the transaction, as the
// hub does.
func (f *fakeFiles) Delete(ctx context.Context, tx *sqlx.Tx, ids ...string) error {
	for _, id := range ids {
		result, err := tx.ExecContext(
			ctx,
			`DELETE FROM public.workspace_files WHERE id = $1 AND workspace_id = $2 AND plugin_id = $3`,
			id,
			pluginapi.ActingWorkspaceFromContext(ctx),
			pluginID,
		)

		var pgErr *pgconn.PgError

		switch {
		case errors.As(err, &pgErr) && pgErr.Code == foreignKeyViolation:
			return pluginapi.ErrFileReferenced
		case err != nil:
			return fmt.Errorf("deleting the file: %w", err)
		}

		if deleted, _ := result.RowsAffected(); deleted == 0 {
			return pluginapi.ErrFileAbsent
		}

		if _, err = tx.ExecContext(ctx,
			`INSERT INTO public.file_purges (file_id) VALUES ($1)`, id); err != nil {
			return fmt.Errorf("recording the purge: %w", err)
		}
	}

	return nil
}

func fileOf(row fileRow) pluginapi.File {
	file := pluginapi.File{
		ID:      row.ID,
		Kind:    row.Kind,
		State:   pluginapi.FileState(row.State),
		Width:   photoWidth,
		Height:  photoHeight,
		Claimed: row.Claimed,
	}

	if file.State == pluginapi.FileStateReady {
		expires := time.Now().Add(time.Hour)
		file.Addresses = map[pluginapi.FileVariant]pluginapi.ReadAddress{
			pluginapi.FileVariantThumbnail: {
				URL:       filesURL + row.ID + "/thumbnail",
				ExpiresAt: expires,
			},
			pluginapi.FileVariantDisplay: {
				URL:       filesURL + row.ID + "/display",
				ExpiresAt: expires,
			},
		}
	}

	return file
}
