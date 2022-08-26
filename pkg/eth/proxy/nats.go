package proxy

import (
	"strconv"
	"sync"

	natseth "github.com/41north/tethys/pkg/eth/nats"

	"github.com/41north/tethys/pkg/eth"
	natsutil "github.com/41north/tethys/pkg/nats"
	"github.com/juju/errors"
	"github.com/nats-io/nats.go"
)

const (
	ErrNatsAlreadyConnected = errors.ConstError("nats has already been connected")
)

var (
	natsConn     *nats.Conn
	stateManager *natseth.StateManager

	rpcClient     natsutil.RpcClient
	subjectPrefix string

	mutex sync.Mutex
)

func connectNats(opts Options) error {
	mutex.Lock()
	defer mutex.Unlock()

	if natsConn != nil {
		return ErrNatsAlreadyConnected
	}

	conn, err := nats.Connect(opts.NatsUrl)
	if err != nil {
		return errors.Annotate(err, "failed to connect to NATS")
	}

	natsConn = conn

	js, err := conn.JetStream()
	if err != nil {
		return errors.Annotate(err, "failed to initialise JetStream context")
	}

	stateManager, err = natseth.NewStateManager(
		js,
		natseth.NetworkAndChainId(opts.NetworkId, opts.ChainId),
		natseth.Create(true),
	)

	if err != nil {
		return errors.Annotate(err, "failed to initialise state stores")
	}

	networkId := strconv.FormatUint(opts.NetworkId, 10)
	chainId := strconv.FormatUint(opts.ChainId, 10)

	subjectPrefix = eth.SubjectName("eth", "rpc", networkId, chainId)

	rpcClient, err = natsutil.NewRpcClient(conn)
	if err != nil {
		return errors.Annotate(err, "failed to create rpc client")
	}

	return nil
}

func closeNats() {
	natsConn.Close()
}
