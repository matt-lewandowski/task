package main

import (
	"errors"
	"fmt"
	"github.com/matt-lewandowski/tasks/internal/workGroup"
)

func main() {
	names := make([]interface{}, 10000)
	for i := 0; i < 10000; i++ {
		names[i] = fmt.Sprintf("Matt-%v", i+1)
	}

	workerFunction := func(name interface{}) (interface{}, error) {
		return name, errors.New(name.(string))
	}

	errorFunction := func(result workGroup.JobData, stop func()) {
		if result.Error.Error() == "Matt-8000" {
			stop()
		}
	}

	resultFunction := func(result workGroup.JobData) {
		fmt.Println(result.Result)
	}

	workerG := workGroup.NewWorkerGroup(workGroup.Config{
		Workers:         3,
		RateLimit:       2000,
		Jobs:            names,
		HandlerFunction: workerFunction,
		ErrorHandler:    errorFunction,
		ResultHandler:   resultFunction,
	})
	workerG.Start()
}
