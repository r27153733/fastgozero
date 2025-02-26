package gateway

import (
	"context"
	"github.com/valyala/fasthttp"
	"log"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/r27153733/fastgozero/core/conf"
	"github.com/r27153733/fastgozero/core/discov"
	"github.com/r27153733/fastgozero/core/logx"
	"github.com/r27153733/fastgozero/core/logx/logtest"
	"github.com/r27153733/fastgozero/internal/mock"
	"github.com/r27153733/fastgozero/rest/httpc"
	"github.com/r27153733/fastgozero/zrpc"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/test/bufconn"
)

func init() {
	logx.Disable()
}

func dialer() func(context.Context, string) (net.Conn, error) {
	listener := bufconn.Listen(1024 * 1024)
	server := grpc.NewServer()
	mock.RegisterDepositServiceServer(server, &mock.DepositServer{})

	reflection.Register(server)

	go func() {
		if err := server.Serve(listener); err != nil {
			log.Fatal(err)
		}
	}()

	return func(context.Context, string) (net.Conn, error) {
		return listener.Dial()
	}
}

func TestMustNewServer(t *testing.T) {
	var c GatewayConf
	assert.NoError(t, conf.FillDefault(&c))
	// avoid popup alert on macos for asking permissions
	c.DevServer.Host = "localhost"
	c.Host = "localhost"
	c.Port = 18881

	s := MustNewServer(c, withDialer(func(conf zrpc.RpcClientConf) zrpc.Client {
		return zrpc.MustNewClient(conf, zrpc.WithDialOption(grpc.WithContextDialer(dialer())))
	}), WithHeaderProcessor(func(header *fasthttp.RequestHeader) []string {
		return []string{"foo"}
	}))
	s.upstreams = []Upstream{
		{
			Mappings: []RouteMapping{
				{
					Method:  "get",
					Path:    "/deposit/:amount",
					RpcPath: "mock.DepositService/Deposit",
				},
			},
			Grpc: zrpc.RpcClientConf{
				Endpoints: []string{"foo"},
				Timeout:   1000,
				Middlewares: zrpc.ClientMiddlewaresConf{
					Trace:      true,
					Duration:   true,
					Prometheus: true,
					Breaker:    true,
					Timeout:    true,
				},
			},
		},
	}

	assert.NoError(t, s.build())
	go s.Server.Start()
	defer s.Stop()

	time.Sleep(time.Millisecond * 200)

	resp, err := httpc.Do(context.Background(), http.MethodGet, "http://localhost:18881/deposit/100", nil)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	resp, err = httpc.Do(context.Background(), http.MethodGet, "http://localhost:18881/deposit_fail/100", nil)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestServer_ensureUpstreamNames(t *testing.T) {
	var s = Server{
		upstreams: []Upstream{
			{
				Grpc: zrpc.RpcClientConf{
					Target: "target",
				},
			},
		},
	}

	assert.NoError(t, s.ensureUpstreamNames())
	assert.Equal(t, "target", s.upstreams[0].Name)
}

func TestServer_ensureUpstreamNames_badEtcd(t *testing.T) {
	var s = Server{
		upstreams: []Upstream{
			{
				Grpc: zrpc.RpcClientConf{
					Etcd: discov.EtcdConf{},
				},
			},
		},
	}

	logtest.PanicOnFatal(t)
	assert.Panics(t, func() {
		s.Start()
	})
}
