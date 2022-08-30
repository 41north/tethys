package main

import (
	"context"

	"github.com/41north/tethys/pkg/eth/proxy"
	"github.com/41north/tethys/pkg/process"
)

type ethProxyCmd struct {
	proxyCmd
}

func (cmd ethProxyCmd) toOptions() []proxy.Option {
	return []proxy.Option{
		proxy.Address(cmd.Address),
		proxy.NetworkId(cmd.NetworkId),
		proxy.ChainId(cmd.ChainId),
		proxy.NatsUrl(cmd.Nats.URL),
		proxy.NatsEmbedded(cmd.Nats.Embedded.Enable),
		proxy.NatsEmbeddedConfigPath(cmd.Nats.Embedded.ConfigPath),
	}
}

func (cmd *ethProxyCmd) Run() error {
	return process.Run(func(ctx context.Context) error {
		return proxy.ListenAndServe(ctx, cmd.toOptions()...)
	})
}
