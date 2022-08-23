package tracking

import (
	"fmt"
	"math/big"

	"github.com/41north/tethys/pkg/util"

	"github.com/tidwall/btree"
)

type Block struct {
	Number          *big.Int
	BlockHash       string
	ParentHash      string
	Difficulty      *big.Int
	TotalDifficulty *big.Int
	ClientIds       btree.Set[string]
}

func (b Block) String() string {
	return fmt.Sprintf(
		"Block{Number: %s, Hash: %s, ParentHash: %s, Difficulty: %s, TotalDifficulty: %s, Clients: %d}",
		b.Number, util.ElideString(b.BlockHash), util.ElideString(b.ParentHash), b.Difficulty, b.TotalDifficulty, b.ClientIds.Len(),
	)
}

type BlocksForNumber struct {
	Number *big.Int
	Blocks []*Block
}
