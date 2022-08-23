package jsonrpc

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/41north/tethys/pkg/async"

	"github.com/41north/tethys/pkg/util"
	"github.com/google/uuid"
	"github.com/juju/errors"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"

	"github.com/gorilla/websocket"
)

const (
	DefaultMaxInFlightRequests = 32

	stateDisconnected = iota
	stateConnecting
	stateConnected
	stateClosing
	stateClosed

	ErrNotConnected = errors.ConstError("client is not connected")
)

type state int32

type ClientOption func(opts *ClientOptions) error

type ClientOptions struct {
	Dialer *websocket.Dialer

	RequestHeader http.Header

	MaxInFlightRequests int

	CloseHandler func(code int, message string)
}

func Dialer(dialer *websocket.Dialer) ClientOption {
	return func(opts *ClientOptions) error {
		opts.Dialer = dialer
		return nil
	}
}

func RequestHeader(header http.Header) ClientOption {
	return func(opts *ClientOptions) error {
		opts.RequestHeader = header
		return nil
	}
}

func MaxInFlightRequests(max int) ClientOption {
	return func(opts *ClientOptions) error {
		opts.MaxInFlightRequests = max
		return nil
	}
}

func GetDefaultClientOptions() ClientOptions {
	return ClientOptions{
		Dialer:              websocket.DefaultDialer,
		MaxInFlightRequests: DefaultMaxInFlightRequests,
	}
}

func newCloseHandler(conn *websocket.Conn, delegate func(code int, message string)) func(code int, message string) error {
	return func(code int, text string) error {
		// default behaviour copied from websocket package
		message := websocket.FormatCloseMessage(code, "")
		_ = conn.WriteControl(websocket.CloseMessage, message, time.Now().Add(time.Second))

		// call delegate function
		delegate(code, text)

		return nil
	}
}

type invocation struct {
	key  string
	req  *Request
	resp chan util.Result[*Response]
}

func newInvocation(key string, req *Request) invocation {
	return invocation{
		key: key,
		req: req,
		// size 1 to prevent rendezvous and improve throughput
		resp: make(chan util.Result[*Response]),
	}
}

func (i invocation) onError(err error) {
	defer close(i.resp)
	i.resp <- util.NewResultErr[*Response](err)
}

func (i invocation) onResponse(resp *Response) {
	defer close(i.resp)
	i.resp <- util.NewResult(resp)
}

func (i invocation) cancel() {
	i.onError(context.Canceled)
}

type Client struct {
	url  string
	opts ClientOptions

	state *atomic.Int32
	log   *log.Entry

	conn *websocket.Conn

	group       *errgroup.Group
	groupCancel context.CancelFunc

	closeHandler func(code int, message string)

	invocations   chan *invocation
	invocationMap sync.Map

	requests chan<- *Request
}

func NewClient(
	url string,
	options ...ClientOption,
) (*Client, error) {

	// process options
	opts := GetDefaultClientOptions()
	for _, opt := range options {
		if err := opt(&opts); err != nil {
			return nil, err
		}
	}

	state := atomic.Int32{}
	state.Store(stateDisconnected)

	return &Client{
		url:   url,
		opts:  opts,
		state: &state,
		log: log.WithFields(log.Fields{
			"component": "JsonRpcClient",
			"url":       url,
		}),
	}, nil
}

func (c *Client) isConnected() bool {
	return c.state.Load() == stateConnected
}

func (c *Client) setState(state state) {
	c.state.Store(int32(state))
}

func (c *Client) transitionState(from state, to state) bool {
	return c.state.CompareAndSwap(int32(from), int32(to))
}

func (c *Client) Connect(
	requests chan<- *Request,
	closeHandler func(code int, message string),
) error {

	if !c.transitionState(stateDisconnected, stateConnecting) {
		return util.ErrNotClosed
	}

	c.log.Debug("connecting")

	conn, _, err := c.opts.Dialer.Dial(c.url, c.opts.RequestHeader)
	if err != nil {
		c.setState(stateDisconnected)
		c.log.WithFields(log.Fields{"error": err}).Errorf("failed to connect")
		return errors.Annotate(err, "failed to connect")
	}

	//
	c.conn = conn
	c.requests = requests
	c.closeHandler = closeHandler
	c.invocations = make(chan *invocation, c.opts.MaxInFlightRequests)

	//
	group := new(errgroup.Group)
	ctx, cancel := context.WithCancel(context.Background())

	group.Go(func() error {
		return c.read(ctx)
	})

	group.Go(func() error {
		return c.write(ctx)
	})

	c.group = group
	c.groupCancel = cancel

	// update state
	if !c.transitionState(stateConnecting, stateConnected) {
		return util.ErrUnexpectedState
	}

	c.log.Info("connected")

	return nil
}

func (c *Client) write(ctx context.Context) error {
	// websocket connection cannot be used concurrently, we serialise the writes instead
	for {
		select {
		case <-ctx.Done():
			c.log.Debug("stopped writing to websocket")
			return nil
		case inv, ok := <-c.invocations:

			if !ok {
				return nil
			}

			// check the invocation is still valid before sending
			_, valid := c.invocationMap.Load(inv.key)
			if !valid {
				// invocation has been cancelled or timed out, do nothing
				break
			}

			if err := c.conn.WriteJSON(inv.req); err != nil {

				if err == websocket.ErrCloseSent {
					return err
				}

				switch err.(type) {
				case *websocket.CloseError:
					return err
				default:
					log.WithError(err).Error("failed to write request to websocket")
				}
			}

		}
	}
}

func (c *Client) read(ctx context.Context) error {
	for {
		select {

		case <-ctx.Done():
			c.log.Debug("stopped reading from websocket")
			return nil

		default:

			var raw json.RawMessage
			if err := c.conn.ReadJSON(&raw); err != nil {
				switch v := err.(type) {
				case *websocket.CloseError:
					if c.closeHandler != nil {
						c.closeHandler(v.Code, v.Text)
					}
					return err
				default:
					log.WithError(err).Error("failed to read from websocket")
				}
			}

			var rawStr = string(raw)

			// TODO tighten up these assertions
			if strings.Contains(rawStr, "method") {
				// we assume this is a request
				var req Request
				if err := json.Unmarshal(raw, &req); err != nil {
					log.WithError(err).Error("failed to unmarshal request")
					break
				}
				// pass along the server request if a channel is configured
				if c.requests == nil {
					log.Warn("server request received, but no channel has been configured")
				} else {
					c.requests <- &req
				}

			} else {
				// we assume this is a response
				var resp Response
				if err := json.Unmarshal(raw, &resp); err != nil {
					log.WithError(err).Error("failed to unmarshal response")
					break
				}

				// dispatch the response
				c.onResponse(&resp)
			}
		}
	}
}

func (c *Client) onResponse(resp *Response) {
	key, err := keyForResponse(resp)
	if err != nil {
		log.WithError(err).Error("failed to construct key for response")
		return
	}

	value, ok := c.invocationMap.LoadAndDelete(key)
	if !ok {
		// timeout or cancel has occurred
		return
	}

	inv := value.(invocation)
	inv.onResponse(resp)
}

func (c *Client) Invoke(ctx context.Context, req *Request) async.Future[Response] {

	// check if connected
	if !c.isConnected() {
		return async.NewFutureFailed[Response](ErrNotConnected)
	}

	// create a new request with a unique id
	uniqueRequest := Request{
		Method:  req.Method,
		Params:  req.Params,
		JsonRpc: "2.0",
	}

	id, err := uuid.NewUUID()
	if err != nil {
		return async.NewFutureFailed[Response](errors.Annotate(err, "failed to generate a uuid"))
	}

	key := id.String()

	if err = uniqueRequest.WithStringId(key); err != nil {
		return async.NewFutureFailed[Response](errors.Annotate(err, "failed to set request id"))
	}

	// create a new invocation and store for later dispatch
	inv := newInvocation(key, &uniqueRequest)
	c.invocationMap.Store(key, inv)

	// handle a timeout or cancellation
	go func() {
		<-ctx.Done()

		value, ok := c.invocationMap.LoadAndDelete(key)
		if !ok {
			return
		}

		inv := value.(invocation)
		if ctx.Err() != nil {
			inv.onError(err)
		} else {
			inv.cancel()
		}
	}()

	// schedule the invocation
	c.invocations <- &inv

	// convert invocation to a future
	return async.NewFuture(inv.resp)
}

// Close stops request processing and releases resources.
// It is idempotent.
func (c *Client) Close() {
	c.log.Debug("close called")
	if !c.transitionState(stateConnected, stateClosing) {
		return // do nothing
	}

	c.log.Debug("closing")

	close(c.invocations)

	// stop the processing loop
	c.log.Debug("stopping processing loop")
	c.groupCancel()
	if err := c.group.Wait(); err != nil {
		c.log.WithError(err).Error("failure whilst waiting for processing to finish")
	}
	c.log.Debug("processing loop stopped")

	// cancel any in flight invocations
	c.log.Debug("cancelling invocations")
	c.invocationMap.Range(func(key, value any) bool {
		inv := value.(invocation)
		inv.cancel()
		return true
	})
	c.log.Debug("invocations cancelled")

	// close the websocket connection
	if err := c.conn.Close(); err != nil {
		c.log.WithError(err).Error("failure whilst closing websocket")
	}

	c.log.Debug("closed")
	c.transitionState(stateClosing, stateClosed)
}

func keyForResponse(resp *Response) (string, error) {
	id, err := resp.UnmarshalId()
	if err != nil {
		return "", errors.Annotate(err, "failed to unmarshal id")
	}
	return keyForId(id)
}

func keyForId(id any) (key string, err error) {
	// todo is there a more performant way of doing this conversion?
	switch v := id.(type) {
	case
		int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		key = fmt.Sprintf("%d", v)
	case string:
		key = v
	default:
		err = errors.Errorf("id must be an integer or a string, received: %v", id)
	}

	return key, err
}
