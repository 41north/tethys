package tracking

import (
	"context"
	"sync"
	"sync/atomic"

	log "github.com/sirupsen/logrus"
	"github.com/tidwall/btree"
)

type LoadBalancer interface {
	NetworkId() uint64
	ChainId() uint64
	Channel() chan<- *CanonicalChain
	NextClientId() (string, bool)
	Close()
}

type latest struct {
	log *log.Entry

	networkId           uint64
	chainId             uint64
	maxDistanceFromHead int

	channel chan *CanonicalChain

	clientIdx atomic.Uint64
	clientIds atomic.Value

	mutex sync.RWMutex
}

func NewLatestBalancer(
	networkId uint64,
	chainId uint64,
	maxDistanceFromHead int,
) (LoadBalancer, error) {
	result := latest{
		networkId:           networkId,
		chainId:             chainId,
		maxDistanceFromHead: maxDistanceFromHead,
		channel:             make(chan *CanonicalChain, 8),
		log: log.WithFields(log.Fields{
			"component":           "LoadBalancer(latest)",
			"maxDistanceFromHead": maxDistanceFromHead,
		}),
	}

	go result.run(context.Background())

	return &result, nil
}

func (l *latest) Channel() chan<- *CanonicalChain {
	return l.channel
}

func (l *latest) NetworkId() uint64 {
	return l.networkId
}

func (l *latest) ChainId() uint64 {
	return l.chainId
}

func (l *latest) run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return

		case chain, ok := <-l.channel:

			if chain == nil || !ok {
				break
			}

			clientIds := btree.Set[string]{}

			head := chain.Head()
			distanceFromHead := 0

			for head != nil && distanceFromHead <= l.maxDistanceFromHead {
				head.ClientIds.Scan(func(clientId string) bool {
					clientIds.Insert(clientId)
					return true
				})

				head, _ = chain.blocksByHash.Get(head.ParentHash)
				distanceFromHead += 1
			}

			// update the client id set
			l.clientIds.Store(&clientIds)

			l.log.WithField("clients", clientIds.Len()).Debug("processed update")
		}
	}
}

func (l *latest) NextClientId() (string, bool) {
	clientIdRef := l.clientIds.Load()
	if clientIdRef == nil {
		return "", false
	}

	clientIds := clientIdRef.(*btree.Set[string])
	if clientIds.Len() == 0 {
		return "", false
	}

	nextIdx := l.clientIdx.Add(1)
	nextIdx = nextIdx % uint64(clientIds.Len())

	return clientIds.GetAt(int(nextIdx))
}

func (l *latest) Close() {
}
