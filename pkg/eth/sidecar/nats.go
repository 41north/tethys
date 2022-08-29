package sidecar

import (
	"github.com/juju/errors"
	"github.com/nats-io/nats.go"
)

var (
	natsConn *nats.EncodedConn
	natsJs   nats.JetStreamContext
)

func connectNats(opts Options) error {
	var err error

	conn, err := nats.Connect(opts.NatsUrl)
	if err != nil {
		return errors.Annotate(err, "failed to connect to NATS")
	}

	natsConn, err = nats.NewEncodedConn(conn, nats.JSON_ENCODER)
	if err != nil {
		return errors.Annotate(err, "failed to create a JSON encoded NATS connection")
	}

	natsJs, err = conn.JetStream()
	if err != nil {
		return errors.Annotate(err, "failed to initialise JetStream context")
	}

	return nil
}

func closeNats() {
	natsConn.Close()
}
