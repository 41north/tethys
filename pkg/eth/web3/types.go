package web3

import (
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"math/big"
	"regexp"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/juju/errors"
)

const clientVersionPattern = "^(.*)/(.*)/(.*)/(.*)$"

func ParseClientVersion(str string) (ClientVersion, error) {
	var result ClientVersion

	regex, err := regexp.Compile(clientVersionPattern)
	if err != nil {
		panic("Failed to compile rpc version pattern")
	}

	matches := regex.FindStringSubmatch(str)
	if len(matches) != 5 {
		return result, errors.Errorf("Expected 5 matches, found %d. Input = %s", len(matches), str)
	}

	result = ClientVersion{
		Name:     matches[1],
		Version:  matches[2],
		OS:       matches[3],
		Language: matches[4],
	}
	return result, nil
}

type NodeInfo struct {
	Id            string          `json:"id"`
	Name          string          `json:"name"`
	Enode         string          `json:"enode"`
	Ports         json.RawMessage `json:"ports"`
	Protocols     json.RawMessage `json:"protocols"`
	ListenAddress string          `json:"listenAddr"`
}

func (n *NodeInfo) ParseClientVersion() (ClientVersion, error) {
	return ParseClientVersion(n.Name)
}

type ClientVersion struct {
	Name     string `json:"name"`
	Version  string `json:"version"`
	OS       string `json:"os"`
	Language string `json:"language"`
}

func (cv ClientVersion) String() string {
	return fmt.Sprintf("%s/%s/%s/%s", cv.Name, cv.Version, cv.OS, cv.Language)
}

type SyncStatus struct {
	Syncing bool `json:"syncing"`
	// todo add progress fields
}

type Head struct {
	BlockNumber     string `json:"blockNumber"`
	BlockHash       string `json:"blockHash"`
	ParentHash      string `json:"parentHash"`
	Difficulty      string `json:"difficulty"`
	TotalDifficulty string `json:"totalDifficulty"`
}

func (h *Head) BlockNumberBI() (*big.Int, error) {
	return hexutil.DecodeBig(h.BlockNumber)
}

func (h *Head) DifficultyBI() (*big.Int, error) {
	return hexutil.DecodeBig(h.Difficulty)
}

func (h *Head) TotalDifficultyBI() (*big.Int, error) {
	return hexutil.DecodeBig(h.TotalDifficulty)
}

type SubscriptionNotification struct {
	SubscriptionId string          `json:"subscription"`
	Result         json.RawMessage `json:"result"`
}

func (sn *SubscriptionNotification) UnmarshalResult(result interface{}) error {
	return json.Unmarshal(sn.Result, &result)
}

type Syncing struct {
	IsSyncing bool `json:"syncing"`
	Status    json.RawMessage
}

type AccessTuple struct {
	Address     common.Address `json:"address"        gencodec:"required"`
	StorageKeys []common.Hash  `json:"storageKeys"    gencodec:"required"`
}

type AccessList = []AccessTuple

type Transaction struct {
	// Common fields
	Hash             string `json:"hash"`
	Nonce            string `json:"nonce"`
	BlockHash        string `json:"blockHash"`
	BlockNumber      string `json:"blockNumber"`
	TransactionIndex string `json:"transactionIndex"`
	From             string `json:"from"`
	To               string `json:"to,omitempty"`
	Value            string `json:"value"`
	Gas              string `json:"gas"`
	Input            string `json:"input"`
	Data             string `json:"data,omitempty"`
	V                string `json:"v"`
	R                string `json:"r"`
	S                string `json:"s"`

	// EIP-155
	ChainID string `json:"chainId,omitempty"`

	// legacy tx pricing
	GasPrice string `json:"gasPrice,omitempty"`

	// EIP-1559
	Type                 string `json:"type,omitempty"`
	MaxFeePerGas         string `json:"MaxFeePerGas,omitempty"`
	MaxPriorityFeePerGas string `json:"maxPriorityFeePerGas,omitempty"`

	// EIP-2930
	AccessList AccessList `json:"accessList,omitempty"`
}

type Block struct {
	Author          string   `json:"author,omitempty"`
	Miner           string   `json:"miner,omitempty"`
	Difficulty      string   `json:"difficulty"`
	TotalDifficulty string   `json:"totalDifficulty"`
	GasLimit        string   `json:"gasLimit"`
	GasUsed         string   `json:"gasUsed"`
	Hash            string   `json:"hash"`
	MixHash         string   `json:"mixHash"`
	LogsBloom       string   `json:"logsBloom"`
	ExtraData       string   `json:"extraData"`
	Nonce           string   `json:"nonce"`
	Number          string   `json:"number"`
	ParentHash      string   `json:"parentHash"`
	ReceiptsRoot    string   `json:"receiptsRoot"`
	SealFields      string   `json:"sealFields,omitempty"`
	Sha3Uncles      string   `json:"sha3Uncles"`
	Size            string   `json:"history,omitempty"`
	StateRoot       string   `json:"stateRoot"`
	Timestamp       string   `json:"timestamp"`
	Uncles          []string `json:"uncles"`

	TransactionsRoot string            `json:"transactionsRoot"`
	Transactions     []json.RawMessage `json:"transactions"` // list of hashes or a list of transactions objects

	TransactionHashes  []string
	TransactionObjects []Transaction

	// EIP-1559
	BaseFeePerGas string `json:"baseFeePerGas,omitempty"`
}

func (b *Block) UnmarshalTransactions() error {
	count := len(b.Transactions)

	if count == 0 || len(b.TransactionHashes) > 0 || len(b.TransactionObjects) > 0 {
		// no transactions or we have already unmarshalled them
		return nil
	}

	// check the first character of the first entry
	switch b.Transactions[0][0] {
	case '"':
		// we assume an array of hashes
		var result []string
		for _, item := range b.Transactions {
			var hash string
			if err := json.Unmarshal(item, &hash); err != nil {
				return err
			}
			result = append(result, hash)
		}
		// cache the result
		b.TransactionHashes = result
	case '{':
		// we assume an array of transaction objects
		// we assume an array of hashes
		var result []Transaction
		for _, item := range b.Transactions {
			var tx Transaction
			if err := json.Unmarshal(item, &tx); err != nil {
				return err
			}
			result = append(result, tx)
		}
		// cache the result
		b.TransactionObjects = result
	default:
		return errors.Errorf("unexpected format encountered: %s", b.Transactions[0])
	}

	return nil
}

type NewHead struct {
	ParentHash       string `json:"parentHash"`
	Sha3Uncles       string `json:"sha3Uncles"`
	Miner            string `json:"miner,omitempty"`
	StateRoot        string `json:"stateRoot"`
	TransactionsRoot string `json:"transactionsRoot"`
	ReceiptsRoot     string `json:"receiptsRoot"`
	LogsBloom        string `json:"logsBloom"`
	Difficulty       string `json:"Difficulty"`
	Number           string `json:"number"`
	GasLimit         string `json:"gasLimit"`
	GasUsed          string `json:"gasUsed"`
	Author           string `json:"author,omitempty"`
	Timestamp        string `json:"timestamp"`
	ExtraData        string `json:"extraData"`
	MixHash          string `json:"mixHash"`
	Nonce            string `json:"nonce"`
	BaseFeePerGas    string `json:"baseFeePerGas,omitempty"`
	Hash             string `json:"hash"`
	SealFields       string `json:"sealFields,omitempty"`

	// Nethermind includes the following fields

	Size            string `json:"history,omitempty"`
	TotalDifficulty string `json:"totalDifficulty,omitempty"`
	Uncles          string `json:"uncles,omitempty"`
	Transactions    string `json:"transactions,omitempty"`
}
