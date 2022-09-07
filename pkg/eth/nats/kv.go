package nats

import (
	"fmt"

	"github.com/41north/tethys/pkg/jsonrpc"

	"github.com/41north/tethys/pkg/eth"
	natsutil "github.com/41north/tethys/pkg/nats"
	"github.com/juju/errors"
	"github.com/nats-io/nats.go"
)

type Option = func(opts *Options) error

func NetworkAndChainId(networkId uint64, chainId uint64) Option {
	return func(opts *Options) error {
		opts.NetworkId = networkId
		opts.ChainId = chainId
		return nil
	}
}

func BucketStatusesHistory(history uint8) Option {
	return func(opts *Options) error {
		opts.BucketConfigStatuses.History = history
		return nil
	}
}

func BucketStatusesFormat(name string) Option {
	return func(opts *Options) error {
		opts.BucketConfigStatuses.Format = name
		return nil
	}
}

func BucketProfilesFormat(name string) Option {
	return func(opts *Options) error {
		opts.BucketConfigProfiles.Format = name
		return nil
	}
}

type bucketConfigStatuses struct {
	Format  string
	History uint8
}

type bucketConfigProfiles struct {
	Format string
}

type Options struct {
	NetworkId uint64
	ChainId   uint64

	BucketConfigStatuses bucketConfigStatuses
	BucketConfigProfiles bucketConfigProfiles
}

func GetDefaultOptions() Options {
	return Options{
		NetworkId: 1,
		ChainId:   1,

		BucketConfigStatuses: bucketConfigStatuses{
			Format:  "eth_%d_%d_client_status",
			History: 12,
		},

		BucketConfigProfiles: bucketConfigProfiles{
			Format: "eth_%d_%d_client_profile",
		},
	}
}

type StatusStore = natsutil.KeyValue[eth.ClientStatus]

type ProfileStore = natsutil.KeyValue[eth.ClientProfile]

type ResponseStore = natsutil.KeyValue[jsonrpc.Response]

type StateManager struct {
	Opts      Options
	Status    StatusStore
	Profiles  ProfileStore
	Responses ResponseStore
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

	responseStore, err := initResponseStore(js, opts)
	if err != nil {
		return nil, errors.Annotate(err, "failed to init response store")
	}

	return &StateManager{
		Opts:      opts,
		Status:    statusStore,
		Profiles:  profileStore,
		Responses: responseStore,
	}, nil
}

func initStatusStore(js nats.JetStreamContext, opts Options) (StatusStore, error) {
	config := opts.BucketConfigStatuses
	bucket := fmt.Sprintf(config.Format, opts.NetworkId, opts.ChainId)
	return natsutil.GetKeyValue[eth.ClientStatus](js, bucket)
}

func initProfileStore(js nats.JetStreamContext, opts Options) (ProfileStore, error) {
	bucket := fmt.Sprintf(opts.BucketConfigProfiles.Format, opts.NetworkId, opts.ChainId)
	return natsutil.GetKeyValue[eth.ClientProfile](js, bucket)
}

func initResponseStore(js nats.JetStreamContext, opts Options) (ResponseStore, error) {
	bucket := fmt.Sprintf("eth_%d_%d_proxy_responses", opts.NetworkId, opts.ChainId)
	return natsutil.GetKeyValue[jsonrpc.Response](js, bucket)
}
