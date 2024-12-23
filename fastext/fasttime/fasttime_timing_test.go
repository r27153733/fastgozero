package fasttime

import (
	"sync/atomic"
	"testing"
	"time"
)

func BenchmarkUnixTimestamp(b *testing.B) {
	time.Sleep(2 * time.Second)

	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		var ts int64
		for pb.Next() {
			ts += UnixTimestamp()
		}
		Sink.Store(ts)
	})
}

func BenchmarkTimeNowUnix(b *testing.B) {
	time.Sleep(2 * time.Second)

	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		var ts int64
		for pb.Next() {
			ts += time.Now().Unix()
		}
		Sink.Store(ts)
	})
}

func BenchmarkTimestamp(b *testing.B) {
	time.Sleep(2 * time.Second)

	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		var ts time.Time
		for pb.Next() {
			ts = Timestamp()
		}
		SinkTime.Store(&ts)
	})
}

func BenchmarkTimeNow(b *testing.B) {
	time.Sleep(2 * time.Second)

	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		var ts time.Time
		for pb.Next() {
			ts = time.Now()
		}
		SinkTime.Store(&ts)
	})
}

// Sink should prevent from code elimination by optimizing compiler
var Sink atomic.Int64

var SinkTime atomic.Pointer[time.Time]
