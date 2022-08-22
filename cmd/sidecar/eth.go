package main

import (
	"context"

	"github.com/41north/web3/pkg/eth/sidecar"
	"github.com/41north/web3/pkg/process"
)

type ethSidecarCmd struct {
	sidecarCmd
}

func (cmd *ethSidecarCmd) toOptions() []sidecar.Option {
	return []sidecar.Option{
		sidecar.ClientUrl(cmd.ClientUrl),
		sidecar.NatsUrl(cmd.NatsUrl),
	}
}

func (cmd *ethSidecarCmd) Run() error {
	return process.Run(func(ctx context.Context) error {
		return sidecar.Run(ctx, cmd.toOptions()...)
	})
}
