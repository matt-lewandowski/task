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

// Task is the interface for the task runner
type Task interface {
	Start(ctx context.Context)
	Stop()
}

// JobData is passed the the results and error handlers
type JobData struct {
	// JobValue is the value passed to the job from the jobs queue
	JobValue interface{}

	// The Result from a job if it is successful
	Result interface{}

	// The Error from a job if it is not successful
	Error error

	// The Count is the number of jobs that have been done
	Count int
}

type task struct {
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
	doneChan        chan struct{}
	doneLock        *sync.Mutex
	clock           clock.Clock
}

// TaskConfig is the struct for creating a new task runner. A task runner will perform a set of tasks once,
// and then stop. If you want to create a task runner for an endless amount of tasks, try a  continuous task
type TaskConfig struct {
	// Workers is the desired amount of concurrent processes. Each worker will consume a job which will
	// use the handler function to process it.
	Workers int

	// RateLimit is the desired RPS for all of the workers combined. For example, if there are 100 works, and
	// the rateLimit is set to 15, only 15 workers will start within every second. This can be used if a worker is expected to
	// take longer then 1 second, and you do not want to send all of them off at once.
	RateLimit int

	// Jobs is a slice of interfaces. It will send each interface to the handler function.
	// For example, if Jobs was a slice of strings, each string will get sent to the handler function
	// to be processed.
	Jobs []interface{}

	// The HandlerFunction will be the function that will be passed a job to handle. If it errors, the error will be passed
	// through the error channel.
	HandlerFunction func(ctx context.Context, v interface{}) (interface{}, error)

	// The ErrorHandler is a function which will receive an error, and a function that can be used to stop job. If stop is called,
	// Any jobs that are already in progress will finish. An error channel is used to return data to the error handler, so that
	// workers do not need to wait for the result to be handled before moving on to the next job
	ErrorHandler func(data JobData, stop func())

	// The ResultHandler is a function that will receive the results from the handler function. A results channel is used to return data to the
	// result handler, so that workers do not need to wait for the result to be handled before moving on to the next job
	ResultHandler func(data JobData)

	// The BufferSize is the size of the buffered results and error channels.
	BufferSize int
}

// NewTask will return a Task which will process jobs concurrently with the provided handler function
func NewTask(tc TaskConfig) Task {
	wg := task{
		clock:   clock.NewClock(),
		workers: safe.NewResourceManager(tc.Workers, len(tc.Jobs)),
		limiter: limiter.NewLimiter(limiter.Config{
			RPS:   tc.RateLimit,
			Clock: clock.NewClock(),
		}),
		handlerFunction: tc.HandlerFunction,
		errorHandler:    tc.ErrorHandler,
		resultHandler:   tc.ResultHandler,
		errorChannel:    make(chan JobData, tc.BufferSize),
		resultsChannel:  make(chan JobData, tc.BufferSize),
		doneChan:        make(chan struct{}, 1),
		doneLock:        &sync.Mutex{},
	}
	wg.loadJobs(tc.Jobs)
	return &wg
}

// Stop will cancel the context, which will stop all requests and return.
func (w *task) Stop() {
	w.doneLock.Lock()
	defer w.doneLock.Unlock()
	select {
	case <-w.ctx.Done():
	default:
		w.cancelCtx()
	}
}

func (w *task) done() {
	w.doneLock.Lock()
	defer w.doneLock.Unlock()
	select {
	case <-w.doneChan:
	default:
		close(w.doneChan)
	}
}

// Start will begin processing the jobs provided
func (w *task) Start(ctx context.Context) {
	notContext, cancel := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	w.cancelCtx = cancel
	w.ctx = notContext

	go w.limiter.Start(notContext)

	flushGroup := sync.WaitGroup{}
	done := make(chan bool)
	go func() {
		w.start(&flushGroup)
		close(done)
	}()
	select {
	case <-done:
		close(w.resultsChannel)
		close(w.errorChannel)
	}
	flushGroup.Wait()
	w.Stop()
}

func (w *task) start(flushGroup *sync.WaitGroup) {
	flushGroup.Add(1)
	go w.sendErrors(flushGroup, w.errorHandler)
	flushGroup.Add(1)
	go w.sendResults(flushGroup, w.resultHandler)

	waitGroup := sync.WaitGroup{}
	waitGroup.Add(1)

	go w.work(&waitGroup)
	waitGroup.Wait()
}

func (w *task) work(waitGroup *sync.WaitGroup) {
	defer waitGroup.Done()
	count := 0
	for {
		select {
		case <-w.ctx.Done():
			return
		case <-w.doneChan:
			return
		default:
			workersAvailable, ok := <-w.limiter.WorkAvailable()
			if ok && workersAvailable > 0 {
				for i := 0; i < workersAvailable; i++ {
					select {
					case <-w.ctx.Done():
						return
					default:
						if w.workers.GetAvailableWorkers() > 0 && w.workers.GetJobsToDo() > 0 {
							w.workers.TakeJob()
							waitGroup.Add(1)
							w.limiter.Record(w.clock.Now())
							count++
							go w.receiveJob(waitGroup, count)
						}
					}
				}
			}
		}
	}
}

func (w *task) receiveJob(waitGroup *sync.WaitGroup, count int) {
	defer waitGroup.Done()
	select {
	case <-w.ctx.Done():
		w.workers.AbortedJob()
	case <-w.doneChan:
		w.workers.AbortedJob()
	case job, open := <-w.jobs:
		if !open {
			w.workers.AbortedJob()
			w.done()
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

func (w *task) handleJob(job interface{}, count int) {
	result, err := w.handlerFunction(w.ctx, job)
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
}

// loadJobs will create a buffered channel containing each job
func (w *task) loadJobs(jobs []interface{}) {
	j := make(chan interface{}, len(jobs))
	for _, job := range jobs {
		j <- job
	}
	w.jobs = j
	close(w.jobs)
}

func (w *task) sendErrors(wg *sync.WaitGroup, handler func(data JobData, stop func())) {
	defer wg.Done()
	for {
		select {
		case <-w.ctx.Done():
			for {
				select {
				case data := <-w.errorChannel:
					if data.Error == nil {
						return
					}
				default:
					return
				}
			}
		case err, ok := <-w.errorChannel:
			if ok && handler != nil {
				if err.Error != nil {
					handler(err, w.Stop)
				}
			} else {
				return
			}
		}
	}
}

func (w *task) sendResults(wg *sync.WaitGroup, handler func(data JobData)) {
	defer wg.Done()
	for {
		select {
		case <-w.ctx.Done():
			for {
				select {
				case data := <-w.resultsChannel:
					if data.Result == nil {
						return
					}
				default:
					return
				}
			}
		case result, ok := <-w.resultsChannel:
			if ok && handler != nil {
				if result.Result != nil {
					handler(result)
				}
			} else {
				return
			}
		}
	}
}
