package main

import (
	"fmt"
	"github.com/matt-lewandowski/task/workers"
)

func main() {
	const (
		amountOfJobToDo = 1000
	)

	// Create a slice of tasks as an interface. I hope to replace this with generics
	// but for now, interfaces and type casting will do
	tasks := make([]interface{}, amountOfJobToDo)
	for i := 0; i < amountOfJobToDo; i++ {
		tasks[i] = fmt.Sprintf("Job-%v", i+1)
	}

	// define a workerFunction to receive each task and handle it. For example if tasks was a slice of
	// userIDs, the worker function could be used to make an api call to delete the user. If required, you can
	// pass the results/error back and they will be received by the resultFunction and errorFunction. This
	// is done so the worker can quickly move on to the next job. If nil is passed for either value, the
	// resulting functions will not be called.
	workerFunction := func(task interface{}) (interface{}, error) {
		fmt.Println(fmt.Sprintf("Finished working on %v", task))
		// For the sake of the example, we a different value to the return function, and an error for the
		// error function to catch.
		returnValue := fmt.Sprintf("%v completed", task.(string))
		errValue := fmt.Errorf("%v error", returnValue)
		return returnValue, errValue
	}

	// The errorFunction will be called when the worker function returns an error. It will receive all of the
	// data for the job. The error function will also be passed a stop function. If stop is called from inside
	// the function handler, the workers will complete their task at hand, and then stop. This function is synchronous
	// and only one error can be handled at a time.
	errorFunction := func(result workers.JobData, stop func()) {
		// An example if killing the process, if a specified error is received. It will stop creating new jobs,
		// but the old jobs will finish. So it will go ~100 more since we have so many workers that need to finish
		if result.Error.Error() == "Job-455 completed error" {
			stop()
			fmt.Println("ERROR: Stopping the job")
		}
	}

	// The resultFunction will be called when the worker function finishes a task, and returns a result.
	// This function is synchronous and it will only handle one result at a time, in the order that the
	// tasks are resolved.
	resultFunction := func(result workers.JobData) {
		fmt.Println(result.Count)
	}

	// Create a new task with a specified amount of amount workers and an RPS rate limit.
	// The Workers is the maximum amount of concurrent jobs being handled at a given time.
	// The RateLimit is the maximum amount of new workers being spawned off between each second.
	// It is important to have more workers then RateLimit if each task might take over 1 second
	// to process. This example task takes 2 seconds to process
	task := workers.NewTask(workers.TaskConfig{
		Workers:         100,
		RateLimit:       100,
		Jobs:            tasks,
		HandlerFunction: workerFunction,
		ErrorHandler:    errorFunction,
		ResultHandler:   resultFunction,
	})
	task.Start()
}
