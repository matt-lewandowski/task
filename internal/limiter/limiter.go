package limiter

import (
	"sync"
	"time"

	"github.com/matt-lewandowski/task/internal/limiter/clock"
)

const (
	nanosecondsInSecond = 1000000000
)

// A Limiter is for controlling the requests per second
type Limiter interface {
	// Stop will close any channels and stop calculating the RPS
	Stop()
	// WorkAvailable will return an integer over a buffered channel of how many requests are
	// allowed to happen
	WorkAvailable() <-chan int
	// Record will record a timestamp that will be used to determine how many slots are available.
	// Each request that uses an available slot should call request
	Record(timestamp time.Time)
}

// The Config is the values needed for creating a limiter
type Config struct {
	// RPS is the desired requests per second that will be allowed
	RPS int
	// Clock is just a stubbed Time, for testing purposes
	Clock clock.Clock
}

type limiter struct {
	mutex     *sync.Mutex
	done      chan bool
	jobsReady chan int
	clock     clock.Clock
	entries   []int64
	rps       int
}

// NewLimiter will return a Limiter interface
func NewLimiter(c Config) Limiter {
	done := make(chan bool, 1)
	jobsReady := make(chan int)
	l := limiter{
		mutex:     &sync.Mutex{},
		clock:     c.Clock,
		rps:       c.RPS,
		done:      done,
		jobsReady: jobsReady,
	}
	l.init()
	return &l
}

// Record will record a timestamp into the rate limiter
func (l *limiter) Record(timestamp time.Time) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	l.entries = append(l.entries, timestamp.UnixNano())
}

// Stop will close channels
func (l *limiter) Stop() {
	close(l.done)
}

// WorkAvailable will return the channel that will receive the amount of jobs ready
func (l *limiter) WorkAvailable() <-chan int {
	return l.jobsReady
}

func (l *limiter) init() {
	ticker := time.NewTicker(1 * time.Millisecond)
	go func() {
		for {
			select {
			case <-l.done:
				close(l.jobsReady)
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
		// consume old value to update
		case _, ok := <-l.jobsReady:
			if ok {
				l.jobsReady <- rate
			}
		default:
			l.jobsReady <- rate
		}
	}
}
