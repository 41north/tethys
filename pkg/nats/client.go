package nats

import (
	"encoding/json"
	"time"

	"github.com/41north/tethys/pkg/jsonrpc"
	"github.com/nats-io/nats.go"
)

type RpcClient interface {
	Invoke(subject string, req *jsonrpc.Request, timeout time.Duration, resp *jsonrpc.Response) error
}

type rpcClient struct {
	conn *nats.Conn
}

func NewRpcClient(conn *nats.Conn) (RpcClient, error) {
	return &rpcClient{
		conn: conn,
	}, nil
}

func (rpc *rpcClient) Invoke(subject string, req *jsonrpc.Request, timeout time.Duration, resp *jsonrpc.Response) error {
	bytes, err := json.Marshal(req)
	msg, err := rpc.conn.Request(subject, bytes, timeout)
	if err != nil {
		return err
	}
	return json.Unmarshal(msg.Data, &resp)
}
