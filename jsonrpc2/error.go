package jsonrpc2

import (
	"encoding/json"
	"fmt"
)

// Standard JSON-RPC 2.0 error codes.
const (
	CodeParseError     int = -32700
	CodeInvalidRequest int = -32600
	CodeMethodNotFound int = -32601
	CodeInvalidParams  int = -32602
	CodeInternalError  int = -32603
)

// Error is a JSON-RPC 2.0 error object. It implements the error interface so
// method handlers can return it directly.
type Error struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

func (e *Error) Error() string {
	return fmt.Sprintf("jsonrpc2: code %d: %s", e.Code, e.Message)
}

// NewError returns an Error with the given code and message. If data is nil the
// data field is omitted; otherwise it is JSON-encoded.
func NewError(code int, message string, data any) *Error {
	e := &Error{Code: code, Message: message}
	if data != nil {
		if raw, err := json.Marshal(data); err == nil {
			e.Data = raw
		}
	}
	return e
}
