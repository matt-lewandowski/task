package workers

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/matt-lewandowski/task/internal/limiter"
	"github.com/matt-lewandowski/task/internal/limiter/clock"
	"github.com/matt-lewandowski/task/internal/safe"
)

type cTask struct {
	workers         safe.ResourceManager
	limiter         limiter.Limiter
	jobs            chan interface{}
	handlerFunction func(ctx context.Context, v interface{}) (interface{}, error)
	errorHandler    func(data JobData, stop func())
	resultHandler   func(data JobData)
	errorChannel    chan JobData
	resultsChannel  chan JobData
	ctx             context.Context
	cancelCtx       context.CancelFunc
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
	// through the error channel. The handler function is passed a cancel context
	HandlerFunction func(ctx context.Context, v interface{}) (interface{}, error)

	// The ErrorHandler is a function which will receive an error, and a function that can be used to stop job. If stop is called,
	// Any jobs that are already in progress will finish. An error channel is used to return data to the error handler, so that
	// workers do not need to wait for the result to be handled before moving on to the next job
	ErrorHandler func(data JobData, stop func())

	// The ResultHandler is a function that will receive the results from the handler function. A results channel is used to return data to the
	// result handler, so that workers do not need to wait for the result to be handled before moving on to the next job
	ResultHandler func(data JobData)

	// The BufferSize is the size of the buffered results and error channels. It is recommended to set a buffer if you do not want the results channel to
	// block workers from moving on to a new job
	BufferSize int
}

// NewContinuousTask will return a ContinuousTask which will process jobs concurrently with the provided handler function
func NewContinuousTask(ct ContinuousTaskConfig) Task {
	l := limiter.NewLimiter(limiter.Config{
		RPS:   ct.RateLimit,
		Clock: clock.NewClock(),
	})
	rc := make(chan JobData, ct.BufferSize)
	ec := make(chan JobData, ct.BufferSize)
	pc := safe.NewResourceManager(ct.Workers, 0)
	clk := clock.NewClock()
	wg := cTask{
		workers:         pc,
		limiter:         l,
		handlerFunction: ct.HandlerFunction,
		errorChannel:    ec,
		resultsChannel:  rc,
		clock:           clk,
		errorHandler:    ct.ErrorHandler,
		resultHandler:   ct.ResultHandler,
		jobs:            ct.Jobs,
	}
	return &wg
}

// Stop will stop creating new jobs and wait for any jobs in progress to finish
func (w *cTask) Stop() {
	if w.cancelCtx != nil {
		w.cancelCtx()
	}
}

// Start will begin processing the jobs provided
func (w *cTask) Start(ctx context.Context) {
	notContext, cancel := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	w.cancelCtx = cancel
	w.ctx = notContext

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
}

func (w *cTask) start(flushGroup *sync.WaitGroup) {
	flushGroup.Add(1)
	go errorHandler(flushGroup, w.errorChannel, w.Stop, w.errorHandler)
	flushGroup.Add(1)
	go resultHandler(flushGroup, w.resultsChannel, w.resultHandler)

	waitGroup := sync.WaitGroup{}
	waitGroup.Add(1)

	go w.work(&waitGroup)
	waitGroup.Wait()
}

func (w *cTask) work(waitGroup *sync.WaitGroup) {
	defer waitGroup.Done()
	for {
		select {
		case <-w.ctx.Done():
			return
		default:
			numberOfWorkers := <-w.limiter.WorkAvailable()
			for i := 0; i < numberOfWorkers; i++ {
				if w.workers.GetAvailableWorkers() > 0 {
					w.workers.TakeJob()
					waitGroup.Add(1)
					w.limiter.Record(w.clock.Now())
					go w.receiveJob(waitGroup, w.workers.GetAllJobsAccepted())
				}
			}
		}
	}
}

func (w *cTask) receiveJob(waitGroup *sync.WaitGroup, count int) {
	defer waitGroup.Done()
	select {
	case <-w.ctx.Done():
		w.workers.AbortedJob()
	case job, open := <-w.jobs:
		if !open {
			w.workers.AbortedJob()
			w.Stop()
		} else {
			if job != nil {
				w.handleJob(job, count)
				w.workers.FinishJob()
			} else {
				w.workers.AbortedJob()
			}
		}
	default:
		w.workers.AbortedJob()
	}
}

// handleJob will send a job to the handler function, returning the results to the appropriate channels.
// It also takes care of cancelling any in flight jobs
func (w *cTask) handleJob(job interface{}, count int) {
	childContext, _ := context.WithCancel(w.ctx)
	innerDone := make(chan interface{})
	go func() {
		result, err := w.handlerFunction(childContext, job)
		select {
		case <-w.ctx.Done():
		default:
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
		}
		close(innerDone)
	}()
	<-innerDone
}
