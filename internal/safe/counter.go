package safe

import "sync"

type ProgressCounter interface {
	Increment()
	Decrement()
	Reset()
	GetCount() int
	GetIncrementTotal() int
}

type progressCounter struct {
	mutex          sync.Mutex
	count          int
	incrementTotal int
}

func NewProgressCounter(count int) ProgressCounter {
	return &progressCounter{
		mutex:          sync.Mutex{},
		count:          count,
		incrementTotal: 0,
	}
}

func (i *progressCounter) Increment() {
	i.mutex.Lock()
	defer i.mutex.Unlock()
	i.count++
	i.incrementTotal++
}

func (i *progressCounter) Decrement() {
	i.mutex.Lock()
	defer i.mutex.Unlock()
	i.count--
}

func (i *progressCounter) Reset() {
	i.mutex.Lock()
	defer i.mutex.Unlock()
	i.count = 0
}

func (i *progressCounter) GetCount() int {
	i.mutex.Lock()
	defer i.mutex.Unlock()
	return i.count
}

func (i *progressCounter) GetIncrementTotal() int {
	i.mutex.Lock()
	defer i.mutex.Unlock()
	return i.incrementTotal
}
