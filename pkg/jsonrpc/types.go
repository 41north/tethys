package jsonrpc

import (
	"encoding/json"
	"fmt"

	"github.com/juju/errors"
)

var (
	ErrInvalidRequest = Error{
		Code:    -32600,
		Message: "invalid request",
	}
	ErrMethodNotFound = Error{
		Code:    -32601,
		Message: "method not found",
	}
	ErrInvalidParams = Error{
		Code:    -32602,
		Message: "invalid params",
	}
	ErrInternal = Error{
		Code:    -32603,
		Message: "internal error",
	}
	ErrParse = Error{
		Code:    -32700,
		Message: "parse error",
	}
)

type IntOrString interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 |
		string
}

type Request struct {
	Id      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
	JsonRpc string          `json:"jsonrpc"`
}

func (r *Request) UnmarshalId() (any, error) {
	return unmarshalId(r.Id)
}

func (r *Request) WithStringId(id string) error {
	bytes, err := json.Marshal(id)
	r.Id = bytes
	return err
}

func (r *Request) WithIntegerId(id uint64) error {
	bytes, err := json.Marshal(id)
	r.Id = bytes
	return err
}

type Error struct {
	Code    int32           `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

// Error renders e to a human-readable string for the error interface.
func (e Error) Error() string { return fmt.Sprintf("[%d] %s", e.Code, e.Message) }

type Response struct {
	Id      json.RawMessage `json:"id,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *Error          `json:"error,omitempty"`
	JsonRpc string          `json:"jsonrpc"`
}

func (r *Response) UnmarshalId() (any, error) {
	return unmarshalId(r.Id)
}

func (r *Response) UnmarshalResult(payload any) error {
	if r.Error != nil {
		return r.Error
	}
	return json.Unmarshal(r.Result, payload)
}

func unmarshalId(id json.RawMessage) (any, error) {
	var key any
	err := json.Unmarshal(id, &key)
	if err != nil {
		return nil, errors.Annotate(err, "failed to unmarshal id")
	}
	switch key.(type) {
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, string:
		return key, nil
	default:
		return nil, errors.Errorf("id field must be an integer or a string, found: %v", key)
	}
}
