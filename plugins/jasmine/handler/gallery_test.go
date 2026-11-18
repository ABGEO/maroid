package handler_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/libs/rest/precondition"
	"github.com/abgeo/maroid/libs/rest/problem"
	"github.com/abgeo/maroid/plugins/jasmine/permission"
)

const (
	typePhotoNotReady = "/problems/dev.maroid.jasmine/photo-not-ready"

	memberFileIDs = "file_ids"
	memberTakenOn = "taken_on"

	pointerFileIDs = "#/file_ids"
	pointerPhotoID = "#/photo_id"
	pointerTakenOn = "#/taken_on"

	stateProcessing = "processing"

	detailFileTaken = "The file is already a photo of a moment."
)

type readAddress struct {
	URL       string `json:"url"`
	ExpiresAt string `json:"expires_at"`
}

type photoAnswer struct {
	ID        string       `json:"id"`
	Width     int          `json:"width"`
	Height    int          `json:"height"`
	Thumbnail *readAddress `json:"thumbnail"`
	Display   *readAddress `json:"display"`
}

type momentAnswer struct {
	ID      string        `json:"id"`
	PlantID string        `json:"plant_id"`
	TakenOn string        `json:"taken_on"`
	Note    *string       `json:"note"`
	Photos  []photoAnswer `json:"photos"`
}

type profile struct {
	ID        string       `json:"id"`
	Thumbnail *readAddress `json:"thumbnail"`
}

type momentPage struct {
	Items []momentAnswer `json:"items"`
	Next  string         `json:"next"`
}

// day answers the UTC date of the run, moved by the count of days.
func day(offset int) string {
	return time.Now().UTC().AddDate(0, 0, offset).Format(time.DateOnly)
}

func homeContext() context.Context {
	return pluginapi.ContextWithActingWorkspace(context.Background(), workspaceH)
}

func momentsOf(plantID string) string {
	return "/plants/" + plantID + "/moments"
}

func postMoment(
	t *testing.T,
	router http.Handler,
	plantID string,
	body map[string]any,
) *httptest.ResponseRecorder {
	t.Helper()

	return send(t, router, http.MethodPost, momentsOf(plantID), encode(t, body), "")
}

// makeMoment adds a moment of the day with one photo of each file, and answers it.
func makeMoment(
	t *testing.T,
	router http.Handler,
	plantID string,
	takenOn string,
	files ...string,
) momentAnswer {
	t.Helper()

	recorder := postMoment(
		t,
		router,
		plantID,
		map[string]any{memberTakenOn: takenOn, memberFileIDs: files},
	)
	require.Equal(t, http.StatusCreated, recorder.Code, recorder.Body.String())

	return decode[momentAnswer](t, recorder)
}

func readMoment(t *testing.T, router http.Handler, plantID, id string) *httptest.ResponseRecorder {
	t.Helper()

	return send(t, router, http.MethodGet, momentsOf(plantID)+"/"+id, "", "")
}

func profileOf(t *testing.T, router http.Handler, plantID string) *profile {
	t.Helper()

	recorder := send(t, router, http.MethodGet, "/plants/"+plantID, "", "")
	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())

	return decode[plantAnswer](t, recorder).ProfilePhoto
}

func putProfile(
	t *testing.T,
	router http.Handler,
	plantID, photoID string,
) *httptest.ResponseRecorder {
	t.Helper()

	return send(t, router, http.MethodPut, "/plants/"+plantID+"/profile-photo",
		encode(t, map[string]any{"photo_id": photoID}), "")
}

func deletePhoto(t *testing.T, router http.Handler, moment momentAnswer, photoID string) {
	t.Helper()

	recorder := send(t, router, http.MethodDelete,
		momentsOf(moment.PlantID)+"/"+moment.ID+"/photos/"+photoID, "", "")
	require.Equal(t, http.StatusNoContent, recorder.Code, recorder.Body.String())
}

func idsOf(photos []photoAnswer) []string {
	ids := make([]string, len(photos))
	for i, photo := range photos {
		ids[i] = photo.ID
	}

	return ids
}

// countIn answers the rows of the query over the files, in H.
func countIn(t *testing.T, fixture *garden, query string, files []string) int {
	t.Helper()

	var count int

	require.NoError(t, fixture.inHome(t, func(tx *sqlx.Tx) error {
		bound, args, err := sqlx.In(query, files)
		if err != nil {
			return err //nolint:wrapcheck // a test helper answers the error of sqlx.
		}

		return tx.Get(&count, tx.Rebind(bound), args...)
	}))

	return count
}

func filesLeft(t *testing.T, fixture *garden, files ...string) int {
	t.Helper()

	return countIn(t, fixture, `SELECT count(*) FROM public.workspace_files WHERE id IN (?)`, files)
}

func purgesOf(t *testing.T, fixture *garden, files ...string) int {
	t.Helper()

	return countIn(
		t,
		fixture,
		`SELECT count(*) FROM public.file_purges WHERE file_id IN (?)`,
		files,
	)
}

func claimed(t *testing.T, fixture *garden, files ...string) int {
	t.Helper()

	return countIn(t, fixture,
		`SELECT count(*) FROM public.workspace_files WHERE claimed AND id IN (?)`, files)
}

func detailsOf(t *testing.T, recorder *httptest.ResponseRecorder) []string {
	t.Helper()

	answer := decode[problem.ValidationProblem](t, recorder)

	details := make([]string, 0, len(answer.Errors))
	for _, failure := range answer.Errors {
		details = append(details, failure.Detail)
	}

	return details
}

// GARDEN-SC-050: A moment holds its note and its photos in the order of the files.
// The files are claimed, and the first photo becomes the profile picture.
func TestAMomentHoldsItsPhotosInOrder(t *testing.T) {
	t.Parallel()

	fixture := gardenUnderTest(t)
	router := fixture.router(workspaceH)
	plant := makePlant(t, router, nameFern, makeEnvironment(t, router, "Office").ID)
	first, second := insertFile(t, fixture, workspaceH), insertFile(t, fixture, workspaceH)

	recorder := postMoment(t, router, plant.ID, map[string]any{
		memberTakenOn: day(0),
		memberNote:    "Repotted into a 14 cm pot",
		memberFileIDs: []string{first, second},
	})
	require.Equal(t, http.StatusCreated, recorder.Code, recorder.Body.String())

	created := decode[momentAnswer](t, recorder)
	assert.Equal(
		t,
		externalURL+momentsOf(plant.ID)+"/"+created.ID,
		recorder.Header().Get("Location"),
	)
	assert.Equal(t, "Repotted into a 14 cm pot", *created.Note)
	require.Len(t, created.Photos, 2)
	assert.Equal(t, filesURL+first+"/thumbnail", created.Photos[0].Thumbnail.URL)
	assert.Equal(t, filesURL+second+"/display", created.Photos[1].Display.URL)
	assert.Equal(t, photoWidth, created.Photos[0].Width)
	assert.Equal(t, 2, claimed(t, fixture, first, second))
	assert.Equal(t, created.Photos[0].ID, profileOf(t, router, plant.ID).ID)
}

// GARDEN-SC-051: A moment with no note holds no member note.
func TestAMomentWithNoNoteHoldsNoNote(t *testing.T) {
	t.Parallel()

	fixture := gardenUnderTest(t)
	router := fixture.router(workspaceH)
	plant := makePlant(t, router, nameFern, makeEnvironment(t, router, "Office").ID)

	recorder := postMoment(t, router, plant.ID, map[string]any{
		memberTakenOn: day(-1), memberFileIDs: []string{insertFile(t, fixture, workspaceH)},
	})
	require.Equal(t, http.StatusCreated, recorder.Code, recorder.Body.String())

	var members map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &members))
	assert.NotContains(t, members, "note")
}

// GARDEN-SC-052: A day after tomorrow, no file, and a note of 1001 characters are
// refused.
func TestAMomentRefusesAnInvalidPayload(t *testing.T) {
	t.Parallel()

	fixture := gardenUnderTest(t)
	router := fixture.router(workspaceH)
	plant := makePlant(t, router, nameFern, makeEnvironment(t, router, "Office").ID)
	file := insertFile(t, fixture, workspaceH)

	cases := []struct {
		body    map[string]any
		pointer string
	}{
		{map[string]any{memberTakenOn: day(2), memberFileIDs: []string{file}}, pointerTakenOn},
		{map[string]any{memberTakenOn: day(0), memberFileIDs: []string{}}, pointerFileIDs},
		{
			map[string]any{
				memberTakenOn: day(0),
				memberNote:    strings.Repeat("a", 1001),
				memberFileIDs: []string{file},
			},
			"#/note",
		},
	}

	for _, refused := range cases {
		recorder := postMoment(t, router, plant.ID, refused.body)
		require.Equal(t, http.StatusUnprocessableEntity, recorder.Code, recorder.Body.String())
		assert.Equal(t, []string{refused.pointer}, pointersOf(t, recorder))
	}

	assert.Zero(t, countRows(t, fixture, "moments", plant.ID))
}

// GARDEN-SC-053: A file that is not ready answers 409. A failed file, a file of
// another plugin, and a photo of another workspace answer 422. Nothing lands.
func TestAMomentAcceptsOnlyReadyPhotosOfTheWorkspace(t *testing.T) {
	t.Parallel()

	fixture := gardenUnderTest(t)
	router := fixture.router(workspaceH)
	plant := makePlant(t, router, nameFern, makeEnvironment(t, router, "Office").ID)
	ready := insertFile(t, fixture, workspaceH)
	processing := insertFileOf(t, fixture, workspaceH, pluginID, "plant-photo", stateProcessing)
	failed := insertFileOf(t, fixture, workspaceH, pluginID, "plant-photo", "failed")
	foreignKind := insertFileOf(t, fixture, workspaceH, "dev.maroid.other", "receipt", "ready")
	foreignWorkspace := insertFile(t, fixture, workspaceG)

	waiting := postMoment(t, router, plant.ID, map[string]any{
		memberTakenOn: day(0), memberFileIDs: []string{ready, processing},
	})
	require.Equal(t, http.StatusConflict, waiting.Code, waiting.Body.String())
	assert.Equal(t, typePhotoNotReady, problemTypeOf(t, waiting))
	assert.Equal(t, "1 photo is not ready yet.", decode[problem.Problem](t, waiting).Detail)

	for _, refused := range []string{failed, foreignKind, foreignWorkspace} {
		recorder := postMoment(t, router, plant.ID, map[string]any{
			memberTakenOn: day(0), memberFileIDs: []string{ready, refused},
		})
		require.Equal(t, http.StatusUnprocessableEntity, recorder.Code, recorder.Body.String())
		assert.Equal(t, []string{pointerFileIDs + "/1"}, pointersOf(t, recorder))
	}

	assert.Zero(t, countRows(t, fixture, "moments", plant.ID))
	assert.Zero(t, claimed(t, fixture, ready))
}

// GARDEN-SC-055: A file that is a photo of a moment, and a file named twice, are
// refused.
func TestAFileIsAPhotoOfOneMoment(t *testing.T) {
	t.Parallel()

	fixture := gardenUnderTest(t)
	router := fixture.router(workspaceH)
	environment := makeEnvironment(t, router, "Office")
	first := makePlant(t, router, nameFern, environment.ID)
	second := makePlant(t, router, "Monstera", environment.ID)
	taken := insertFile(t, fixture, workspaceH)
	makeMoment(t, router, first.ID, day(0), taken)
	loose := insertFile(t, fixture, workspaceH)

	again := postMoment(
		t,
		router,
		second.ID,
		map[string]any{memberTakenOn: day(0), memberFileIDs: []string{taken}},
	)
	require.Equal(t, http.StatusUnprocessableEntity, again.Code, again.Body.String())
	assert.Equal(t, []string{detailFileTaken}, detailsOf(t, again))

	twice := postMoment(t, router, second.ID, map[string]any{
		memberTakenOn: day(0), memberFileIDs: []string{loose, loose},
	})
	require.Equal(t, http.StatusUnprocessableEntity, twice.Code, twice.Body.String())
	assert.Equal(t, []string{pointerFileIDs}, pointersOf(t, twice))

	assert.Zero(t, countRows(t, fixture, "moments", second.ID))
}

// GARDEN-SC-056: A put replaces the day and the note, and the photos stay.
func TestAPutReplacesTheDayAndTheNote(t *testing.T) {
	t.Parallel()

	fixture := gardenUnderTest(t)
	router := fixture.router(workspaceH)
	plant := makePlant(t, router, nameFern, makeEnvironment(t, router, "Office").ID)

	created := postMoment(t, router, plant.ID, map[string]any{
		memberTakenOn: "2026-06-02",
		memberNote:    "New leaf",
		memberFileIDs: []string{insertFile(t, fixture, workspaceH)},
	})
	require.Equal(t, http.StatusCreated, created.Code, created.Body.String())

	moment := decode[momentAnswer](t, created)
	target := momentsOf(plant.ID) + "/" + moment.ID

	replaced := send(
		t,
		router,
		http.MethodPut,
		target,
		encode(t, map[string]any{memberTakenOn: "2026-06-01"}),
		"",
	)
	require.Equal(t, http.StatusOK, replaced.Code, replaced.Body.String())

	answered := decode[momentAnswer](t, replaced)
	assert.Equal(t, "2026-06-01", answered.TakenOn)
	assert.Nil(t, answered.Note)
	assert.Equal(t, idsOf(moment.Photos), idsOf(answered.Photos))
	assert.NotEqual(
		t,
		created.Header().
			Get(precondition.ETagHeader),
		replaced.Header().Get(precondition.ETagHeader),
	)

	future := send(
		t,
		router,
		http.MethodPut,
		target,
		encode(t, map[string]any{memberTakenOn: day(2)}),
		"",
	)
	require.Equal(t, http.StatusUnprocessableEntity, future.Code, future.Body.String())
	assert.Equal(t, []string{pointerTakenOn}, pointersOf(t, future))
}

// GARDEN-SC-057: Added photos follow the photos of the moment. A file that is not
// ready adds nothing.
func TestPhotosJoinAMomentInOrder(t *testing.T) {
	t.Parallel()

	fixture := gardenUnderTest(t)
	router := fixture.router(workspaceH)
	plant := makePlant(t, router, nameFern, makeEnvironment(t, router, "Office").ID)
	moment := makeMoment(t, router, plant.ID, day(0), insertFile(t, fixture, workspaceH))
	second, third := insertFile(t, fixture, workspaceH), insertFile(t, fixture, workspaceH)
	processing := insertFileOf(t, fixture, workspaceH, pluginID, "plant-photo", stateProcessing)
	target := momentsOf(plant.ID) + "/" + moment.ID + "/photos"

	added := send(t, router, http.MethodPost, target,
		encode(t, map[string]any{memberFileIDs: []string{second, third}}), "")
	require.Equal(t, http.StatusCreated, added.Code, added.Body.String())

	photos := decode[momentAnswer](t, added).Photos
	require.Len(t, photos, 3)
	assert.Equal(t, moment.Photos[0].ID, photos[0].ID)
	assert.Equal(t, filesURL+second+"/display", photos[1].Display.URL)
	assert.Equal(t, filesURL+third+"/display", photos[2].Display.URL)

	waiting := send(t, router, http.MethodPost, target,
		encode(t, map[string]any{memberFileIDs: []string{processing}}), "")
	require.Equal(t, http.StatusConflict, waiting.Code, waiting.Body.String())
	assert.Len(t, decode[momentAnswer](t, readMoment(t, router, plant.ID, moment.ID)).Photos, 3)
}

// GARDEN-SC-058: The delete of the last photo of a moment deletes the moment, so no
// moment holds no photo.
func TestTheLastPhotoTakesItsMoment(t *testing.T) {
	t.Parallel()

	fixture := gardenUnderTest(t)
	router := fixture.router(workspaceH)
	plant := makePlant(t, router, nameFern, makeEnvironment(t, router, "Office").ID)
	full := makeMoment(
		t,
		router,
		plant.ID,
		day(0),
		insertFile(
			t,
			fixture,
			workspaceH,
		),
		insertFile(t, fixture, workspaceH),
		insertFile(t, fixture, workspaceH),
	)
	single := makeMoment(t, router, plant.ID, day(0), insertFile(t, fixture, workspaceH))

	deletePhoto(t, router, full, full.Photos[1].ID)
	deletePhoto(t, router, single, single.Photos[0].ID)

	assert.Len(t, decode[momentAnswer](t, readMoment(t, router, plant.ID, full.ID)).Photos, 2)
	assert.Equal(t, http.StatusNotFound, readMoment(t, router, plant.ID, single.ID).Code)

	var empty int

	require.NoError(t, fixture.inHome(t, func(tx *sqlx.Tx) error {
		return tx.Get(&empty, `SELECT count(*) FROM moments m
			WHERE NOT EXISTS (SELECT 1 FROM photos p WHERE p.moment_id = m.id)`)
	}))
	assert.Zero(t, empty)
}

// GARDEN-SC-059: The delete of a moment deletes its photos and their files, and
// purges the bytes.
func TestAMomentTakesItsFiles(t *testing.T) {
	t.Parallel()

	fixture := gardenUnderTest(t)
	router := fixture.router(workspaceH)
	plant := makePlant(t, router, nameFern, makeEnvironment(t, router, "Office").ID)
	first, second := insertFile(t, fixture, workspaceH), insertFile(t, fixture, workspaceH)
	moment := makeMoment(t, router, plant.ID, day(0), first, second)

	deleted := send(t, router, http.MethodDelete, momentsOf(plant.ID)+"/"+moment.ID, "", "")
	require.Equal(t, http.StatusNoContent, deleted.Code, deleted.Body.String())

	assert.Zero(
		t,
		countIn(
			t,
			fixture,
			`SELECT count(*) FROM photos WHERE file_id IN (?)`,
			[]string{first, second},
		),
	)

	files, err := fixture.files.Get(homeContext(), first, second)
	require.NoError(t, err)
	assert.Empty(t, files)
	assert.Equal(t, 2, purgesOf(t, fixture, first, second))
}

// GARDEN-SC-060: The delete of a plant takes its moments, its photos, and their
// files. A stale If-Match deletes nothing.
func TestAPlantTakesItsGallery(t *testing.T) {
	t.Parallel()

	fixture := gardenUnderTest(t)
	router := fixture.router(workspaceH)
	plant := makePlant(t, router, nameFern, makeEnvironment(t, router, "Office").ID)
	older := send(
		t,
		router,
		http.MethodGet,
		"/plants/"+plant.ID,
		"",
		"",
	).Header().
		Get(precondition.ETagHeader)
	files := []string{
		insertFile(t, fixture, workspaceH),
		insertFile(t, fixture, workspaceH),
		insertFile(t, fixture, workspaceH),
	}
	makeMoment(t, router, plant.ID, day(0), files[0], files[1])
	makeMoment(t, router, plant.ID, day(-1), files[2])
	require.NotNil(t, profileOf(t, router, plant.ID))

	stale := send(t, router, http.MethodDelete, "/plants/"+plant.ID, "", older)
	require.Equal(t, http.StatusPreconditionFailed, stale.Code, stale.Body.String())
	assert.Equal(t, 2, countRows(t, fixture, "moments", plant.ID))
	assert.Equal(t, 3, countRows(t, fixture, "photos", plant.ID))
	assert.Equal(t, 3, filesLeft(t, fixture, files...))
	assert.Zero(t, purgesOf(t, fixture, files...))

	deleted := send(t, router, http.MethodDelete, "/plants/"+plant.ID, "", "")
	require.Equal(t, http.StatusNoContent, deleted.Code, deleted.Body.String())
	assert.Zero(t, countRows(t, fixture, "moments", plant.ID))
	assert.Zero(t, countRows(t, fixture, "photos", plant.ID))
	assert.Zero(t, filesLeft(t, fixture, files...))
	assert.Equal(t, 3, purgesOf(t, fixture, files...))
}

// GARDEN-SC-061: An inactive plant takes moments, photos, and a profile picture.
func TestAnInactivePlantKeepsItsGallery(t *testing.T) {
	t.Parallel()

	fixture := gardenUnderTest(t)
	router := fixture.router(workspaceH)
	plant := makePlant(t, router, nameFern, makeEnvironment(t, router, "Office").ID)
	inactive := replacePlant(t, router, plant, map[string]any{memberStatus: statusInactive})
	require.Equal(t, http.StatusOK, inactive.Code, inactive.Body.String())

	moment := makeMoment(t, router, plant.ID, day(0), insertFile(t, fixture, workspaceH))

	added := send(t, router, http.MethodPost, momentsOf(plant.ID)+"/"+moment.ID+"/photos",
		encode(t, map[string]any{memberFileIDs: []string{insertFile(t, fixture, workspaceH)}}), "")
	require.Equal(t, http.StatusCreated, added.Code, added.Body.String())

	photos := decode[momentAnswer](t, added).Photos
	chosen := putProfile(t, router, plant.ID, photos[1].ID)
	require.Equal(t, http.StatusOK, chosen.Code, chosen.Body.String())
}

// GARDEN-SC-062: The moments read the latest day first, and the latest moment of a
// day first.
func TestTheGalleryReadsTheLatestDayFirst(t *testing.T) {
	t.Parallel()

	fixture := gardenUnderTest(t)
	router := fixture.router(workspaceH)
	plant := makePlant(t, router, nameFern, makeEnvironment(t, router, "Office").ID)
	earliest := makeMoment(t, router, plant.ID, "2026-03-12", insertFile(t, fixture, workspaceH))
	first := makeMoment(t, router, plant.ID, "2026-06-01", insertFile(t, fixture, workspaceH))
	latest := makeMoment(t, router, plant.ID, "2026-06-02", insertFile(t, fixture, workspaceH))
	second := makeMoment(t, router, plant.ID, "2026-06-01", insertFile(t, fixture, workspaceH))

	head := send(t, router, http.MethodGet, momentsOf(plant.ID)+"?limit=2", "", "")
	require.Equal(t, http.StatusOK, head.Code, head.Body.String())

	headPage := decode[momentPage](t, head)
	require.Len(t, headPage.Items, 2)
	assert.Equal(
		t,
		[]string{latest.ID, second.ID},
		[]string{headPage.Items[0].ID, headPage.Items[1].ID},
	)
	require.NotEmpty(t, headPage.Next)

	tail := send(t, router, http.MethodGet, strings.TrimPrefix(headPage.Next, externalURL), "", "")
	require.Equal(t, http.StatusOK, tail.Code, tail.Body.String())

	tailPage := decode[momentPage](t, tail)
	require.Len(t, tailPage.Items, 2)
	assert.Equal(
		t,
		[]string{first.ID, earliest.ID},
		[]string{tailPage.Items[0].ID, tailPage.Items[1].ID},
	)
	assert.Empty(t, tailPage.Next)
}

// GARDEN-SC-063: A viewer reads the gallery, and every write answers 403.
func TestAViewerReadsTheGallery(t *testing.T) {
	t.Parallel()

	fixture := gardenUnderTest(t)
	router := fixture.router(workspaceH)
	plant := makePlant(t, router, nameFern, makeEnvironment(t, router, "Office").ID)
	moment := makeMoment(t, router, plant.ID, day(0), insertFile(t, fixture, workspaceH))
	loose := insertFile(t, fixture, workspaceH)
	viewer := fixture.routerAs(workspaceH, []string{permission.PlantsRead})
	target := momentsOf(plant.ID) + "/" + moment.ID

	assert.Equal(
		t,
		http.StatusOK,
		send(t, viewer, http.MethodGet, momentsOf(plant.ID), "", "").Code,
	)
	assert.Equal(t, http.StatusOK, readMoment(t, viewer, plant.ID, moment.ID).Code)

	writes := []struct{ method, target, body string }{
		{
			http.MethodPost,
			momentsOf(plant.ID),
			encode(t, map[string]any{memberTakenOn: day(0), memberFileIDs: []string{loose}}),
		},
		{http.MethodPut, target, encode(t, map[string]any{memberTakenOn: day(-1)})},
		{
			http.MethodPost,
			target + "/photos",
			encode(t, map[string]any{memberFileIDs: []string{loose}}),
		},
		{http.MethodDelete, target + "/photos/" + moment.Photos[0].ID, ""},
		{http.MethodDelete, target, ""},
		{
			http.MethodPut, "/plants/" + plant.ID + "/profile-photo",
			encode(t, map[string]any{"photo_id": moment.Photos[0].ID}),
		},
	}

	for _, write := range writes {
		recorder := send(t, viewer, write.method, write.target, write.body, "")
		assert.Equal(t, http.StatusForbidden, recorder.Code, write.method+" "+write.target)
	}

	assert.Equal(t, 1, countRows(t, fixture, "moments", plant.ID))
	assert.Equal(t, moment, decode[momentAnswer](t, readMoment(t, router, plant.ID, moment.ID)))
	assert.Zero(t, claimed(t, fixture, loose))
}

// GARDEN-SC-064: Any photo of the plant becomes its profile picture and stays in its
// moment. A photo of another plant and an unknown photo are refused.
func TestAPhotoOfThePlantBecomesItsProfilePicture(t *testing.T) {
	t.Parallel()

	fixture := gardenUnderTest(t)
	router := fixture.router(workspaceH)
	environment := makeEnvironment(t, router, "Office")
	first := makePlant(t, router, nameFern, environment.ID)
	second := makePlant(t, router, "Monstera", environment.ID)
	moment := makeMoment(
		t,
		router,
		first.ID,
		day(0),
		insertFile(t, fixture, workspaceH),
		insertFile(t, fixture, workspaceH),
	)
	foreign := makeMoment(t, router, second.ID, day(0), insertFile(t, fixture, workspaceH))
	before := send(
		t,
		router,
		http.MethodGet,
		"/plants/"+first.ID,
		"",
		"",
	).Header().
		Get(precondition.ETagHeader)

	chosen := putProfile(t, router, first.ID, moment.Photos[1].ID)
	require.Equal(t, http.StatusOK, chosen.Code, chosen.Body.String())
	assert.Equal(t, moment.Photos[1].ID, decode[plantAnswer](t, chosen).ProfilePhoto.ID)
	assert.NotEqual(t, before, chosen.Header().Get(precondition.ETagHeader))
	assert.Equal(
		t,
		idsOf(moment.Photos),
		idsOf(decode[momentAnswer](t, readMoment(t, router, first.ID, moment.ID)).Photos),
	)

	for _, refused := range []string{foreign.Photos[0].ID, "01927f4e-3c2a-7b1d-9e8f-0a1b2c3d4e99"} {
		recorder := putProfile(t, router, first.ID, refused)
		require.Equal(t, http.StatusUnprocessableEntity, recorder.Code, recorder.Body.String())
		assert.Equal(t, []string{pointerPhotoID}, pointersOf(t, recorder))
	}
}

// GARDEN-SC-065: The first photo of a plant with no profile picture becomes it, and
// a later photo leaves it.
func TestTheFirstPhotoFillsTheProfilePicture(t *testing.T) {
	t.Parallel()

	fixture := gardenUnderTest(t)
	router := fixture.router(workspaceH)
	plant := makePlant(t, router, nameFern, makeEnvironment(t, router, "Office").ID)

	first := makeMoment(
		t,
		router,
		plant.ID,
		day(0),
		insertFile(t, fixture, workspaceH),
		insertFile(t, fixture, workspaceH),
	)
	assert.Equal(t, first.Photos[0].ID, profileOf(t, router, plant.ID).ID)

	makeMoment(t, router, plant.ID, day(0), insertFile(t, fixture, workspaceH))
	assert.Equal(t, first.Photos[0].ID, profileOf(t, router, plant.ID).ID)

	deletePhoto(t, router, first, first.Photos[0].ID)
	assert.Nil(t, profileOf(t, router, plant.ID))

	last := makeMoment(t, router, plant.ID, day(0), insertFile(t, fixture, workspaceH))
	assert.Equal(t, last.Photos[0].ID, profileOf(t, router, plant.ID).ID)
}

// GARDEN-SC-066: The delete of the profile picture leaves the plant with none, and
// the other photos stay.
func TestTheDeleteOfTheProfilePictureLeavesNone(t *testing.T) {
	t.Parallel()

	fixture := gardenUnderTest(t)
	router := fixture.router(workspaceH)
	plant := makePlant(t, router, nameFern, makeEnvironment(t, router, "Office").ID)
	moment := makeMoment(
		t,
		router,
		plant.ID,
		day(0),
		insertFile(t, fixture, workspaceH),
		insertFile(t, fixture, workspaceH),
	)
	require.Equal(t, moment.Photos[0].ID, profileOf(t, router, plant.ID).ID)

	deletePhoto(t, router, moment, moment.Photos[0].ID)

	assert.Nil(t, profileOf(t, router, plant.ID))
	assert.Equal(t, []string{moment.Photos[1].ID},
		idsOf(decode[momentAnswer](t, readMoment(t, router, plant.ID, moment.ID)).Photos))
}

// GARDEN-SC-067: A plant with a profile picture answers its thumbnail, in the list and
// alone. A plant with none answers no profile picture.
func TestAPlantAnswersItsProfilePicture(t *testing.T) {
	t.Parallel()

	fixture := gardenUnderTest(t)
	router := fixture.router(workspaceH)
	environment := makeEnvironment(t, router, "Office")
	pictured := makePlant(t, router, nameFern, environment.ID)
	bare := makePlant(t, router, "Monstera", environment.ID)
	file := insertFile(t, fixture, workspaceH)
	moment := makeMoment(t, router, pictured.ID, day(0), file)

	listed := send(t, router, http.MethodGet, "/plants", "", "")
	require.Equal(t, http.StatusOK, listed.Code, listed.Body.String())

	plants := map[string]*profile{}
	for _, plant := range decode[struct {
		Items []plantAnswer `json:"items"`
	}](t, listed).Items {
		plants[plant.ID] = plant.ProfilePhoto
	}

	for _, read := range []*profile{plants[pictured.ID], profileOf(t, router, pictured.ID)} {
		require.NotNil(t, read)
		assert.Equal(t, moment.Photos[0].ID, read.ID)
		assert.Equal(t, filesURL+file+"/thumbnail", read.Thumbnail.URL)
	}

	assert.Nil(t, plants[bare.ID])
	assert.Nil(t, profileOf(t, router, bare.ID))
}
