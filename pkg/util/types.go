package util

import "github.com/juju/errors"

const (
	ErrClosed          = errors.ConstError("component is closed")
	ErrNotClosed       = errors.ConstError("component is not closed")
	ErrUnexpectedState = errors.ConstError("component is in an unexpected state")
)
