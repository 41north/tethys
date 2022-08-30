package proxy

import (
	"context"
	"net/url"

	"github.com/juju/errors"
)

const (
	DefaultAddress                = ":8080"
	DefaultNetworkId              = uint64(1)
	DefaultChainId                = uint64(1)
	DefaultNatsUrl                = "ns://127.0.0.1:4222"
	DefaultNatsEmbedded           = false
	DefaultNatsEmbeddedConfigPath = ""
	DefaultClientStatusBucket     = "eth_client_statuses"
	DefaultClientProfileBucket    = "eth_client_profiles"
	DefaultMaxDistanceFromHead    = 3
)

type Option func(opts *Options) error

type Options struct {
	Address string

	NetworkId uint64

	ChainId uint64

	NatsUrl string

	NatsEmbedded bool

	NatsEmbeddedUseDefaultConfig bool

	NatsEmbeddedConfigPath string

	ClientStatusBucket string

	ClientProfileBucket string

	MaxDistanceFromHead int
}

func Address(addr string) Option {
	return func(opts *Options) error {
		opts.Address = addr
		return nil
	}
}

func NetworkId(networkId uint64) Option {
	return func(opts *Options) error {
		opts.NetworkId = networkId
		return nil
	}
}

func ChainId(chainId uint64) Option {
	return func(opts *Options) error {
		opts.ChainId = chainId
		return nil
	}
}

func NatsUrl(url *url.URL) Option {
	return func(opts *Options) error {
		opts.NatsUrl = url.String()
		return nil
	}
}

func NatsEmbedded(embed bool) Option {
	return func(opts *Options) error {
		opts.NatsEmbedded = embed
		return nil
	}
}

func NatsEmbeddedConfigPath(path string) Option {
	return func(opts *Options) error {
		opts.NatsEmbeddedConfigPath = path
		return nil
	}
}

func ClientStatusBucket(bucket string) Option {
	return func(opts *Options) error {
		opts.ClientStatusBucket = bucket
		return nil
	}
}

func ClientProfileBucket(bucket string) Option {
	return func(opts *Options) error {
		opts.ClientProfileBucket = bucket
		return nil
	}
}

func GetDefaultOptions() Options {
	return Options{
		Address:                DefaultAddress,
		NetworkId:              DefaultNetworkId,
		ChainId:                DefaultChainId,
		NatsUrl:                DefaultNatsUrl,
		NatsEmbedded:           DefaultNatsEmbedded,
		NatsEmbeddedConfigPath: DefaultNatsEmbeddedConfigPath,
		ClientStatusBucket:     DefaultClientStatusBucket,
		ClientProfileBucket:    DefaultClientProfileBucket,
		MaxDistanceFromHead:    DefaultMaxDistanceFromHead,
	}
}

func ListenAndServe(ctx context.Context, options ...Option) error {
	opts := GetDefaultOptions()
	for _, opt := range options {
		if err := opt(&opts); err != nil {
			return err
		}
	}

	if err := startNatsServer(opts); err != nil {
		return errors.Annotate(err, "failed to start NATS server")
	}
	defer closeNatsServer() // stop embedded server (if applicable)

	if err := connectNats(opts); err != nil {
		return errors.Annotate(err, "failed to initialise NATS")
	}
	defer closeNats() // stop connection to server first

	if err := InitRouter(opts); err != nil {
		return errors.Annotate(err, "failed to initialise router")
	}
	defer closeRouter()

	return listenAndServe(ctx, opts)
}
