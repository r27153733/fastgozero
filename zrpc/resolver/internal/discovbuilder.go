package internal

import (
	"strings"

	"github.com/r27153733/fastgozero/core/discov"
	"github.com/r27153733/fastgozero/core/logx"
	"github.com/r27153733/fastgozero/zrpc/resolver/internal/targets"
	"google.golang.org/grpc/resolver"
)

type discovBuilder struct{}

func (b *discovBuilder) Build(target resolver.Target, cc resolver.ClientConn, _ resolver.BuildOptions) (
	resolver.Resolver, error) {
	hosts := strings.FieldsFunc(targets.GetAuthority(target), func(r rune) bool {
		return r == EndpointSepChar
	})
	sub, err := discov.NewSubscriber(hosts, targets.GetEndpoints(target))
	if err != nil {
		return nil, err
	}

	update := func() {
		vals := subset(sub.Values(), subsetSize)
		addrs := make([]resolver.Address, 0, len(vals))
		for _, val := range vals {
			addrs = append(addrs, resolver.Address{
				Addr: val,
			})
		}
		if err := cc.UpdateState(resolver.State{
			Addresses: addrs,
		}); err != nil {
			logx.Error(err)
		}
	}
	sub.AddListener(update)
	update()

	return &nopResolver{cc: cc}, nil
}

func (b *discovBuilder) Scheme() string {
	return DiscovScheme
}
