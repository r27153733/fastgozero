package internal

import (
	"errors"
	"fmt"
	"github.com/valyala/fasthttp"
	"net"
	"net/http"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/proc"
	"github.com/zeromicro/go-zero/internal/health"
)

const probeNamePrefix = "rest"

// StartOption defines the method to customize http.Server.
type StartOption func(svr *fasthttp.Server)

// StartHttp starts a http server.
func StartHttp(host string, port int, handler fasthttp.RequestHandler, opts ...StartOption) error {
	return start(host, port, handler, func(svr *fasthttp.Server) error {
		addr := fmt.Sprintf("%s:%d", host, port)
		ln, err := net.Listen("tcp", addr)
		if err != nil {
			return err
		}
		return svr.Serve(ln)
	}, opts...)
}

// StartHttps starts a https server.
func StartHttps(host string, port int, certFile, keyFile string, handler fasthttp.RequestHandler,
	opts ...StartOption) error {
	return start(host, port, handler, func(svr *fasthttp.Server) error {
		// certFile and keyFile are set in buildHttpsServer
		addr := fmt.Sprintf("%s:%d", host, port)
		ln, err := net.Listen("tcp", addr)
		if err != nil {
			return err
		}
		return svr.ServeTLS(ln, certFile, keyFile)
	}, opts...)
}

func start(host string, port int, handler fasthttp.RequestHandler, run func(svr *fasthttp.Server) error,
	opts ...StartOption) (err error) {
	server := &fasthttp.Server{
		Handler: handler,
	}
	for _, opt := range opts {
		opt(server)
	}
	healthManager := health.NewHealthManager(fmt.Sprintf("%s-%s:%d", probeNamePrefix, host, port))

	waitForCalled := proc.AddShutdownListener(func() {
		healthManager.MarkNotReady()
		if e := server.Shutdown(); e != nil {
			logx.Error(e)
		}
	})
	defer func() {
		if errors.Is(err, http.ErrServerClosed) {
			waitForCalled()
		}
	}()

	healthManager.MarkReady()
	health.AddProbe(healthManager)
	return run(server)
}
