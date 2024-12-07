package syncx

import (
	"sync/atomic"
)

// AtomicLimit controls the concurrent requests.
type AtomicLimit struct {
	n, m uint32
}

// NewAtomicLimit creates a AtomicLimit that can borrow n elements from it concurrently.
func NewAtomicLimit(n int) *AtomicLimit {
	return &AtomicLimit{n: uint32(n), m: 0}
}

// Return returns the borrowed resource, returns error only if returned more than borrowed.
func (l *AtomicLimit) Return() {
	atomic.AddUint32(&l.m, ^uint32(0))
}

// TryBorrow tries to borrow an element from AtomicLimit, in non-blocking mode.
// If success, true returned, false for otherwise.
func (l *AtomicLimit) TryBorrow() bool {
	v := atomic.AddUint32(&l.m, 1)
	if v > l.n {
		atomic.AddUint32(&l.m, ^uint32(0))
		return false
	}
	return true
}
