package testdb

import (
	"slices"
	"testing"

	"github.com/jmoiron/sqlx"
)

// The delete actions of a foreign key, as pg_constraint.confdeltype holds them.
const (
	deleteCascade = "c"
	deleteSetNull = "n"
)

// RequireWorkspaceCascade fails the test unless each foreign key of the schema that
// references public.workspaces deletes with CASCADE. A table that setNull names must
// set NULL instead. A schema with no such key fails, so a wrong name never passes.
func RequireWorkspaceCascade(t *testing.T, db *sqlx.DB, schema string, setNull ...string) {
	t.Helper()

	var keys []struct {
		Table  string `db:"table_name"`
		Name   string `db:"conname"`
		Action string `db:"confdeltype"`
	}

	err := db.Select(&keys, `
		SELECT c.relname AS table_name, con.conname, con.confdeltype::text AS confdeltype
		FROM pg_constraint con
		JOIN pg_class c ON c.oid = con.conrelid
		JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE con.contype = 'f'
		  AND con.confrelid = 'public.workspaces'::regclass
		  AND n.nspname = $1
		ORDER BY c.relname, con.conname;`, schema)
	if err != nil {
		t.Fatalf("reading the foreign keys of the schema %q: %v", schema, err)
	}

	if len(keys) == 0 {
		t.Fatalf("the schema %q holds no foreign key to public.workspaces", schema)
	}

	for _, key := range keys {
		want := deleteCascade
		if slices.Contains(setNull, key.Table) {
			want = deleteSetNull
		}

		if key.Action != want {
			t.Errorf("%s.%s: the key %s deletes with %q, want %q",
				schema, key.Table, key.Name, key.Action, want)
		}
	}
}
