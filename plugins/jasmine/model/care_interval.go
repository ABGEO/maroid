package model

import (
	"fmt"
	"time"

	"github.com/abgeo/maroid/plugins/jasmine/errs"
)

// CareState tells how a care interval of an active plant stands.
type CareState string

const (
	// CareStateOnSchedule marks fewer days since the latest entry than the interval.
	CareStateOnSchedule CareState = "on_schedule"
	// CareStateDue marks from the interval to 1.5 times the interval.
	CareStateDue CareState = "due"
	// CareStateOverdue marks more than 1.5 times the interval.
	CareStateOverdue CareState = "overdue"
)

// ParseCareState converts a string to a CareState, returning an error for an
// unknown value.
func ParseCareState(s string) (CareState, error) {
	switch CareState(s) {
	case CareStateOnSchedule, CareStateDue, CareStateOverdue:
		return CareState(s), nil
	default:
		return "", fmt.Errorf("%w: %q", errs.ErrUnknownCareState, s)
	}
}

// CareInterval holds the days between two care actions of one maintenance type
// for one plant, with the count of days that the read computed.
type CareInterval struct {
	ID              string          `db:"id"`
	Plant           PlantReference  `db:"plant"`
	Type            MaintenanceType `db:"type"`
	Days            int             `db:"days"`
	SetAt           time.Time       `db:"set_at"`
	LastPerformedAt *time.Time      `db:"last_performed_at"`
	CountedFrom     time.Time       `db:"counted_from"`
	DaysSince       int             `db:"days_since"`
	DueIn           int             `db:"due_in"`
	State           *CareState      `db:"state"`
	UpdatedAt       time.Time       `db:"updated_at"`
}
