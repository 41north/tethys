package tracking

import (
	"context"
	"math/big"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/41north/tethys/pkg/eth"
	natsutil "github.com/41north/tethys/pkg/nats"
	"github.com/41north/tethys/pkg/util"
	"github.com/nats-io/nats.go"
	log "github.com/sirupsen/logrus"
	"github.com/tidwall/btree"
)

type CanonicalChain struct {
	networkId           uint64
	chainId             uint64
	maxDistanceFromHead int

	log *log.Entry

	head         atomic.Value
	blocksByHash btree.Map[string, *Block]

	wg     *sync.WaitGroup
	cancel context.CancelFunc

	updates <-chan natsutil.KeyValueEntry[eth.ClientStatus]

	listeners []chan<- *CanonicalChain
}

func (cc CanonicalChain) Head() *Block {
	head := cc.head.Load()
	if head == nil {
		return nil
	} else {
		return head.(*Block)
	}
}

func (cc *CanonicalChain) AddListener(ch chan<- *CanonicalChain) {
	cc.listeners = append(cc.listeners, ch)
}

func (cc CanonicalChain) String() string {
	var sb strings.Builder
	block := cc.Head()
	for block != nil {

		sb.WriteString("(")
		sb.WriteString(block.Number.String())
		sb.WriteString(", ")
		sb.WriteString(util.ElideString(block.BlockHash))
		sb.WriteString(", ")
		sb.WriteString(strconv.Itoa(block.ClientIds.Len()))
		sb.WriteString(")")

		// fetch the parent
		nextBlock, _ := cc.blocksByHash.Get(block.ParentHash)

		if nextBlock != nil {
			sb.WriteString(" -> ")
		}

		block = nextBlock
	}

	return sb.String()
}

func NewCanonicalChain(
	networkId uint64,
	chainId uint64,
	updates <-chan natsutil.KeyValueEntry[eth.ClientStatus],
	maxDistanceFromHead int,
) (*CanonicalChain, error) {
	bc := CanonicalChain{
		networkId:           networkId,
		chainId:             chainId,
		maxDistanceFromHead: maxDistanceFromHead,
		updates:             updates,
		log:                 log.WithField("component", "CanonicalChain"),
	}

	return &bc, nil
}

func (cc CanonicalChain) Start() {
	wg := sync.WaitGroup{}
	wg.Add(1)

	ctx, cancel := context.WithCancel(context.Background())

	go cc.process(ctx)

	cc.wg = &wg
	cc.cancel = cancel
}

func (cc *CanonicalChain) process(ctx context.Context) {
	// countdown the wait group after processing has finished
	defer cc.wg.Done()

	for {
		select {

		case <-ctx.Done():
			// ctx was cancelled so return
			return

		case update, ok := <-cc.updates:
			if update != nil {

				switch update.Operation() {

				case nats.KeyValuePut:
					status, err := update.Value()
					if err != nil {
						cc.log.WithError(err).Error("failed to retrieve client status from update")
						continue
					}

					head := status.Head

					var block *Block

					block, ok := cc.blocksByHash.Get(status.Head.BlockHash)
					if !ok {

						number, err := head.BlockNumberBI()
						if err != nil {
							cc.log.WithError(err).Error("failed to process update")
							continue
						}

						difficulty, err := head.DifficultyBI()
						if err != nil {
							cc.log.WithError(err).Error("failed to process update")
							continue
						}

						totalDifficulty, err := head.TotalDifficultyBI()
						if err != nil {
							cc.log.WithError(err).Error("failed to process update")
							continue
						}

						// create a new block entry
						block = &Block{
							Number:          number,
							BlockHash:       head.BlockHash,
							ParentHash:      head.ParentHash,
							Difficulty:      difficulty,
							TotalDifficulty: totalDifficulty,
							ClientIds:       btree.Set[string]{},
						}

						// add it to the map
						cc.blocksByHash.Set(block.BlockHash, block)
					}

					// register that this client has the specified block
					block.ClientIds.Insert(update.Key())

					cc.log.WithField("block", block).Debug("updated block")

					// check if we have a new head by comparing the total difficulty of both blocks
					// the one with the greatest total difficulty is the head
					currentHead := cc.Head()
					if currentHead == nil || currentHead.TotalDifficulty.Cmp(block.TotalDifficulty) == -1 {
						cc.head.Store(block)
					}

				case nats.KeyValueDelete, nats.KeyValuePurge:

					clientId := update.Key()

					cc.blocksByHash.Scan(func(key string, value *Block) bool {
						// remove the client from the block
						value.ClientIds.Delete(clientId)

						if value.ClientIds.Len() == 0 {
							// remove the block entry as there are no clients
							cc.blocksByHash.Delete(key)
						}

						// continue scanning
						return true
					})

				default:
					cc.log.Errorf("unexpected kv operation: %s", update.Operation())

				}

				if cc.blocksByHash.Len() > cc.maxDistanceFromHead {

					headNumber := cc.Head().Number
					maxDistanceFromHead := big.NewInt(int64(cc.maxDistanceFromHead))

					cc.blocksByHash.Scan(func(key string, value *Block) bool {
						distanceFromHead := big.Int{}
						distanceFromHead.Sub(headNumber, value.Number)

						if distanceFromHead.Cmp(maxDistanceFromHead) > 0 {
							// too far from head, remove the entry
							cc.blocksByHash.Delete(key)
						}

						// continue scanning
						return true
					})
				}

				// notify listeners
				for _, listener := range cc.listeners {
					go func(listener chan<- *CanonicalChain) {
						listener <- cc
					}(listener)
				}

				cc.log.WithField("chain", cc).Debug("chain updated")
			}
			if !ok {
				// update channel has been closed
				return
			}
		}
	}
}

func (cc *CanonicalChain) Close() {
	cc.cancel()
	cc.wg.Wait()
}
