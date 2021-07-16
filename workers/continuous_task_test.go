package workers

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"
)

func loadChannel(job interface{}, jobChannel chan interface{}, size int) chan interface{} {
	for i := 0; i < size; i++ {
		jobChannel <- fmt.Sprintf("%v-%v", job, i+1)
	}
	return jobChannel
}

func loadNil(job interface{}, jobChannel chan interface{}, size int) chan interface{} {
	for i := 0; i < size; i++ {
		jobChannel <- job
	}
	return jobChannel
}

func TestNewContinuousTask(t *testing.T) {
	channelSize := 9

	t.Parallel()
	tests := []struct {
		name            string
		jobs            chan interface{}
		workers         int
		rateLimit       int
		handlerFunction func(ctx context.Context, v interface{}) (interface{}, error)
		errorHandler    func(data JobData, stop func())
		resultHandler   func(data JobData)
	}{
		{
			name:      "create a job with nil values",
			jobs:      loadNil(nil, make(chan interface{}, channelSize), channelSize),
			workers:   10,
			rateLimit: 10,
			handlerFunction: func(ctx context.Context, i interface{}) (interface{}, error) {
				return nil, nil
			},
		},
		{
			name:      "return results to the result function",
			jobs:      loadChannel("result", make(chan interface{}, channelSize), channelSize),
			workers:   10,
			rateLimit: 10,
			handlerFunction: func(ctx context.Context, i interface{}) (interface{}, error) {
				return i, nil
			},
			resultHandler: func(data JobData) {
				return
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			finish := make(chan bool, 1)
			jobsChannel := test.jobs
			continousTask := NewContinuousTask(ContinuousTaskConfig{
				Workers:         test.workers,
				RateLimit:       test.rateLimit,
				Jobs:            jobsChannel,
				HandlerFunction: test.handlerFunction,
				ErrorHandler:    test.errorHandler,
				ResultHandler:   test.resultHandler,
				BufferSize:      100,
			})
			close(jobsChannel)
			go func() {
				continousTask.Start(context.Background())
				finish <- true
			}()

			_ = <-finish
			continousTask.Stop()
		})
	}
}

func TestNewContinuousTaskStopping(t *testing.T) {

	t.Parallel()
	tests := []struct {
		name            string
		jobs            chan interface{}
		workers         int
		rateLimit       int
		handlerFunction func(ctx context.Context, v interface{}) (interface{}, error)
		errorHandler    func(data JobData, stop func())
		resultHandler   func(data JobData)
	}{
		{
			name:      "stop the job from the error handler",
			jobs:      loadChannel("stop", make(chan interface{}, 10), 10),
			workers:   10,
			rateLimit: 2,
			handlerFunction: func(ctx context.Context, i interface{}) (interface{}, error) {
				return nil, fmt.Errorf(i.(string))
			},
			errorHandler: func(data JobData, stop func()) {
				if data.Error != nil {
					stop()
				}
			},
		},
		{
			name:      "stop the job from the error handler with a lot of jobs",
			jobs:      loadChannel("stop", make(chan interface{}, 10000), 10000),
			workers:   1000,
			rateLimit: 1000,
			handlerFunction: func(ctx context.Context, i interface{}) (interface{}, error) {
				return nil, fmt.Errorf(i.(string))
			},
			errorHandler: func(data JobData, stop func()) {
				if data.Error != nil {
					stop()
				}
			},
		},
		{
			name:      "stop the job from the error handler, while jobs are being created",
			jobs:      loadChannel("stop", make(chan interface{}, 10000), 10000),
			workers:   4000,
			rateLimit: 4000,
			handlerFunction: func(ctx context.Context, i interface{}) (interface{}, error) {
				return nil, fmt.Errorf(i.(string))
			},
			errorHandler: func(data JobData, stop func()) {
				if strings.Contains(data.Error.Error(), "stop-5000") {
					stop()
				}
			},
		},
		{
			name:      "stop the job when it completes",
			jobs:      loadChannel("close", make(chan interface{}, 10), 10),
			workers:   10,
			rateLimit: 100,
			handlerFunction: func(ctx context.Context, i interface{}) (interface{}, error) {
				return nil, fmt.Errorf(i.(string))
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			abort := make(chan bool, 1)
			continousTask := NewContinuousTask(ContinuousTaskConfig{
				Workers:         test.workers,
				RateLimit:       test.rateLimit,
				Jobs:            test.jobs,
				HandlerFunction: test.handlerFunction,
				ErrorHandler:    test.errorHandler,
				ResultHandler:   test.resultHandler,
				BufferSize:      100,
			})
			close(test.jobs)
			go func() {
				continousTask.Start(context.Background())
				abort <- false
			}()
			go func() {
				time.Sleep(time.Second * 5)
				abort <- true
			}()

			aborted := <-abort
			if aborted {
				t.Error("ERROR: Test did not finish in time. Test aborted")
				continousTask.Stop()
			}
		})
	}
}
