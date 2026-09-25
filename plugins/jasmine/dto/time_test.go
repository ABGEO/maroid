package dto_test

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/plugins/jasmine/dto"
	"github.com/abgeo/maroid/plugins/jasmine/model"
)

// tbilisi is a zone four hours ahead of UTC. A value that the database answers
// carries the zone of the session, which no code of Maroid sets.
func tbilisi() *time.Location {
	return time.FixedZone("Asia/Tbilisi", 4*60*60)
}

// moment is one instant, held in a zone that is not UTC.
func moment() time.Time {
	return time.Date(2026, time.September, 25, 14, 30, 0, 0, tbilisi())
}

// APIFMT-SC-009: A point in time answers UTC, whatever zone the value carries
// when it reaches the DTO. Z-169 asks for the upper case Z and recommends the
// form with no offset.
func TestAPlantAnswersEveryMomentInUTC(t *testing.T) {
	t.Parallel()

	answer := dto.NewPlantResponse(&model.Plant{
		ID:            "01a0cae5-eb36-777a-824e-6e7e28d7a6b1",
		Name:          "Jasmine",
		EnvironmentID: "01a0cae5-eb36-777a-824e-71285629ec17",
		CreatedAt:     moment(),
		UpdatedAt:     moment(),
	})

	for name, value := range map[string]string{
		"created_at": answer.CreatedAt,
		"updated_at": answer.UpdatedAt,
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			assert.True(t, strings.HasSuffix(value, "Z"),
				"%s answers %q, which carries an offset", name, value)

			parsed, err := time.Parse(time.RFC3339, value)
			require.NoError(t, err)
			assert.True(t, moment().Equal(parsed), "the value names the same instant")
		})
	}
}

// APIFMT-SC-009: The environment answers the same way, so no route of the plugin
// carries a zone of its own.
func TestAnEnvironmentAnswersEveryMomentInUTC(t *testing.T) {
	t.Parallel()

	answer := dto.NewEnvironmentResponse(&model.Environment{
		ID:        "01a0cae5-eb36-777a-824e-6e7e28d7a6b1",
		Name:      "Balcony",
		CreatedAt: moment(),
		UpdatedAt: moment(),
	})

	assert.True(t, strings.HasSuffix(answer.CreatedAt, "Z"), answer.CreatedAt)
	assert.True(t, strings.HasSuffix(answer.UpdatedAt, "Z"), answer.UpdatedAt)

	parsed, err := time.Parse(time.RFC3339, answer.CreatedAt)
	require.NoError(t, err)
	assert.True(t, moment().Equal(parsed))
}
