package model

import (
	"time"

	"github.com/google/uuid"
)

// Contribution represents a pension contribution record.
type Contribution struct {
	Hash             string        `db:"hash"`
	BasisID          uuid.UUID     `db:"basis_id"`
	Date             time.Time     `db:"date"`
	ClosingDate      *time.Time    `db:"closing_date"`
	Year             *int          `db:"year"`
	Month            *int          `db:"month"`
	GrossSalary      float64       `db:"gross_salary"`
	Type             string        `db:"type"`
	Source           string        `db:"source"`
	Amount           *float64      `db:"amount"`
	Units            *float64      `db:"units"`
	OrganizationCode *string       `db:"organization_code"`
	Organization     *Organization `db:"-"`
}
