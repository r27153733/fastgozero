package fasttime

import (
	"testing"
	"time"
)

func TestUnixTimestamp(t *testing.T) {
	time.Sleep(3 * time.Second)
	tsExpected := time.Now().Unix()
	ts := UnixTimestamp()
	if ts-tsExpected > 1 {
		t.Fatalf("unexpected UnixTimestamp; got %d; want %d", ts, tsExpected)
	}
}

func TestTimestamp(t *testing.T) {
	time.Sleep(3 * time.Second)
	tsExpected := time.Now()
	ts := Timestamp()
	if ts.Sub(tsExpected) > time.Second {
		t.Fatalf("unexpected Timestamp; got %v; want %v", ts, tsExpected)
	}
}
