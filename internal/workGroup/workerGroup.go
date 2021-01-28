package workGroup

import (
	"github.com/matt-lewandowski/tasks/internal/limiter"
	"github.com/matt-lewandowski/tasks/internal/limiter/clock"
	"github.com/matt-lewandowski/tasks/internal/safe"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

// A WorkGroup will process any number of tasks in parallel. Limited by the provided RPS and workers parameters
type WorkGroup interface {
	Start()
	Stop()
}

// JobData is passed the the results and error handlers
type JobData struct {
	// JobValue is the value passed to the job from the jobs queue
	JobValue interface{}

	// The Result from a job if it is successful
	Result   interface{}

	// The Error from a job if it is not successful
	Error    error

	// The Count is the number of jobs that have been done
	Count    int
}

type workgroup struct {
	workers         safe.Integer
	limiter         limiter.Limiter
	jobs            chan interface{}
	handlerFunction func(interface{}) (interface{}, error)
	errorHandler    func(data JobData, stop func())
	resultHandler   func(data JobData)
	errorChannel    chan JobData
	resultsChannel  chan JobData
	stop            chan os.Signal
	clock           clock.Clock
}

type Config struct {
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
	HandlerFunction func(interface{}) (interface{}, error)

	// The ErrorHandler is a function which will receive an error, and a function that can be used to stop job. If stop is called,
	// Any jobs that are already in progress will finish. An error channel is used to return data to the error handler, so that
	// workers do not need to wait for the result to be handled before moving on to the next job
	ErrorHandler func(data JobData, stop func())

	// The ResultHandler is a function that will receive the results from the handler function. A results channel is used to return data to the
	// result handler, so that workers do not need to wait for the result to be handled before moving on to the next job
	ResultHandler func(data JobData)
}

// NewWorkerGroup will return WorkGroup which will process jobs concurrently with the provided handler function
func NewWorkerGroup(c Config) WorkGroup {
	l := limiter.NewLimiter(limiter.Config{RPS: c.RateLimit})
	s := make(chan os.Signal)
	rc := make(chan JobData)
	ec := make(chan JobData)
	i := safe.NewInteger(c.Workers)
	clk := clock.NewClock()
	signal.Notify(s, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	wg := workgroup{
		workers:         i,
		limiter:         l,
		handlerFunction: c.HandlerFunction,
		errorChannel:    ec,
		resultsChannel:  rc,
		stop:            s,
		clock:           clk,
		errorHandler:    c.ErrorHandler,
		resultHandler:   c.ResultHandler,
	}
	wg.loadJobs(c.Jobs)
	return &wg
}

// Stop will stop creating new jobs and wait for any jobs in progress to finish
func (w *workgroup) Stop() {
	w.stop <- os.Interrupt
}

// Start will begin processing the jobs provided
func (w *workgroup) Start() {
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

func (w *workgroup) start(flushGroup *sync.WaitGroup) {
	flushGroup.Add(1)
	go errorHandler(flushGroup, w.errorChannel, w.Stop, w.errorHandler)
	flushGroup.Add(1)
	go resultHandler(flushGroup, w.resultsChannel, w.resultHandler)

	workerStops := make(chan bool, w.workers.GetCount())
	waitGroup := sync.WaitGroup{}
	waitGroup.Add(1)
	go w.work(workerStops, &waitGroup)
	workersDone := make(chan bool)
	go func() {
		waitGroup.Wait()
		workersDone <- true
		close(workersDone)
	}()
	// Now we wait
	for {
		select {
		case <-w.stop:
			for {
				select {
				case workerStops <- true:
				default:
					break
				}
				break
			}
		case <-workersDone:
			close(workerStops)
			return
		}
	}
}

// loadJobs will create a buffered channel containing each job
func (w *workgroup) loadJobs(jobs []interface{}) {
	j := make(chan interface{}, len(jobs))
	for _, job := range jobs {
		j <- job
	}
	w.jobs = j
	close(w.jobs)
}

func (w *workgroup) work(workerStops chan bool, waitGroup *sync.WaitGroup) {
	count := 0
	for len(w.jobs) > 0 && len(workerStops) == 0 {
		numberOfJobs := <-w.limiter.JobsChannel()
		for i := 0; i < numberOfJobs; i++ {
			if w.workers.GetCount() > 0 && len(w.jobs) > 0 && len(workerStops) == 0 {
				w.workers.Decrement()
				waitGroup.Add(1)
				w.limiter.Record(w.clock.Now())
				count++
				go w.receiveJob(waitGroup, workerStops, count)
			}
		}
	}
	waitGroup.Done()
}

func (w *workgroup) receiveJob(waitGroup *sync.WaitGroup, workerStops chan bool, count int) {
	select {
	case job := <-w.jobs:
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
		} else {
			w.Stop()
		}
		waitGroup.Done()
		w.workers.Increment()
	case <-workerStops:
		waitGroup.Done()
		return
	default:
		waitGroup.Done()
		return
	}
}

func errorHandler(wg *sync.WaitGroup, errs chan JobData, stop func(), handler func(data JobData, stop func())) {
	for err := range errs {
		handler(err, stop)
	}
	wg.Done()
}

func resultHandler(wg *sync.WaitGroup, results chan JobData, handler func(data JobData)) {
	for result := range results {
		handler(result)
	}
	wg.Done()
}
