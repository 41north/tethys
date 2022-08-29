package proxy

import (
	"sync"
	"time"

	natseth "github.com/41north/tethys/pkg/eth/nats"

	"github.com/juju/errors"
	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
)

const (
	ErrNatsAlreadyConnected = errors.ConstError("nats has already been connected")
)

var (
	ns           *server.Server
	natsConn     *nats.EncodedConn
	jsContext    nats.JetStreamContext
	stateManager *natseth.StateManager

	mutex sync.Mutex
)

func startNatsServer(opts Options) error {
	mutex.Lock()
	defer mutex.Unlock()

	if !opts.NatsEmbedded {
		return nil
	}

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

	natsConn, err = nats.NewEncodedConn(conn, nats.JSON_ENCODER)
	if err != nil {
		return errors.Annotate(err, "failed to create a json encoded NATS connection")
	}

	jsContext, err = conn.JetStream()
	if err != nil {
		return errors.Annotate(err, "failed to initialise JetStream context")
	}

	stateManager, err = natseth.NewStateManager(
		jsContext,
		natseth.NetworkAndChainId(opts.NetworkId, opts.ChainId),
		natseth.Create(true),
	)

	if err != nil {
		return errors.Annotate(err, "failed to initialise state stores")
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
