package fasttime

import (
	"sync/atomic"
	"time"
)

func Init() {
	if !currentTimestamp.CompareAndSwap(0, time.Now().Unix()) {
		return
	}

	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for tm := range ticker.C {
			t := tm.Unix()
			currentTimestamp.Store(t)
		}
	}()
}

var currentTimestamp atomic.Int64

// UnixTimestamp returns the current unix timestamp in seconds.
// It is faster than time.Now().Unix()
func UnixTimestamp() int64 {
	return currentTimestamp.Load()
}

func Timestamp() time.Time {
	return time.Unix(UnixTimestamp(), 0)
}
