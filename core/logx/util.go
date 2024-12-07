package logx

import (
	"fmt"
	"github.com/r27153733/fastgozero/fastext"
	"github.com/r27153733/fastgozero/fastext/fasttime"
	"runtime"
	"strings"
	"sync"
	"time"
)

func getCaller(callDepth int) string {
	_, file, line, ok := runtime.Caller(callDepth)
	if !ok {
		return ""
	}

	return prettyCaller(file, line)
}

var timestampPoll sync.Pool

func getTimestamp() string {
	v := timestampPoll.Get()
	var buf []byte
	if v == nil {
		buf = make([]byte, 0, len(timeFormat))
	} else {
		buf = v.([]byte)
	}
	var t time.Time
	if isSecondPrecision {
		t = fasttime.Timestamp()
	} else {
		t = time.Now()
	}
	return fastext.B2s(t.AppendFormat(buf, timeFormat))
}

func releaseTimestamp(timestamp string) {
	timestampPoll.Put(fastext.S2B(timestamp)[:0])
}

func prettyCaller(file string, line int) string {
	idx := strings.LastIndexByte(file, '/')
	if idx < 0 {
		return fmt.Sprintf("%s:%d", file, line)
	}

	idx = strings.LastIndexByte(file[:idx], '/')
	if idx < 0 {
		return fmt.Sprintf("%s:%d", file, line)
	}

	return fmt.Sprintf("%s:%d", file[idx+1:], line)
}
