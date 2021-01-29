package limiter_test

import (
	"github.com/matt-lewandowski/task/internal/limiter"
	"github.com/matt-lewandowski/task/internal/limiter/mock"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestNewLimiter(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		rps        int
		timestamps []time.Time
		clockMock  func(c *mock.Clock)
		validator  func(t *testing.T, l limiter.Limiter)
	}{
		{
			name: "create a new limiter with no records",
			rps:  10,
			clockMock: func(c *mock.Clock) {
				c.On("Now").Return(time.Now())
			},
			validator: func(t *testing.T, l limiter.Limiter) {
				allowance := <-l.SlotsAvailable()
				assert.Equal(t, 10, allowance)
			},
		},
		{
			name: "should have 5 slots",
			rps:  10,
			timestamps: []time.Time{
				time.Now().Add(1 * time.Hour),
				time.Now().Add(1 * time.Hour),
				time.Now().Add(1 * time.Hour),
				time.Now().Add(1 * time.Hour),
				time.Now().Add(1 * time.Hour),
			},
			clockMock: func(c *mock.Clock) {
				c.On("Now").Return(time.Now())
			},
			validator: func(t *testing.T, l limiter.Limiter) {
				allowance := <-l.SlotsAvailable()
				assert.Equal(t, 5, allowance)
			},
		},
		{
			name: "should have 0 slots available",
			rps:  5,
			timestamps: []time.Time{
				time.Now().Add(1 * time.Hour),
				time.Now().Add(1 * time.Hour),
				time.Now().Add(1 * time.Hour),
				time.Now().Add(1 * time.Hour),
				time.Now().Add(1 * time.Hour),
			},
			clockMock: func(c *mock.Clock) {
				c.On("Now").Return(time.Now())
			},
			validator: func(t *testing.T, l limiter.Limiter) {
				select {
				case <-l.SlotsAvailable():
					t.Error("limiter should not return when there are no slots available")
				default:
					return
				}
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			clock := mock.Clock{}
			if test.clockMock != nil {
				test.clockMock(&clock)
			}
			l := limiter.NewLimiter(limiter.Config{
				RPS:   test.rps,
				Clock: &clock,
			})
			for _, entry := range test.timestamps {
				l.Record(entry)
			}
			test.validator(t, l)
			l.Stop()
		})
	}
}
