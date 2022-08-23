package util

import "github.com/juju/errors"

const (
	ErrClosed          = errors.ConstError("component is closed")
	ErrNotClosed       = errors.ConstError("component is not closed")
	ErrUnexpectedState = errors.ConstError("component is in an unexpected state")
)

type Result[T any] interface {
	Value() (*T, error)
}

func NewResult[T any](value *T) Result[T] {
	return result[T]{value: value}
}

func NewResultErr[T any](err error) Result[T] {
	return result[T]{err: err}
}

func NewResultOrErr[T any](value *T, err error) Result[T] {
	return result[T]{value: value, err: err}
}

type result[T any] struct {
	value *T
	err   error
}

func (r result[T]) Value() (*T, error) {
	return r.value, r.err
}
