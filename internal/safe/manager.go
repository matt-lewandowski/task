package safe

import "sync"

// ResourceManager is a thread safe manager that will keep track of jobs
// and workers
type ResourceManager interface {
	FinishJob()
	AbortedJob()
	TakeJob()
	Reset()
	GetAvailableWorkers() int
	GetWorkersWorking() int
	GetAllJobsAccepted() int
	GetJobsToDo() int
	GetTotalWorkers() int
}

type manager struct {
	mutex        sync.Mutex
	totalWorkers int
	workerCount  int
	jobsAccepted int
	jobsToDo     int
}

// NewResourceManager will return a new ResourceManager
func NewResourceManager(workerCount int, jobCount int) ResourceManager {
	return &manager{
		mutex:        sync.Mutex{},
		totalWorkers: workerCount,
		workerCount:  workerCount,
		jobsAccepted: 0,
		jobsToDo:     jobCount,
	}
}

// FinishJob will increment the workerCount by 1
func (i *manager) FinishJob() {
	i.mutex.Lock()
	defer i.mutex.Unlock()
	i.workerCount++
}

// AbortedJob will reverse TakeJob, because the job was never started
func (i *manager) AbortedJob() {
	i.mutex.Lock()
	defer i.mutex.Unlock()
	i.workerCount++
	i.jobsToDo++
	i.jobsAccepted--
}

// TakeJob will subtract 1 from the workerCount
func (i *manager) TakeJob() {
	i.mutex.Lock()
	defer i.mutex.Unlock()
	i.workerCount--
	i.jobsToDo--
	i.jobsAccepted++
}

// Reset will set the workerCount to 0
func (i *manager) Reset() {
	i.mutex.Lock()
	defer i.mutex.Unlock()
	i.workerCount = 0
}

// GetAvailableWorkers will return the current workerCount
func (i *manager) GetAvailableWorkers() int {
	i.mutex.Lock()
	defer i.mutex.Unlock()
	return i.workerCount
}

// GetJobsToDo will return the amount of jobs that are left,
// when a specified amount of jobs has been set
func (i *manager) GetJobsToDo() int {
	i.mutex.Lock()
	defer i.mutex.Unlock()
	return i.jobsToDo
}

// GetAllJobsAccepted will return the total number of increments
func (i *manager) GetAllJobsAccepted() int {
	i.mutex.Lock()
	defer i.mutex.Unlock()
	return i.jobsAccepted
}

// GetWorkersWorking will return the number of workers that have not
// released their job yet
func (i *manager) GetWorkersWorking() int {
	i.mutex.Lock()
	defer i.mutex.Unlock()
	return i.workerCount - i.jobsAccepted
}

// GetTotalWorkers will return the amount of workers that there are in total
// both working and not working
func (i *manager) GetTotalWorkers() int {
	return i.totalWorkers
}
