package fasttime

import (
	"sync/atomic"
	"time"
)

func init() {
	t := time.Now()
	currentTimestamp.Store(&t)
	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for tm := range ticker.C {
			t := time.Unix(tm.Unix(), 0)
			currentTimestamp.Store(&t)
		}
	}()
}

var currentTimestamp atomic.Pointer[time.Time]

func Timestamp() time.Time {
	return *currentTimestamp.Load()
}
