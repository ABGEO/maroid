package service_test

import (
	"context"
	"encoding/base64"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/plugins/gwp/config"
	"github.com/abgeo/maroid/plugins/gwp/service"
)

// proxied is one request that the fake proxy received.
type proxied struct {
	url           string
	authorization string
}

// fakeProxy plays an HTTP proxy. It records each request and answers the list of the
// customers of the water service.
func fakeProxy(t *testing.T) (*httptest.Server, func() []proxied) {
	t.Helper()

	var (
		mu       sync.Mutex
		received []proxied
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()

		received = append(received, proxied{
			url:           r.URL.String(),
			authorization: r.Header.Get("Proxy-Authorization"),
		})
		mu.Unlock()

		_, _ = w.Write([]byte(`{"success": true, "items": []}`))
	}))
	t.Cleanup(server.Close)

	return server, func() []proxied {
		mu.Lock()
		defer mu.Unlock()

		return append([]proxied(nil), received...)
	}
}

func proxyOf(t *testing.T, server *httptest.Server, username, password string) config.Proxy {
	t.Helper()

	host, rawPort, err := net.SplitHostPort(server.Listener.Addr().String())
	require.NoError(t, err)

	port, err := strconv.Atoi(rawPort)
	require.NoError(t, err)

	return config.Proxy{
		Enabled:  true,
		Scheme:   "http",
		Host:     host,
		Port:     port,
		Username: username,
		Password: password,
	}
}

func TestACallGoesThroughTheProxy(t *testing.T) {
	t.Parallel()

	server, received := fakeProxy(t)

	client := service.NewAPIClient(&config.Config{
		BaseURL:        "http://water.example/api",
		RequestTimeout: time.Second,
		Proxy:          proxyOf(t, server, "customer-maroid", "p@ss:w/rd+1"),
	})

	_, err := client.GetCustomers(context.Background(), "token-1")
	require.NoError(t, err)

	calls := received()
	require.Len(t, calls, 1)
	require.Equal(t, "http://water.example/api/customer/ListCustomers", calls[0].url)
	require.Equal(
		t,
		"Basic "+base64.StdEncoding.EncodeToString([]byte("customer-maroid:p@ss:w/rd+1")),
		calls[0].authorization,
	)
}

func TestAProxyWithNoUsernameSendsNoCredentials(t *testing.T) {
	t.Parallel()

	server, received := fakeProxy(t)

	client := service.NewAPIClient(&config.Config{
		BaseURL:        "http://water.example/api",
		RequestTimeout: time.Second,
		Proxy:          proxyOf(t, server, "", ""),
	})

	_, err := client.GetCustomers(context.Background(), "token-1")
	require.NoError(t, err)

	calls := received()
	require.Len(t, calls, 1)
	require.Empty(t, calls[0].authorization)
}
