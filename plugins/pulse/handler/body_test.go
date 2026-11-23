package handler_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/abgeo/maroid/plugins/pulse/pulsetest"
)

// PULSE-SC-041: A body of 64 KiB and one byte answers 413 content-too-large on each
// write, and no row changes.
func TestABodyOverTheLimitAnswers413(t *testing.T) {
	t.Parallel()

	app := newAPI(t)
	app.TurnOn(t, nina)

	counts := func() []int {
		out := make([]int, 0, len(pulsetest.Tables()))
		for _, table := range pulsetest.Tables() {
			out = append(out, pulsetest.Count(t, app.DB, nina, "SELECT count(*) FROM "+table))
		}

		return out
	}
	before := counts()

	const opening = `{"prompt":"`

	body := opening + strings.Repeat("a", 64<<10+1-len(opening)-len(`"}`)) + `"}`

	for _, write := range []struct{ method, target string }{
		{http.MethodPut, "/settings"},
		{http.MethodPost, "/questions"},
		{http.MethodPost, "/checkins"},
	} {
		recorder := app.as(t, write.method, write.target, body)

		assert.Equal(t, http.StatusRequestEntityTooLarge, recorder.Code, write.target)
		assert.Equal(
			t,
			"/problems/http/content-too-large",
			decode[map[string]any](t, recorder)["type"],
			write.target,
		)
	}

	assert.Len(t, body, 64<<10+1)
	assert.Equal(t, before, counts())
}
