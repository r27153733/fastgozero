package fileserver

import (
	"github.com/r27153733/fastgozero/fastext"
	"github.com/valyala/fasthttp"
	"io/fs"
	"strings"
	"sync"
)

// Middleware returns a middleware that serves files from the given file system.
func Middleware(path string, fs fs.FS) func(fasthttp.RequestHandler) fasthttp.RequestHandler {
	pathWithoutTrailSlash := ensureNoTrailingSlash(path)
	canServe := createServeChecker(path, fs)

	return func(next fasthttp.RequestHandler) fasthttp.RequestHandler {
		return func(ctx *fasthttp.RequestCtx) {
			if canServe(&ctx.Request) {
				fasthttp.ServeFS(ctx, fs, fastext.B2s(ctx.URI().Path()[len(pathWithoutTrailSlash):]))
			} else {
				next(ctx)
			}
		}
	}
}

func createFileChecker(fs fs.FS) func(string) bool {
	var lock sync.RWMutex
	fileChecker := make(map[string]bool)

	return func(path string) bool {
		lock.RLock()
		exist, ok := fileChecker[path]
		lock.RUnlock()
		if ok {
			return exist
		}

		lock.Lock()
		defer lock.Unlock()

		file, err := fs.Open(path)
		exist = err == nil
		fileChecker[path] = exist
		if err != nil {
			return false
		}

		_ = file.Close()
		return true
	}
}

func createServeChecker(path string, fs fs.FS) func(r *fasthttp.Request) bool {
	pathWithTrailSlash := ensureTrailingSlash(path)
	fileChecker := createFileChecker(fs)

	return func(r *fasthttp.Request) bool {
		return r.Header.IsGet() &&
			strings.HasPrefix(fastext.B2s(r.URI().Path()), pathWithTrailSlash) &&
			fileChecker(fastext.B2s(r.URI().Path()[len(pathWithTrailSlash):]))
	}
}

func ensureTrailingSlash(path string) string {
	if strings.HasSuffix(path, "/") {
		return path
	}

	return path + "/"
}

func ensureNoTrailingSlash(path string) string {
	if strings.HasSuffix(path, "/") {
		return path[:len(path)-1]
	}

	return path
}
