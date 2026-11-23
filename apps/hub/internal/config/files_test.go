package config_test

import (
	"testing"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/mcuadros/go-defaults"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/apps/hub/internal/config"
)

func validFiles() config.Files {
	files := config.Files{
		Endpoint:       "http://rustfs.storage.svc:9000",
		PublicEndpoint: "https://files.example.com",
		Bucket:         "maroid",
		AccessKey:      "maroid-hub",
		SecretKey:      "a-secret",
	}
	defaults.SetDefaults(&files)

	return files
}

// FILES-SC-037: FILES-SC-038: An upload address lasts 15 minutes and a read address
// lasts one hour unless the deployment says less.
func TestTheAddressesOfTheFilesDefaultToTheirLimits(t *testing.T) {
	t.Parallel()

	files := validFiles()

	assert.Equal(t, 15*time.Minute, files.UploadTTL)
	assert.Equal(t, time.Hour, files.ReadTTL)
	assert.Equal(t, "us-east-1", files.Region)
	require.NoError(t, validator.New().Struct(files))
}

// FILES-SC-037: FILES-SC-038: No deployment gives an upload address more than 15
// minutes or a read address more than one hour.
func TestTheAddressesOfTheFilesStayWithinTheirLimits(t *testing.T) {
	t.Parallel()

	validate := validator.New()

	for name, testCase := range map[string]struct {
		change func(*config.Files)
		valid  bool
	}{
		"a shorter upload":         {change: func(f *config.Files) { f.UploadTTL = 5 * time.Minute }, valid: true},
		"an upload past 15m":       {change: func(f *config.Files) { f.UploadTTL = 16 * time.Minute }},
		"a read past one hour":     {change: func(f *config.Files) { f.ReadTTL = 61 * time.Minute }},
		"a read of zero":           {change: func(f *config.Files) { f.ReadTTL = 0 }},
		"a public endpoint absent": {change: func(f *config.Files) { f.PublicEndpoint = "" }},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			files := validFiles()
			testCase.change(&files)

			err := validate.Struct(files)

			if testCase.valid {
				require.NoError(t, err)

				return
			}

			require.Error(t, err)
		})
	}
}
