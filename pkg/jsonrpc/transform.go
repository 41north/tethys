package jsonrpc

import (
	"encoding/json"

	"github.com/juju/errors"
)

type RequestTransform = func(req *Request) error

func NewRequestPipeline(transforms ...RequestTransform) RequestTransform {
	return func(req *Request) error {
		for _, transform := range transforms {
			if err := transform(req); err != nil {
				return err
			}
		}
		return nil
	}
}

func unmarshalParamsArray(req *Request) ([]any, error) {
	var params []any
	err := json.Unmarshal(req.Params, &params)
	return params, err
}

func marshalParamsArray(req *Request, params []any) error {
	bytes, err := json.Marshal(params)
	if err != nil {
		return errors.Annotate(err, "failed to marshal params array")
	}
	req.Params = bytes
	return nil
}

func ReplaceParameterByIndex(position int, valueFn func(current any) (any, error)) RequestTransform {
	return func(req *Request) error {
		params, err := unmarshalParamsArray(req)
		if err != nil {
			return errors.Annotate(err, "failed to unmarshal params array")
		}
		if position >= len(params) {
			// not enough params, do nothing
			return nil
		}

		// update params
		value, err := valueFn(params[position])
		if err != nil {
			return errors.Annotate(err, "value fn returned an error")
		}

		params[position] = value
		return marshalParamsArray(req, params)
	}
}
