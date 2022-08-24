package proxy

import (
	"github.com/41north/tethys/pkg/jsonrpc"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/juju/errors"
)

var (
	overrideLatestBlockParameter = func(current any) (any, error) {
		blockParameter := current.(string)
		if blockParameter != LatestBlockParameter {
			// do not override
			return current, nil
		}
		// set the latest block parameter based on the latest tracked head
		head := canonicalChain.Head()
		if head == nil {
			return "", errors.New("no head available")
		}
		return hexutil.EncodeBig(head.Number), nil
	}

	transformsByMethod = map[string]jsonrpc.RequestTransform{
		EthGetBlockByNumber: jsonrpc.NewRequestPipeline(
			jsonrpc.ReplaceParameterByIndex(0, overrideLatestBlockParameter),
		),
		EthGetBalance: jsonrpc.NewRequestPipeline(
			jsonrpc.ReplaceParameterByIndex(1, overrideLatestBlockParameter),
		),
		EthGetStorageAt: jsonrpc.NewRequestPipeline(
			jsonrpc.ReplaceParameterByIndex(2, overrideLatestBlockParameter),
		),
		EthGetTransactionCount: jsonrpc.NewRequestPipeline(
			jsonrpc.ReplaceParameterByIndex(1, overrideLatestBlockParameter),
		),
		EthGetBlockTransactionCountByNumber: jsonrpc.NewRequestPipeline(
			jsonrpc.ReplaceParameterByIndex(0, overrideLatestBlockParameter),
		),
		EthGetUncleCountByNumber: jsonrpc.NewRequestPipeline(
			jsonrpc.ReplaceParameterByIndex(0, overrideLatestBlockParameter),
		),
		EthGetCode: jsonrpc.NewRequestPipeline(
			jsonrpc.ReplaceParameterByIndex(1, overrideLatestBlockParameter),
		),
		EthGetTransactionByBlockNumberAndIndex: jsonrpc.NewRequestPipeline(
			jsonrpc.ReplaceParameterByIndex(0, overrideLatestBlockParameter),
		),
		EthGetUncleByBlockNumberAndIndex: jsonrpc.NewRequestPipeline(
			jsonrpc.ReplaceParameterByIndex(0, overrideLatestBlockParameter),
		),
	}
)
