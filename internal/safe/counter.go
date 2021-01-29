package safe

import "sync"

// ProgressCounter is a thread safe counter that will also keep track of
// the total number of increments
type ProgressCounter interface {
	Increment()
	Decrement()
	Reset()
	GetAvailableWorkers() int
	GetJobsAccepted() int
	GetJobsToDo() int
}

type progressCounter struct {
	mutex        sync.Mutex
	workerCount  int
	jobsAccepted int
	jobsToDo     int
}

// NewProgressCounter will return a thread safe counter
func NewProgressCounter(workerCount int, jobCount int) ProgressCounter {
	return &progressCounter{
		mutex:        sync.Mutex{},
		workerCount:  workerCount,
		jobsAccepted: 0,
		jobsToDo:     jobCount,
	}
}

// Increment will increment the workerCount and total by 1
func (i *progressCounter) Increment() {
	i.mutex.Lock()
	defer i.mutex.Unlock()
	i.workerCount++
	i.jobsAccepted++
}

// Decrement will subtract 1 from the workerCount
func (i *progressCounter) Decrement() {
	i.mutex.Lock()
	defer i.mutex.Unlock()
	i.workerCount--
	i.jobsToDo--
}

// Reset will set the workerCount to 0
func (i *progressCounter) Reset() {
	i.mutex.Lock()
	defer i.mutex.Unlock()
	i.workerCount = 0
}

// GetAvailableWorkers will return the current workerCount
func (i *progressCounter) GetAvailableWorkers() int {
	i.mutex.Lock()
	defer i.mutex.Unlock()
	return i.workerCount
}

// GetJobsToDo will return the amount of jobs that are left
func (i *progressCounter) GetJobsToDo() int {
	i.mutex.Lock()
	defer i.mutex.Unlock()
	return i.jobsToDo
}

// GetJobsAccepted will return the total number of increments
func (i *progressCounter) GetJobsAccepted() int {
	i.mutex.Lock()
	defer i.mutex.Unlock()
	return i.jobsAccepted
}
