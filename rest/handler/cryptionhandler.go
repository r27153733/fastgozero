package handler

import (
	"encoding/base64"
	"errors"
	"io"

	"github.com/valyala/fasthttp"
	"github.com/zeromicro/go-zero/core/codec"
	"github.com/zeromicro/go-zero/fastext"
)

const maxBytes = 1 << 20 // 1 MiB

var errContentLengthExceeded = errors.New("content length exceeded")

// CryptionHandler returns a middleware to handle cryption.
func CryptionHandler(key []byte) func(fasthttp.RequestHandler) fasthttp.RequestHandler {
	return LimitCryptionHandler(maxBytes, key)
}

// LimitCryptionHandler returns a middleware to handle cryption.
func LimitCryptionHandler(limitBytes int64, key []byte) func(handler fasthttp.RequestHandler) fasthttp.RequestHandler {
	return func(next fasthttp.RequestHandler) fasthttp.RequestHandler {
		return func(ctx *fasthttp.RequestCtx) {
			if ctx.Request.Header.ContentLength() <= 0 {
				next(ctx)
				return
			}

			if err := decryptBody(limitBytes, key, &ctx.Request); err != nil {
				ctx.SetStatusCode(fasthttp.StatusBadRequest)
				return
			}

			next(ctx)
		}
	}
}

func decryptBody(limitBytes int64, key []byte, r *fasthttp.Request) error {
	contentLength := r.Header.ContentLength()
	if limitBytes > 0 && int64(contentLength) > limitBytes {
		return errContentLengthExceeded
	}

	var content []byte
	var err error
	if contentLength > 0 {
		content = r.Body()
	} else {
		content, err = io.ReadAll(io.LimitReader(r.BodyStream(), maxBytes))
	}
	if err != nil {
		return err
	}

	content, err = base64.StdEncoding.DecodeString(fastext.B2s(content))
	if err != nil {
		return err
	}

	output, err := codec.EcbDecrypt(key, content)
	if err != nil {
		return err
	}

	r.SetBody(output)

	return nil
}
