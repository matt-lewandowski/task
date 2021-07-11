[![codecov](https://codecov.io/gh/matt-lewandowski/task/branch/main/graph/badge.svg?token=GHTD4J5BQL)](https://codecov.io/gh/matt-lewandowski/task)
[![Go Report Card](https://goreportcard.com/badge/github.com/matt-lewandowski/task)](https://goreportcard.com/report/github.com/matt-lewandowski/task)
[![Build Status](https://travis-ci.com/matt-lewandowski/task.svg?branch=main)](https://travis-ci.com/matt-lewandowski/task)
[![contributions welcome](https://img.shields.io/badge/contributions-welcome-brightgreen.svg?style=flat)](https://github.com/dwyl/esta/issues)
# task
Tasks are useful for handling large, simple, repetitive jobs. <br>
A Task has a defined amount of workers, which share an RPS rate limit. <br>
## Installation
`go get github.com/matt-lewandowski/task/workers`

### Getting Started
Three functions need to be passed in when creating a task worker

- The workerFunction will receive each task, process it, and return the results/error.
``` go
workerFunction := func(ctx context.Context, task interface{}) (interface{}, error) {
		err, response := someClientRequest(ctx, task.(requestStruct))
		return response, err
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