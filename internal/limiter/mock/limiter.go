// Code generated by mockery v1.0.0. DO NOT EDIT.

package mock

import (
	time "time"

	mock "github.com/stretchr/testify/mock"
)

// Limiter is an autogenerated mock type for the Limiter type
type Limiter struct {
	mock.Mock
}

// Record provides a mock function with given fields: timestamp
func (_m *Limiter) Record(timestamp time.Time) {
	_m.Called(timestamp)
}

// SlotsAvailable provides a mock function with given fields:
func (_m *Limiter) SlotsAvailable() chan int {
	ret := _m.Called()

	var r0 chan int
	if rf, ok := ret.Get(0).(func() chan int); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(chan int)
		}
	}

	return r0
}

// Stop provides a mock function with given fields:
func (_m *Limiter) Stop() {
	_m.Called()
}
