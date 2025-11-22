package model

// Organization represents an organization entity.
type Organization struct {
	Code string `db:"code"`
	Name string `db:"name"`
}
