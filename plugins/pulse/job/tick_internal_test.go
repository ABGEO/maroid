package job

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/plugins/pulse/model"
)

func tbilisi(t *testing.T) *time.Location {
	t.Helper()

	zone, err := time.LoadLocation("Asia/Tbilisi")
	require.NoError(t, err)

	return zone
}

func at(t *testing.T, hour int, minute int) time.Time {
	t.Helper()

	return time.Date(2026, time.November, 24, hour, minute, 0, 0, tbilisi(t))
}

// PULSE-SC-016: The tick at 08:59:59 fires nothing, and the tick at 09:01:59 fires
// the occurrence of 09:00.
func TestTheOccurrenceFiresWithinTwoMinutes(t *testing.T) {
	t.Parallel()

	morning := model.Checkin{
		ID:         "morning",
		Name:       "Morning",
		LocalTimes: model.Clocks{{Hour: 9}},
		FiresAfter: at(t, 6, 0),
	}
	checkins := []model.Checkin{morning}

	early := at(t, 8, 59).Add(59 * time.Second)
	assert.Nil(t, decide(checkins, nil, tbilisi(t), early))

	late := at(t, 9, 1).Add(59 * time.Second)
	plan := decide(checkins, nil, tbilisi(t), late)
	require.NotNil(t, plan)
	assert.True(t, plan.Moment.Equal(at(t, 9, 0)))
	assert.Equal(t, []model.Checkin{morning}, plan.Fire)
}
