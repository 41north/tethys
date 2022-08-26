package nats

import (
	"context"
	"strings"
	"time"

	"github.com/41north/tethys/pkg/jsonrpc"
	"github.com/juju/errors"

	"github.com/nats-io/nats.go"
)

const (
	ErrNoClientsAvailable = errors.ConstError("no clients available")
)

func SubjectName(keys ...string) string {
	var sb strings.Builder
	for idx, key := range keys {
		if idx > 0 {
			sb.WriteString(".")
		}
		sb.WriteString(key)
	}
	return sb.String()
}

type Router interface {
	Request(data []byte, timeout time.Duration) (*nats.Msg, error)

	RequestMsg(msg *nats.Msg, timeout time.Duration) (*nats.Msg, error)

	RequestWithContext(ctx context.Context, data []byte) (*nats.Msg, error)

	RequestMsgWithContext(ctx context.Context, msg *nats.Msg) (*nats.Msg, error)

	RequestJsonRpc(req *jsonrpc.Request, timeout time.Duration, resp *jsonrpc.Response) error

	RequestJsonRpcWithContext(ctx context.Context, req *jsonrpc.Request, resp *jsonrpc.Response) error
}
