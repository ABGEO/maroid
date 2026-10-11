package notice_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/abgeo/maroid/libs/pluginapi"
	"github.com/abgeo/maroid/plugins/parking/service"
	"github.com/abgeo/maroid/plugins/parking/telegram/notice"
)

const accountURL = "https://deck.example/w/W/plugins/dev.maroid.parking/account"

var errOther = errors.New("other")

// fakePages answers the address that the test gives, or its error.
type fakePages struct {
	address string
	err     error
	path    string
}

func (p *fakePages) PageURL(_ context.Context, _ *pluginapi.PluginID, path string) (string, error) {
	p.path = path

	return p.address, p.err
}

func noticeOf(pages *fakePages) *notice.Notice {
	return notice.New(pages, pluginapi.ParsePluginID("dev.maroid.parking"))
}

// PARKACC-SC-018: The answer to a user with no connection says so and links to the
// account page.
func TestNotConnectedLinksToTheAccountPage(t *testing.T) {
	t.Parallel()

	pages := &fakePages{address: accountURL}

	text, known := noticeOf(
		pages,
	).Text(context.Background(), fmt.Errorf("fetching person: %w", service.ErrNotConnected))

	require.True(t, known)
	require.Contains(t, text, "not connected")
	require.Contains(t, text, `href="`+accountURL+`"`)
	require.Equal(t, "/account", pages.path)
}

// PARKACC-SC-019: With no page address, the answer names the account page and holds no link.
func TestNotConnectedWithNoAddressNamesThePage(t *testing.T) {
	t.Parallel()

	text, known := noticeOf(&fakePages{err: pluginapi.ErrPageAddressAbsent}).
		Text(context.Background(), service.ErrNotConnected)

	require.True(t, known)
	require.Contains(t, text, "not connected")
	require.Contains(t, text, "Account page")
	require.NotContains(t, text, "href")
}

// The bot answer of section 4.5: a failure of the parking service says so, and any
// other failure goes back to the caller.
func TestOtherFailures(t *testing.T) {
	t.Parallel()

	answers := noticeOf(&fakePages{address: accountURL})

	text, known := answers.Text(context.Background(), &service.StatusError{Status: 503})
	require.True(t, known)
	require.Contains(t, text, "unavailable")

	_, known = answers.Text(context.Background(), errOther)
	require.False(t, known)
}
