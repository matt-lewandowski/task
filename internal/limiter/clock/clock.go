package clock

import "time"

type Clock interface {
	Now() time.Time
}

func NewClock() Clock {
	return &clock{}
}

type clock struct{}

func (c *clock) Now() time.Time {
	return time.Now()
}
