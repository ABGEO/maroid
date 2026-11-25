package model

// Snapshot is one read of the whole car log account. A collection stores it in one
// transaction, or stores nothing.
type Snapshot struct {
	Settings       CarLogSettings
	ReferenceItems []ReferenceItem
	Vehicles       []Vehicle
	LogEntries     []LogEntry
	Reminders      []Reminder
}
