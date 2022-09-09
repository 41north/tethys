package web3

import (
	"context"
	"math/big"

	"github.com/41north/go-jsonrpc"

	"github.com/ethereum/go-ethereum/common/math"
	"github.com/juju/errors"
	"golang.org/x/sync/errgroup"
)

type Client struct {
	rpc          jsonrpc.Client
	sm           *subManager
	group        *errgroup.Group
	closeHandler func(code int, text string)
}

func NewClient(url string) (*Client, error) {
	client := Client{
		group: new(errgroup.Group),
	}

	dialer := jsonrpc.WebSocketDialer{
		Url: url,
	}
	client.rpc = jsonrpc.NewClient(dialer)

	return &client, nil
}

func (c *Client) Connect(
	closeHandler func(error error),
) error {
	c.sm = &subManager{}
	serverRequestsCh := make(chan *jsonrpc.Request, 256)

	c.rpc.SetCloseHandler(closeHandler)
	c.rpc.SetRequestHandler(func(req jsonrpc.Request) {
		serverRequestsCh <- &req
	})

	c.group.Go(func() error {
		for request := range serverRequestsCh {
			c.handleRequest(request)
		}
		return nil
	})

	return c.rpc.Connect()
}

func (c *Client) Close() {
	c.sm.close()
	c.rpc.Close()
}

func (c *Client) Invoke(
	ctx context.Context,
	method string,
	params any,
	resp *jsonrpc.Response,
) error {
	req, err := jsonrpc.NewRequest(method, params)
	if err != nil {
		return err
	}
	return c.rpc.SendContext(ctx, *req, resp)
}

func (c *Client) InvokeRequest(
	ctx context.Context,
	req jsonrpc.Request,
	resp *jsonrpc.Response,
) error {
	return c.rpc.SendContext(ctx, req, resp)
}

func (c *Client) BlockNumber(ctx context.Context) (*big.Int, error) {
	var resp jsonrpc.Response
	if err := c.Invoke(ctx, "eth_blockNumber", nil, &resp); err != nil {
		return nil, err
	}

	var hex string
	if err := resp.UnmarshalResult(&hex); err != nil {
		return nil, err
	}

	blockNumber, ok := math.ParseBig256(hex)
	if !ok {
		return nil, errors.Errorf("Failed to parse block number: %s", blockNumber)
	}
	return blockNumber, nil
}

func (c *Client) NetVersion(ctx context.Context) (string, error) {
	var resp jsonrpc.Response
	if err := c.Invoke(ctx, "net_version", nil, &resp); err != nil {
		return "", err
	}
	var result string
	err := resp.UnmarshalResult(&result)
	return result, err
}

func (c *Client) ChainId(ctx context.Context) (string, error) {
	var resp jsonrpc.Response
	if err := c.Invoke(ctx, "eth_chainId", nil, &resp); err != nil {
		return "", err
	}
	var result string
	err := resp.UnmarshalResult(&result)
	return result, err
}

func (c *Client) NodeInfo(ctx context.Context) (*NodeInfo, error) {
	var resp jsonrpc.Response
	if err := c.Invoke(ctx, "admin_nodeInfo", nil, &resp); err != nil {
		return nil, err
	}
	var result NodeInfo
	err := resp.UnmarshalResult(&result)
	return &result, err
}

func (c *Client) Web3ClientVersion(ctx context.Context) (*ClientVersion, error) {
	var resp jsonrpc.Response
	if err := c.Invoke(ctx, "web3_clientVersion", nil, &resp); err != nil {
		return nil, err
	}
	var result string
	if err := resp.UnmarshalResult(&result); err != nil {
		return nil, err
	}
	cv, err := ParseClientVersion(result)
	return &cv, err
}

func (c *Client) SyncProgress(ctx context.Context) (bool, error) {
	var resp jsonrpc.Response
	if err := c.Invoke(ctx, "eth_syncing", nil, &resp); err != nil {
		return false, err
	}
	var syncing bool
	if err := resp.UnmarshalResult(&syncing); err == nil {
		return syncing, nil
	}
	return syncing, nil
}

func (c *Client) LatestBlock(ctx context.Context) (*Block, error) {
	var resp jsonrpc.Response
	if err := c.Invoke(ctx, "eth_getBlockByNumber", []interface{}{"latest", false}, &resp); err != nil {
		return nil, err
	}
	var result Block
	err := resp.UnmarshalResult(&result)
	return &result, err
}
