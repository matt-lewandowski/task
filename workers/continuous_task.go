package workers

import (
	"fmt"
	"github.com/matt-lewandowski/task/internal/limiter"
	"github.com/matt-lewandowski/task/internal/limiter/clock"
	"github.com/matt-lewandowski/task/internal/safe"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

type cTask struct {
	workers         safe.ResourceManager
	limiter         limiter.Limiter
	jobs            chan interface{}
	handlerFunction func(interface{}) (interface{}, error)
	errorHandler    func(data JobData, stop func())
	resultHandler   func(data JobData)
	errorChannel    chan JobData
	resultsChannel  chan JobData
	stop            chan os.Signal
	abort           bool
	clock           clock.Clock
}

// ContinuousTaskConfig is the struct for creating a new task runner that will create and handle jobs
// that come from provided jobs channel until the channel is closed or the task runner is stopped
type ContinuousTaskConfig struct {
	// Workers is the desired amount of concurrent processes. Each worker will consume a job which will
	// use the handler function to process it.
	Workers int

	// RateLimit is the desired RPS for all of the workers combined. For example, if there are 100 works, and
	// the rateLimit is set to 15, only 15 workers will start within every second. This can be used if a worker is expected to
	// take longer then 1 second, and you do not want to send all of them off at once.
	RateLimit int

	// Jobs is an interface channel that will be used to create jobs from. The task will not end if the channel is empty,
	// but will end if the channel is closed and the channel is empty.
	Jobs chan interface{}

	// The HandlerFunction will be the function that will be passed a job to handle. If it errors, the error will be passed
	// through the error channel.
	HandlerFunction func(interface{}) (interface{}, error)

	// The ErrorHandler is a function which will receive an error, and a function that can be used to stop job. If stop is called,
	// Any jobs that are already in progress will finish. An error channel is used to return data to the error handler, so that
	// workers do not need to wait for the result to be handled before moving on to the next job
	ErrorHandler func(data JobData, stop func())

	// The ResultHandler is a function that will receive the results from the handler function. A results channel is used to return data to the
	// result handler, so that workers do not need to wait for the result to be handled before moving on to the next job
	ResultHandler func(data JobData)
}

// NewContinuousTask will return a ContinuousTask which will process jobs concurrently with the provided handler function
func NewContinuousTask(ct ContinuousTaskConfig) Task {
	l := limiter.NewLimiter(limiter.Config{
		RPS:   ct.RateLimit,
		Clock: clock.NewClock(),
	})
	s := make(chan os.Signal)
	rc := make(chan JobData)
	ec := make(chan JobData)
	pc := safe.NewResourceManager(ct.Workers, 0)
	clk := clock.NewClock()
	signal.Notify(s, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	wg := cTask{
		workers:         pc,
		limiter:         l,
		handlerFunction: ct.HandlerFunction,
		errorChannel:    ec,
		resultsChannel:  rc,
		stop:            s,
		clock:           clk,
		errorHandler:    ct.ErrorHandler,
		resultHandler:   ct.ResultHandler,
		jobs:            ct.Jobs,
		abort:           false,
	}
	return &wg
}

// Stop will stop creating new jobs and wait for any jobs in progress to finish
func (w *cTask) Stop() {
	w.stop <- os.Interrupt
}

// Start will begin processing the jobs provided
func (w *cTask) Start() {
	flushGroup := sync.WaitGroup{}
	done := make(chan bool)
	go func() {
		w.start(&flushGroup)
		done <- true
	}()
	select {
	case <-done:
		close(w.resultsChannel)
		close(w.errorChannel)
	}
	flushGroup.Wait()
	w.limiter.Stop()
	fmt.Println("I'm done")
}

func (w *cTask) start(flushGroup *sync.WaitGroup) {
	flushGroup.Add(1)
	go errorHandler(flushGroup, w.errorChannel, w.Stop, w.errorHandler)
	flushGroup.Add(1)
	go resultHandler(flushGroup, w.resultsChannel, w.resultHandler)

	waitGroup := sync.WaitGroup{}
	waitGroup.Add(1)

	go w.work(&waitGroup)
	workersDone := make(chan bool)
	go func() {
		waitGroup.Wait()
		workersDone <- true
	}()
	// Now we wait
	for {
		select {
		case <-w.stop:
			w.abort = true
		case <-workersDone:
			return
		}
	}
}

func (w *cTask) work(waitGroup *sync.WaitGroup) {
	for !w.abort {
		numberOfWorkers := <-w.limiter.WorkAvailable()
		for i := 0; i < numberOfWorkers; i++ {
			if w.workers.GetAvailableWorkers() > 0 && !w.abort {
				w.workers.TakeJob()
				waitGroup.Add(1)
				w.limiter.Record(w.clock.Now())
				go w.receiveJob(waitGroup, w.workers.GetAllJobsAccepted())
			}
		}
	}
	w.Stop()
	waitGroup.Done()
}

func (w *cTask) receiveJob(waitGroup *sync.WaitGroup, count int) {
	select {
	case job, open := <-w.jobs:
		if !open {
			w.abort = true
			w.workers.AbortedJob()
		} else {
			if job != nil {
				result, err := w.handlerFunction(job)
				if err != nil {
					w.errorChannel <- JobData{
						JobValue: job,
						Error:    err,
						Count:    count,
					}
				}
				if result != nil {
					w.resultsChannel <- JobData{
						JobValue: job,
						Result:   result,
						Count:    count,
					}
				}
				w.workers.FinishJob()
			} else {
				w.workers.AbortedJob()
			}
		}
	default:
		w.workers.AbortedJob()
	}
	waitGroup.Done()
}
