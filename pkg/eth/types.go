package eth

import (
	"fmt"
	"strings"

	"github.com/41north/tethys/pkg/eth/web3"
)

type NetworkAndChainId struct {
	NetworkId uint64 `json:"networkId"`
	ChainId   uint64 `json:"chainId"`
}

type ClientProfile struct {
	NetworkId     uint64             `json:"networkId"`
	ChainId       uint64             `json:"chainId"`
	NodeInfo      web3.NodeInfo      `json:"nodeInfo"`
	ClientVersion web3.ClientVersion `json:"clientVersion"`
}

func (cp ClientProfile) Id() string {
	return cp.NodeInfo.Id
}

func (cp ClientProfile) String() string {
	return fmt.Sprintf(
		"ClientProfile{networkId: %d, chainId: %d, nodeId: %s, clientVersion: %s}",
		cp.NetworkId, cp.ChainId, cp.NodeInfo.Id, cp.ClientVersion,
	)
}

type ClientStatus struct {
	Id         string           `json:"id"`
	Head       *web3.Head       `json:"head,omitempty"`
	SyncStatus *web3.SyncStatus `json:"syncStatus,omitempty"`
}

func (cs *ClientStatus) Merge(src *ClientStatus) (*ClientStatus, error) {
	merged := &ClientStatus{
		Id:         cs.Id,
		Head:       cs.Head,
		SyncStatus: cs.SyncStatus,
	}

	if src.Head != nil {
		merged.Head = src.Head
	}

	if src.SyncStatus != nil {
		merged.SyncStatus = src.SyncStatus
	}

	return merged, nil
}

func SubjectName(keys ...string) string {
	var sb strings.Builder
	for idx, key := range keys {
		if idx > 0 {
			sb.WriteString(".")
		}
		sb.WriteString(key)
	}
	return sb.String()
}

func SanitizeVersion(version string) string {
	version = strings.ReplaceAll(version, ".", "_")
	version = strings.ReplaceAll(version, "-", "_")
	return version
}
