package model

// Employer is an employer that the pension service names on a contribution.
type Employer struct {
	Code string `db:"code"`
	Name string `db:"name"`
}
