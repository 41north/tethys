package proxy

import (
	"context"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/41north/tethys/pkg/eth"
	natseth "github.com/41north/tethys/pkg/eth/nats"
	"github.com/viney-shih/go-cache"

	proxymethods "github.com/41north/tethys/pkg/eth/proxy/methods"
	"github.com/41north/tethys/pkg/proxy"
	"github.com/juju/errors"

	"github.com/41north/tethys/pkg/eth/tracking"
	"github.com/41north/tethys/pkg/jsonrpc"
	natsutil "github.com/41north/tethys/pkg/nats"
	"github.com/nats-io/nats.go"
	log "github.com/sirupsen/logrus"
	"github.com/tidwall/btree"
)

var (
	canonicalChain    *tracking.CanonicalChain
	latestBlockRouter natsutil.Router
	cachingRouter     natsutil.Router

	proxyMethods map[string]proxy.Method
)

func InitRouter(opts Options) error {
	var err error

	watcher, err := stateManager.Status.WatchAll()
	if err != nil {
		return errors.Annotate(err, "failed to create client status watcher")
	}

	canonicalChain, err = tracking.NewCanonicalChain(opts.NetworkId, opts.ChainId, watcher.Updates(), 12)
	if err != nil {
		return errors.Annotate(err, "failed to create canonical chain tracker")
	}

	profileCache := natsutil.NewCache[eth.ClientProfile](
		1024,
		stateManager.Profiles,
		24*time.Hour,
	)

	latestBlockRouter = NewLatestBlockRouter(natsConn, canonicalChain, stateManager.Profiles, profileCache, 0)

	canonicalChain.Start()

	// init the response cache
	respCache := natsutil.NewCache[jsonrpc.Response](
		1024*10,
		stateManager.Responses,
		1*time.Hour,
	)

	// create a caching router backed by the latest block router
	cachingRouter = natsutil.NewCachingRouter(respCache, stateManager.Responses.Bucket(), latestBlockRouter)

	// construct a map of supported methods
	proxyMethods, err = proxymethods.Build(canonicalChain, cachingRouter)

	return err
}

func closeRouter() {
	canonicalChain.Close()
}

func invoke(ctx context.Context, req jsonrpc.Request, resp *jsonrpc.Response) {
	// set the resp id to match the request
	resp.Id = req.Id
	resp.JsonRpc = "2.0"

	// check if the method is supported
	method, ok := proxyMethods[req.Method]
	if !ok {
		// todo make a const error for this
		errorResponse(errors.New("method not supported"), resp)
		return
	}

	var err error
	req, err = method.BeforeRequest(req)
	if err != nil {
		errorResponse(errors.Annotate(err, "failed to apply request transform"), resp)
		return
	}

	if err = method.Router().RequestWithContext(ctx, req, resp, method.RouteOpts()...); err != nil {
		errorResponse(err, resp)
		return
	}

	if err = method.AfterResponse(resp); err != nil {
		errorResponse(err, resp)
		return
	}
}

func errorResponse(err error, resp *jsonrpc.Response) {
	// todo sanitize errors and distinguish between error types
	resp.Error = &jsonrpc.Error{
		Code:    -326000,
		Message: err.Error(),
	}
}

type currentClients struct {
	connectionType      eth.ConnectionType
	clientsByConnection map[eth.ConnectionType]*btree.Set[string]
}

func (cc currentClients) clientIds() *btree.Set[string] {
	return cc.clientsByConnection[cc.connectionType]
}

func (cc currentClients) String() string {
	var sb strings.Builder
	sb.WriteString("currentClients(")
	sb.WriteString(cc.connectionType.String())
	sb.WriteString("){ ")
	for key, value := range cc.clientsByConnection {
		if key > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(key.String())
		sb.WriteString(" -> ")
		sb.WriteString(strconv.Itoa(value.Len()))
	}
	sb.WriteString("}")
	return sb.String()
}

type LatestBlockRouter struct {
	conn *nats.EncodedConn

	chain               *tracking.CanonicalChain
	maxDistanceFromHead int

	subjectPrefix string

	clientIdx      atomic.Uint64
	currentClients atomic.Value

	profileStore natseth.ProfileStore
	profileCache cache.Cache

	log *log.Entry
}

func NewLatestBlockRouter(
	conn *nats.EncodedConn,
	chain *tracking.CanonicalChain,
	profileStore natseth.ProfileStore,
	profileCache cache.Cache,
	maxDistanceFromHead int,
) natsutil.Router {
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
		profileStore:        profileStore,
		profileCache:        profileCache,
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

func (r *LatestBlockRouter) getClientProfile(id string) (*eth.ClientProfile, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var profile eth.ClientProfile
	err := r.profileCache.GetByFunc(ctx, r.profileStore.Bucket(), id, &profile, func() (interface{}, error) {
		entry, err := r.profileStore.Get(id)
		if err != nil {
			return nil, err
		}
		return entry.Value()
	})
	return &profile, err
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
	clientsByConnection := make(map[eth.ConnectionType]*btree.Set[string])

	head := chain.Head()
	distanceFromHead := 0

	for head != nil && distanceFromHead <= r.maxDistanceFromHead {
		head.ClientIds.Scan(func(clientId string) bool {
			profile, err := r.getClientProfile(clientId)
			if err != nil {
				r.log.WithError(err).WithField("clientId", clientId).Error("failed to load client profile")
				return true
			}

			clientIds, ok := clientsByConnection[profile.ConnectionType]
			if !ok {
				clientIds = &btree.Set[string]{}
				clientsByConnection[profile.ConnectionType] = clientIds
			}

			clientIds.Insert(clientId)
			return true
		})

		head, _ = chain.BlockByHash(head.ParentHash)
		distanceFromHead += 1
	}

	var update currentClients

	if len(clientsByConnection) == 0 {
		r.currentClients.Store(nil)
	} else {
		// determine the preferred connection type
		var preferredConnection eth.ConnectionType
		for _, connectionType := range eth.ConnectionTypes {
			_, ok := clientsByConnection[connectionType]
			if ok {
				preferredConnection = connectionType
				break
			}
		}

		update = currentClients{
			connectionType:      preferredConnection,
			clientsByConnection: clientsByConnection,
		}
		r.currentClients.Store(update)
	}

	r.log.WithField("clients", update).Debug("processed update")
}

func (r *LatestBlockRouter) nextSubject() (string, error) {
	currentClientsRef := r.currentClients.Load()
	if currentClientsRef == nil {
		return "", natsutil.ErrNoClientsAvailable
	}

	currentClients := currentClientsRef.(currentClients)
	clientIds := currentClients.clientIds()

	nextIdx := r.clientIdx.Add(1)
	nextIdx = nextIdx % uint64(clientIds.Len())

	clientId, ok := clientIds.GetAt(int(nextIdx))
	if !ok {
		return "", natsutil.ErrNoClientsAvailable
	}

	return natsutil.SubjectName(r.subjectPrefix, clientId), nil
}

func (r *LatestBlockRouter) Request(req jsonrpc.Request, resp *jsonrpc.Response, timeout time.Duration, options ...natsutil.RouteOpt) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return r.RequestWithContext(ctx, req, resp, options...)
}

func (r *LatestBlockRouter) RequestWithContext(ctx context.Context, req jsonrpc.Request, resp *jsonrpc.Response, _ ...natsutil.RouteOpt) error {
	subject, err := r.nextSubject()
	if err != nil {
		return err
	}
	return r.conn.RequestWithContext(ctx, subject, req, resp)
}
