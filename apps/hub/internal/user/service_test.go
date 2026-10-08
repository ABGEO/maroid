package user_test

import (
	"io/fs"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/apps/hub/db"
	"github.com/abgeo/maroid/apps/hub/internal/domain/errs"
	"github.com/abgeo/maroid/apps/hub/internal/model"
	"github.com/abgeo/maroid/apps/hub/internal/user"
	"github.com/abgeo/maroid/libs/testdb"
)

// world holds Zura, the only administrator, and Ana, who is none.
type world struct {
	instance *testdb.Instance
	service  *user.Manager
	zura     string
	ana      string
}

func newWorld(t *testing.T) *world {
	t.Helper()

	instance := testdb.Start(t)

	migrations, err := fs.Sub(db.GetMigrationsFS(), "migrations")
	require.NoError(t, err)

	instance.Migrate(t, "public", migrations)

	insert := func(name string, administrator bool) string {
		var id string

		require.NoError(t, instance.DB.Get(&id,
			`INSERT INTO public.users (first_name, is_administrator) VALUES ($1, $2) RETURNING id;`,
			name, administrator))

		return id
	}

	return &world{
		instance: instance,
		service:  user.NewManager(instance.DB, nil, nil, 0),
		zura:     insert("Zura", true),
		ana:      insert("Ana", false),
	}
}

func (w *world) activeAdministrators(t *testing.T) int {
	t.Helper()

	var count int

	require.NoError(t, w.instance.DB.Get(&count,
		`SELECT count(*) FROM public.users WHERE is_administrator AND status = 'active';`))

	return count
}

func marked(value bool) user.Change {
	return user.Change{Administrator: &value}
}

// PLUGACC-SC-005: The last administrator keeps the mark and the active status, and a
// second administrator joins.
// PLUGACC-INV-003: The instance keeps one active administrator.
func TestTheLastAdministratorStays(t *testing.T) {
	t.Parallel()

	scene := newWorld(t)

	_, err := scene.service.Change(t.Context(), scene.zura, marked(false), nil)
	require.ErrorIs(t, err, errs.ErrAdministratorLast)

	blocked := model.StatusBlocked
	_, err = scene.service.Change(t.Context(), scene.zura,
		user.Change{Status: &blocked}, nil)
	require.ErrorIs(t, err, errs.ErrAdministratorLast)
	assert.Equal(t, 1, scene.activeAdministrators(t))

	ana, err := scene.service.Change(t.Context(), scene.ana, marked(true), nil)
	require.NoError(t, err)
	assert.True(t, ana.IsAdministrator)
	assert.Equal(t, 2, scene.activeAdministrators(t))
}

// PLUGACC-SC-005: Two administrators who take the mark of each other at the same
// moment leave one, because the lock orders them.
func TestTwoAdministratorsWhoUnmarkEachOtherKeepOne(t *testing.T) {
	t.Parallel()

	scene := newWorld(t)

	_, err := scene.service.Change(t.Context(), scene.ana, marked(true), nil)
	require.NoError(t, err)

	failures := make([]error, 2)

	var group sync.WaitGroup

	for index, target := range []string{scene.zura, scene.ana} {
		group.Go(func() {
			_, failures[index] = scene.service.Change(t.Context(), target, marked(false), nil)
		})
	}

	group.Wait()

	refused := 0

	for _, failure := range failures {
		if failure != nil {
			require.ErrorIs(t, failure, errs.ErrAdministratorLast)

			refused++
		}
	}

	assert.Equal(t, 1, refused)
	assert.Equal(t, 1, scene.activeAdministrators(t))
}
