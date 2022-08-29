package methods

import (
	natsutil "github.com/41north/tethys/pkg/nats"
	"github.com/41north/tethys/pkg/proxy"
)

const (
	// todo make this configurable/dynamic
	ClientVersion = "Tethys/0.1.0/linux/go1.19"

	Web3Sha3          = "web3_sha3"
	Web3ClientVersion = "web3_clientVersion"
)

func web3Methods(router natsutil.Router) []proxy.Method {

	return []proxy.Method{
		proxy.NewMethod(Web3Sha3, router),
		proxy.NewMethod(Web3ClientVersion, natsutil.NewStaticResult(ClientVersion)),
	}
}
