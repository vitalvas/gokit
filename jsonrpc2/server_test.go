package jsonrpc2

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testServer() *Server {
	s := NewServer()
	s.Register("subtract", func(_ context.Context, p json.RawMessage) (any, error) {
		var arr []int
		if err := json.Unmarshal(p, &arr); err == nil {
			if len(arr) != 2 {
				return nil, NewError(CodeInvalidParams, "Invalid params", nil)
			}
			return arr[0] - arr[1], nil
		}
		var obj struct{ Minuend, Subtrahend int }
		if err := json.Unmarshal(p, &obj); err != nil {
			return nil, NewError(CodeInvalidParams, "Invalid params", nil)
		}
		return obj.Minuend - obj.Subtrahend, nil
	})
	s.Register("update", func(_ context.Context, _ json.RawMessage) (any, error) { return nil, nil })
	s.Register("boom", func(_ context.Context, _ json.RawMessage) (any, error) {
		return nil, errors.New("kaboom")
	})
	s.Register("unmarshalable", func(_ context.Context, _ json.RawMessage) (any, error) {
		return make(chan int), nil
	})
	s.Register("panic", func(_ context.Context, _ json.RawMessage) (any, error) {
		panic("boom")
	})
	return s
}

func TestHandleSingle(t *testing.T) {
	s := testServer()
	ctx := context.Background()

	t.Run("positional params", func(t *testing.T) {
		out := s.Handle(ctx, []byte(`{"jsonrpc":"2.0","method":"subtract","params":[42,23],"id":1}`))
		assert.JSONEq(t, `{"jsonrpc":"2.0","result":19,"id":1}`, string(out))
	})

	t.Run("named params", func(t *testing.T) {
		out := s.Handle(ctx, []byte(`{"jsonrpc":"2.0","method":"subtract","params":{"minuend":42,"subtrahend":23},"id":3}`))
		assert.JSONEq(t, `{"jsonrpc":"2.0","result":19,"id":3}`, string(out))
	})

	t.Run("notification returns nothing", func(t *testing.T) {
		out := s.Handle(ctx, []byte(`{"jsonrpc":"2.0","method":"update","params":[1,2,3]}`))
		assert.Nil(t, out)
	})

	t.Run("method not found", func(t *testing.T) {
		out := s.Handle(ctx, []byte(`{"jsonrpc":"2.0","method":"foobar","id":"1"}`))
		assert.JSONEq(t, `{"jsonrpc":"2.0","error":{"code":-32601,"message":"Method not found"},"id":"1"}`, string(out))
	})

	t.Run("unknown notification returns nothing", func(t *testing.T) {
		out := s.Handle(ctx, []byte(`{"jsonrpc":"2.0","method":"foobar"}`))
		assert.Nil(t, out)
	})

	t.Run("parse error", func(t *testing.T) {
		out := s.Handle(ctx, []byte(`{"jsonrpc":"2.0","method":"foobar,"params":"bar","baz]`))
		assert.JSONEq(t, `{"jsonrpc":"2.0","error":{"code":-32700,"message":"Parse error"},"id":null}`, string(out))
	})

	t.Run("invalid request wrong type", func(t *testing.T) {
		out := s.Handle(ctx, []byte(`{"jsonrpc":"2.0","method":1,"params":"bar"}`))
		assert.JSONEq(t, `{"jsonrpc":"2.0","error":{"code":-32600,"message":"Invalid Request"},"id":null}`, string(out))
	})

	t.Run("wrong version", func(t *testing.T) {
		out := s.Handle(ctx, []byte(`{"jsonrpc":"1.0","method":"subtract","params":[1,2],"id":5}`))
		assert.JSONEq(t, `{"jsonrpc":"2.0","error":{"code":-32600,"message":"Invalid Request"},"id":5}`, string(out))
	})

	t.Run("handler error not jsonrpc becomes internal", func(t *testing.T) {
		out := s.Handle(ctx, []byte(`{"jsonrpc":"2.0","method":"boom","id":9}`))
		assert.JSONEq(t, `{"jsonrpc":"2.0","error":{"code":-32603,"message":"Internal error"},"id":9}`, string(out))
	})

	t.Run("handler jsonrpc error preserved", func(t *testing.T) {
		out := s.Handle(ctx, []byte(`{"jsonrpc":"2.0","method":"subtract","params":"nope","id":9}`))
		assert.JSONEq(t, `{"jsonrpc":"2.0","error":{"code":-32602,"message":"Invalid params"},"id":9}`, string(out))
	})

	t.Run("unmarshalable result becomes internal error", func(t *testing.T) {
		out := s.Handle(ctx, []byte(`{"jsonrpc":"2.0","method":"unmarshalable","id":9}`))
		assert.JSONEq(t, `{"jsonrpc":"2.0","error":{"code":-32603,"message":"Internal error"},"id":9}`, string(out))
	})

	t.Run("handler panic becomes internal error", func(t *testing.T) {
		out := s.Handle(ctx, []byte(`{"jsonrpc":"2.0","method":"panic","id":9}`))
		assert.JSONEq(t, `{"jsonrpc":"2.0","error":{"code":-32603,"message":"Internal error"},"id":9}`, string(out))
	})
}

func TestHandleBatch(t *testing.T) {
	s := testServer()
	ctx := context.Background()

	t.Run("empty array is invalid request", func(t *testing.T) {
		out := s.Handle(ctx, []byte(`[]`))
		assert.JSONEq(t, `{"jsonrpc":"2.0","error":{"code":-32600,"message":"Invalid Request"},"id":null}`, string(out))
	})

	t.Run("invalid non-empty batch", func(t *testing.T) {
		out := s.Handle(ctx, []byte(`[1,2,3]`))
		var resps []json.RawMessage
		require.NoError(t, json.Unmarshal(out, &resps))
		require.Len(t, resps, 3)
		for _, r := range resps {
			assert.JSONEq(t, `{"jsonrpc":"2.0","error":{"code":-32600,"message":"Invalid Request"},"id":null}`, string(r))
		}
	})

	t.Run("not valid json", func(t *testing.T) {
		out := s.Handle(ctx, []byte(`[{"jsonrpc":"2.0","method"]`))
		assert.JSONEq(t, `{"jsonrpc":"2.0","error":{"code":-32700,"message":"Parse error"},"id":null}`, string(out))
	})

	t.Run("mixed batch", func(t *testing.T) {
		in := `[{"jsonrpc":"2.0","method":"subtract","params":[42,23],"id":"1"},{"jsonrpc":"2.0","method":"update","params":[7]},{"foo":"boo"},{"jsonrpc":"2.0","method":"foobar","id":"5"}]`
		out := s.Handle(ctx, []byte(in))
		var resps []json.RawMessage
		require.NoError(t, json.Unmarshal(out, &resps))
		require.Len(t, resps, 3)
		assert.JSONEq(t, `{"jsonrpc":"2.0","result":19,"id":"1"}`, string(resps[0]))
		assert.JSONEq(t, `{"jsonrpc":"2.0","error":{"code":-32600,"message":"Invalid Request"},"id":null}`, string(resps[1]))
		assert.JSONEq(t, `{"jsonrpc":"2.0","error":{"code":-32601,"message":"Method not found"},"id":"5"}`, string(resps[2]))
	})

	t.Run("all notifications returns nothing", func(t *testing.T) {
		out := s.Handle(ctx, []byte(`[{"jsonrpc":"2.0","method":"update","params":[1]},{"jsonrpc":"2.0","method":"update"}]`))
		assert.Nil(t, out)
	})
}

func TestRegisterReplaces(t *testing.T) {
	s := NewServer()
	s.Register("m", func(_ context.Context, _ json.RawMessage) (any, error) { return 1, nil })
	s.Register("m", func(_ context.Context, _ json.RawMessage) (any, error) { return 2, nil })
	out := s.Handle(context.Background(), []byte(`{"jsonrpc":"2.0","method":"m","id":1}`))
	assert.JSONEq(t, `{"jsonrpc":"2.0","result":2,"id":1}`, string(out))
}

func FuzzHandle(f *testing.F) {
	s := testServer()
	seeds := []string{
		`{"jsonrpc":"2.0","method":"subtract","params":[1,2],"id":1}`,
		`[]`, `[1]`, `not json`, `{}`, `null`, `[{"jsonrpc":"2.0","method":"update"}]`,
	}
	for _, s := range seeds {
		f.Add([]byte(s))
	}
	f.Fuzz(func(t *testing.T, payload []byte) {
		out := s.Handle(context.Background(), payload)
		if out != nil {
			assert.True(t, json.Valid(out), "output must be valid JSON: %s", out)
		}
	})
}

func BenchmarkHandleSingle(b *testing.B) {
	s := testServer()
	ctx := context.Background()
	payload := []byte(`{"jsonrpc":"2.0","method":"subtract","params":[42,23],"id":1}`)
	b.ReportAllocs()
	for range b.N {
		s.Handle(ctx, payload)
	}
}

func BenchmarkHandleBatch(b *testing.B) {
	s := testServer()
	ctx := context.Background()
	payload := []byte(`[{"jsonrpc":"2.0","method":"subtract","params":[42,23],"id":1},{"jsonrpc":"2.0","method":"update","params":[7]},{"jsonrpc":"2.0","method":"subtract","params":[5,1],"id":2}]`)
	b.ReportAllocs()
	for range b.N {
		s.Handle(ctx, payload)
	}
}
