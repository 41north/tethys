package nats

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/viney-shih/go-cache"

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

type RouteOpt = func(opts *RouteOpts) error

type RouteOpts struct {
	Cache bool
}

func CacheRoute(cache bool) RouteOpt {
	return func(opts *RouteOpts) error {
		opts.Cache = cache
		return nil
	}
}

func DefaultRouteOpts() RouteOpts {
	return RouteOpts{
		Cache: false,
	}
}

type Router interface {
	Request(req jsonrpc.Request, resp *jsonrpc.Response, timeout time.Duration, options ...RouteOpt) error

	RequestWithContext(ctx context.Context, req jsonrpc.Request, resp *jsonrpc.Response, options ...RouteOpt) error
}

type staticRouter struct {
	resp jsonrpc.Response
}

func (r *staticRouter) Request(req jsonrpc.Request, resp *jsonrpc.Response, timeout time.Duration, options ...RouteOpt) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return r.RequestWithContext(ctx, req, resp, options...)
}

func (r *staticRouter) RequestWithContext(_ context.Context, _ jsonrpc.Request, resp *jsonrpc.Response, _ ...RouteOpt) error {
	resp.Id = r.resp.Id
	resp.JsonRpc = r.resp.JsonRpc
	resp.Result = r.resp.Result
	resp.Error = r.resp.Error
	return nil
}

func NewStaticResult(result any) Router {

	bytes, err := json.Marshal(result)
	if err != nil {
		panic("could not marshal result to json")
	}

	resp := jsonrpc.Response{
		JsonRpc: "2.0",
		Result:  bytes,
	}

	return &staticRouter{resp: resp}
}

func NewStaticError(error jsonrpc.Error) Router {
	resp := jsonrpc.Response{
		JsonRpc: "2.0",
		Error:   &error,
	}
	return &staticRouter{resp: resp}
}

type cachingRouter struct {
	cache       cache.Cache
	cachePrefix string
	router      Router
	log         *log.Entry
}

func (r *cachingRouter) Request(req jsonrpc.Request, resp *jsonrpc.Response, timeout time.Duration, options ...RouteOpt) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return r.RequestWithContext(ctx, req, resp, options...)
}

func (r *cachingRouter) RequestWithContext(ctx context.Context, req jsonrpc.Request, resp *jsonrpc.Response, options ...RouteOpt) error {
	// build options
	opts := DefaultRouteOpts()
	for _, opt := range options {
		if err := opt(&opts); err != nil {
			return err
		}
	}

	if !opts.Cache {
		// caching is disabled for this request
		return r.router.RequestWithContext(ctx, req, resp, options...)
	}

	// unmarshal params array
	var params []any
	err := json.Unmarshal(req.Params, &params)
	if err != nil {
		return errors.Annotate(err, "failed to unmarshal params array")
	}

	// build cache key
	// todo this approach will not work well for requests like eth_call that can have large request params

	sb := strings.Builder{}
	sb.WriteString(req.Method)

	for _, param := range params {
		// todo implement a more efficient approach to constructing the key
		sb.WriteString(fmt.Sprintf("_%v", param))
	}

	//
	key := sb.String()
	l := r.log.WithFields(log.Fields{
		"reqId":     string(req.Id),
		"reqMethod": req.Method,
		"cacheKey":  key,
	})
	l.Debug("loading from cache")
	return r.cache.GetByFunc(ctx, r.cachePrefix, key, resp, func() (interface{}, error) {
		l.Debug("cache miss")
		err := r.router.RequestWithContext(ctx, req, resp)
		return resp, err
	})
}

func NewCachingRouter(cache cache.Cache, cachePrefix string, router Router) Router {
	return &cachingRouter{
		cache:       cache,
		cachePrefix: cachePrefix,
		router:      router,
		log: log.WithFields(log.Fields{
			"component":   "cachingRouter",
			"cachePrefix": cachePrefix,
		}),
	}
}
