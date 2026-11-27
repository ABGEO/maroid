package service_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/plugins/parking/config"
	"github.com/abgeo/maroid/plugins/parking/errs"
	"github.com/abgeo/maroid/plugins/parking/service"
)

const historyBody = `{"result":{"dataType":"PAGE","data":[{"id":45304769,"status":"FINISHED",` +
	`"startDate":"2026-11-21T13:26:08","endDate":"2026-11-21T14:14:21","amount":0.55}],` +
	`"page":{"total":468,"number":3,"size":20}}}`

// PARKHIST-SC-012: The client asks for one page of 20 rows, and decodes the page.
func TestTheClientReadsOnePageOfTheHistory(t *testing.T) {
	t.Parallel()

	var queries []string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		queries = append(queries, r.URL.Path+"?"+r.URL.RawQuery)

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(historyBody))
	}))
	t.Cleanup(server.Close)

	secrets := &memorySecrets{values: map[string]string{userA + "/" + service.SecretSession: "T1"}}
	client := service.NewAPIClient(&config.Config{BaseURL: server.URL}, secrets)

	page, err := client.ParkingHistory(as(userA), 3)
	require.NoError(t, err)

	_, err = client.PaymentHistory(as(userA), 0)
	require.NoError(t, err)

	assert.Equal(t, []string{
		"/parking/history?page=3&size=20",
		"/parking/person/payment-history?page=0&size=20",
	}, queries)
	assert.Equal(t, 468, page.Meta.Total)
	require.Len(t, page.Items, 1)
	assert.Equal(t, int64(45304769), page.Items[0].ID)
}

// PARKHIST-SC-010: A rejection of a history call removes the token, and answers not
// connected.
func TestARejectedHistoryCallRemovesTheToken(t *testing.T) {
	t.Parallel()

	fix := newFixture(t, http.StatusUnauthorized, `{}`)
	fix.secrets.values[userA+"/"+service.SecretSession] = "T1"

	_, err := fix.client.ParkingHistory(as(userA), 0)

	require.ErrorIs(t, err, errs.ErrNotConnected)
	require.NotContains(t, fix.secrets.values, userA+"/"+service.SecretSession)
}
