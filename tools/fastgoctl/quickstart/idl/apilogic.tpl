package logic

import (
	"context"

	"github.com/r27153733/fastgozero/core/logx"

	"{{.svcPkg}}"
	"{{.typesPkg}}"{{if .callRPC}}
	"{{.rpcClientPkg}}"{{end}}
)

type PingLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewPingLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PingLogic {
	return &PingLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *PingLogic) Ping() (resp *types.Resp, err error) {
	{{if .callRPC}}if _, err = l.svcCtx.GreetRpc.Ping(l.ctx, new(greet.Placeholder)); err != nil {
		return
	}

	{{end}}resp = new(types.Resp)
	resp.Msg = "pong"

	return
}
