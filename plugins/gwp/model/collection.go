package model

import "time"

// Collection is the outcome of one collection of the readings of the assigned customer.
type Collection struct {
	// Count is the count of the readings that the water service reported for the customer.
	Count          int
	CustomerNumber string
	// NewestDate is the latest date of a reading that the collection stored for the
	// first time, or nil when it stored none.
	NewestDate *time.Time
}
