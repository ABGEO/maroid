package handler

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/libs/rest/problem"
	"github.com/abgeo/maroid/plugins/jasmine/model"
)

const (
	detailNotAPhoto   = "The file is not a plant photo of this workspace."
	detailPhotoFailed = "The photo failed. Remove it and upload it again."
)

// readFiles answers the files under the identifiers, by identifier. It reads them in
// groups that one call of Get takes.
func readFiles(
	ctx context.Context,
	files pluginapi.PluginFiles,
	ids []string,
) (map[string]pluginapi.File, error) {
	found := make(map[string]pluginapi.File, len(ids))

	for start := 0; start < len(ids); start += pluginapi.MaxFilesPerGet {
		group := ids[start:min(start+pluginapi.MaxFilesPerGet, len(ids))]

		read, err := files.Get(ctx, group...)
		if err != nil {
			return nil, fmt.Errorf("reading the files: %w", err)
		}

		for _, file := range read {
			found[file.ID] = file
		}
	}

	return found, nil
}

// addressesOf answers the files under the identifiers, so an answer carries their
// addresses. A failure leaves every address out, because the record itself stands.
func addressesOf(
	ctx context.Context,
	logger *slog.Logger,
	files pluginapi.PluginFiles,
	ids []string,
) map[string]pluginapi.File {
	found, err := readFiles(ctx, files, ids)
	if err != nil {
		logger.ErrorContext(
			ctx,
			"failed to read the addresses of the photos",
			slog.Any("error", err),
		)

		return nil
	}

	return found
}

// acceptPhotoFiles answers whether every file is a ready plant photo of the acting
// workspace. It writes the problem and answers false otherwise.
func acceptPhotoFiles(
	w http.ResponseWriter,
	r *http.Request,
	logger *slog.Logger,
	files pluginapi.PluginFiles,
	ids []string,
) bool {
	found, err := readFiles(r.Context(), files, ids)
	if err != nil {
		logger.ErrorContext(r.Context(), "failed to check the photos", slog.Any("error", err))
		problem.Write(w, r, problem.NewInternal())

		return false
	}

	waiting := 0

	for i, id := range ids {
		file, ok := found[id]
		field := fmt.Sprintf("file_ids/%d", i)

		switch {
		case !ok || file.Kind != model.PhotoKind:
			writeFieldProblem(w, r, field, detailNotAPhoto)

			return false
		case file.State == pluginapi.FileStateFailed:
			writeFieldProblem(w, r, field, detailPhotoFailed)

			return false
		case file.State != pluginapi.FileStateReady:
			waiting++
		}
	}

	if waiting > 0 {
		writePhotoNotReady(w, r, waiting)

		return false
	}

	return true
}

// claimAfterCommit claims each file once the rows that name it committed. A failed
// claim leaves the answer as it is, because the expiry of the hub meets the
// reference and claims the file.
func claimAfterCommit(
	ctx context.Context,
	logger *slog.Logger,
	files pluginapi.PluginFiles,
	ids []string,
) {
	for _, id := range ids {
		if err := files.Claim(ctx, id); err != nil {
			logger.WarnContext(ctx, "failed to claim a photo", slog.Any("error", err))
		}
	}
}

func fileIDsOf(photos []model.Photo) []string {
	ids := make([]string, len(photos))
	for i, photo := range photos {
		ids[i] = photo.FileID
	}

	return ids
}
