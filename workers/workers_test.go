package workers

import (
	"fmt"
	"testing"
	"time"
)

// createJobs is a helper function for testing different job types
func createJobs(job interface{}) []interface{} {
	jobs := make([]interface{}, 10)
	for i := 0; i < 10; i++ {
		jobs[i] = job
	}
	return jobs
}

func TestNewTask(t *testing.T) {
	tests := []struct {
		name            string
		jobs            []interface{}
		workers         int
		rateLimit       int
		handlerFunction func(interface{}) (interface{}, error)
	}{
		{
			name:      "stop the job from the error handler",
			jobs:      createJobs("Cancel Me"),
			workers:   20,
			rateLimit: 100,
			handlerFunction: func(i interface{}) (interface{}, error) {
				return i, fmt.Errorf(i.(string))
			},
		},
		{
			name:      "create a job of nil values",
			jobs:      createJobs(nil),
			workers:   20,
			rateLimit: 100,
			handlerFunction: func(i interface{}) (interface{}, error) {
				return i, nil
			},
		},
		{
			name:      "sends errors to error function",
			jobs:      createJobs("Test Error"),
			workers:   20,
			rateLimit: 100,
			handlerFunction: func(i interface{}) (interface{}, error) {
				return i, fmt.Errorf(i.(string))
			},
		},
		{
			name:      "create a job of strings",
			jobs:      createJobs("string"),
			workers:   20,
			rateLimit: 100,
			handlerFunction: func(i interface{}) (interface{}, error) {
				return i, nil
			},
		},
		{
			name:      "create a job of ints",
			jobs:      createJobs(10),
			workers:   20,
			rateLimit: 100,
			handlerFunction: func(i interface{}) (interface{}, error) {
				return i, nil
			},
		},
		{
			name:      "create a job of runes",
			jobs:      createJobs('a'),
			workers:   20,
			rateLimit: 100,
			handlerFunction: func(i interface{}) (interface{}, error) {
				return i, nil
			},
		},
		{
			name:      "create a job of slices",
			jobs:      createJobs([]string{"string", "string"}),
			workers:   20,
			rateLimit: 100,
			handlerFunction: func(i interface{}) (interface{}, error) {
				return i, nil
			},
		},
		{
			name:      "create a job of channels",
			jobs:      createJobs([]chan bool{}),
			workers:   20,
			rateLimit: 100,
			handlerFunction: func(i interface{}) (interface{}, error) {
				return i, nil
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			abort := make(chan bool)
			resultFunction := func(data JobData) {
				if data.Count == 10 {
					abort <- false
				}
			}
			errorFunction := func(data JobData, stop func()) {
				if data.Error.Error() == "Cancel Me" {
					stop()
					return
				}
				if data.Error.Error() == "Test Error" {
					return
				}
				if data.Error != nil {
					t.Fatalf("ERROR: %v", data.Error)
				}
			}
			worker := NewTask(Config{
				Workers:         test.workers,
				RateLimit:       test.rateLimit,
				Jobs:            test.jobs,
				HandlerFunction: test.handlerFunction,
				ErrorHandler:    errorFunction,
				ResultHandler:   resultFunction,
			})

			go func() {
				time.Sleep(time.Second * 1)
				abort <- true
			}()
			go func() {
				worker.Start()
				abort <- false
			}()

			aborted := <-abort

			if aborted {
				t.Error("ERROR: Test did not finish in time. Test aborted")
				worker.Stop()
			}
		})
	}
}

func TestTask_Stop(t *testing.T) {
	tests := []struct {
		name            string
		jobs            []interface{}
		workers         int
		rateLimit       int
		handlerFunction func(interface{}) (interface{}, error)
		errorFunction   func(data JobData, stop func())
	}{
		{
			name:            "The job will stop properly",
			jobs:            createJobs("Cancel Me"),
			workers:         20,
			rateLimit:       1,
			handlerFunction: func(i interface{}) (interface{}, error) { return nil, nil },
			errorFunction:   func(data JobData, stop func()) {},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			abort := make(chan bool)
			resultFunction := func(data JobData) {
				if data.Count == 2 {
					abort <- true
				}
			}
			errorFunction := func(data JobData, stop func()) {}
			worker := NewTask(Config{
				Workers:         test.workers,
				RateLimit:       test.rateLimit,
				Jobs:            test.jobs,
				HandlerFunction: test.handlerFunction,
				ErrorHandler:    errorFunction,
				ResultHandler:   resultFunction,
			})

			go func() {
				time.Sleep(time.Second * 2)
				abort <- true
			}()
			go func() {
				worker.Start()
				abort <- false
			}()
			worker.Stop()

			aborted := <-abort

			if aborted {
				t.Error("ERROR: Test did not finish in time. Test aborted")
				worker.Stop()
			}
		})
	}
}
