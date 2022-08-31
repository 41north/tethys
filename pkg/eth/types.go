package eth

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/41north/tethys/pkg/eth/web3"
)

const (
	ConnectionTypeDirect = iota
	ConnectionTypeManaged
)

var ConnectionTypes = []ConnectionType{ConnectionTypeDirect, ConnectionTypeManaged}

type ConnectionType int

func (ct ConnectionType) String() (result string) {
	types := []string{"ConnectionTypeDirect", "ConnectionTypeManaged"}
	if len(types) < int(ct) {
		return ""
	}
	return types[ct]
}

func ToConnectionType(s string) ConnectionType {
	switch s {
	case "ConnectionTypeDirect":
		return ConnectionTypeDirect
	case "ConnectionTypeManaged":
		return ConnectionTypeManaged
	default:
		return -1
	}
}

// MarshalJSON implements a custom json marshaller for ConnectionType.
func (ct ConnectionType) MarshalJSON() ([]byte, error) {
	// we remove the 'ConnectionType' prefix
	return json.Marshal(ct.String())
}

// UnmarshalJSON implements a custom json unmarshaller for ConnectionType.
func (ct *ConnectionType) UnmarshalJSON(data []byte) error {
	var ctStr string
	if err := json.Unmarshal(data, &ctStr); err != nil {
		return err
	}
	// we add the 'ConnectionType' prefix before parsing
	*ct = ToConnectionType(ctStr)
	return nil
}

type NetworkAndChainId struct {
	NetworkId uint64 `json:"networkId"`
	ChainId   uint64 `json:"chainId"`
}

type ClientProfile struct {
	Id             string         `json:"id"`
	ConnectionType ConnectionType `json:"connectionType"`

	NetworkId uint64 `json:"networkId"`
	ChainId   uint64 `json:"chainId"`

	ClientVersion web3.ClientVersion `json:"clientVersion"`
	NodeInfo      *web3.NodeInfo     `json:"nodeInfo,omitempty"` // unavailable from third party providers e.g. alchemy
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

func SanitizeVersion(version string) string {
	version = strings.ReplaceAll(version, ".", "_")
	version = strings.ReplaceAll(version, "-", "_")
	return version
}
