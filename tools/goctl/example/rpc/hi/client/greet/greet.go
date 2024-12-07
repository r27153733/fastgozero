// Code generated by goctl. DO NOT EDIT!
// Source: hi.proto

package client

import (
	"context"

	"github.com/r27153733/fastgozero/tools/fastgoctl/example/rpc/hi/pb/hi"
	"github.com/r27153733/fastgozero/zrpc"
	"google.golang.org/grpc"
)

type (
	EventReq  = hi.EventReq
	EventResp = hi.EventResp
	HelloReq  = hi.HelloReq
	HelloResp = hi.HelloResp
	HiReq     = hi.HiReq
	HiResp    = hi.HiResp

	Greet interface {
		SayHi(ctx context.Context, in *HiReq, opts ...grpc.CallOption) (*HiResp, error)
		SayHello(ctx context.Context, in *HelloReq, opts ...grpc.CallOption) (*HelloResp, error)
	}

	defaultGreet struct {
		cli zrpc.Client
	}
)

func NewGreet(cli zrpc.Client) Greet {
	return &defaultGreet{
		cli: cli,
	}
}

func (m *defaultGreet) SayHi(ctx context.Context, in *HiReq, opts ...grpc.CallOption) (*HiResp, error) {
	client := hi.NewGreetClient(m.cli.Conn())
	return client.SayHi(ctx, in, opts...)
}

func (m *defaultGreet) SayHello(ctx context.Context, in *HelloReq, opts ...grpc.CallOption) (*HelloResp, error) {
	client := hi.NewGreetClient(m.cli.Conn())
	return client.SayHello(ctx, in, opts...)
}
