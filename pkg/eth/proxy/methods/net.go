package methods

import (
	"github.com/41north/tethys/pkg/eth/tracking"
	natsutil "github.com/41north/tethys/pkg/nats"
	"github.com/41north/tethys/pkg/proxy"
)

const (
	NetVersion   = "net_version"
	NetListening = "net_listening"
	NetPeerCount = "net_peerCount"
)

func netMethods(chain *tracking.CanonicalChain) []proxy.Method {

	return []proxy.Method{
		proxy.NewMethod(NetVersion, natsutil.NewStaticResult(chain.NetworkId)),
		proxy.NewMethod(NetListening, natsutil.NewStaticResult(true)),
		proxy.NewMethod(NetPeerCount, natsutil.NewStaticResult(1)),
	}
}
