package filestore_test

import (
	"bytes"
	"context"
	"io"
	"maps"
	"net/http"
	"net/url"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/apps/hub/internal/filestore"
	"github.com/abgeo/maroid/apps/hub/internal/teststore"
)

const (
	megabyte   = 1 << 20
	jpegType   = "image/jpeg"
	webpType   = "image/webp"
	fileKey    = "plugins/dev.maroid.probe/workspaces/w1/f1/"
	uploadKey  = fileKey + "upload"
	displayKey = fileKey + "display.webp"
)

func startClient(t *testing.T, opts ...filestore.Option) (*filestore.Client, *teststore.Server) {
	t.Helper()

	if testing.Short() {
		t.Skip("skipping the integration test, it needs Docker")
	}

	server := teststore.Start(t)

	return filestore.New(server.Config(), opts...), server
}

// answer holds the status, the headers, and the body of one response.
type answer struct {
	status int
	header http.Header
	body   []byte
}

// send runs one request against an address, with the headers, and answers the response.
func send(t *testing.T, method, address string, headers map[string]string, body []byte) answer {
	t.Helper()

	request, err := http.NewRequestWithContext(t.Context(), method, address, bytes.NewReader(body))
	require.NoError(t, err)

	for name, value := range headers {
		request.Header.Set(name, value)
	}

	request.ContentLength = int64(len(body))

	response, err := http.DefaultClient.Do(request)
	require.NoError(t, err)

	defer func() { _ = response.Body.Close() }()

	read, err := io.ReadAll(response.Body)
	require.NoError(t, err)

	return answer{status: response.StatusCode, header: response.Header, body: read}
}

func expiresOf(t *testing.T, address string) int {
	t.Helper()

	parsed, err := url.Parse(address)
	require.NoError(t, err)

	seconds, err := strconv.Atoi(parsed.Query().Get("X-Amz-Expires"))
	require.NoError(t, err)

	return seconds
}

// FILES-SC-008: The file store refuses a body larger than the signed size, and a body
// of another type than the signed one, so no upload lands.
func TestTheFileStoreRefusesABodyThatDiffersFromTheSignature(t *testing.T) {
	t.Parallel()

	client, _ := startClient(t)

	upload, err := client.PresignPut(t.Context(), uploadKey, megabyte, jpegType)
	require.NoError(t, err)
	require.Equal(t, http.MethodPut, upload.Method)

	larger := send(t, upload.Method, upload.URL, upload.Headers, make([]byte, 2*megabyte))
	require.Equal(t, http.StatusForbidden, larger.status)

	headers := maps.Clone(upload.Headers)

	headers["Content-Type"] = "text/plain"

	otherType := send(t, upload.Method, upload.URL, headers, make([]byte, megabyte))
	require.Equal(t, http.StatusForbidden, otherType.status)

	_, err = client.Head(t.Context(), uploadKey)
	require.ErrorIs(t, err, filestore.ErrObjectAbsent)
}

// FILES-SC-008: A body of the signed size and the signed type lands, and the file store
// answers its facts.
func TestTheFileStoreTakesABodyThatMatchesTheSignature(t *testing.T) {
	t.Parallel()

	client, _ := startClient(t)

	upload, err := client.PresignPut(t.Context(), uploadKey, megabyte, jpegType)
	require.NoError(t, err)

	landed := send(t, upload.Method, upload.URL, upload.Headers, make([]byte, megabyte))
	require.Equal(t, http.StatusOK, landed.status)

	object, err := client.Head(t.Context(), uploadKey)
	require.NoError(t, err)
	require.Equal(t, int64(megabyte), object.Size)
	require.Equal(t, jpegType, object.ContentType)
}

// FILES-SC-037: An upload address lasts at most 15 minutes, and the file store refuses
// it after that.
func TestAnUploadAddressStopsAfterFifteenMinutes(t *testing.T) {
	t.Parallel()

	signedAt := time.Now().Add(-(15*time.Minute + time.Second))
	client, _ := startClient(t, filestore.WithClock(func() time.Time { return signedAt }))

	upload, err := client.PresignPut(t.Context(), uploadKey, megabyte, jpegType)
	require.NoError(t, err)
	require.LessOrEqual(t, expiresOf(t, upload.URL), 900)
	require.True(t, upload.ExpiresAt.Before(time.Now()))

	late := send(t, upload.Method, upload.URL, upload.Headers, make([]byte, megabyte))
	require.Equal(t, http.StatusForbidden, late.status)
}

// FILES-SC-027: A read address answers the object with its type.
func TestAReadAddressAnswersTheObject(t *testing.T) {
	t.Parallel()

	client, _ := startClient(t)
	content := []byte("RIFF....WEBP")

	require.NoError(t, client.Write(t.Context(), displayKey, content, webpType))

	address, err := client.PresignGet(t.Context(), displayKey, "")
	require.NoError(t, err)

	response := send(t, http.MethodGet, address.URL, nil, nil)
	require.Equal(t, http.StatusOK, response.status)
	require.Equal(t, webpType, response.header.Get("Content-Type"))

	require.Equal(t, content, response.body)
}

// FILES-DD-010: The read address of an original makes the browser save it under the
// name of the file.
func TestTheReadAddressOfAnOriginalSavesItUnderItsName(t *testing.T) {
	t.Parallel()

	client, _ := startClient(t)

	require.NoError(t, client.Write(t.Context(), uploadKey, []byte("%PDF-1.7"), "application/pdf"))

	address, err := client.PresignGet(t.Context(), uploadKey, "lease contract.pdf")
	require.NoError(t, err)

	response := send(t, http.MethodGet, address.URL, nil, nil)
	require.Equal(t, http.StatusOK, response.status)
	require.Equal(
		t,
		`attachment; filename="lease contract.pdf"`,
		response.header.Get("Content-Disposition"),
	)
}

// FILES-SC-040: The file store refuses a read and a write that carry no signature.
func TestTheFileStoreRefusesAnUnsignedRequest(t *testing.T) {
	t.Parallel()

	client, server := startClient(t)

	require.NoError(t, client.Write(t.Context(), displayKey, []byte("RIFF"), webpType))

	address := server.Endpoint + "/" + teststore.Bucket + "/" + displayKey

	require.Equal(t, http.StatusForbidden, send(t, http.MethodGet, address, nil, nil).status)
	require.Equal(
		t,
		http.StatusForbidden,
		send(
			t,
			http.MethodPut,
			address,
			map[string]string{"Content-Type": webpType},
			[]byte("RIFF"),
		).status,
	)
}

// FILES-SC-039: The purge removes every object of a file, and keeps the objects of
// another file.
func TestDeletePrefixRemovesEveryObjectOfAFile(t *testing.T) {
	t.Parallel()

	client, _ := startClient(t)
	other := "plugins/dev.maroid.probe/workspaces/w1/f2/display.webp"

	for _, key := range []string{uploadKey, displayKey, fileKey + "thumbnail.webp", other} {
		require.NoError(t, client.Write(t.Context(), key, []byte("x"), webpType))
	}

	require.NoError(t, client.DeletePrefix(t.Context(), fileKey))

	for _, key := range []string{uploadKey, displayKey, fileKey + "thumbnail.webp"} {
		_, err := client.Head(t.Context(), key)
		require.ErrorIs(t, err, filestore.ErrObjectAbsent)
	}

	_, err := client.Head(t.Context(), other)
	require.NoError(t, err)
}

// FILES-SC-013: The start of an object reads at most the asked bytes, and an absent
// upload answers ErrObjectAbsent.
func TestReadRangeAnswersTheStartOfAnObject(t *testing.T) {
	t.Parallel()

	client, _ := startClient(t)
	content := bytes.Repeat([]byte("abcdefgh"), 1024)

	require.NoError(t, client.Write(t.Context(), uploadKey, content, jpegType))

	start, err := client.ReadRange(t.Context(), uploadKey, 3072)
	require.NoError(t, err)
	require.Equal(t, content[:3072], start)

	_, err = client.ReadRange(t.Context(), fileKey+"absent", 3072)
	require.ErrorIs(t, err, filestore.ErrObjectAbsent)
}

// FILES-SC-030: Ping answers while the file store runs, and fails within the limit
// once it stops.
func TestPingFailsWithinTheLimitWhenTheFileStoreStops(t *testing.T) {
	t.Parallel()

	client, server := startClient(t)

	require.NoError(t, client.Ping(t.Context()))

	server.Stop(t)

	started := time.Now()

	require.Error(t, client.Ping(context.Background()))
	require.Less(t, time.Since(started), filestore.PingLimit+time.Second)
}

// LOG-008: An error of an unreachable file store names no key, so neither a log
// record nor the error of a task holds one.
func TestAnErrorOfTheFileStoreNamesNoKey(t *testing.T) {
	t.Parallel()

	client, server := startClient(t)
	server.Stop(t)

	key := fileKey + "a-key-that-must-not-leak"

	_, headErr := client.Head(t.Context(), key)
	_, readErr := client.ReadRange(t.Context(), key, 3072)

	for _, err := range []error{
		headErr,
		readErr,
		client.Write(t.Context(), key, []byte("x"), webpType),
		client.DeletePrefix(t.Context(), fileKey),
	} {
		require.ErrorIs(t, err, filestore.ErrUnreachable)
		require.NotContains(t, err.Error(), "a-key-that-must-not-leak")
		require.NotContains(t, err.Error(), fileKey)
	}
}
