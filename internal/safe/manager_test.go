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
		validator    func(t *testing.T, availableWorkers int, totalJobsAccepted int, jobsToDo int, working int)
	}{
		{
			name:         "should create a progress counter with a zero value",
			initialValue: 0,
			amountOfJobs: 0,
			validator: func(t *testing.T, availableWorkers int, totalJobsAccepted int, jobsToDo int, working int) {
				assert.Equal(t, availableWorkers, 0)
				assert.Equal(t, totalJobsAccepted, 0)
				assert.Equal(t, jobsToDo, 0)
				assert.Equal(t, working, 0)
			},
		},
		{
			name:         "should increase worker Count by 10",
			initialValue: 0,
			amountOfJobs: 10,
			useCounter: func(p safe.ResourceManager) {
				for i := 0; i < 10; i++ {
					p.FinishJob()
				}
			},
			validator: func(t *testing.T, availableWorkers int, totalJobsAccepted int, jobsToDo int, working int) {
				assert.Equal(t, availableWorkers, 10)
				assert.Equal(t, totalJobsAccepted, 0)
				assert.Equal(t, jobsToDo, 10)
				assert.Equal(t, working, 10)
			},
		},
		{
			name:         "should abort the same amount of jobs that were taken",
			initialValue: 0,
			amountOfJobs: 10,
			useCounter: func(p safe.ResourceManager) {
				for i := 0; i < 10; i++ {
					p.TakeJob()
				}
				for i := 0; i < 10; i++ {
					p.AbortedJob()
				}
			},
			validator: func(t *testing.T, availableWorkers int, totalJobsAccepted int, jobsToDo int, working int) {
				assert.Equal(t, availableWorkers, 0)
				assert.Equal(t, totalJobsAccepted, 0)
				assert.Equal(t, jobsToDo, 10)
				assert.Equal(t, working, 0)
			},
		},
		{
			name:         "should decrease workerCount down by 10",
			initialValue: 10,
			amountOfJobs: 10,
			useCounter: func(p safe.ResourceManager) {
				for i := 0; i < 10; i++ {
					p.TakeJob()
				}
			},
			validator: func(t *testing.T, availableWorkers int, totalJobsAccepted int, jobsToDo int, working int) {
				assert.Equal(t, availableWorkers, 0)
				assert.Equal(t, totalJobsAccepted, 10)
				assert.Equal(t, jobsToDo, 0)
				assert.Equal(t, working, -10)
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
			validator: func(t *testing.T, availableWorkers int, totalJobsAccepted int, jobsToDo int, working int) {
				assert.Equal(t, availableWorkers, 0)
				assert.Equal(t, totalJobsAccepted, 10)
				assert.Equal(t, jobsToDo, 0)
				assert.Equal(t, working, -10)
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
			validator: func(t *testing.T, availableWorkers int, totalJobsAccepted int, jobsToDo int, working int) {
				assert.Equal(t, availableWorkers, 0)
				assert.Equal(t, totalJobsAccepted, 0)
				assert.Equal(t, jobsToDo, 10)
				assert.Equal(t, working, 0)
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			p := safe.NewResourceManager(test.initialValue, test.amountOfJobs)
			if test.useCounter != nil {
				test.useCounter(p)
			}
			availableWorkers := p.GetAvailableWorkers()
			totalJobsAccepted := p.GetAllJobsAccepted()
			jobsToDo := p.GetJobsToDo()
			workersWorking := p.GetWorkersWorking()
			test.validator(t, availableWorkers, totalJobsAccepted, jobsToDo, workersWorking)
		})
	}
}
