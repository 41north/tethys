package nats

import (
	"encoding/json"

	"github.com/juju/errors"
	"github.com/nats-io/nats.go"
)

type Publisher[T any] struct {
	Subject string
	js      nats.JetStreamContext
}

func (p Publisher[T]) Publish(payload T, opts ...nats.PubOpt) (*nats.PubAck, error) {
	bytes, err := json.Marshal(payload)
	if err != nil {
		return nil, errors.Annotate(err, "failed to serialize to json")
	}
	return p.js.Publish(p.Subject, bytes, opts...)
}

func (p Publisher[T]) PublishRaw(payload json.RawMessage, opts ...nats.PubOpt) (*nats.PubAck, error) {
	bytes, err := json.Marshal(payload)
	if err != nil {
		return nil, errors.Annotate(err, "failed to serialize to json")
	}
	return p.js.Publish(p.Subject, bytes, opts...)
}

func (p Publisher[T]) PublishAsync(payload T, opts ...nats.PubOpt) (nats.PubAckFuture, error) {
	bytes, err := json.Marshal(payload)
	if err != nil {
		return nil, errors.Annotate(err, "failed to serialize to json")
	}
	return p.js.PublishAsync(p.Subject, bytes, opts...)
}

func (p Publisher[T]) PublishAsyncRaw(payload json.RawMessage, opts ...nats.PubOpt) (nats.PubAckFuture, error) {
	bytes, err := json.Marshal(payload)
	if err != nil {
		return nil, errors.Annotate(err, "failed to serialize to json")
	}
	return p.js.PublishAsync(p.Subject, bytes, opts...)
}

func NewPublisher[T any](
	js nats.JetStreamContext,
	subject string,
) (*Publisher[T], error) {
	return &Publisher[T]{
		Subject: subject,
		js:      js,
	}, nil
}
