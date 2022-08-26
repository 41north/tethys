package proxy

import (
	"context"
	"encoding/json"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/41north/tethys/pkg/eth/tracking"
	"github.com/41north/tethys/pkg/jsonrpc"
	natsutil "github.com/41north/tethys/pkg/nats"
	"github.com/juju/errors"
	"github.com/nats-io/nats.go"
	log "github.com/sirupsen/logrus"
	"github.com/tidwall/btree"
)

type LatestBlockRouter struct {
	conn *nats.Conn

	chain               *tracking.CanonicalChain
	maxDistanceFromHead int

	subjectPrefix string

	clientIdx atomic.Uint64
	clientIds atomic.Value

	log *log.Entry
}

func NewLatestBlockRouter(conn *nats.Conn, chain *tracking.CanonicalChain, maxDistanceFromHead int) natsutil.Router {
	subjectPrefix := natsutil.SubjectName(
		"eth", "rpc",
		strconv.FormatUint(chain.NetworkId, 10),
		strconv.FormatUint(chain.ChainId, 10),
	)

	router := &LatestBlockRouter{
		conn:                conn,
		chain:               chain,
		maxDistanceFromHead: maxDistanceFromHead,
		subjectPrefix:       subjectPrefix,
		log: log.WithFields(log.Fields{
			"component":           "LatestBlockRouter(latest)",
			"maxDistanceFromHead": maxDistanceFromHead,
		}),
	}

	chainUpdates := make(chan *tracking.CanonicalChain, 32)
	chain.AddListener(chainUpdates)

	go router.listenForUpdates(chainUpdates, 100*time.Millisecond)

	return router
}

func (r *LatestBlockRouter) listenForUpdates(updates <-chan *tracking.CanonicalChain, timeout time.Duration) {
	var updatedChain *tracking.CanonicalChain

	for {
		timeAfter := time.After(timeout)

		select {
		case <-timeAfter:
			// timeout has occurred
			if updatedChain != nil {
				r.onUpdate(updatedChain)
				// reset
				updatedChain = nil
			}
		case update, ok := <-updates:
			if !ok {
				// channel has been closed, stop
				r.log.Debug("chain update channel has been closed, no more updates will be processed")
				return
			}
			updatedChain = update
		}
	}
}

func (r *LatestBlockRouter) onUpdate(chain *tracking.CanonicalChain) {
	clientIds := btree.Set[string]{}

	head := chain.Head()
	distanceFromHead := 0

	for head != nil && distanceFromHead <= r.maxDistanceFromHead {
		head.ClientIds.Scan(func(clientId string) bool {
			clientIds.Insert(clientId)
			return true
		})

		head, _ = chain.BlockByHash(head.ParentHash)
		distanceFromHead += 1
	}

	// update the client id set
	r.clientIds.Store(&clientIds)
	r.log.WithField("clients", clientIds.Len()).Debug("processed update")
}

func (r *LatestBlockRouter) nextSubject() (string, error) {
	clientIdRef := r.clientIds.Load()
	if clientIdRef == nil {
		return "", natsutil.ErrNoClientsAvailable
	}

	clientIds := clientIdRef.(*btree.Set[string])
	if clientIds.Len() == 0 {
		return "", natsutil.ErrNoClientsAvailable
	}

	nextIdx := r.clientIdx.Add(1)
	nextIdx = nextIdx % uint64(clientIds.Len())

	clientId, ok := clientIds.GetAt(int(nextIdx))
	if !ok {
		return "", natsutil.ErrNoClientsAvailable
	}

	return natsutil.SubjectName(r.subjectPrefix, clientId), nil
}

func (r *LatestBlockRouter) Request(data []byte, timeout time.Duration) (*nats.Msg, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return r.RequestWithContext(ctx, data)
}

func (r *LatestBlockRouter) RequestMsg(msg *nats.Msg, timeout time.Duration) (*nats.Msg, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return r.RequestMsgWithContext(ctx, msg)
}

func (r *LatestBlockRouter) RequestWithContext(ctx context.Context, data []byte) (*nats.Msg, error) {
	subject, err := r.nextSubject()
	if err != nil {
		return nil, err
	}
	return r.conn.RequestWithContext(ctx, subject, data)
}

func (r *LatestBlockRouter) RequestMsgWithContext(ctx context.Context, msg *nats.Msg) (*nats.Msg, error) {
	subject, err := r.nextSubject()
	if err != nil {
		return nil, err
	}
	// override the msg subject
	msg.Subject = subject
	return r.conn.RequestMsgWithContext(ctx, msg)
}

func (r *LatestBlockRouter) RequestJsonRpc(req *jsonrpc.Request, timeout time.Duration, resp *jsonrpc.Response) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return r.RequestJsonRpcWithContext(ctx, req, resp)
}

func (r *LatestBlockRouter) RequestJsonRpcWithContext(ctx context.Context, req *jsonrpc.Request, resp *jsonrpc.Response) error {
	bytes, err := json.Marshal(req)
	if err != nil {
		return errors.Annotate(err, "failed to marshal request to json")
	}
	msg, err := r.RequestWithContext(ctx, bytes)
	if err != nil {
		return err
	}
	return json.Unmarshal(msg.Data, resp)
}
