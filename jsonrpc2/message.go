// Package jsonrpc2 implements the JSON-RPC 2.0 protocol.
//
// See https://www.jsonrpc.org/specification
package jsonrpc2

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// Version is the required jsonrpc protocol version string.
const Version string = "2.0"

// Request is a JSON-RPC 2.0 request or notification. A request with a nil ID is
// a notification: the server must not reply to it.
type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
	ID      *ID             `json:"id,omitempty"`
}

// IsNotification reports whether the request is a notification (has no ID).
func (r *Request) IsNotification() bool {
	return r.ID == nil
}

// Response is a JSON-RPC 2.0 response. Exactly one of Result or Error is set.
type Response struct {
	JSONRPC string          `json:"jsonrpc"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *Error          `json:"error,omitempty"`
	ID      *ID             `json:"id"`
}

// ID is a JSON-RPC request identifier. Per the spec it may be a string, a
// number, or null. A null id is represented by a non-nil *ID holding isNull.
type ID struct {
	num    float64
	str    string
	isStr  bool
	isNull bool
}

// NewStringID returns an ID holding a string value.
func NewStringID(s string) *ID { return &ID{str: s, isStr: true} }

// NewNumberID returns an ID holding a numeric value.
func NewNumberID(n float64) *ID { return &ID{num: n} }

// IsNull reports whether the ID is JSON null.
func (id *ID) IsNull() bool { return id.isNull }

// String returns a stable string form of the ID for logging and map keys.
func (id *ID) String() string {
	switch {
	case id == nil || id.isNull:
		return "null"
	case id.isStr:
		return id.str
	default:
		return fmt.Sprintf("%v", id.num)
	}
}

func (id ID) MarshalJSON() ([]byte, error) {
	switch {
	case id.isNull:
		return []byte("null"), nil
	case id.isStr:
		return json.Marshal(id.str)
	default:
		return json.Marshal(id.num)
	}
}

func (id *ID) UnmarshalJSON(data []byte) error {
	if bytes.Equal(data, []byte("null")) {
		id.isNull = true
		return nil
	}
	if len(data) > 0 && data[0] == '"' {
		id.isStr = true
		return json.Unmarshal(data, &id.str)
	}
	return json.Unmarshal(data, &id.num)
}
