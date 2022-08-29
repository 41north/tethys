package nats

import (
	"context"
	"strings"
	"time"

	"github.com/41north/tethys/pkg/jsonrpc"
	"github.com/juju/errors"
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
	Request(req jsonrpc.Request, resp *jsonrpc.Response, timeout time.Duration) error

	RequestWithContext(ctx context.Context, req jsonrpc.Request, resp *jsonrpc.Response) error
}
