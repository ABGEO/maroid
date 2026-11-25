package dto

import "github.com/abgeo/maroid/plugins/drivvo/figures"

// ReminderResponse is one reminder with its due point. A due point that the reminder does
// not hold is absent, and a value past due is negative.
type ReminderResponse struct {
	ID           string   `json:"id"`
	Kind         string   `json:"kind"`
	Type         string   `json:"type"`
	Repeats      bool     `json:"repeats"`
	Note         string   `json:"note,omitempty"`
	Due          bool     `json:"due"`
	DueOn        *string  `json:"due_on,omitempty"`
	DueOdometer  *float64 `json:"due_odometer,omitempty"`
	DaysLeft     *int     `json:"days_left,omitempty"`
	DistanceLeft *float64 `json:"distance_left,omitempty"`
}

// NewReminderResponseList converts the reminders in their order.
func NewReminderResponseList(reminders []figures.DueReminder) []ReminderResponse {
	answer := make([]ReminderResponse, 0, len(reminders))

	for _, reminder := range reminders {
		answer = append(answer, ReminderResponse{
			ID:           reminder.ID,
			Kind:         string(reminder.Kind),
			Type:         reminder.Type,
			Repeats:      reminder.Repeats,
			Note:         reminder.Note,
			Due:          reminder.Due,
			DueOn:        dayText(reminder.DueOn),
			DueOdometer:  reminder.DueOdometer,
			DaysLeft:     reminder.DaysLeft,
			DistanceLeft: reminder.DistanceLeft,
		})
	}

	return answer
}
