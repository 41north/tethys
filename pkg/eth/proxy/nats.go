package proxy

import (
	"strconv"
	"sync"
	"time"

	natseth "github.com/41north/tethys/pkg/eth/nats"

	"github.com/41north/tethys/pkg/eth"
	natsutil "github.com/41north/tethys/pkg/nats"
	"github.com/juju/errors"
	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
)

const (
	ErrNatsEmbeddedEmptyConfig = errors.ConstError("nats embedded config path is empty or undefined")
	ErrNatsAlreadyConnected    = errors.ConstError("nats has already been connected")
)

var (
	ns           *server.Server
	natsConn     *nats.Conn
	stateManager *natseth.StateManager

	rpcClient     natsutil.RpcClient
	subjectPrefix string

	mutex sync.Mutex
)

func startNatsServer(opts Options) error {
	if !opts.NatsEmbedded {
		return nil
	}

	if opts.NatsEmbeddedConfigPath == "" {
		return ErrNatsEmbeddedEmptyConfig
	}

	mutex.Lock()
	defer mutex.Unlock()

	nsOpts, err := server.ProcessConfigFile(opts.NatsEmbeddedConfigPath)
	if err != nil {
		return errors.Annotate(err, "failed to parse NATS config options from path")
	}

	server, err := server.NewServer(nsOpts)
	if err != nil {
		return errors.Annotate(err, "failed to create NATS server in embedded mode")
	}

	ns = server

	// start server directly in a goroutine
	go ns.Start()

	// Wait for server to be ready for connections
	if !ns.ReadyForConnections(5 * time.Second) {
		return errors.Annotate(err, "failed to start NATS server in embedded mode")
	}

	return nil
}

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

func closeNatsServer() {
	ns.Shutdown()
	ns.WaitForShutdown()
}
