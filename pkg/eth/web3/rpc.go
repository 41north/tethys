package web3

import (
	"context"
	"encoding/json"
	"math/big"

	"github.com/41north/tethys/pkg/jsonrpc"
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
	params any,
) (*jsonrpc.Response, error) {
	req := &jsonrpc.Request{
		Method: method,
	}

	if params != nil {
		bytes, err := json.Marshal(params)
		if err != nil {
			return nil, errors.Annotate(err, "failed to marshal params")
		}
		req.Params = bytes
	}

	return c.rpc.Invoke(ctx, req)
}

func (c *Client) InvokeRequest(
	ctx context.Context,
	req *jsonrpc.Request,
) (*jsonrpc.Response, error) {
	return c.rpc.Invoke(ctx, req)
}

func unmarshal[T any](resp *jsonrpc.Response, proto *T) error {
	if resp.Error != nil {
		return errors.Errorf("json rpc error, code = %d message = '%s'", resp.Error.Code, resp.Error.Message)
	}
	return json.Unmarshal(resp.Result, proto)
}

func (c *Client) BlockNumber(ctx context.Context) (*big.Int, error) {
	resp, err := c.Invoke(ctx, "eth_blockNumber", nil)
	if err != nil {
		return nil, err
	}

	var hex string
	if err = unmarshal[string](resp, &hex); err != nil {
		return nil, err
	}

	blockNumber, ok := math.ParseBig256(hex)
	if !ok {
		return nil, errors.Errorf("Failed to parse block number: %s", blockNumber)
	}
	return blockNumber, nil
}

func (c *Client) NetVersion(ctx context.Context) (string, error) {
	resp, err := c.Invoke(ctx, "net_version", nil)
	if err != nil {
		return "", err
	}
	var result string
	err = unmarshal[string](resp, &result)
	return result, err
}

func (c *Client) ChainId(ctx context.Context) (string, error) {
	resp, err := c.Invoke(ctx, "eth_chainId", nil)
	if err != nil {
		return "", err
	}
	var result string
	err = unmarshal[string](resp, &result)
	return result, err
}

func (c *Client) NodeInfo(ctx context.Context) (*NodeInfo, error) {
	resp, err := c.Invoke(ctx, "admin_nodeInfo", nil)
	if err != nil {
		return nil, err
	}
	var result NodeInfo
	err = unmarshal[NodeInfo](resp, &result)
	return &result, err
}

func (c *Client) Web3ClientVersion(ctx context.Context) (*ClientVersion, error) {
	resp, err := c.Invoke(ctx, "web3_clientVersion", nil)
	if err != nil {
		return nil, err
	}
	var result string
	if err = unmarshal[string](resp, &result); err != nil {
		return nil, err
	}

	cv, err := ParseClientVersion(result)
	return &cv, err
}

func (c *Client) SyncProgress(ctx context.Context) (bool, error) {
	resp, err := c.Invoke(ctx, "eth_syncing", nil)
	if err != nil {
		return false, err
	}
	var syncing bool
	if err := json.Unmarshal(resp.Result, &syncing); err == nil {
		return syncing, nil
	} else {
		syncing = true
	}
	return syncing, nil
}

func (c *Client) LatestBlock(ctx context.Context) (*Block, error) {
	resp, err := c.Invoke(ctx, "eth_getBlockByNumber", []interface{}{"latest", false})
	if err != nil {
		return nil, err
	}
	var result Block
	err = unmarshal[Block](resp, &result)
	return &result, err
}
