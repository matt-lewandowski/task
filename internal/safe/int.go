package safe

import "sync"

type Integer interface {
	Increment()
	Decrement()
	Reset()
	GetCount() int
	GetRunningTotal() int
}

type integer struct {
	mutex        sync.Mutex
	count        int
	runningTotal int
}

func NewInteger(count int) Integer {
	return &integer{
		mutex:        sync.Mutex{},
		count:        count,
		runningTotal: 0,
	}
}

func (i *integer) Increment() {
	i.mutex.Lock()
	defer i.mutex.Unlock()
	i.count++
	i.runningTotal++
}

func (i *integer) Decrement() {
	i.mutex.Lock()
	defer i.mutex.Unlock()
	i.count--
}

func (i *integer) Reset() {
	i.mutex.Lock()
	defer i.mutex.Unlock()
	i.count = 0
}

func (i *integer) GetCount() int {
	i.mutex.Lock()
	defer i.mutex.Unlock()
	return i.count
}

func (i *integer) GetRunningTotal() int {
	i.mutex.Lock()
	defer i.mutex.Unlock()
	return i.runningTotal
}
