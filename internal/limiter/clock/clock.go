package clock

import "time"

// Clock will return a wrapper for the built in Time interface
type Clock interface {
	Now() time.Time
}

// NewClock returns a Clock interface
func NewClock() Clock {
	return &clock{}
}

type clock struct{}

// Now will return the current time
func (c *clock) Now() time.Time {
	return time.Now()
}
