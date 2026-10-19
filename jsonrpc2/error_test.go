package jsonrpc2

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestError(t *testing.T) {
	t.Run("message and format", func(t *testing.T) {
		e := NewError(CodeInvalidParams, "bad", nil)
		assert.Equal(t, "jsonrpc2: code -32602: bad", e.Error())
	})

	t.Run("data encoded", func(t *testing.T) {
		e := NewError(CodeInternalError, "x", map[string]int{"x": 1})
		assert.JSONEq(t, `{"x":1}`, string(e.Data))
	})

	t.Run("nil data omitted", func(t *testing.T) {
		assert.Nil(t, NewError(CodeInternalError, "x", nil).Data)
	})

	t.Run("unmarshalable data left nil", func(t *testing.T) {
		assert.Nil(t, NewError(CodeInternalError, "x", make(chan int)).Data)
	})

	t.Run("marshals to spec shape", func(t *testing.T) {
		out, err := json.Marshal(NewError(CodeMethodNotFound, "Method not found", nil))
		assert.NoError(t, err)
		assert.JSONEq(t, `{"code":-32601,"message":"Method not found"}`, string(out))
	})
}
