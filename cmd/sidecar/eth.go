package main

import (
	"context"

	"github.com/41north/tethys/pkg/eth"

	"github.com/41north/tethys/pkg/eth/sidecar"
	"github.com/41north/tethys/pkg/process"
)

type ethSidecarCmd struct {
	sidecarCmd
}

func (cmd *ethSidecarCmd) toOptions() []sidecar.Option {
	result := []sidecar.Option{
		sidecar.ClientUrl(cmd.ClientUrl),
		sidecar.NatsUrl(cmd.NatsUrl),
		sidecar.ClientConnectionType(eth.ToConnectionType(cmd.ClientConnectionType)),
	}

	if cmd.ClientConnectionType == eth.ConnectionType(eth.ConnectionTypeManaged).String() {
		result = append(result, sidecar.ClientId(cmd.ClientId))
	}

	return result
}

func (cmd *ethSidecarCmd) Run() error {
	return process.Run(func(ctx context.Context) error {
		return sidecar.Run(ctx, cmd.toOptions()...)
	})
}
