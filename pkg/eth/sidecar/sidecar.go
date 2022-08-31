package sidecar

import (
	"context"

	"github.com/41north/tethys/pkg/eth"
	"github.com/juju/errors"

	log "github.com/sirupsen/logrus"
)

// Default Constants
const (
	DefaultClientURL            = "ws://127.0.0.1:8546"
	DefaultClientConnectionType = eth.ConnectionTypeDirect
	DefaultNatsURL              = "ns://127.0.0.1:4222"
	DefaultBucketClientProfile  = "eth_client_profiles"
	DefaultBucketClientStatus   = "eth_client_statuses"
)

type Option func(opts *Options) error

// Options can be used to create a customized connection.
type Options struct {
	ClientUrl            string
	ClientId             *string
	ClientConnectionType eth.ConnectionType

	NatsUrl string

	BucketClientProfile string
	BucketClientStatus  string
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

func BucketClientProfile(bucket string) Option {
	return func(opts *Options) error {
		opts.BucketClientProfile = bucket
		return nil
	}
}

func BucketClientStatus(bucket string) Option {
	return func(opts *Options) error {
		opts.BucketClientStatus = bucket
		return nil
	}
}

func ClientId(id string) Option {
	return func(opts *Options) error {
		opts.ClientId = &id
		return nil
	}
}

func ClientConnectionType(connectionType eth.ConnectionType) Option {
	return func(opts *Options) error {
		if connectionType == -1 {
			return errors.New("invalid connection type")
		}
		opts.ClientConnectionType = connectionType
		return nil
	}
}

// GetDefaultOptions returns default configuration options for the sidecar.
func GetDefaultOptions() Options {
	return Options{
		ClientUrl:            DefaultClientURL,
		ClientConnectionType: DefaultClientConnectionType,
		NatsUrl:              DefaultNatsURL,
		BucketClientProfile:  DefaultBucketClientProfile,
		BucketClientStatus:   DefaultBucketClientStatus,
	}
}

func Run(ctx context.Context, options ...Option) error {
	opts := GetDefaultOptions()
	for _, opt := range options {
		if err := opt(&opts); err != nil {
			return err
		}
	}

	// extra options validation
	if opts.ClientConnectionType == eth.ConnectionTypeManaged {
		if opts.ClientId == nil {
			return errors.New("clientId option must be specified when connection type is managed")
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
