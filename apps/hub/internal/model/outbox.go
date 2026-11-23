package model

import (
	"encoding/json"
	"time"
)

// OutboxScope names the kind of the owner of a task of the outbox.
type OutboxScope string

// The kinds of an owner.
const (
	OutboxScopeWorkspace OutboxScope = "workspace"
	OutboxScopeUser      OutboxScope = "user"
)

// OutboxOwner is the workspace or the user that a task acts for.
type OutboxOwner struct {
	Scope OutboxScope
	ID    string
}

// OutboxTask is one task of the outbox. The worker runs it in the scope of its owner,
// when it names one.
type OutboxTask struct {
	ID          string          `db:"id"`
	Type        string          `db:"type"`
	Payload     json.RawMessage `db:"payload"`
	OwnerScope  *OutboxScope    `db:"owner_scope"`
	OwnerID     *string         `db:"owner_id"`
	DedupeKey   *string         `db:"dedupe_key"`
	RunAfter    time.Time       `db:"run_after"`
	LockedUntil *time.Time      `db:"locked_until"`
	Attempts    int             `db:"attempts"`
	LastError   *string         `db:"last_error"`
	CreatedAt   time.Time       `db:"created_at"`
}

// Owner answers the owner of the task, and false for a task that acts for nobody.
func (t OutboxTask) Owner() (OutboxOwner, bool) {
	if t.OwnerScope == nil || t.OwnerID == nil {
		return OutboxOwner{}, false
	}

	return OutboxOwner{Scope: *t.OwnerScope, ID: *t.OwnerID}, true
}

// OutboxDraft holds a task to enqueue. Delay holds the time from the commit to the
// first run. A draft whose DedupeKey a waiting task holds adds no task.
type OutboxDraft struct {
	Type      string
	Payload   any
	Owner     *OutboxOwner
	DedupeKey string
	Delay     time.Duration
}
