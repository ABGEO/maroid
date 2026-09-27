package telegram

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/abgeo/maroid/apps/hub/internal/config"
)

// APIFMT-SC-014: The webhook address carries the scheme and the host of the one
// stored external address, and the path of the webhook.
func TestTheWebhookAddressComesFromTheExternalAddress(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{}
	cfg.Server.ExternalURL = "https://hub.example.com/"
	cfg.Telegram.Webhook.Path = "/telegram/webhook"

	params := webhookParams(cfg, "the-secret")

	assert.Equal(t, "https://hub.example.com/telegram/webhook", params.URL)
	assert.Equal(t, "the-secret", params.SecretToken)
}
