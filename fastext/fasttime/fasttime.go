package fasttime

import (
	"sync/atomic"
	"time"
)

func init() {
	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for tm := range ticker.C {
			t := tm.Unix()
			currentTimestamp.Store(t)
		}
	}()
}

var currentTimestamp = func() *atomic.Int64 {
	var x atomic.Int64
	x.Store(time.Now().Unix())
	return &x
}()

// UnixTimestamp returns the current unix timestamp in seconds.
// It is faster than time.Now().Unix()
func UnixTimestamp() int64 {
	return currentTimestamp.Load()
}

func Timestamp() time.Time {
	return time.Unix(UnixTimestamp(), 0)
}
