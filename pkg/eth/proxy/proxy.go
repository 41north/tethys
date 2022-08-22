package proxy

import (
	"context"
	"net/url"
)

const (
	DefaultAddress             = ":8080"
	DefaultNetworkId           = uint64(1)
	DefaultChainId             = uint64(1)
	DefaultNatsUrl             = "ns://127.0.0.1:4222"
	DefaultClientStatusBucket  = "eth_client_statuses"
	DefaultClientProfileBucket = "eth_client_profiles"
	DefaultMaxDistanceFromHead = 3
)

type Option func(opts *Options) error

type Options struct {
	Address string

	NetworkId uint64

	ChainId uint64

	NatsUrl string

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
		Address:             DefaultAddress,
		NetworkId:           DefaultNetworkId,
		ChainId:             DefaultChainId,
		NatsUrl:             DefaultNatsUrl,
		ClientStatusBucket:  DefaultClientStatusBucket,
		ClientProfileBucket: DefaultClientProfileBucket,
		MaxDistanceFromHead: DefaultMaxDistanceFromHead,
	}
}

func ListenAndServe(ctx context.Context, options ...Option) error {
	opts := GetDefaultOptions()
	for _, opt := range options {
		if err := opt(&opts); err != nil {
			return err
		}
	}

	if err := connectNats(opts); err != nil {
		return err
	}
	defer closeNats()

	if err := startTracking(opts); err != nil {
		return err
	}
	defer stopTracking()

	return listenAndServe(ctx, opts)
}
