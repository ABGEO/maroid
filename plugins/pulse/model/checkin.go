package model

import (
	"database/sql/driver"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/abgeo/maroid/plugins/pulse/errs"
)

const clockLayout = "15:04"

// Clock is a local time of day, with a minute precision.
type Clock struct {
	Hour   int
	Minute int
}

// ParseClock reads a clock in the form HH:MM, from 00:00 to 23:59.
func ParseClock(value string) (Clock, error) {
	moment, err := time.Parse(clockLayout, value)
	if err != nil || len(value) != len(clockLayout) {
		return Clock{}, fmt.Errorf("%w: %q", errs.ErrClockInvalid, value)
	}

	return Clock{Hour: moment.Hour(), Minute: moment.Minute()}, nil
}

// String renders the clock as HH:MM.
func (c Clock) String() string {
	return fmt.Sprintf("%02d:%02d", c.Hour, c.Minute)
}

// Clocks is the list of local times of a check-in. The repository moves it as a
// comma list, because the driver maps no TIME[].
//
//nolint:recvcheck // database/sql calls Scan on a pointer and Value on a value.
type Clocks []Clock

// Scan reads a comma list of HH:MM.
func (c *Clocks) Scan(src any) error {
	parts, err := commaList(src)
	if err != nil {
		return err
	}

	clocks := make(Clocks, 0, len(parts))

	for _, part := range parts {
		clock, parseErr := ParseClock(part)
		if parseErr != nil {
			return parseErr
		}

		clocks = append(clocks, clock)
	}

	*c = clocks

	return nil
}

// Value writes a comma list of HH:MM.
func (c Clocks) Value() (driver.Value, error) {
	parts := make([]string, 0, len(c))
	for _, clock := range c {
		parts = append(parts, clock.String())
	}

	return strings.Join(parts, ","), nil
}

// Weekdays holds ISO weekdays, 1 for Monday. An empty list is every day.
//
//nolint:recvcheck // database/sql calls Scan on a pointer and Value on a value.
type Weekdays []int

// Holds reports whether the list allows the weekday of the day.
func (w Weekdays) Holds(day time.Time) bool {
	if len(w) == 0 {
		return true
	}

	iso := int(day.Weekday())
	if iso == 0 {
		iso = 7
	}

	return slices.Contains(w, iso)
}

// Scan reads a comma list of numbers.
func (w *Weekdays) Scan(src any) error {
	parts, err := commaList(src)
	if err != nil {
		return err
	}

	days := make(Weekdays, 0, len(parts))

	for _, part := range parts {
		day, parseErr := strconv.Atoi(part)
		if parseErr != nil {
			return fmt.Errorf("model: reading a weekday: %w", parseErr)
		}

		days = append(days, day)
	}

	*w = days

	return nil
}

// Value writes a comma list of numbers.
func (w Weekdays) Value() (driver.Value, error) {
	parts := make([]string, 0, len(w))
	for _, day := range w {
		parts = append(parts, strconv.Itoa(day))
	}

	return strings.Join(parts, ","), nil
}

// IDs holds identifiers in an order, moved as a comma list.
//
//nolint:recvcheck // database/sql calls Scan on a pointer and Value on a value.
type IDs []string

// Scan reads a comma list of identifiers.
func (i *IDs) Scan(src any) error {
	parts, err := commaList(src)
	if err != nil {
		return err
	}

	*i = parts

	return nil
}

// Value writes a comma list of identifiers.
func (i IDs) Value() (driver.Value, error) {
	return strings.Join(i, ","), nil
}

// Checkin is a named list of questions with the local times and the weekdays at which
// pulse asks them.
type Checkin struct {
	ID          string    `db:"id"`
	Name        string    `db:"name"`
	LocalTimes  Clocks    `db:"local_times"`
	Weekdays    Weekdays  `db:"weekdays"`
	Enabled     bool      `db:"enabled"`
	FiresAfter  time.Time `db:"fires_after"`
	QuestionIDs IDs       `db:"question_ids"`
	CreatedAt   time.Time `db:"created_at"`
	UpdatedAt   time.Time `db:"updated_at"`
}

func commaList(src any) ([]string, error) {
	var text string

	switch value := src.(type) {
	case nil:
		return []string{}, nil
	case string:
		text = value
	case []byte:
		text = string(value)
	default:
		return nil, fmt.Errorf("%w: %T", errs.ErrListType, src)
	}

	if text == "" {
		return []string{}, nil
	}

	return strings.Split(text, ","), nil
}
