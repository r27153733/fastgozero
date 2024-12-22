package handler

import (
	"encoding/base64"
	"errors"
	"github.com/r27153733/fastgozero/fastext/bytesconv"
	"io"
	"net/http"

	"github.com/r27153733/fastgozero/core/codec"
	"github.com/valyala/fasthttp"
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
			defer func() {
				content, err := codec.EcbEncrypt(key, ctx.Response.Body())
				if err != nil {
					ctx.SetStatusCode(http.StatusInternalServerError)
					return
				}
				ctx.Response.SetBody([]byte(base64.StdEncoding.EncodeToString(content)))
			}()
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
	if !r.IsBodyStream() {
		content = r.Body()
	} else {
		content, err = io.ReadAll(io.LimitReader(r.BodyStream(), maxBytes))
	}
	if err != nil {
		return err
	}

	content, err = base64.StdEncoding.DecodeString(bytesconv.BToS(content))
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
