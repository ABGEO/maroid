package pluginapi_test

import (
	"database/sql"
	"reflect"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"

	"github.com/abgeo/maroid/libs/pluginapi"
)

// PLGBIND-SC-016: No method of the host, and no method of a value that the host
// answers, takes a plugin identifier or answers a database pool.
func TestTheHostOffersNoIdentifierParameterAndNoPool(t *testing.T) {
	t.Parallel()

	var (
		pluginID = reflect.TypeFor[*pluginapi.PluginID]()
		pools    = []reflect.Type{reflect.TypeFor[*sqlx.DB](), reflect.TypeFor[*sql.DB]()}
	)

	host := reflect.TypeFor[pluginapi.Host]()

	for method := range host.Methods() {
		for answer := range method.Type.Outs() {
			assert.NotContains(t, pools, answer, "Host.%s answers a pool", method.Name)

			if answer.Kind() != reflect.Interface || answer.PkgPath() != host.PkgPath() {
				continue
			}

			for service := range answer.Methods() {
				for param := range service.Type.Ins() {
					assert.NotEqual(t, pluginID, param,
						"%s.%s takes a plugin identifier", answer.Name(), service.Name)
				}

				for out := range service.Type.Outs() {
					assert.NotContains(t, pools, out,
						"%s.%s answers a pool", answer.Name(), service.Name)
				}
			}
		}

		for param := range method.Type.Ins() {
			assert.NotEqual(t, pluginID, param, "Host.%s takes a plugin identifier", method.Name)
		}
	}
}
