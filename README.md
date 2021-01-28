[![codecov](https://codecov.io/gh/matt-lewandowski/task/branch/main/graph/badge.svg?token=GHTD4J5BQL)](https://codecov.io/gh/matt-lewandowski/task)

# task
Task workers are useful for handling large, simple, repetitive jobs concurrently.
## Installation
`go get github.com/matt-lewandowski/task/workers`

### Getting Started
Three functions need to be passed in when creating a task worker

- The workerFunction will receive each task, process it, and return the results/error.
``` go
workerFunction := func(task interface{}) (interface{}, error) {
		fmt.Println(fmt.Sprintf("Finished working on %v", task))
		returnValue := fmt.Sprintf("%v completed", task.(string))
		errValue := errors.New(fmt.Sprintf("%v error", returnValue))
		return returnValue, errValue
	}
```

- The errorFunction will receive errors, and has the ability to stop the process.
``` go
errorFunction := func(result workers.JobData, stop func()) {
		fmt.Println("ERROR: Stopping the job")
                stop()
	}
```

- The resultFunction will handle further processing of results from the workerFunction.
``` go
resultFunction := func(result workers.JobData) {
		fmt.Println(result.Result)
	}
```

Then you can just create a new task and tell it to start

``` go
task := workers.NewTask(workers.Config{
		Workers:         250,
		RateLimit:       50,
		Jobs:            tasks,
		HandlerFunction: workerFunction,
		ErrorHandler:    errorFunction,
		ResultHandler:   resultFunction,
	})
	task.Start()
```
---
A more detailed example can be found in example.go