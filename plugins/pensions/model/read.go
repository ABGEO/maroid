package model

import "time"

// SummaryView is what the pages and the widget read first: whether Maroid holds a
// session of the acting user, and the stored summary when a collection wrote one.
type SummaryView struct {
	Connected bool
	Summary   *AccountSummary
}

// ContributionFilter narrows the contributions of the acting user. A nil member keeps
// every contribution. From and To are dates, each one included.
type ContributionFilter struct {
	EmployerCode *string
	From         *time.Time
	To           *time.Time
}

// ContributionTotal is the sum and the count of the contributions that a filter matches.
type ContributionTotal struct {
	Amount float64 `db:"amount"`
	Count  int     `db:"count"`
}

// MonthSum is the sum of the contributions paid in one month.
type MonthSum struct {
	// Month is the first day of the month.
	Month  time.Time `db:"month"`
	Amount float64   `db:"amount"`
}
