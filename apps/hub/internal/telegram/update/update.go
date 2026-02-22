// Package update provides utilities for working with Telegram updates.
package update

import "github.com/mymmrac/telego"

// SentFrom extracts the user who sent the given Telegram update.
// It returns nil if no user can be determined.
func SentFrom(update telego.Update) *telego.User {
	switch {
	case update.Message != nil:
		return update.Message.From
	case update.EditedMessage != nil:
		return update.EditedMessage.From
	case update.InlineQuery != nil:
		return &update.InlineQuery.From
	case update.ChosenInlineResult != nil:
		return &update.ChosenInlineResult.From
	case update.CallbackQuery != nil:
		return &update.CallbackQuery.From
	case update.ShippingQuery != nil:
		return &update.ShippingQuery.From
	case update.PreCheckoutQuery != nil:
		return &update.PreCheckoutQuery.From
	default:
		return nil
	}
}
