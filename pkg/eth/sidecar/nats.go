package sidecar

import (
	"github.com/juju/errors"
	"github.com/nats-io/nats.go"
)

var (
	natsConn *nats.Conn
	natsJs   nats.JetStreamContext
)

func connectNats(opts Options) error {
	var err error

	natsConn, err = nats.Connect(opts.NatsUrl)
	if err != nil {
		return errors.Annotate(err, "failed to connect to NATS")
	}

	natsJs, err = natsConn.JetStream()
	if err != nil {
		return errors.Annotate(err, "failed to initialise JetStream context")
	}

	return nil
}

func closeNats() {
	natsConn.Close()
}
