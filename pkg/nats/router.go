package nats

import (
	"context"
	"github.com/41north/go-async"
	"strings"
	"time"

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
	Request(method string, params any, response *any, timeout time.Duration, options ...RouteOpt) error

	RequestWithContext(ctx context.Context, method string, params any, response *any, options ...RouteOpt) error
}

type staticRouter struct {
	result async.Result[any]
}

func (r *staticRouter) Request(method string, params any, response *any, timeout time.Duration, options ...RouteOpt) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return r.RequestWithContext(ctx, method, params, response, options...)
}

func (r *staticRouter) RequestWithContext(_ context.Context, _ string, _ any, resp *any, _ ...RouteOpt) error {
	v, err := r.result.Unwrap()
	if err != nil {
		return err
	}
	resp = &v
	return nil
}

func NewStaticResult(result async.Result[any]) Router {
	return &staticRouter{result: result}
}
