package internal

import (
	"fmt"
	"github.com/r27153733/fastgozero/fastext/bytesconv"
	"github.com/valyala/fasthttp"
	"strings"
)

const (
	metadataHeaderPrefix = "Grpc-Metadata-"
	metadataPrefix       = "gateway-"
)

// ProcessHeaders builds the headers for the gateway from HTTP headers.
func ProcessHeaders(header *fasthttp.RequestHeader) []string {
	var headers []string

	header.VisitAll(func(k []byte, v []byte) {
		sk := bytesconv.BToS(k)
		if !strings.HasPrefix(sk, metadataHeaderPrefix) {
			return
		}

		key := fmt.Sprintf("%s%s", metadataPrefix, strings.TrimPrefix(sk, metadataHeaderPrefix))
		headers = append(headers, key+":"+string(v))
	})

	return headers
}
