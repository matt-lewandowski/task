package safe_test

import (
	"github.com/matt-lewandowski/task/internal/safe"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewProgressCounter(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		initialValue int
		useCounter   func(p safe.ProgressCounter)
		validator    func(t *testing.T, count int, total int)
	}{
		{
			name:         "should create a progress counter with a zero value",
			initialValue: 0,
			validator: func(t *testing.T, count int, total int) {
				assert.Equal(t, count, 0)
				assert.Equal(t, total, 0)
			},
		},
		{
			name:         "should count up by 10",
			initialValue: 0,
			useCounter: func(p safe.ProgressCounter) {
				for i := 0; i < 10; i++ {
					p.Increment()
				}
			},
			validator: func(t *testing.T, count int, total int) {
				assert.Equal(t, count, 10)
				assert.Equal(t, total, 10)
			},
		},
		{
			name:         "should count down by 10",
			initialValue: 10,
			useCounter: func(p safe.ProgressCounter) {
				for i := 0; i < 10; i++ {
					p.Decrement()
				}
			},
			validator: func(t *testing.T, count int, total int) {
				assert.Equal(t, count, 0)
				assert.Equal(t, total, 0)
			},
		},
		{
			name:         "should have a different TotalIncrements and count",
			initialValue: 0,
			useCounter: func(p safe.ProgressCounter) {
				for i := 0; i < 10; i++ {
					p.Increment()
				}
				for i := 0; i < 10; i++ {
					p.Decrement()
				}
			},
			validator: func(t *testing.T, count int, total int) {
				assert.Equal(t, count, 0)
				assert.Equal(t, total, 10)
			},
		},
		{
			name:         "should reset the count",
			initialValue: 10,
			useCounter: func(p safe.ProgressCounter) {
				for i := 0; i < 10; i++ {
					p.Increment()
				}
				p.Reset()
			},
			validator: func(t *testing.T, count int, total int) {
				assert.Equal(t, count, 0)
				assert.Equal(t, total, 10)
			},
		},
	}
	for _, test := range tests {
		p := safe.NewProgressCounter(test.initialValue)
		if test.useCounter != nil {
			test.useCounter(p)
		}
		value := p.GetCount()
		total := p.GetIncrementTotal()
		test.validator(t, value, total)
	}
}
