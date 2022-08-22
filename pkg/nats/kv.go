package nats

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/juju/errors"
	"github.com/nats-io/nats.go"
)

type KeyValueEntry[T any] interface {
	// Bucket is the bucket the data was loaded from.
	Bucket() string
	// Key is the key that was retrieved.
	Key() string
	// Value is the retrieved value unmarshalled from json.
	Value() (T, error)
	// ValueRaw is the original []byte value.
	ValueRaw() []byte
	// Revision is a unique sequence for this value.
	Revision() uint64
	// Created is the time the data was put in the bucket.
	Created() time.Time
	// Delta is distance from the latest value.
	Delta() uint64
	// Operation returns Put or Delete or Purge.
	Operation() nats.KeyValueOp
}

type kve[T any] struct {
	delegate nats.KeyValueEntry

	value        T
	err          error
	unmarshalled bool
	mutex        sync.RWMutex
}

func (e kve[T]) Bucket() string             { return e.delegate.Bucket() }
func (e kve[T]) Key() string                { return e.delegate.Key() }
func (e kve[T]) ValueRaw() []byte           { return e.delegate.Value() }
func (e kve[T]) Revision() uint64           { return e.delegate.Revision() }
func (e kve[T]) Created() time.Time         { return e.delegate.Created() }
func (e kve[T]) Delta() uint64              { return e.delegate.Delta() }
func (e kve[T]) Operation() nats.KeyValueOp { return e.delegate.Operation() }

func (e kve[T]) Value() (T, error) {

	// first we check if the values have already been unmarshalled
	e.mutex.RLock()
	if e.unmarshalled {
		defer e.mutex.RUnlock()
		// return cached values
		return e.value, e.err
	}

	// if not we unmarshal the raw bytes returned from nats and cache the results
	e.mutex.RUnlock()
	e.mutex.Lock()
	defer e.mutex.Unlock()

	var value T
	err := json.Unmarshal(e.delegate.Value(), &value)

	e.value = value
	e.err = err

	return value, err
}

type KeyWatcher[T any] interface {
	// Context returns watcher context optionally provided by nats.Context option.
	Context() context.Context
	// Updates returns a channel to read any updates to entries.
	Updates() <-chan KeyValueEntry[T]
	// Stop will stop this watcher.
	Stop() error
}

type kw[T any] struct {
	delegate nats.KeyWatcher
}

func (w kw[T]) Context() context.Context {
	return w.delegate.Context()
}

func (w kw[T]) Updates() <-chan KeyValueEntry[T] {
	// channel size matches the nats implementation
	ch := make(chan KeyValueEntry[T], 256)

	// todo maybe this can be done by a shared errgroup or a single go routine?
	go func() {
		for entry := range w.delegate.Updates() {
			// TODO find out why we get a nil sometimes
			if entry == nil {
				continue
			}
			ch <- kve[T]{delegate: entry}
		}
	}()

	return ch
}

func (w kw[T]) Stop() error {
	return w.delegate.Stop()
}

type KeyValue[T any] interface {
	Get(key string) (KeyValueEntry[T], error)

	Put(key string, value T) (uint64, error)

	Watch(key string, opts ...nats.WatchOpt) (KeyWatcher[T], error)

	WatchAll(opts ...nats.WatchOpt) (KeyWatcher[T], error)

	Delete(key string) error
}

func CreateKeyValue[T any](js nats.JetStreamContext, cfg *nats.KeyValueConfig) (KeyValue[T], error) {
	keyValue, err := js.CreateKeyValue(cfg)
	if err != nil {
		return nil, errors.Annotatef(err, "could not create kv store with bucket = %s", cfg.Bucket)
	}
	return kv[T]{keyValue}, nil
}

func GetKeyValue[T any](js nats.JetStreamContext, bucket string) (KeyValue[T], error) {
	keyValue, err := js.KeyValue(bucket)
	if err != nil {
		return nil, errors.Annotatef(err, "could not retrieve kv store with bucket = %s", bucket)
	}
	return kv[T]{keyValue}, nil
}

type kv[T any] struct {
	kv nats.KeyValue
}

func (s kv[T]) Get(key string) (KeyValueEntry[T], error) {
	entry, err := s.kv.Get(key)
	return &kve[T]{delegate: entry}, err
}

func (s kv[T]) Put(key string, value T) (uint64, error) {
	bytes, err := json.Marshal(value)
	if err != nil {
		return 0, errors.Annotate(err, "failed to marshal value to json")
	}
	return s.kv.Put(key, bytes)
}

func (s kv[T]) Delete(key string) error {
	return s.kv.Delete(key)
}

func (s kv[T]) Watch(key string, opts ...nats.WatchOpt) (KeyWatcher[T], error) {
	watcher, err := s.kv.Watch(key, opts...)
	return kw[T]{delegate: watcher}, err
}

func (s kv[T]) WatchAll(opts ...nats.WatchOpt) (KeyWatcher[T], error) {
	watcher, err := s.kv.WatchAll(opts...)
	return kw[T]{delegate: watcher}, err
}
