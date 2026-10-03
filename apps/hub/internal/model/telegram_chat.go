package model

import "time"

// TelegramChat is the state of one chat of one person: the workspace it acts in.
type TelegramChat struct {
	ID                  string    `db:"id"`
	ChatID              int64     `db:"chat_id"`
	SelectedWorkspaceID *string   `db:"selected_workspace_id"`
	CreatedAt           time.Time `db:"created_at"`
	UpdatedAt           time.Time `db:"updated_at"`
}
