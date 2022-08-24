package proxy

import (
	"context"
	"fmt"
	"time"

	"github.com/juju/errors"

	"github.com/41north/tethys/pkg/jsonrpc"
	natsutil "github.com/41north/tethys/pkg/nats"
	"github.com/viney-shih/go-cache"
)

const (
	LatestBlockParameter   = "latest"
	EarliestBlockParameter = "earliest"
	PendingBlockParameter  = "pending"
)

type (
	CacheType      = int
	BlockParameter = string
)

var (
	methodsToCache = map[string]bool{
		EthGetBalance:                          true,
		EthGetStorageAt:                        true,
		EthGetBlockByNumber:                    true,
		EthGetBlockByHash:                      true,
		EthGetTransactionCount:                 true,
		EthGetBlockTransactionCountByHash:      true,
		EthGetBlockTransactionCountByNumber:    true,
		EthGetUncleCountByBlockHash:            true,
		EthGetUncleCountByNumber:               true,
		EthGetCode:                             true,
		EthGetTransactionByHash:                true,
		EthGetTransactionByBlockHashAndIndex:   true,
		EthGetTransactionByBlockNumberAndIndex: true,
		EthGetTransactionReceipt:               true,
		EthGetUncleByBlockHashAndIndex:         true,
		EthGetUncleByBlockNumberAndIndex:       true,
	}

	respCachePrefix string
	respCache       cache.Cache
)

func InitCaches(opts Options) error {
	respCachePrefix = fmt.Sprintf("eth_resp_cache_%d_%d", opts.NetworkId, opts.ChainId)

	respCache = natsutil.NewCache[jsonrpc.Response](
		respCachePrefix,
		1024*10,
		1*time.Hour,
		jsContext,
	)

	return nil
}

func invokeWithCache(subject string, req *jsonrpc.Request, timeout time.Duration, respCh chan<- *jsonrpc.Response) error {
	_, doCache := methodsToCache[req.Method]
	if !doCache {
		return invoke(subject, req, timeout, respCh)
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	downstreamReq := req
	transform, ok := transformsByMethod[req.Method]

	if ok {
		// copy the original request and apply the transform
		downstreamReq = &jsonrpc.Request{
			JsonRpc: req.JsonRpc, Id: req.Id, Method: req.Method, Params: req.Params,
		}
		if err := transform(downstreamReq); err != nil {
			return errors.Annotate(err, "failed to apply request transform")
		}
	}

	key := fmt.Sprintf("%s_%s", downstreamReq.Method, downstreamReq.Params)

	var resp jsonrpc.Response
	err := respCache.GetByFunc(ctx, respCachePrefix, key, &resp, func() (interface{}, error) {
		resp := jsonrpc.Response{
			Id: req.Id,
		}
		err := rpcClient.Invoke(subject, downstreamReq, timeout, &resp)
		return resp, err
	})
	if err != nil {
		return err
	}

	respCh <- &resp
	return nil
}

func invoke(subject string, req *jsonrpc.Request, timeout time.Duration, respCh chan<- *jsonrpc.Response) error {
	resp := jsonrpc.Response{
		Id: req.Id,
	}
	err := rpcClient.Invoke(subject, req, timeout, &resp)
	if err != nil {
		resp.Error = &jsonrpc.ErrInternal
	}
	respCh <- &resp
	return err
}
