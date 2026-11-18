package host

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"

	"github.com/abgeo/maroid/libs/pluginapi"
)

// FileStore reads, claims, and deletes the files of every plugin in the acting scope.
type FileStore interface {
	ReadFiles(
		ctx context.Context,
		pluginID *pluginapi.PluginID,
		ids ...string,
	) ([]pluginapi.File, error)
	ClaimFile(ctx context.Context, pluginID *pluginapi.PluginID, id string) error
	DeleteFiles(
		ctx context.Context,
		tx *sqlx.Tx,
		pluginID *pluginapi.PluginID,
		ids ...string,
	) error
}

type boundFiles struct {
	store    FileStore
	pluginID *pluginapi.PluginID
}

var _ pluginapi.PluginFiles = (*boundFiles)(nil)

func (f *boundFiles) Get(ctx context.Context, ids ...string) ([]pluginapi.File, error) {
	files, err := f.store.ReadFiles(ctx, f.pluginID, ids...)
	if err != nil {
		return nil, fmt.Errorf("reading the files: %w", err)
	}

	return files, nil
}

func (f *boundFiles) Claim(ctx context.Context, id string) error {
	if err := f.store.ClaimFile(ctx, f.pluginID, id); err != nil {
		return fmt.Errorf("claiming the file %q: %w", id, err)
	}

	return nil
}

func (f *boundFiles) Delete(ctx context.Context, tx *sqlx.Tx, ids ...string) error {
	if err := f.store.DeleteFiles(ctx, tx, f.pluginID, ids...); err != nil {
		return fmt.Errorf("deleting the files: %w", err)
	}

	return nil
}
