package methods

import (
	"github.com/41north/tethys/pkg/eth/tracking"
	natsutil "github.com/41north/tethys/pkg/nats"
	"github.com/41north/tethys/pkg/proxy"
)

const (
	EthBlockNumber                         = "eth_blockNumber"
	EthGetBalance                          = "eth_getBalance"
	EthGetStorageAt                        = "eth_getStorageAt"
	EthGetBlockByNumber                    = "eth_getBlockByNumber"
	EthGetBlockByHash                      = "eth_getBlockByHash"
	EthGetTransactionCount                 = "eth_getTransactionCount"
	EthGetBlockTransactionCountByHash      = "eth_getBlockTransactionCountByHash"
	EthGetBlockTransactionCountByNumber    = "eth_getBlockTransactionCountByNumber"
	EthGetUncleCountByBlockHash            = "eth_getUncleCountByBlockHash"
	EthGetUncleCountByNumber               = "eth_getUncleCountByBlockNumber"
	EthGetCode                             = "eth_getCode"
	EthGetTransactionByHash                = "eth_getTransactionByHash"
	EthGetTransactionByBlockHashAndIndex   = "eth_getTransactionByBlockHashAndIndex"
	EthGetTransactionByBlockNumberAndIndex = "eth_getTransactionByBlockNumberAndIndex"
	EthGetTransactionReceipt               = "eth_getTransactionReceipt"
	EthGetUncleByBlockHashAndIndex         = "eth_getUncleByBlockHashAndIndex"
	EthGetUncleByBlockNumberAndIndex       = "eth_getUncleByBlockNumberAndIndex"
)

func ethMethods(
	chain *tracking.CanonicalChain,
	router natsutil.Router,
) []proxy.Method {
	cacheRouteOpt := proxy.RouteOpts(natsutil.CacheRoute(true))

	overrideLatestBlockOpt := func(idx int) proxy.MethodOpt {
		return proxy.BeforeRequest(proxy.ReplaceParameterByIndex(idx, overrideLatestBlockParam(chain)))
	}

	return []proxy.Method{
		proxy.NewMethod(EthBlockNumber, router),
		proxy.NewMethod(EthGetBalance, router, cacheRouteOpt, overrideLatestBlockOpt(1)),
		proxy.NewMethod(EthGetStorageAt, router, cacheRouteOpt, overrideLatestBlockOpt(2)),
		proxy.NewMethod(EthGetBlockByNumber, router, cacheRouteOpt, overrideLatestBlockOpt(0)),
		proxy.NewMethod(EthGetBlockByHash, router, cacheRouteOpt),
		proxy.NewMethod(EthGetTransactionCount, router, cacheRouteOpt, overrideLatestBlockOpt(1)),
		proxy.NewMethod(EthGetBlockTransactionCountByHash, router, cacheRouteOpt),
		proxy.NewMethod(EthGetBlockTransactionCountByNumber, router, cacheRouteOpt, overrideLatestBlockOpt(0)),
		proxy.NewMethod(EthGetUncleCountByBlockHash, router, cacheRouteOpt),
		proxy.NewMethod(EthGetUncleCountByNumber, router, cacheRouteOpt, overrideLatestBlockOpt(0)),
		proxy.NewMethod(EthGetCode, router, cacheRouteOpt, overrideLatestBlockOpt(1)),
		proxy.NewMethod(EthGetTransactionByHash, router, cacheRouteOpt),
		proxy.NewMethod(EthGetTransactionByBlockHashAndIndex, router, cacheRouteOpt),
		proxy.NewMethod(EthGetTransactionByBlockNumberAndIndex, router, cacheRouteOpt, overrideLatestBlockOpt(0)),
		proxy.NewMethod(EthGetTransactionReceipt, router, cacheRouteOpt),
		proxy.NewMethod(EthGetUncleByBlockHashAndIndex, router, cacheRouteOpt),
		proxy.NewMethod(EthGetUncleByBlockNumberAndIndex, router, cacheRouteOpt, overrideLatestBlockOpt(0)),
	}
}
