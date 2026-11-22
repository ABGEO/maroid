package model

import "time"

// TelegramMessage is the record of one plugin message: the plugin that sent it, the
// message that Telegram gave it, and the reply claim on it.
type TelegramMessage struct {
	ID         string     `db:"id"`
	PluginID   string     `db:"plugin_id"`
	ChatID     int64      `db:"chat_id"`
	MessageID  int64      `db:"message_id"`
	ClaimUntil *time.Time `db:"claim_until"`
	// ClaimOpen reports whether the claim holds at the moment of the read, by the
	// clock of the database.
	ClaimOpen bool      `db:"claim_open"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}
