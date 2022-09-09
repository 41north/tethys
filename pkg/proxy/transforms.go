package proxy

import (
	"encoding/json"

	"github.com/41north/go-jsonrpc"
	"github.com/juju/errors"
)

type RequestTransform = func(req jsonrpc.Request) (jsonrpc.Request, error)

type ResponseTransform = func(resp *jsonrpc.Response) error

func marshalParamsArray(req *jsonrpc.Request, params []any) error {
	bytes, err := json.Marshal(params)
	if err != nil {
		return errors.Annotate(err, "failed to marshal params array")
	}
	req.Params = bytes
	return nil
}

func ReplaceParameterByIndex(position int, valueFn func(current any) (any, error)) RequestTransform {
	return func(req jsonrpc.Request) (jsonrpc.Request, error) {
		var params []any
		if err := req.UnmarshalParams(&params); err != nil {
			return req, errors.Annotate(err, "failed to unmarshal params array")
		}
		if position >= len(params) {
			// not enough params, do nothing
			return req, nil
		}
		// update params
		value, err := valueFn(params[position])
		if err != nil {
			return req, errors.Annotate(err, "value fn returned an error")
		}

		params[position] = value
		return req, marshalParamsArray(&req, params)
	}
}
