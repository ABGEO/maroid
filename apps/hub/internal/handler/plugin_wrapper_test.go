package handler_test

import (
	"encoding/json"
	"net/http"
	"sync/atomic"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	hubdatabase "github.com/abgeo/maroid/apps/hub/internal/database"
	"github.com/abgeo/maroid/libs/rest/problem"
)

const probePluginID = "dev.maroid.probe"

// listNotes answers the bodies of the notes that the acting workspace holds, as a
// plugin reads its own scoped table.
func listNotes(database *sqlx.DB, runs *atomic.Int32) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		runs.Add(1)

		ctx := r.Context()
		bodies := []string{}

		err := hubdatabase.WithScopeTx(ctx, database, func(tx *sqlx.Tx) error {
			return tx.SelectContext(ctx, &bodies,
				`SELECT body FROM public.workspace_test_notes ORDER BY body;`)
		})
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)

			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(bodies)
	}
}

func notesPath(workspaceID string) string {
	return "/workspaces/" + workspaceID + "/plugins/" + probePluginID + "/api/notes"
}

// WSPACE-SC-009: One route of a plugin answers the records of the workspace that its
// address names, and no record of another workspace of the same person.
func TestARouteOfAPluginReadsTheWorkspaceOfItsAddress(t *testing.T) {
	t.Parallel()

	fixture := workspaceUnderTest(t)
	garden := fixture.create(t, fixture.ana, "G")
	fixture.writeNotes(t, fixture.h, "first of H", "second of H")
	fixture.writeNotes(t, garden, "one of G")

	read := func(workspaceID string) []string {
		response := fixture.call(t, fixture.ana, http.MethodGet, notesPath(workspaceID), nil, "")
		require.Equal(t, http.StatusOK, response.Code, response.Body.String())

		var bodies []string

		require.NoError(t, json.Unmarshal(response.Body.Bytes(), &bodies))

		return bodies
	}

	assert.Equal(t, []string{"first of H", "second of H"}, read(fixture.h))
	assert.Equal(t, []string{"one of G"}, read(garden))
}

// IDENT-SC-013: A person who is not a member of the workspace of the address reads
// 404, and the route of the plugin does not run.
func TestARouteOfAPluginRefusesANonMember(t *testing.T) {
	t.Parallel()

	fixture := workspaceUnderTest(t)

	response := fixture.call(t, fixture.nino, http.MethodGet, notesPath(fixture.h), nil, "")

	require.Equal(t, http.StatusNotFound, response.Code, response.Body.String())

	var answered problem.Problem

	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &answered))
	assert.Equal(t, problem.TypeNotFound, answered.Type)
	assert.Zero(t, fixture.probeRuns.Load())
}
