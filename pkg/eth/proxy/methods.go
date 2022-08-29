package proxy

import (
	"github.com/41north/tethys/pkg/jsonrpc"
	natsutil "github.com/41north/tethys/pkg/nats"
)

const (
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

type Method interface {
	Name() string
	Router() natsutil.Router
	BeforeRequest() *jsonrpc.RequestTransform
}
