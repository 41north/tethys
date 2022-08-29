package methods

import (
	"github.com/41north/tethys/pkg/eth/tracking"
	natsutil "github.com/41north/tethys/pkg/nats"
	"github.com/41north/tethys/pkg/proxy"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/juju/errors"
)

type BlockParameter = string

const (
	LatestBlockParameter   BlockParameter = "latest"
	EarliestBlockParameter BlockParameter = "earliest"
	PendingBlockParameter  BlockParameter = "pending"
)

func overrideLatestBlockParam(chain *tracking.CanonicalChain) func(any) (any, error) {
	return func(current any) (any, error) {
		blockParameter := current.(string)
		if blockParameter != LatestBlockParameter {
			// do not override
			return current, nil
		}
		// set the latest block parameter based on the latest tracked head
		head := chain.Head()
		if head == nil {
			return "", errors.New("no head available")
		}
		return hexutil.EncodeBig(head.Number), nil
	}
}

func register(methodMap map[string]proxy.Method, methods []proxy.Method) error {
	for _, method := range methods {
		name := method.Name()
		_, exists := methodMap[name]
		if exists {
			return errors.Errorf("a method is already registered with the name '%s'", name)
		}
		methodMap[name] = method
	}
	return nil
}

func Build(
	chain *tracking.CanonicalChain,
	latestBlockRouter natsutil.Router,
) (map[string]proxy.Method, error) {

	result := make(map[string]proxy.Method)

	// web3 methods
	if err := register(result, web3Methods(latestBlockRouter)); err != nil {
		return nil, err
	}

	// net methods
	if err := register(result, netMethods(chain)); err != nil {
		return nil, err
	}

	// eth methods
	if err := register(result, ethMethods(chain, latestBlockRouter)); err != nil {
		return nil, err
	}

	return result, nil
}
