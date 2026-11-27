package model

import (
	"time"

	"github.com/google/uuid"
)

// Contribution is one payment to the pension account of a person: one source of one basis.
type Contribution struct {
	ID               uuid.UUID  `db:"id"`
	BasisID          uuid.UUID  `db:"basis_id"`
	Source           string     `db:"source"`
	ContributionType string     `db:"contribution_type"`
	PaidOn           time.Time  `db:"paid_on"`
	ClosedOn         *time.Time `db:"closed_on"`
	SalaryYear       *int       `db:"salary_year"`
	SalaryMonth      *int       `db:"salary_month"`
	GrossSalary      float64    `db:"gross_salary"`
	Amount           *float64   `db:"amount"`
	Units            *float64   `db:"units"`
	EmployerCode     *string    `db:"employer_code"`
	// Employer is nil when the contribution names no employer. The store writes it to
	// the shared employers before the contribution.
	Employer *Employer `db:"-"`
}
