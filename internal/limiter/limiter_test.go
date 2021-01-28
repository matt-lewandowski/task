package limiter_test

import (
	"github.com/matt-lewandowski/task/internal/limiter"
	"github.com/matt-lewandowski/task/internal/limiter/mock"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestNewLimiter(t *testing.T) {
	tests := []struct {
		name string
		rps  int
		clockMock func(c *mock.MockClock)
		validator func(t *testing.T, allowance int)
	}{
		{
			name: "create a new limiter with no records",
			rps: 10,
			clockMock: func(c *mock.MockClock) {
				c.On("Now").Return(time.Now())
			},
			validator: func(t *testing.T, allowance int) {
				assert.Equal(t, 10, allowance)
			},
		},
	}
	for _, test := range tests {
		clock := mock.MockClock{}
		if test.clockMock != nil {
			test.clockMock(&clock)
		}
		l := limiter.NewLimiter(limiter.Config{
			RPS:   test.rps,
			Clock: &clock,
		})
		allowance := <- l.JobsChannel()
		test.validator(t, allowance)
	}
}
