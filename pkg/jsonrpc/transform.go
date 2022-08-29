package jsonrpc

import (
	"encoding/json"

	"github.com/juju/errors"
)

type RequestTransform = func(req Request) (Request, error)

type ResponseTransform = func(resp *Response) (Response, error)

func NewRequestPipeline(transforms ...RequestTransform) RequestTransform {
	return func(req Request) (result Request, err error) {
		result = req
		for _, transform := range transforms {
			result, err = transform(result)
			if err != nil {
				// exit early and return the early
				break
			}
		}
		return result, err
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
	return func(req Request) (Request, error) {
		params, err := unmarshalParamsArray(&req)
		if err != nil {
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
