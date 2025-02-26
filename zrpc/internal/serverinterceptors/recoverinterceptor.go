package serverinterceptors

import (
	"context"
	"runtime/debug"

	"github.com/r27153733/fastgozero/core/logx"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// StreamRecoverInterceptor catches panics in processing stream requests and recovers.
func StreamRecoverInterceptor(svr any, stream grpc.ServerStream, _ *grpc.StreamServerInfo,
	handler grpc.StreamHandler) (err error) {
	defer handleCrash(func(r any) {
		err = toPanicError(r)
	})

	return handler(svr, stream)
}

// UnaryRecoverInterceptor catches panics in processing unary requests and recovers.
func UnaryRecoverInterceptor(ctx context.Context, req any, _ *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler) (resp any, err error) {
	defer handleCrash(func(r any) {
		err = toPanicError(r)
	})

	return handler(ctx, req)
}

func handleCrash(handler func(any)) {
	if r := recover(); r != nil {
		handler(r)
	}
}

func toPanicError(r any) error {
	logx.Errorf("%+v\n\n%s", r, debug.Stack())
	return status.Errorf(codes.Internal, "panic: %v", r)
}
