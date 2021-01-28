package limiter

import (
	"github.com/matt-lewandowski/tasks/internal/limiter/clock"
	"sync"
	"time"
)

const (
	nanosecondsInSecond = 1000000000
)

type Limiter interface {
	Stop()
	JobsChannel() chan int
	Record(timestamp time.Time)
}

type Config struct {
	RPS int
}

type limiter struct {
	mutex     *sync.Mutex
	done      chan bool
	jobsReady chan int
	clock     clock.Clock
	entries   []int64
	rps       int
}

func NewLimiter(c Config) Limiter {
	done := make(chan bool)
	jobsReady := make(chan int, 1)
	clk := clock.NewClock()
	l := limiter{
		mutex:     &sync.Mutex{},
		clock:     clk,
		rps:       c.RPS,
		done:      done,
		jobsReady: jobsReady,
		entries:   nil,
	}
	l.init()
	return &l
}

func (l *limiter) init() {
	ticker := time.NewTicker(1 * time.Millisecond)
	go func() {
		for {
			select {
			case <-l.done:
				return
			case <-ticker.C:
				l.calculateAllowance()
			}
		}
	}()
}

func (l *limiter) calculateAllowance() {
	now := l.clock.Now().UnixNano()
	secondWindow := now - nanosecondsInSecond

	var second int
	var currentEntries []int64

	l.mutex.Lock()
	for _, entry := range l.entries {
		isValid := false
		if entry > secondWindow {
			second++
			isValid = true
		}
		if isValid {
			currentEntries = append(currentEntries, entry)
		}
	}
	l.entries = currentEntries
	l.mutex.Unlock()

	rate := l.rps - second
	if rate > 0 {
		select {
		case l.jobsReady <- rate:
		default:
			return
		}
	}
}

// Record will record a timestamp into the rate limiter
func (l *limiter) Record(timestamp time.Time) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	l.entries = append(l.entries, timestamp.UnixNano())
}

// Stop will close channels
func (l *limiter) Stop() {
	l.done <- true
	close(l.jobsReady)
	close(l.done)
}

// JobsChannel will return the channel that will recieve the amount of jobs ready
func (l *limiter) JobsChannel() chan int {
	return l.jobsReady
}
