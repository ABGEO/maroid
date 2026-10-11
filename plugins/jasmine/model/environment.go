package model

import "time"

// Environment represents a place in a workspace where plants live.
type Environment struct {
	ID               string    `db:"id"`
	Number           int       `db:"number"`
	Name             string    `db:"name"`
	ActivePlantCount int       `db:"active_plant_count"`
	CreatedAt        time.Time `db:"created_at"`
	UpdatedAt        time.Time `db:"updated_at"`
}

// DisplayID answers the identifier that a person reads for the environment.
func (e *Environment) DisplayID() string {
	return FormatDisplayID(PrefixEnvironment, e.Number)
}

// EnvironmentReference names the environment of another record.
type EnvironmentReference struct {
	ID     string `db:"id"`
	Number int    `db:"number"`
	Name   string `db:"name"`
}

// DisplayID answers the identifier that a person reads for the environment.
func (e *EnvironmentReference) DisplayID() string {
	return FormatDisplayID(PrefixEnvironment, e.Number)
}
