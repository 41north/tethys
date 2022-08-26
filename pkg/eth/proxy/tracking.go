package proxy

import (
	"github.com/41north/tethys/pkg/eth/tracking"
	"github.com/41north/tethys/pkg/nats"
	"github.com/juju/errors"
)

var (
	canonicalChain    *tracking.CanonicalChain
	latestBlockRouter nats.Router
)

func startTracking(opts Options) error {
	watcher, err := stateManager.Status.WatchAll()
	if err != nil {
		return errors.Annotate(err, "failed to create client status watcher")
	}

	canonicalChain, err = tracking.NewCanonicalChain(opts.NetworkId, opts.ChainId, watcher.Updates(), 12)
	if err != nil {
		return errors.Annotate(err, "failed to create canonical chain tracker")
	}

	latestBlockRouter = NewLatestBlockRouter(natsConn, canonicalChain, 0)

	canonicalChain.Start()

	return nil
}

func stopTracking() {
	canonicalChain.Close()
}
