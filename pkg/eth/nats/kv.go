package nats

import (
	"fmt"

	"github.com/41north/tethys/pkg/eth"
	natsutil "github.com/41north/tethys/pkg/nats"
	"github.com/juju/errors"
	"github.com/nats-io/nats.go"
)

type Option = func(opts *Options) error

func Create(create bool) Option {
	return func(opts *Options) error {
		opts.Create = create
		return nil
	}
}

func NetworkAndChainId(networkId uint64, chainId uint64) Option {
	return func(opts *Options) error {
		opts.NetworkId = networkId
		opts.ChainId = chainId
		return nil
	}
}

func BucketStatuses(name string) Option {
	return func(opts *Options) error {
		opts.BucketStatuses = name
		return nil
	}
}

func StatusHistory(history uint8) Option {
	return func(opts *Options) error {
		opts.StatusHistory = history
		return nil
	}
}

func BucketProfiles(name string) Option {
	return func(opts *Options) error {
		opts.BucketProfiles = name
		return nil
	}
}

type Options struct {
	Create bool

	NetworkId uint64
	ChainId   uint64

	StatusHistory uint8

	BucketStatuses string
	BucketProfiles string
}

func GetDefaultOptions() Options {
	return Options{
		Create:         false,
		NetworkId:      1,
		ChainId:        1,
		BucketStatuses: "eth_client_statuses",
		StatusHistory:  12,
		BucketProfiles: "eth_client_profiles",
	}
}

type StatusStore = natsutil.KeyValue[eth.ClientStatus]

type ProfileStore = natsutil.KeyValue[eth.ClientProfile]

type StateManager struct {
	Opts     Options
	Status   StatusStore
	Profiles ProfileStore
}

func NewStateManager(js nats.JetStreamContext, options ...Option) (*StateManager, error) {
	opts := GetDefaultOptions()
	for _, option := range options {
		if err := option(&opts); err != nil {
			return nil, err
		}
	}

	statusStore, err := initStatusStore(js, opts)
	if err != nil {
		return nil, errors.Annotate(err, "failed to init status store")
	}

	profileStore, err := initProfileStore(js, opts)
	if err != nil {
		return nil, errors.Annotate(err, "failed to init profile store")
	}

	return &StateManager{
		Opts:     opts,
		Status:   statusStore,
		Profiles: profileStore,
	}, nil
}

func initStatusStore(js nats.JetStreamContext, opts Options) (StatusStore, error) {
	bucket := fmt.Sprintf("%s_%d_%d", opts.BucketStatuses, opts.NetworkId, opts.ChainId)

	if !opts.Create {
		return natsutil.GetKeyValue[eth.ClientStatus](js, bucket)
	}

	return natsutil.CreateKeyValue[eth.ClientStatus](js, &nats.KeyValueConfig{
		Bucket:  bucket,
		History: opts.StatusHistory,
	})
}

func initProfileStore(js nats.JetStreamContext, opts Options) (ProfileStore, error) {
	bucket := fmt.Sprintf("%s_%d_%d", opts.BucketProfiles, opts.NetworkId, opts.ChainId)

	if !opts.Create {
		return natsutil.GetKeyValue[eth.ClientProfile](js, bucket)
	}

	return natsutil.CreateKeyValue[eth.ClientProfile](js, &nats.KeyValueConfig{
		Bucket: bucket,
	})
}
