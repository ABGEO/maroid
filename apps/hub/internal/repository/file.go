package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jmoiron/sqlx"

	"github.com/abgeo/maroid/apps/hub/internal/domain/errs"
	"github.com/abgeo/maroid/apps/hub/internal/model"
	"github.com/abgeo/maroid/libs/pluginapi"
)

// The codes of a delete that a reference refuses: RESTRICT answers the first, and the
// default NO ACTION answers the second.
const (
	restrictViolation   = "23001"
	foreignKeyViolation = "23503"
)

// FileRepository defines the data access contract for the files of the workspace
// and of the user. A plugin reaches a file by its plugin and its identifier in both
// tables. The worker reaches a file by its scope and its identifier.
type FileRepository interface {
	Create(
		ctx context.Context,
		scope pluginapi.FileScope,
		draft model.FileDraft,
	) (*model.File, error)
	List(ctx context.Context, pluginID string, ids []string) ([]model.File, error)
	ListOfWorkspace(ctx context.Context) ([]model.File, error)
	Claim(ctx context.Context, pluginID string, id string) (bool, error)
	Delete(ctx context.Context, pluginID string, id string) (*model.File, error)
	Find(ctx context.Context, scope pluginapi.FileScope, id string) (*model.File, error)
	StartProcessing(ctx context.Context, scope pluginapi.FileScope, id string) (bool, error)
	MarkReady(
		ctx context.Context,
		scope pluginapi.FileScope,
		id string,
		ready model.FileReady,
	) (bool, error)
	MarkFailed(
		ctx context.Context,
		scope pluginapi.FileScope,
		id string,
		failure pluginapi.FileFailure,
		mediaType string,
	) (bool, error)
	Expire(ctx context.Context, scope pluginapi.FileScope, id string) (bool, error)
}

// File is a SQL based implementation of FileRepository.
type File struct {
	tx *sqlx.Tx
}

var _ FileRepository = (*File)(nil)

// NewFile creates a new File repository instance.
func NewFile(tx *sqlx.Tx) *File {
	return &File{tx: tx}
}

// fileTable names one table of the files, the column of its owner, and the column
// of its uploader.
type fileTable struct {
	scope    pluginapi.FileScope
	name     string
	owner    string
	uploader string
}

func fileTables() []fileTable {
	return []fileTable{
		{
			scope:    pluginapi.FileScopeWorkspace,
			name:     "public.workspace_files",
			owner:    workspaceOwner,
			uploader: "uploaded_by",
		},
		{
			scope:    pluginapi.FileScopeUser,
			name:     "public.user_files",
			owner:    userOwner,
			uploader: userOwner,
		},
	}
}

func fileTableOf(scope pluginapi.FileScope) (fileTable, error) {
	for _, table := range fileTables() {
		if table.scope == scope {
			return table, nil
		}
	}

	return fileTable{}, fmt.Errorf("%w: %q", errs.ErrUnknownFileScope, scope)
}

func (t fileTable) columns() string {
	return `id, '` + string(t.scope) + `' AS scope, ` + t.owner + ` AS owner_id, plugin_id, kind,
		state, failure, name, media_type, size, width, height, ` + t.uploader + ` AS uploaded_by,
		claimed_at, created_at, updated_at`
}

// Create writes a pending file of the scope. The session gives the owner and the uploader.
func (r *File) Create(
	ctx context.Context,
	scope pluginapi.FileScope,
	draft model.FileDraft,
) (*model.File, error) {
	table, err := fileTableOf(scope)
	if err != nil {
		return nil, err
	}

	query := `INSERT INTO ` + table.name + ` (plugin_id, kind, name, media_type, size)
		VALUES ($1, $2, $3, $4, $5) RETURNING ` + table.columns() + `;`

	var entity model.File

	err = r.tx.GetContext(
		ctx, &entity, query, draft.PluginID, draft.Kind, draft.Name, draft.MediaType, draft.Size,
	)
	if err != nil {
		return nil, fmt.Errorf("creating a File of the %s: %w", scope, err)
	}

	return &entity, nil
}

// List reads the files of the plugin under the identifiers, from both tables. The
// policies leave out every file outside the acting scope. Each identifier must be a UUID.
func (r *File) List(ctx context.Context, pluginID string, ids []string) ([]model.File, error) {
	if len(ids) == 0 {
		return []model.File{}, nil
	}

	selects := make([]string, 0, len(fileTables()))
	for _, table := range fileTables() {
		selects = append(selects, `SELECT `+table.columns()+` FROM `+table.name+`
			WHERE plugin_id = $1 AND id = ANY($2::uuid[])`)
	}

	query := strings.Join(selects, ` UNION ALL `) + ` ORDER BY id;`

	entities := []model.File{}

	if err := r.tx.SelectContext(ctx, &entities, query, pluginID, ids); err != nil {
		return nil, fmt.Errorf("listing the Files: %w", err)
	}

	return entities, nil
}

// ListOfWorkspace retrieves every file of the acting workspace, of every plugin.
func (r *File) ListOfWorkspace(ctx context.Context) ([]model.File, error) {
	table, err := fileTableOf(pluginapi.FileScopeWorkspace)
	if err != nil {
		return nil, err
	}

	entities := []model.File{}

	if err = r.tx.SelectContext(ctx, &entities,
		`SELECT `+table.columns()+` FROM `+table.name+` ORDER BY id;`); err != nil {
		return nil, fmt.Errorf("listing the Files of the workspace: %w", err)
	}

	return entities, nil
}

// Claim marks the file of the plugin as used, and keeps the time of the first claim.
// It answers false when the plugin and the acting scope hold no file under the identifier.
func (r *File) Claim(ctx context.Context, pluginID string, id string) (bool, error) {
	for _, table := range fileTables() {
		claimed, err := r.affected(ctx, `UPDATE `+table.name+`
			SET claimed_at = COALESCE(claimed_at, now()) WHERE plugin_id = $1 AND id = $2;`,
			pluginID, id)
		if err != nil {
			return false, fmt.Errorf("claiming a File of the %s: %w", table.scope, err)
		}

		if claimed {
			return true, nil
		}
	}

	return false, nil
}

// Delete removes the file of the plugin, and answers the row it removed, or nil when
// the plugin and the acting scope hold no file under the identifier. It answers
// pluginapi.ErrFileReferenced while a row references the file. The refusal aborts
// the transaction.
func (r *File) Delete(ctx context.Context, pluginID string, id string) (*model.File, error) {
	for _, table := range fileTables() {
		var entity model.File

		err := r.tx.GetContext(ctx, &entity, `DELETE FROM `+table.name+`
			WHERE plugin_id = $1 AND id = $2 RETURNING `+table.columns()+`;`, pluginID, id)

		switch {
		case err == nil:
			return &entity, nil
		case errors.Is(err, sql.ErrNoRows):
			continue
		case isReferenced(err):
			return nil, fmt.Errorf(
				"deleting a File of the %s: %w",
				table.scope,
				pluginapi.ErrFileReferenced,
			)
		default:
			return nil, fmt.Errorf("deleting a File of the %s: %w", table.scope, err)
		}
	}

	return nil, nil //nolint:nilnil // no row is an absent file, not a failure.
}

// Find reads the file of the scope under the identifier, of any plugin. It answers nil
// when the acting scope holds none.
func (r *File) Find(
	ctx context.Context,
	scope pluginapi.FileScope,
	id string,
) (*model.File, error) {
	table, err := fileTableOf(scope)
	if err != nil {
		return nil, err
	}

	var entity model.File

	err = r.tx.GetContext(ctx, &entity,
		`SELECT `+table.columns()+` FROM `+table.name+` WHERE id = $1;`, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil //nolint:nilnil // no row is an absent file, not a failure.
		}

		return nil, fmt.Errorf("finding a File of the %s: %w", scope, err)
	}

	return &entity, nil
}

// StartProcessing moves a pending file to processing. It answers false for a file in
// any other state.
func (r *File) StartProcessing(
	ctx context.Context,
	scope pluginapi.FileScope,
	id string,
) (bool, error) {
	table, err := fileTableOf(scope)
	if err != nil {
		return false, err
	}

	moved, err := r.affected(ctx, `UPDATE `+table.name+
		` SET state = 'processing' WHERE id = $1 AND state = 'pending';`, id)
	if err != nil {
		return false, fmt.Errorf("processing a File of the %s: %w", scope, err)
	}

	return moved, nil
}

// MarkReady moves a processing file to ready with the facts that the content gave.
// It answers false for a file in any other state.
func (r *File) MarkReady(
	ctx context.Context,
	scope pluginapi.FileScope,
	id string,
	ready model.FileReady,
) (bool, error) {
	table, err := fileTableOf(scope)
	if err != nil {
		return false, err
	}

	moved, err := r.affected(ctx, `UPDATE `+table.name+`
		SET state = 'ready', media_type = $2, width = $3, height = $4
		WHERE id = $1 AND state = 'processing';`, id, ready.MediaType, ready.Width, ready.Height)
	if err != nil {
		return false, fmt.Errorf("marking a File of the %s ready: %w", scope, err)
	}

	return moved, nil
}

// MarkFailed moves a pending or a processing file to failed with the reason. An empty
// media type keeps the declared one. It answers false for a file in any other state.
func (r *File) MarkFailed(
	ctx context.Context,
	scope pluginapi.FileScope,
	id string,
	failure pluginapi.FileFailure,
	mediaType string,
) (bool, error) {
	table, err := fileTableOf(scope)
	if err != nil {
		return false, err
	}

	moved, err := r.affected(ctx, `UPDATE `+table.name+`
		SET state = 'failed', failure = $2, media_type = COALESCE(NULLIF($3, ''), media_type)
		WHERE id = $1 AND state IN ('pending', 'processing');`, id, failure, mediaType)
	if err != nil {
		return false, fmt.Errorf("marking a File of the %s failed: %w", scope, err)
	}

	return moved, nil
}

// Expire removes an unclaimed file, and answers true when it did. A file that a row
// references takes a claim in its place, because the row proves the claim.
func (r *File) Expire(ctx context.Context, scope pluginapi.FileScope, id string) (bool, error) {
	table, err := fileTableOf(scope)
	if err != nil {
		return false, err
	}

	if _, err = r.tx.ExecContext(ctx, `SAVEPOINT file_expire;`); err != nil {
		return false, fmt.Errorf("expiring a File of the %s: %w", scope, err)
	}

	removed, err := r.affected(ctx,
		`DELETE FROM `+table.name+` WHERE id = $1 AND claimed_at IS NULL;`, id)
	if err == nil {
		return removed, nil
	}

	if !isReferenced(err) {
		return false, fmt.Errorf("expiring a File of the %s: %w", scope, err)
	}

	if _, err = r.tx.ExecContext(ctx, `ROLLBACK TO SAVEPOINT file_expire;`); err != nil {
		return false, fmt.Errorf("expiring a File of the %s: %w", scope, err)
	}

	if _, err = r.tx.ExecContext(ctx,
		`UPDATE `+table.name+` SET claimed_at = now() WHERE id = $1;`, id); err != nil {
		return false, fmt.Errorf("claiming a referenced File of the %s: %w", scope, err)
	}

	return false, nil
}

func (r *File) affected(ctx context.Context, query string, args ...any) (bool, error) {
	result, err := r.tx.ExecContext(ctx, query, args...)
	if err != nil {
		return false, err //nolint:wrapcheck // each caller names the operation.
	}

	count, err := result.RowsAffected()
	if err != nil {
		return false, err //nolint:wrapcheck // each caller names the operation.
	}

	return count > 0, nil
}

func isReferenced(err error) bool {
	var pgErr *pgconn.PgError

	return errors.As(err, &pgErr) &&
		(pgErr.Code == restrictViolation || pgErr.Code == foreignKeyViolation)
}
