package nats

import (
	"context"
	"encoding/json"
	"time"

	"github.com/creachadair/jrpc2"

	"github.com/41north/go-jsonrpc"

	"github.com/41north/tethys/pkg/eth/web3"
	"github.com/41north/tethys/pkg/util"
	"github.com/juju/errors"
	"github.com/nats-io/nats.go"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

const (
	DefaultMaxInFlightRequests = 256
)

type RpcServerOption func(*RpcServerOptions) error

type RpcServerOptions struct {
	// ClientId is the unique node id (public key) for the web3 client
	ClientId string

	// NetworkId is the ethereum network id that the web3 client is connected to.
	NetworkId uint64

	// ChainId is the ethereum chain id that the web3 client is connect to.
	ChainId uint64

	// MaxInFlightRequests constrains the max number of rpc requests that can be
	// awaiting a response from the web3 client.
	MaxInFlightRequests int
}

func GetDefaultRpcServerOptions() RpcServerOptions {
	return RpcServerOptions{
		MaxInFlightRequests: DefaultMaxInFlightRequests,
	}
}

// MaxInFlightRequests is an RpcServerOption to set the max number of rpc requests
// that can be awaiting a response from the web3 client.
func MaxInFlightRequests(max int) RpcServerOption {
	return func(o *RpcServerOptions) error {
		o.MaxInFlightRequests = max
		return nil
	}
}

type RpcServer struct {
	Options RpcServerOptions

	// conn is the NATS connection used for communicating with the NATS server.
	conn *nats.EncodedConn

	// client is used for making rpc requests against the web3 client.
	client *web3.Client

	group  *errgroup.Group
	cancel context.CancelFunc

	log *log.Entry
}

func NewRpcServer(
	clientId string,
	conn *nats.EncodedConn,
	client *web3.Client,
	options ...RpcServerOption,
) (*RpcServer, error) {
	opts := GetDefaultRpcServerOptions()
	opts.ClientId = clientId

	for _, opt := range options {
		if opt != nil {
			if err := opt(&opts); err != nil {
				return nil, err
			}
		}
	}

	l := log.WithFields(log.Fields{
		"component":   "NatsRpcServer",
		"url":         conn.Conn.ConnectedUrlRedacted(),
		"ethClientId": util.ElideString(clientId),
	})

	return &RpcServer{
		Options: opts,
		conn:    conn,
		client:  client,
		log:     l,
	}, nil
}

func (srv *RpcServer) ListenAndServe(
	ctx context.Context,
	subFn func(conn *nats.Conn, msgs chan *nats.Msg) ([]*nats.Subscription, error),
) error {
	opts := srv.Options
	msgs := make(chan *nats.Msg, opts.MaxInFlightRequests)

	subs, err := subFn(srv.conn.Conn, msgs)
	if err != nil {
		return errors.Annotate(err, "failed to initialise subscriptions")
	}

	for _, sub := range subs {
		srv.log.WithField("subject", sub.Subject).Debug("subscription created")
	}

	g := new(errgroup.Group)
	g.SetLimit(opts.MaxInFlightRequests)

	g.Go(func() error {
		for {
			select {

			case <-ctx.Done():
				srv.log.Info("draining rpc requests")
				for _, sub := range subs {
					if err := sub.Drain(); err != nil {
						log.WithError(err).
							WithField("subject", sub.Subject).
							Error("failed to drain rpc requests")
					}
				}
				srv.log.Info("stopped listening for rpc requests")
				return nil

			case msg := <-msgs:
				g.Go(func() error {
					srv.onRequest(ctx, msg)
					return nil
				})

			}
		}
	})

	srv.log.Info("listening for rpc requests")

	return g.Wait()
}

func (srv *RpcServer) Close() {
	srv.log.Info("closing")
	srv.cancel()
	_ = srv.group.Wait()
	srv.log.Info("closed")
}

func (srv *RpcServer) onRequest(ctx context.Context, msg *nats.Msg) {
	requests, err := jrpc2.ParseRequests(msg.Data)
	if err != nil {
		respondWithError(msg, &request, &jsonrpc.Error{
			Code:    -32700,
			Message: "Parse error",
		})
		return
	}

	// capture original request id
	id := request.Id

	go func() {
		ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()

		var resp jrpc2.Response
		if err := srv.client.InvokeRequest(ctx, request, &resp); err != nil {
			respondWithError(msg, &request, &jsonrpc.Error{
				Code:    -32603,
				Message: "Internal error",
			})
			return
		}
		// replace id with original
		resp.Id = id

		respond(msg, &resp)
	}()
}

func respond(msg *nats.Msg, resp *jsonrpc.Response) {
	bytes, err := json.Marshal(resp)
	if err != nil {
		log.WithError(err).Error("failed to serialize error response to json")
	}
	if err = msg.Respond(bytes); err != nil {
		log.WithError(err).Error("failed to send error response to nats")
	}
}

func respondWithError(msg *nats.Msg, request *jsonrpc.Request, error *jsonrpc.Error) {
	response := jsonrpc.Response{
		Error:   error,
		Version: "2.0",
	}
	if request != nil {
		response.Id = request.Id
	}
	bytes, err := json.Marshal(response)
	if err != nil {
		log.WithError(err).Error("failed to serialize error response to json")
	}
	if err = msg.Respond(bytes); err != nil {
		log.WithError(err).Error("failed to send error response to nats", err)
	}
}
