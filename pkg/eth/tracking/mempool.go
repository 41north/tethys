package tracking

import (
	natsutil "github.com/41north/tethys/pkg/nats"
	"github.com/juju/errors"
	"github.com/nats-io/nats.go"
	cuckoo "github.com/seiflotfy/cuckoofilter"
)

type MemPool interface {
	Add(hash string, tx []byte) error
	Remove(hash string) error
	Close() error
}

func NewNatsMemPool(kv natsutil.KeyValue[[]byte]) (MemPool, error) {
	result := natsMemPool{
		kv: kv,
	}

	if err := result.listen(); err != nil {
		return nil, err
	}

	return &result, nil
}

type natsMemPool struct {
	kv natsutil.KeyValue[[]byte]

	filter  *cuckoo.ScalableCuckooFilter
	watcher natsutil.KeyWatcher[[]byte]
}

func (mp *natsMemPool) Add(hash string, tx []byte) error {
	if !mp.filter.InsertUnique([]byte(hash)) {
		// already present in the kv store, do nothing
		return nil
	}
	_, err := mp.kv.Create(hash, tx)
	// handle error if already exists
	if err != nil {
		return errors.Annotate(err, "failed to write to kv store")
	}
	return nil
}

func (mp *natsMemPool) Remove(hash string) error {
	if !mp.filter.Delete([]byte(hash)) {
		// already removed, do nothing
		return nil
	}
	err := mp.kv.Delete(hash)
	// todo handle error if already deleted
	if err != nil {
		return errors.Annotate(err, "failed to remove from kv store")
	}
	return nil
}

func (mp *natsMemPool) Close() error {
	if mp.watcher != nil {
		return mp.watcher.Stop()
	}
	return nil
}

func (mp *natsMemPool) listen() error {
	// only interested in keys not values
	watcher, err := mp.kv.WatchAll(nats.MetaOnly())
	if err != nil {
		return errors.Annotate(err, "failed to initialise kv watcher")
	}

	// create a cuckoo filter for tracking keys
	filter := cuckoo.NewScalableCuckooFilter()

	go func() {
		for update := range watcher.Updates() {
			key := []byte(update.Key())
			switch update.Operation() {
			case nats.KeyValuePut:
				filter.Insert(key)
			case nats.KeyValueDelete, nats.KeyValuePurge:
				filter.Delete(key)
			}
		}
	}()

	// store for later
	mp.filter = filter
	mp.watcher = watcher

	return nil
}
