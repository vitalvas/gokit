package jsonrpc2

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// loopback returns a Transport that feeds the payload straight into s and
// records the last request bytes sent.
func loopback(s *Server, last *[]byte) Transport {
	return func(ctx context.Context, payload []byte) ([]byte, error) {
		if last != nil {
			*last = payload
		}
		return s.Handle(ctx, payload), nil
	}
}

func TestClientCall(t *testing.T) {
	s := testServer()
	ctx := context.Background()

	t.Run("decodes result", func(t *testing.T) {
		var sent []byte
		c := NewClient(loopback(s, &sent))
		var out int
		require.NoError(t, c.Call(ctx, "subtract", []int{42, 23}, &out))
		assert.Equal(t, 19, out)
		assert.JSONEq(t, `{"jsonrpc":"2.0","method":"subtract","params":[42,23],"id":1}`, string(sent))
	})

	t.Run("auto-increments id", func(t *testing.T) {
		var sent []byte
		c := NewClient(loopback(s, &sent))
		require.NoError(t, c.Call(ctx, "subtract", []int{5, 1}, nil))
		assert.Contains(t, string(sent), `"id":1`)
		require.NoError(t, c.Call(ctx, "subtract", []int{5, 1}, nil))
		assert.Contains(t, string(sent), `"id":2`)
	})

	t.Run("server error returned as *Error", func(t *testing.T) {
		c := NewClient(loopback(s, nil))
		err := c.Call(ctx, "subtract", []int{1}, nil)
		var rpcErr *Error
		require.True(t, errors.As(err, &rpcErr))
		assert.Equal(t, CodeInvalidParams, rpcErr.Code)
	})

	t.Run("method not found", func(t *testing.T) {
		c := NewClient(loopback(s, nil))
		err := c.Call(ctx, "nope", nil, nil)
		var rpcErr *Error
		require.True(t, errors.As(err, &rpcErr))
		assert.Equal(t, CodeMethodNotFound, rpcErr.Code)
	})

	t.Run("nil out discards result", func(t *testing.T) {
		c := NewClient(loopback(s, nil))
		assert.NoError(t, c.Call(ctx, "subtract", []int{9, 4}, nil))
	})

	t.Run("transport error wrapped", func(t *testing.T) {
		boom := errors.New("down")
		c := NewClient(func(context.Context, []byte) ([]byte, error) { return nil, boom })
		err := c.Call(ctx, "subtract", []int{1, 2}, nil)
		assert.ErrorIs(t, err, boom)
	})

	t.Run("params encode error", func(t *testing.T) {
		c := NewClient(loopback(s, nil))
		err := c.Call(ctx, "subtract", make(chan int), nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "encode params")
	})

	t.Run("result decode error", func(t *testing.T) {
		c := NewClient(loopback(s, nil))
		var out string // subtract returns a number
		err := c.Call(ctx, "subtract", []int{9, 4}, &out)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "decode result")
	})

	t.Run("bad response bytes", func(t *testing.T) {
		c := NewClient(func(context.Context, []byte) ([]byte, error) { return []byte("not json"), nil })
		err := c.Call(ctx, "subtract", nil, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "decode response")
	})
}

func TestClientNotify(t *testing.T) {
	s := testServer()
	ctx := context.Background()

	t.Run("sends no id and gets no error", func(t *testing.T) {
		var sent []byte
		c := NewClient(loopback(s, &sent))
		require.NoError(t, c.Notify(ctx, "update", map[string]int{"x": 7}))
		assert.JSONEq(t, `{"jsonrpc":"2.0","method":"update","params":{"x":7}}`, string(sent))
		assert.NotContains(t, string(sent), `"id"`)
	})

	t.Run("nil params omits member", func(t *testing.T) {
		var sent []byte
		c := NewClient(loopback(s, &sent))
		require.NoError(t, c.Notify(ctx, "update", nil))
		assert.JSONEq(t, `{"jsonrpc":"2.0","method":"update"}`, string(sent))
	})

	t.Run("transport error wrapped", func(t *testing.T) {
		boom := errors.New("down")
		c := NewClient(func(context.Context, []byte) ([]byte, error) { return nil, boom })
		assert.ErrorIs(t, c.Notify(ctx, "update", nil), boom)
	})

	t.Run("params encode error", func(t *testing.T) {
		c := NewClient(loopback(s, nil))
		err := c.Notify(ctx, "update", make(chan int))
		assert.Contains(t, err.Error(), "encode params")
	})
}

func BenchmarkClientCall(b *testing.B) {
	s := testServer()
	c := NewClient(loopback(s, nil))
	ctx := context.Background()
	var out int
	b.ReportAllocs()
	for range b.N {
		_ = c.Call(ctx, "subtract", []int{42, 23}, &out)
	}
}

func FuzzClientRoundTrip(f *testing.F) {
	s := testServer()
	c := NewClient(loopback(s, nil))
	f.Add("subtract", `[42,23]`)
	f.Add("update", `{"x":1}`)
	f.Add("nope", `null`)
	f.Fuzz(func(t *testing.T, method, params string) {
		if !json.Valid([]byte(params)) {
			t.Skip()
		}
		// Must never panic regardless of method/params.
		_ = c.Call(context.Background(), method, json.RawMessage(params), nil)
	})
}
