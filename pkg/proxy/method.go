package proxy

import (
	"github.com/41north/tethys/pkg/jsonrpc"
	natsutil "github.com/41north/tethys/pkg/nats"
	"github.com/juju/errors"
)

type MethodOpt = func(opts *MethodOpts) error

type MethodOpts struct {
	routeOpts     []natsutil.RouteOpt
	beforeRequest jsonrpc.RequestTransform
	afterResponse jsonrpc.ResponseTransform
}

func RouteOpts(routeOpts ...natsutil.RouteOpt) MethodOpt {
	return func(opts *MethodOpts) error {
		opts.routeOpts = routeOpts
		return nil
	}
}

func BeforeRequest(transform jsonrpc.RequestTransform) MethodOpt {
	return func(opts *MethodOpts) error {
		opts.beforeRequest = transform
		return nil
	}
}

func AfterResponse(transform jsonrpc.ResponseTransform) MethodOpt {
	return func(opts *MethodOpts) error {
		opts.afterResponse = transform
		return nil
	}
}

func DefaultMethodOpts() MethodOpts {
	return MethodOpts{
		// by default no caching
		routeOpts: []natsutil.RouteOpt{natsutil.CacheRoute(false)},
	}
}

type Method interface {
	Name() string
	Router() natsutil.Router
	RouteOpts() []natsutil.RouteOpt
	BeforeRequest(req jsonrpc.Request) (jsonrpc.Request, error)
	AfterResponse(resp *jsonrpc.Response) error
}

type method struct {
	name   string
	router natsutil.Router
	opts   MethodOpts
}

func NewMethod(name string, router natsutil.Router, options ...MethodOpt) Method {
	opts := DefaultMethodOpts()
	for _, opt := range options {
		if err := opt(&opts); err != nil {
			// panic instead of returning error to keep return object clean for static configuration
			panic(errors.Annotate(err, "bad proxy method config"))
		}
	}
	return &method{
		name: name, router: router, opts: opts,
	}
}

func (m method) Name() string {
	return m.name
}

func (m method) Router() natsutil.Router {
	return m.router
}

func (m method) RouteOpts() []natsutil.RouteOpt {
	return m.opts.routeOpts
}

func (m method) BeforeRequest(req jsonrpc.Request) (jsonrpc.Request, error) {
	if m.opts.beforeRequest == nil {
		return req, nil
	} else {
		return m.opts.beforeRequest(req)
	}
}

func (m method) AfterResponse(resp *jsonrpc.Response) error {
	if m.opts.afterResponse == nil {
		return nil
	} else {
		return m.opts.afterResponse(resp)
	}
}
