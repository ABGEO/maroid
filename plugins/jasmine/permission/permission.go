// Package permission names the permissions that the entries of jasmine name.
package permission

const (
	// EnvironmentsRead reads the environments.
	EnvironmentsRead = "environments.read"
	// EnvironmentsWrite creates, renames, and deletes the environments.
	EnvironmentsWrite = "environments.write"
	// PlantsRead reads the plants.
	PlantsRead = "plants.read"
	// PlantsWrite creates, changes, and deletes the plants.
	PlantsWrite = "plants.write"
	// MaintenanceRead reads the maintenance entries.
	MaintenanceRead = "maintenance.read"
	// MaintenanceWrite records, changes, and deletes the maintenance entries.
	MaintenanceWrite = "maintenance.write"
)
