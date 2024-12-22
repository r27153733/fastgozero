package internal

import (
	"github.com/r27153733/fastgozero/fastext/bytesconv"
	"github.com/valyala/fasthttp"
	"time"
)

const grpcTimeoutHeader = "Grpc-Timeout"

// GetTimeout returns the timeout from the header, if not set, returns the default timeout.
func GetTimeout(header *fasthttp.RequestHeader, defaultTimeout time.Duration) time.Duration {
	if timeout := header.Peek(grpcTimeoutHeader); len(timeout) > 0 {
		if t, err := time.ParseDuration(bytesconv.BToS(timeout)); err == nil {
			return t
		}
	}

	return defaultTimeout
}
