package job

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/abgeo/maroid/plugins/jasmine/model"
)

// GARDEN-SC-035: The names that a member typed reach Telegram as text, not as HTML.
func TestTheDigestEscapesTheNames(t *testing.T) {
	t.Parallel()

	due := model.CareStateDue
	body := digest([]model.CareInterval{{
		Plant: model.PlantReference{
			ID:          "p1",
			Number:      1,
			CommonName:  "Rose <Big> & Co",
			Environment: model.EnvironmentReference{ID: "e1", Number: 1, Name: "Office & Hall"},
		},
		Type:      model.MaintenanceTypeWatering,
		Days:      7,
		DaysSince: 7,
		State:     &due,
	}}).Body

	assert.Contains(t, body, "Office &amp; Hall")
	assert.Contains(t, body, "Rose &lt;Big&gt; &amp; Co")
	assert.NotContains(t, body, "<Big>")
}
