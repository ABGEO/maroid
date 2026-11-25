package model_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/plugins/drivvo/errs"
	"github.com/abgeo/maroid/plugins/drivvo/model"
)

// DRIVVO-SC-039: The formats 1 to 3 and 7 to 9 read as a distance per volume, and the
// formats 4 to 6 and 10 to 12 as a volume per 100 of the distance.
func TestAConsumptionFormatGivesItsShape(t *testing.T) {
	t.Parallel()

	for format := model.FirstConsumptionFormat; format <= model.LastConsumptionFormat; format++ {
		want := model.ShapeDistancePerVolume
		if (format >= 4 && format <= 6) || format >= 10 {
			want = model.ShapeVolumePer100
		}

		assert.True(t, format.Valid())
		assert.Equal(t, want, format.Shape(), "format %d", format)
	}

	assert.False(t, model.ConsumptionFormat(0).Valid())
	assert.False(t, model.ConsumptionFormat(13).Valid())
}

// DRIVVO-SC-037: An amount sums with no error of a binary fraction, and reads back from a
// NUMERIC column.
func TestAnAmountSumsExactly(t *testing.T) {
	t.Parallel()

	sum := model.AmountOf(0.1) + model.AmountOf(0.2)
	assert.Equal(t, model.AmountOf(0.3), sum)
	assert.Equal(t, "0.300", sum.String())
	assert.Equal(t, "-2.859", model.AmountOf(-2.859).String())

	var read model.Amount
	require.NoError(t, read.Scan("120.5"))
	assert.Equal(t, model.AmountOf(120.5), read)
	assert.InDelta(t, 120.5, read.Float(), 0)
}

// DRIVVO-SC-010: A session ends at its moment, and a stored session with no token is no
// connection.
func TestASessionEndsAtItsMoment(t *testing.T) {
	t.Parallel()

	end := time.Date(2026, 11, 25, 10, 0, 0, 0, time.UTC)
	session := model.Session{Token: "T", ExpiresAt: end}

	assert.False(t, session.Expired(end.Add(-time.Minute)))
	assert.True(t, session.Expired(end))

	stored, err := session.MarshalSecret()
	require.NoError(t, err)

	read, err := model.ParseSession(stored)
	require.NoError(t, err)
	assert.Equal(t, session, read)

	_, err = model.ParseSession(`{"token":""}`)
	require.ErrorIs(t, err, errs.ErrNotConnected)
}
