package service_test

import (
	"context"
	"crypto/md5" //nolint:gosec // G501: the car log service demands the MD5 of the password.
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/plugins/drivvo/errs"
	"github.com/abgeo/maroid/plugins/drivvo/internal/fakeservice"
	"github.com/abgeo/maroid/plugins/drivvo/model"
	"github.com/abgeo/maroid/plugins/drivvo/service"
)

const (
	email         = "driver@example.com"
	spacedSecret  = " secret "
	pathRefuelsOf = "/veiculo/11/abastecimento/web"
)

//nolint:gochecknoglobals // the fixed now of every test of this file.
var now = time.Date(2026, 11, 25, 6, 0, 0, 0, time.UTC)

// secretMD5 answers the lowercase hex MD5 of the password with its outer spaces cut.
func secretMD5() string {
	plain := []byte(strings.TrimSpace(spacedSecret))
	digest := md5.Sum(plain) //nolint:gosec // G401: see the import.

	return hex.EncodeToString(digest[:])
}

func newClient(t *testing.T) (*service.APIClient, *fakeservice.Service) {
	t.Helper()

	fake := fakeservice.New(t)

	return service.NewAPIClient(testConfig(fake.URL()), func() time.Time { return now }), fake
}

// DRIVVO-SC-005: The sign in sends the email address and the MD5 of the trimmed password
// as the web site does, and the session ends at now plus expira_em_segundos.
func TestTheSignInSendsTheDigestAndKeepsTheToken(t *testing.T) {
	t.Parallel()

	client, fake := newClient(t)

	session, err := client.SignIn(context.Background(), email, spacedSecret)
	require.NoError(t, err)

	assert.Equal(t, model.Session{
		Token:     fakeservice.Token,
		ExpiresAt: now.Add(fakeservice.TokenSeconds * time.Second),
	}, session)

	signIns := fake.CallsTo(fakeservice.PathLogIn)
	require.Len(t, signIns, 1)
	assert.Equal(t, "Web", signIns[0].Platform)
	assert.Equal(t, "23", signIns[0].Version)
	assert.Equal(t, "14.2.3", signIns[0].VersionName)

	var body map[string]string
	require.NoError(t, json.Unmarshal(signIns[0].Body, &body))
	assert.Equal(t, map[string]string{"email": email, "senha": secretMD5(), "idioma": "en"}, body)
}

// DRIVVO-SC-006: A 401, a 403, and the exception code 1908 each reject the credentials.
func TestTheSignInNamesARejection(t *testing.T) {
	t.Parallel()

	rejection := map[string]any{"result": false, memberException: map[string]any{memberCode: 1908}}
	cases := map[string]struct {
		status int
		body   any
	}{
		"401":           {status: http.StatusUnauthorized, body: map[string]any{}},
		"403":           {status: http.StatusForbidden, body: map[string]any{}},
		"1908 with 200": {status: http.StatusOK, body: rejection},
		"1908 with 400": {status: http.StatusBadRequest, body: rejection},
	}

	for name, setup := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			client, fake := newClient(t)
			fake.Set(func(state *fakeservice.Service) {
				state.SignInStatus, state.SignInBody = setup.status, setup.body
			})

			_, err := client.SignIn(context.Background(), email, spacedSecret)
			require.ErrorIs(t, err, errs.ErrCredentialsRejected)
			assert.True(t, service.IsRefusal(err))
		})
	}
}

// DRIVVO-SC-007: An answer with alterar_senha asks for a new password, even with a token.
func TestTheSignInRefusesAnAccountThatAsksForANewPassword(t *testing.T) {
	t.Parallel()

	client, fake := newClient(t)
	fake.Set(func(state *fakeservice.Service) {
		state.SignInBody = map[string]any{memberToken: "T", "alterar_senha": true}
	})

	_, err := client.SignIn(context.Background(), email, spacedSecret)
	require.ErrorIs(t, err, errs.ErrNewPasswordRequired)
	assert.True(t, service.IsRefusal(err))
}

// DRIVVO-SC-004: A failure of the service that is not a rejection is ErrRequestFailed, and
// no refusal.
func TestAFailedSignInIsNoRefusal(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		status int
		body   any
	}{
		"503":              {status: http.StatusServiceUnavailable, body: map[string]any{}},
		"200 and no token": {status: http.StatusOK, body: map[string]any{memberToken: ""}},
		"another code": {
			status: http.StatusOK,
			body:   map[string]any{memberException: map[string]any{memberCode: 7}},
		},
	}

	for name, setup := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			client, fake := newClient(t)
			fake.Set(func(state *fakeservice.Service) {
				state.SignInStatus, state.SignInBody = setup.status, setup.body
			})

			_, err := client.SignIn(context.Background(), email, spacedSecret)
			require.ErrorIs(t, err, errs.ErrRequestFailed)
			assert.False(t, service.IsRefusal(err))
		})
	}
}

// DRIVVO-SC-027: The error of a failed sign in names no email address, though the answer
// repeats it, because the error reaches the log.
func TestAFailedSignInNamesNoEmail(t *testing.T) {
	t.Parallel()

	client, fake := newClient(t)
	fake.Set(func(state *fakeservice.Service) {
		state.SignInStatus = http.StatusBadRequest
		state.SignInBody = map[string]any{"message": "no account for " + email}
	})

	_, err := client.SignIn(context.Background(), email, spacedSecret)
	require.ErrorIs(t, err, errs.ErrRequestFailed)
	assert.NotContains(t, err.Error(), email)
}

// DRIVVO-SC-003: The vehicles come from /veiculo/web, read with the token.
func TestTheVehiclesComeWithTheToken(t *testing.T) {
	t.Parallel()

	client, fake := newClient(t)

	vehicles, err := client.Vehicles(context.Background(), fakeservice.Token)
	require.NoError(t, err)

	require.Len(t, vehicles, 2)
	assert.Equal(t, nameHatchback, vehicles[0].Name)
	assert.True(t, vehicles[0].Active)
	assert.True(t, vehicles[0].Principal)
	assert.Equal(t, "Make Hatchback", vehicles[0].Make)
	assert.Equal(t, "Model Hatchback", vehicles[0].Model)
	assert.Equal(t, model.DistanceKilometre, vehicles[0].DistanceUnit)
	assert.False(t, vehicles[1].Active)

	calls := fake.CallsTo(fakeservice.PathVehicles)
	require.Len(t, calls, 1)
	assert.Equal(t, fakeservice.Token, calls[0].Token)
	assert.Equal(t, "Web", calls[0].Platform)
}

// DRIVVO-SC-009: A data call that the service answers with 401 is a rejected session.
func TestARejectedTokenIsARejectedSession(t *testing.T) {
	t.Parallel()

	client, _ := newClient(t)

	_, err := client.Vehicles(context.Background(), "OLD")
	require.ErrorIs(t, err, errs.ErrSessionRejected)
}

// DRIVVO-SC-004, DRIVVO-SC-017: A data call that fails names the path and the status.
func TestAFailedDataCallNamesThePath(t *testing.T) {
	t.Parallel()

	client, fake := newClient(t)
	fake.Set(
		func(state *fakeservice.Service) { state.Status[pathRefuelsOf] = http.StatusServiceUnavailable },
	)

	_, err := client.VehicleEntries(context.Background(), fakeservice.Token, fakeservice.Hatchback)
	require.ErrorIs(t, err, errs.ErrRequestFailed)
	assert.Contains(t, err.Error(), pathRefuelsOf)
	assert.Contains(t, err.Error(), "[503]")
}

// DRIVVO-SC-004: A unit outside the lists of the service fails the read, and names the
// member.
func TestAnUnknownUnitFailsTheRead(t *testing.T) {
	t.Parallel()

	client, fake := newClient(t)
	fake.Set(func(state *fakeservice.Service) {
		state.Answers[fakeservice.PathSettings] = []any{
			map[string]any{
				"unidade_medida":         9,
				"eficiencia_combustivel": 4,
				"formato_valor":          currencyGEL,
			},
		}
	})

	_, err := client.Settings(context.Background(), fakeservice.Token)
	require.ErrorIs(t, err, errs.ErrRequestFailed)
	assert.Contains(t, err.Error(), "unidade_medida")
}

// DRIVVO-SC-033: The settings name the volume unit, the consumption format, and the
// currency.
func TestTheSettingsNameTheUnitsAndTheCurrency(t *testing.T) {
	t.Parallel()

	client, _ := newClient(t)

	settings, err := client.Settings(context.Background(), fakeservice.Token)
	require.NoError(t, err)

	assert.Equal(t, model.CarLogSettings{
		VolumeUnit:        model.VolumeLitre,
		ConsumptionFormat: 4,
		Currency:          currencyGEL,
	}, settings)
}

// DRIVVO-SC-014: A collection reads the five lists of entries and the reminders of each
// vehicle, an inactive vehicle too.
func TestEachVehicleIsReadOnSixPaths(t *testing.T) {
	t.Parallel()

	client, fake := newClient(t)
	fake.Set(func(state *fakeservice.Service) {
		state.Answers["/veiculo/12/abastecimento/web"] = []any{
			petrolRefuel(301, "2020-01-01 10:00:00"),
			petrolRefuel(302, "2020-02-01 10:00:00"),
			petrolRefuel(303, "2020-03-01 10:00:00"),
		}
		state.Answers["/veiculo/12/lembrete/web"] = []any{
			map[string]any{
				"id_lembrete":         401,
				memberMaintenanceType: 5,
				memberOdometer:        90000,
				"unico_repetir":       true,
			},
		}
	})

	entries, err := client.VehicleEntries(
		context.Background(),
		fakeservice.Token,
		fakeservice.OldSedan,
	)
	require.NoError(t, err)

	reminders, err := client.Reminders(
		context.Background(),
		fakeservice.Token,
		fakeservice.OldSedan,
	)
	require.NoError(t, err)

	require.Len(t, entries, 3)

	for _, entry := range entries {
		assert.Equal(t, int32(fakeservice.OldSedan), entry.VehicleSource)
	}

	require.Len(t, reminders, 1)
	assert.Equal(t, int32(fakeservice.OldSedan), reminders[0].VehicleSource)

	for _, path := range fakeservice.VehiclePaths(fakeservice.OldSedan) {
		assert.Len(t, fake.CallsTo(path), 1, path)
	}
}

// DRIVVO-SC-015: The reference items of each list carry their kind, identifier, and name.
func TestTheReferenceItemsComeFromEightLists(t *testing.T) {
	t.Parallel()

	client, fake := newClient(t)

	items, err := client.ReferenceItems(context.Background(), fakeservice.Token)
	require.NoError(t, err)

	assert.Contains(
		t,
		items,
		model.ReferenceItem{Kind: model.ReferenceStation, SourceID: 4, Name: "Gulf"},
	)
	assert.Contains(
		t,
		items,
		model.ReferenceItem{Kind: model.ReferenceFuel, SourceID: 2, Name: "LPG"},
	)
	assert.Len(t, items, 8)

	paths := []string{
		"/combustivel", "/tipoServico", "/tipoDespesa", "/tipoReceita",
		"/tipoMotivo", "/formaPagamento", "/postoCombustivel", "/local",
	}
	for _, path := range paths {
		assert.Len(t, fake.CallsTo(path), 1, path)
	}
}

// DRIVVO-SC-027: The session holds the token and its end, and no value of the holder.
func TestTheSessionHoldsNoValueOfTheHolder(t *testing.T) {
	t.Parallel()

	client, _ := newClient(t)

	session, err := client.SignIn(context.Background(), email, spacedSecret)
	require.NoError(t, err)

	stored, err := session.MarshalSecret()
	require.NoError(t, err)

	for _, value := range []string{
		fakeservice.HolderName, fakeservice.HolderEmail, fakeservice.HolderPhone,
		fakeservice.HolderLicence, secretMD5(), strings.TrimSpace(spacedSecret),
	} {
		assert.NotContains(t, stored, value)
	}
}

func petrolRefuel(id int, moment string) map[string]any {
	return map[string]any{
		"id_abastecimento": id,
		memberDay:          moment,
		"id_combustivel":   1,
		"valor_total":      10,
	}
}
