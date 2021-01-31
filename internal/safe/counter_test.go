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
		amountOfJobs int
		useCounter   func(p safe.ResourceManager)
		validator    func(t *testing.T, count int, total int, todo int)
	}{
		{
			name:         "should create a progress counter with a zero value",
			initialValue: 0,
			amountOfJobs: 0,
			validator: func(t *testing.T, count int, total int, todo int) {
				assert.Equal(t, count, 0)
				assert.Equal(t, total, 0)
				assert.Equal(t, total, 0)
			},
		},
		{
			name:         "should workerCount up by 10",
			initialValue: 0,
			amountOfJobs: 10,
			useCounter: func(p safe.ResourceManager) {
				for i := 0; i < 10; i++ {
					p.FinishJob()
				}
			},
			validator: func(t *testing.T, count int, total int, todo int) {
				assert.Equal(t, count, 10)
				assert.Equal(t, total, 10)
				assert.Equal(t, todo, 10)
			},
		},
		{
			name:         "should workerCount down by 10",
			initialValue: 10,
			amountOfJobs: 10,
			useCounter: func(p safe.ResourceManager) {
				for i := 0; i < 10; i++ {
					p.TakeJob()
				}
			},
			validator: func(t *testing.T, count int, total int, todo int) {
				assert.Equal(t, count, 0)
				assert.Equal(t, total, 0)
				assert.Equal(t, todo, 0)
			},
		},
		{
			name:         "should have a different TotalIncrements and workerCount",
			initialValue: 0,
			amountOfJobs: 10,
			useCounter: func(p safe.ResourceManager) {
				for i := 0; i < 10; i++ {
					p.FinishJob()
				}
				for i := 0; i < 10; i++ {
					p.TakeJob()
				}
			},
			validator: func(t *testing.T, count int, total int, todo int) {
				assert.Equal(t, count, 0)
				assert.Equal(t, total, 10)
				assert.Equal(t, todo, 0)
			},
		},
		{
			name:         "should reset the workerCount",
			initialValue: 10,
			amountOfJobs: 10,
			useCounter: func(p safe.ResourceManager) {
				for i := 0; i < 10; i++ {
					p.FinishJob()
				}
				p.Reset()
			},
			validator: func(t *testing.T, count int, total int, todo int) {
				assert.Equal(t, count, 0)
				assert.Equal(t, total, 10)
				assert.Equal(t, todo, 10)
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			p := safe.NewResourceManager(test.initialValue, test.amountOfJobs)
			if test.useCounter != nil {
				test.useCounter(p)
			}
			value := p.GetAvailableWorkers()
			total := p.GetAllJobsAccepted()
			jobsLeft := p.GetJobsToDo()
			test.validator(t, value, total, jobsLeft)
		})
	}
}
