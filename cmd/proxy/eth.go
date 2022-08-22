package main

import (
	"context"

	"github.com/41north/web3/pkg/eth/proxy"
	"github.com/41north/web3/pkg/process"
)

type ethProxyCmd struct {
	proxyCmd
}

func (cmd ethProxyCmd) toOptions() []proxy.Option {
	return []proxy.Option{
		proxy.Address(cmd.Address),
		proxy.NetworkId(cmd.NetworkId),
		proxy.ChainId(cmd.ChainId),
		proxy.NatsUrl(cmd.NatsURL),
	}
}

func (cmd *ethProxyCmd) Run() error {
	return process.Run(func(ctx context.Context) error {
		return proxy.ListenAndServe(ctx, cmd.toOptions()...)
	})
}
