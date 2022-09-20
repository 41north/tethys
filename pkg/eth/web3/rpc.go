package web3

import (
	"context"
	"math/big"

	"github.com/creachadair/jrpc2"
	"github.com/creachadair/wschannel"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/juju/errors"
)

type Client struct {
	url          string
	rpc          *jrpc2.Client
	sm           *subManager
	closeHandler func(code int, text string)
}

func NewClient(url string) Client {
	return Client{url: url}
}

func (c *Client) Connect(
	closeHandler func(error error),
) error {
	// TODO use a different channel based on the transport specified in the url
	// TODO close handler
	conn, err := wschannel.Dial(c.url, nil)
	if err != nil {
		return err
	}

	go func() {
		<-conn.Done()
		// TODO refine the concept of close handler
		closeHandler(nil)
	}()

	c.rpc = jrpc2.NewClient(conn, &jrpc2.ClientOptions{
		OnNotify: c.onNotify,
	})

	c.sm = &subManager{}

	return nil
}

func (c *Client) onNotify(req *jrpc2.Request) {
	c.handleRequest(req)
}

func (c *Client) Close() {
	c.sm.close()
	c.rpc.Close()
}

func (c *Client) CallResult(
	ctx context.Context,
	method string,
	params any,
	result any,
) error {
	return c.rpc.CallResult(ctx, method, params, result)
}

func (c *Client) Batch(
	ctx context.Context,
	specs []jrpc2.Spec,
) ([]*jrpc2.Response, error) {
	return c.rpc.Batch(ctx, specs)
}

func (c *Client) BlockNumber(ctx context.Context) (*big.Int, error) {
	var hex string
	if err := c.CallResult(ctx, "eth_blockNumber", nil, &hex); err != nil {
		return nil, err
	}
	blockNumber, ok := math.ParseBig256(hex)
	if !ok {
		return nil, errors.Errorf("Failed to parse block number: %s", blockNumber)
	}
	return blockNumber, nil
}

func (c *Client) NetVersion(ctx context.Context) (string, error) {
	var result string
	err := c.CallResult(ctx, "net_version", nil, &result)
	return result, err
}

func (c *Client) ChainId(ctx context.Context) (string, error) {
	var result string
	err := c.CallResult(ctx, "eth_chainId", nil, &result)
	return result, err
}

func (c *Client) NodeInfo(ctx context.Context) (NodeInfo, error) {
	var result NodeInfo
	err := c.CallResult(ctx, "admin_nodeInfo", nil, &result)
	return result, err
}

func (c *Client) Web3ClientVersion(ctx context.Context) (ClientVersion, error) {
	var result string
	if err := c.CallResult(ctx, "web3_clientVersion", nil, &result); err != nil {
		return ClientVersion{}, err
	}
	return ParseClientVersion(result)
}

func (c *Client) SyncProgress(ctx context.Context) (bool, error) {
	var result bool
	err := c.CallResult(ctx, "eth_syncing", nil, &result)
	return result, err
}

func (c *Client) LatestBlock(ctx context.Context) (*Block, error) {
	var result Block
	err := c.CallResult(ctx, "eth_getBlockByNumber", []interface{}{"latest", false}, &result)
	return &result, err
}
