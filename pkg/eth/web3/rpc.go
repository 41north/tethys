package web3

import (
	"context"
	"encoding/json"
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

func (c *Client) Invoke(
	ctx context.Context,
	method string,
	params any,
	resp *jrpc2.Response,
) error {
	return c.rpc.CallResult(ctx, method, params, resp)
}

func (c *Client) InvokeRequest(
	ctx context.Context,
	req jrpc2.Request,
	resp *jrpc2.Response,
) error {
	// we unmarshal to raw message to avoid unmarshalling just to marshal again
	var params json.RawMessage
	if err := req.UnmarshalParams(params); err != nil {
		return err
	}

	return c.rpc.CallResult(ctx, req.Method(), params, resp)
}

func (c *Client) BlockNumber(ctx context.Context) (*big.Int, error) {
	var resp jrpc2.Response
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
	var resp jrpc2.Response
	if err := c.Invoke(ctx, "net_version", nil, &resp); err != nil {
		return "", err
	}
	var result string
	err := resp.UnmarshalResult(&result)
	return result, err
}

func (c *Client) ChainId(ctx context.Context) (string, error) {
	var resp jrpc2.Response
	if err := c.Invoke(ctx, "eth_chainId", nil, &resp); err != nil {
		return "", err
	}
	var result string
	err := resp.UnmarshalResult(&result)
	return result, err
}

func (c *Client) NodeInfo(ctx context.Context) (*NodeInfo, error) {
	var resp jrpc2.Response
	if err := c.Invoke(ctx, "admin_nodeInfo", nil, &resp); err != nil {
		return nil, err
	}
	var result NodeInfo
	err := resp.UnmarshalResult(&result)
	return &result, err
}

func (c *Client) Web3ClientVersion(ctx context.Context) (*ClientVersion, error) {
	var resp jrpc2.Response
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
	var resp jrpc2.Response
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
	var resp jrpc2.Response
	if err := c.Invoke(ctx, "eth_getBlockByNumber", []interface{}{"latest", false}, &resp); err != nil {
		return nil, err
	}
	var result Block
	err := resp.UnmarshalResult(&result)
	return &result, err
}
