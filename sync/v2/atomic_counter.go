package v2

import (
	"sync/atomic"
)

type ICounter interface {
	Inc()
	Value() int64
}

// Counter will increment a number safely in concurrent environment.
type AtomicCounter struct {
	value int64
}

// Inc increments the counter atomically.
func (c *AtomicCounter) Inc() {
	atomic.AddInt64(&c.value, 1)
}

// Value returns the current count atomically.
func (c *AtomicCounter) Value() int64 {
	return atomic.LoadInt64(&c.value)
}
