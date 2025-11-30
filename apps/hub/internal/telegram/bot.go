package telegram

import (
	"fmt"
	"time"

	"github.com/mymmrac/telego"
	ta "github.com/mymmrac/telego/telegoapi"

	"github.com/abgeo/maroid/apps/hub/internal/config"
)

const (
	retryMaxAttempts  = 4
	retryExponentBase = 2
	retryStartDelay   = 10 * time.Millisecond
	retryMaxDelay     = 1 * time.Second
)

// NewBot creates and returns a new Telego Bot instance.
func NewBot(cfg *config.Config) (*telego.Bot, error) {
	options := []telego.BotOption{
		telego.WithAPICaller(&ta.RetryCaller{
			Caller:       ta.DefaultFastHTTPCaller,
			MaxAttempts:  retryMaxAttempts,
			ExponentBase: retryExponentBase,
			StartDelay:   retryStartDelay,
			MaxDelay:     retryMaxDelay,
		}),
	}

	if cfg.Telegram.Debug {
		options = append(options, telego.WithDefaultDebugLogger())
	}

	bot, err := telego.NewBot(cfg.Telegram.Token, options...)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Telego Bot: %w", err)
	}

	return bot, nil
}
