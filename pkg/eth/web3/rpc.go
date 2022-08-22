package web3

import (
	"context"
	"encoding/json"
	"math/big"

	"github.com/41north/web3/pkg/async"
	"github.com/41north/web3/pkg/jsonrpc"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/juju/errors"
	"golang.org/x/sync/errgroup"
)

type Client struct {
	rpc          *jsonrpc.Client
	sm           *subManager
	group        *errgroup.Group
	closeHandler func(code int, text string)
}

func NewClient(url string) (*Client, error) {

	client := Client{
		group: new(errgroup.Group),
	}

	rpc, err := jsonrpc.NewClient(url)

	if err != nil {
		return nil, err
	}

	client.rpc = rpc

	return &client, nil
}

func (c *Client) Connect(
	closeHandler func(code int, message string),
) error {
	c.sm = &subManager{}
	serverRequestsCh := make(chan *jsonrpc.Request, 256)

	c.group.Go(func() error {
		for request := range serverRequestsCh {
			c.handleRequest(request)
		}
		return nil
	})

	return c.rpc.Connect(serverRequestsCh, closeHandler)
}

func (c *Client) Close() {
	c.sm.close()
	c.rpc.Close()
}

func (c *Client) Invoke(
	ctx context.Context,
	method string,
	params any) async.Future[jsonrpc.Response] {

	req := &jsonrpc.Request{
		Method: method,
	}

	if params != nil {
		bytes, err := json.Marshal(params)
		if err != nil {
			return async.NewFutureFailed[jsonrpc.Response](errors.Annotate(err, "failed to marshal params"))
		}
		req.Params = bytes
	}

	return c.rpc.Invoke(ctx, req)
}

func (c *Client) InvokeRequest(
	ctx context.Context,
	req *jsonrpc.Request,
) async.Future[jsonrpc.Response] {
	return c.rpc.Invoke(ctx, req)
}

func unmarshal[T any](ctx context.Context, resp async.Future[jsonrpc.Response], proto T) async.Future[T] {
	return async.Map(ctx, resp, func(from jsonrpc.Response) (*T, error) {
		if from.Error != nil {
			return nil, errors.Errorf("json rpc error, code = %d message = '%s'", from.Error.Code, from.Error.Message)
		}
		err := json.Unmarshal(from.Result, &proto)
		return &proto, err
	})
}

func (c *Client) BlockNumber(ctx context.Context) async.Future[big.Int] {

	resp := c.Invoke(ctx, "eth_blockNumber", nil)
	unmarshalled := unmarshal[string](ctx, resp, "")

	return async.Map(ctx, unmarshalled, func(from string) (*big.Int, error) {
		blockNumber, ok := math.ParseBig256(from)
		if !ok {
			return nil, errors.Errorf("Failed to parse block number: %s", blockNumber)
		}
		return blockNumber, nil
	})
}

func (c *Client) NetVersion(ctx context.Context) async.Future[string] {
	resp := c.Invoke(ctx, "net_version", nil)
	return unmarshal[string](ctx, resp, "")
}

func (c *Client) ChainId(ctx context.Context) async.Future[string] {
	resp := c.Invoke(ctx, "eth_chainId", nil)
	return unmarshal[string](ctx, resp, "")
}

func (c *Client) NodeInfo(ctx context.Context) async.Future[NodeInfo] {
	resp := c.Invoke(ctx, "admin_nodeInfo", nil)
	return unmarshal[NodeInfo](ctx, resp, NodeInfo{})
}

func (c *Client) Web3ClientVersion(ctx context.Context) async.Future[ClientVersion] {
	resp := c.Invoke(ctx, "web3_clientVersion", nil)
	unmarshalled := unmarshal[string](ctx, resp, "")
	return async.Map(ctx, unmarshalled, func(from string) (*ClientVersion, error) {
		cv, err := ParseClientVersion(from)
		return &cv, err
	})
}

func (c *Client) SyncProgress(ctx context.Context) async.Future[bool] {
	resp := c.Invoke(ctx, "eth_syncing", nil)
	unmarshalled := unmarshal[json.RawMessage](ctx, resp, json.RawMessage{})

	// todo support progress fields
	return async.Map(ctx, unmarshalled, func(from json.RawMessage) (*bool, error) {
		var syncing bool
		if err := json.Unmarshal(from, &syncing); err == nil {
			return &syncing, nil
		} else {
			syncing = true
		}
		return &syncing, nil
	})
}

func (c *Client) LatestBlock(ctx context.Context) async.Future[Block] {
	resp := c.Invoke(ctx, "eth_getBlockByNumber", []interface{}{"latest", false})
	return unmarshal[Block](ctx, resp, Block{})
}
