package proxy

import (
	"github.com/41north/tethys/pkg/eth/tracking"
	"github.com/juju/errors"
)

var (
	canonicalChain *tracking.CanonicalChain

	balancer tracking.LoadBalancer
)

func startTracking(opts Options) error {
	watcher, err := stateManager.Status.WatchAll()
	if err != nil {
		return errors.Annotate(err, "failed to create client status watcher")
	}

	chain, err := tracking.NewCanonicalChain(opts.NetworkId, opts.ChainId, watcher.Updates(), 12)
	if err != nil {
		return errors.Annotate(err, "failed to create canonical chain tracker")
	}

	lb, err := tracking.NewLatestBalancer(opts.NetworkId, opts.ChainId, 0)
	if err != nil {
		return errors.Annotate(err, "failed to create load balancer")
	}

	chain.AddListener(lb.Channel())
	chain.Start()

	// store in module context
	canonicalChain = chain
	balancer = lb

	return nil
}

func stopTracking() {
	canonicalChain.Close()
}
