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

// ChatOf returns the identifier of the chat of a message or of a tap on a button.
// It reports false for an update that belongs to no chat.
func ChatOf(update telego.Update) (int64, bool) {
	switch {
	case update.Message != nil:
		return update.Message.Chat.ID, true
	case update.CallbackQuery != nil && update.CallbackQuery.Message != nil:
		return update.CallbackQuery.Message.GetChat().ID, true
	default:
		return 0, false
	}
}
