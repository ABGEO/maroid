package pluginmsg

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/mymmrac/telego"

	messageapi "github.com/abgeo/maroid/libs/pluginapi/telegram/message"
)

// Validate answers an error that wraps messageapi.ErrInvalid for a
// message that Telegram would refuse.
func Validate(msg messageapi.Message, now time.Time) error {
	if err := validateText(msg.Text); err != nil {
		return err
	}

	if err := validateButtons(msg.Buttons); err != nil {
		return err
	}

	if err := validateKeyboard(msg.Keyboard); err != nil {
		return err
	}

	if markupCount(msg) > 1 {
		return invalid("a message holds one of buttons, a keyboard, or a keyboard removal")
	}

	if msg.ClaimReplyUntil.IsZero() {
		return nil
	}

	if !msg.ClaimReplyUntil.After(now) {
		return invalid("the claim ends at or before now")
	}

	if msg.ClaimReplyUntil.After(now.Add(messageapi.Lifetime)) {
		return invalid("the claim ends after the lifetime of the message")
	}

	return nil
}

func validateText(text string) error {
	if strings.TrimSpace(text) == "" {
		return invalid("the text is empty")
	}

	if utf8.RuneCountInString(text) > messageapi.TextMax {
		return invalid(fmt.Sprintf("the text holds over %d characters", messageapi.TextMax))
	}

	return nil
}

func validateButtons(rows [][]messageapi.Button) error {
	for _, row := range rows {
		if len(row) == 0 {
			return invalid("a row of buttons is empty")
		}

		for _, button := range row {
			if strings.TrimSpace(button.Label) == "" {
				return invalid("a button has an empty label")
			}

			if button.Value == "" || len(button.Value) > messageapi.ButtonValueMax {
				return invalid(fmt.Sprintf(
					"a button value holds 1 to %d bytes", messageapi.ButtonValueMax,
				))
			}
		}
	}

	return nil
}

func validateKeyboard(rows [][]messageapi.Key) error {
	for _, row := range rows {
		if len(row) == 0 {
			return invalid("a row of keys is empty")
		}

		for _, key := range row {
			if strings.TrimSpace(key.Label) == "" {
				return invalid("a key has an empty label")
			}
		}
	}

	return nil
}

func markupCount(msg messageapi.Message) int {
	count := 0

	for _, present := range []bool{len(msg.Buttons) > 0, len(msg.Keyboard) > 0, msg.RemoveKeyboard} {
		if present {
			count++
		}
	}

	return count
}

func invalid(reason string) error {
	return fmt.Errorf("%w: %s", messageapi.ErrInvalid, reason)
}

// Markup answers the markup of telego for the message, or nil for none.
//
//nolint:ireturn // the markup field of telego takes the interface.
func Markup(msg messageapi.Message) telego.ReplyMarkup {
	switch {
	case len(msg.Buttons) > 0:
		return InlineMarkup(msg.Buttons)
	case len(msg.Keyboard) > 0:
		return replyKeyboard(msg.Keyboard)
	case msg.RemoveKeyboard:
		return &telego.ReplyKeyboardRemove{RemoveKeyboard: true}
	default:
		return nil
	}
}

// InlineMarkup answers the inline keyboard for the buttons, or nil for none. The value
// of a button is its callback data.
func InlineMarkup(rows [][]messageapi.Button) *telego.InlineKeyboardMarkup {
	if len(rows) == 0 {
		return nil
	}

	keyboard := make([][]telego.InlineKeyboardButton, 0, len(rows))

	for _, row := range rows {
		buttons := make([]telego.InlineKeyboardButton, 0, len(row))
		for _, button := range row {
			buttons = append(buttons, telego.InlineKeyboardButton{
				Text:         button.Label,
				CallbackData: button.Value,
			})
		}

		keyboard = append(keyboard, buttons)
	}

	return &telego.InlineKeyboardMarkup{InlineKeyboard: keyboard}
}

func replyKeyboard(rows [][]messageapi.Key) *telego.ReplyKeyboardMarkup {
	keyboard := make([][]telego.KeyboardButton, 0, len(rows))

	for _, row := range rows {
		keys := make([]telego.KeyboardButton, 0, len(row))
		for _, key := range row {
			keys = append(keys, telego.KeyboardButton{
				Text:            key.Label,
				RequestLocation: key.RequestLocation,
			})
		}

		keyboard = append(keyboard, keys)
	}

	return &telego.ReplyKeyboardMarkup{
		Keyboard:        keyboard,
		ResizeKeyboard:  true,
		OneTimeKeyboard: true,
	}
}
