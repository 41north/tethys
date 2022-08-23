package nats

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/41north/tethys/pkg/async"
	"github.com/41north/tethys/pkg/util"
	"github.com/nats-io/nats.go"
	"github.com/viney-shih/go-cache"

	"github.com/juju/errors"
)

var sanitizeKeyRegex = regexp.MustCompile(`[^-/_=.a-zA-Z\d]+`)

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

type CacheEntry struct{}

type kvCacheAdapter struct {
	kv nats.KeyValue
}

func (c kvCacheAdapter) sanitizeKey(key string) string {
	prefix := fmt.Sprintf("ca:%s:", c.kv.Bucket())
	// remove the prefix
	result := strings.ReplaceAll(key, prefix, "")
	// replace any invalid characters that remain
	result = sanitizeKeyRegex.ReplaceAllString(result, "_")
	return result
}

func (c kvCacheAdapter) MGet(ctx context.Context, keys []string) ([]cache.Value, error) {
	resultsCh := make(chan util.Result[cache.Value])

	go func() {
		for _, key := range keys {
			select {
			case <-ctx.Done():
				resultsCh <- util.NewResultErr[cache.Value](ctx.Err())
			default:
				entry, err := c.kv.Get(c.sanitizeKey(key))
				if err != nil {
					resultsCh <- util.NewResultErr[cache.Value](err)
					continue
				}
				value := cache.Value{Valid: entry != nil}
				if entry != nil {
					value.Bytes = entry.Value()
				}
				resultsCh <- util.NewResult[cache.Value](&value)
			}
		}
		close(resultsCh)
	}()

	results, err := async.NewFuture[cache.Value](resultsCh).AwaitAll(ctx)
	if err != nil {
		return nil, errors.Annotate(err, "failed to get all keys")
	}

	var values []cache.Value
	for _, result := range results {
		v, _ := result.Value()
		//if err != nil {
		//	return nil, errors.Annotate(err, "failed to get all keys")
		//}

		if v == nil {
			v = &cache.Value{Valid: false}
		}

		values = append(values, *v)
	}

	return values, nil
}

func (c kvCacheAdapter) MSet(ctx context.Context, keyValues map[string][]byte, _ time.Duration, _ ...cache.MSetOptions) error {
	for key, value := range keyValues {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			_, err := c.kv.Put(c.sanitizeKey(key), value)
			if err != nil {
				return errors.Annotate(err, "failed to write value to kv store")
			}
		}
	}
	return nil
}

func (c kvCacheAdapter) Del(ctx context.Context, keys ...string) error {
	for _, key := range keys {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			err := c.kv.Delete(c.sanitizeKey(key))
			if err != nil {
				return errors.Annotate(err, "failed to delete from kv store")
			}
		}
	}
	return nil
}

func NewCache[V any](
	name string,
	localCacheSize int,
	ttl time.Duration,
	js nats.JetStreamContext,
) cache.Cache {
	kv, err := js.KeyValue(name)
	// TODO for now we assume any error means the kv store doesn't exist
	// TODO there are storage types and replica settings which we currently don't pass
	// TODO how do we evolve cache settings?
	if err != nil {
		kv, err = js.CreateKeyValue(&nats.KeyValueConfig{
			Bucket: name,
			TTL:    ttl,
		})
	}

	localCache := cache.NewTinyLFU(localCacheSize)
	sharedCache := kvCacheAdapter{kv: kv}

	factory := cache.NewFactory(sharedCache, localCache)
	return factory.NewCache([]cache.Setting{
		{
			Prefix: name, // todo what's the correct mapping for this?
			MarshalFunc: func(value interface{}) ([]byte, error) {
				return json.Marshal(value)
			},
			UnmarshalFunc: func(bytes []byte, value interface{}) error {
				return json.Unmarshal(bytes, value)
			},
			CacheAttributes: map[cache.Type]cache.Attribute{
				cache.LocalCacheType:  {TTL: ttl}, // match the overall ttl for now
				cache.SharedCacheType: {TTL: ttl}, // has no effect, but we set for consistency
			},
		},
	})
}
