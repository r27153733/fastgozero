package propagation

import (
	"github.com/valyala/fasthttp"
)

// ReqHeaderCarrier adapts http.Header to satisfy the TextMapCarrier interface.
type ReqHeaderCarrier fasthttp.RequestHeader

// Get returns the value associated with the passed key.
func (hc *ReqHeaderCarrier) Get(key string) string {
	return string((*fasthttp.RequestHeader)(hc).Peek(key))
}

// Set stores the key-value pair.
func (hc *ReqHeaderCarrier) Set(key string, value string) {
	(*fasthttp.RequestHeader)(hc).Set(key, value)
}

// Keys lists the keys stored in this carrier.
func (hc *ReqHeaderCarrier) Keys() []string {
	keysBytes := (*fasthttp.RequestHeader)(hc).PeekKeys()
	keys := make([]string, 0, len(keysBytes))
	for _, k := range keysBytes {
		keys = append(keys, string(k))
	}
	return keys
}

func ConvertReq(reqHeader *fasthttp.RequestHeader) *ReqHeaderCarrier {
	return (*ReqHeaderCarrier)(reqHeader)
}

// RespHeaderCarrier adapts http.Header to satisfy the TextMapCarrier interface.
type RespHeaderCarrier fasthttp.ResponseHeader

// Get returns the value associated with the passed key.
func (hc *RespHeaderCarrier) Get(key string) string {
	return string((*fasthttp.ResponseHeader)(hc).Peek(key))
}

// Set stores the key-value pair.
func (hc *RespHeaderCarrier) Set(key string, value string) {
	(*fasthttp.ResponseHeader)(hc).Set(key, value)
}

// Keys lists the keys stored in this carrier.
func (hc *RespHeaderCarrier) Keys() []string {
	keysBytes := (*fasthttp.ResponseHeader)(hc).PeekKeys()
	keys := make([]string, 0, len(keysBytes))
	for _, k := range keysBytes {
		keys = append(keys, string(k))
	}
	return keys
}

func ConvertResp(reqHeader *fasthttp.ResponseHeader) *RespHeaderCarrier {
	return (*RespHeaderCarrier)(reqHeader)
}
