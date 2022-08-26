package sidecar

import (
	"context"

	log "github.com/sirupsen/logrus"
)

// Default Constants
const (
	DefaultClientURL           = "ws://127.0.0.1:8546"
	DefaultNatsURL             = "ns://127.0.0.1:4222"
	DefaultClientProfileBucket = "eth_client_profiles"
	DefaultClientStatusBucket  = "eth_client_statuses"
)

type Option func(opts *Options) error

// Options can be used to create a customized connection.
type Options struct {
	// Websocket url for connecting to a eth client
	ClientUrl string

	// NATS server url
	NatsUrl string

	ClientProfileBucket string

	ClientStatusBucket string
}

func ClientUrl(url string) Option {
	return func(opts *Options) error {
		opts.ClientUrl = url
		return nil
	}
}

func NatsUrl(url string) Option {
	return func(opts *Options) error {
		opts.NatsUrl = url
		return nil
	}
}

func ClientProfileBucket(bucket string) Option {
	return func(opts *Options) error {
		opts.ClientProfileBucket = bucket
		return nil
	}
}

func ClientStatusBucket(bucket string) Option {
	return func(opts *Options) error {
		opts.ClientStatusBucket = bucket
		return nil
	}
}

// GetDefaultOptions returns default configuration options for the sidecar.
func GetDefaultOptions() Options {
	return Options{
		ClientUrl:           DefaultClientURL,
		NatsUrl:             DefaultNatsURL,
		ClientProfileBucket: DefaultClientProfileBucket,
		ClientStatusBucket:  DefaultClientStatusBucket,
	}
}

func Run(ctx context.Context, options ...Option) error {
	opts := GetDefaultOptions()
	for _, opt := range options {
		if err := opt(&opts); err != nil {
			return err
		}
	}

	if err := connectNats(opts); err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			closeNats()
			return nil

		default:

			// keep trying to initiate a session until the ctx is done
			session := newClientSession(opts)
			if err := session.connect(ctx); err != nil {
				log.WithError(err).Error("client session connect failed")
			}

		}
	}
}
