package pulsetest

import (
	"sync"
	"time"
)

// Clock is a clock that a test sets.
type Clock struct {
	mu  sync.Mutex
	now time.Time
}

// NewClock answers a clock at the moment.
func NewClock(moment time.Time) *Clock {
	return &Clock{now: moment}
}

// Now answers the moment of the clock.
func (c *Clock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.now
}

// Set moves the clock to the moment.
func (c *Clock) Set(moment time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.now = moment
}

// At answers the moment of the local time of Nina on the day of November 2026.
func At(day int, hour int, minute int) time.Time {
	return time.Date(2026, time.November, day, hour, minute, 0, 0, Tbilisi())
}
