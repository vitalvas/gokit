package jsonrpc2

import (
	"context"
	"encoding/json"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// rwc joins a pipe reader and writer into an io.ReadWriteCloser; Close shuts
// both ends so a blocked Serve decoder unblocks.
type rwc struct {
	*io.PipeReader
	*io.PipeWriter
}

func (c rwc) Read(p []byte) (int, error)  { return c.PipeReader.Read(p) }
func (c rwc) Write(p []byte) (int, error) { return c.PipeWriter.Write(p) }
func (c rwc) Close() error {
	_ = c.PipeReader.Close()
	return c.PipeWriter.Close()
}

// connPair wires two Conns together over in-memory pipes and runs Serve on
// both. It returns the pair and a cleanup func that stops them.
func connPair(t *testing.T) (a, b *Conn, stop func()) {
	t.Helper()
	ar, bw := io.Pipe()
	br, aw := io.Pipe()
	a = NewConn(rwc{ar, aw})
	b = NewConn(rwc{br, bw})

	var wg sync.WaitGroup
	wg.Add(2)
	go func() { defer wg.Done(); a.Serve(context.Background()) }()
	go func() { defer wg.Done(); b.Serve(context.Background()) }()

	return a, b, func() {
		_ = a.Close()
		_ = b.Close()
		wg.Wait()
	}
}

func TestConnCall(t *testing.T) {
	a, b, stop := connPair(t)
	defer stop()

	b.Register("add", func(_ context.Context, p json.RawMessage) (any, error) {
		var args []int
		require.NoError(t, json.Unmarshal(p, &args))
		return args[0] + args[1], nil
	})
	b.Register("fail", func(_ context.Context, _ json.RawMessage) (any, error) {
		return nil, NewError(CodeInvalidParams, "Invalid params", nil)
	})

	ctx := context.Background()

	t.Run("result", func(t *testing.T) {
		var sum int
		require.NoError(t, a.Call(ctx, "add", []int{2, 3}, &sum))
		assert.Equal(t, 5, sum)
	})

	t.Run("error surfaces as *Error", func(t *testing.T) {
		err := a.Call(ctx, "fail", nil, nil)
		var rpcErr *Error
		require.ErrorAs(t, err, &rpcErr)
		assert.Equal(t, CodeInvalidParams, rpcErr.Code)
	})

	t.Run("method not found", func(t *testing.T) {
		err := a.Call(ctx, "nope", nil, nil)
		var rpcErr *Error
		require.ErrorAs(t, err, &rpcErr)
		assert.Equal(t, CodeMethodNotFound, rpcErr.Code)
	})
}

func TestConnBidirectional(t *testing.T) {
	a, b, stop := connPair(t)
	defer stop()

	a.Register("ping", func(_ context.Context, _ json.RawMessage) (any, error) {
		return "pong", nil
	})
	// b's handler calls back into a mid-flight: exercises re-entrancy.
	b.Register("echo_ping", func(ctx context.Context, _ json.RawMessage) (any, error) {
		var pong string
		if err := b.Call(ctx, "ping", nil, &pong); err != nil {
			return nil, err
		}
		return pong, nil
	})

	var got string
	require.NoError(t, a.Call(context.Background(), "echo_ping", nil, &got))
	assert.Equal(t, "pong", got)
}

func TestConnNotify(t *testing.T) {
	a, b, stop := connPair(t)
	defer stop()

	done := make(chan json.RawMessage, 1)
	b.Register("event", func(_ context.Context, p json.RawMessage) (any, error) {
		done <- p
		return nil, nil
	})

	require.NoError(t, a.Notify(context.Background(), "event", map[string]int{"n": 7}))
	select {
	case p := <-done:
		assert.JSONEq(t, `{"n":7}`, string(p))
	case <-time.After(time.Second):
		t.Fatal("notification not received")
	}
}

func TestConnCallContextCancel(t *testing.T) {
	a, b, stop := connPair(t)
	defer stop()

	block := make(chan struct{})
	b.Register("slow", func(_ context.Context, _ json.RawMessage) (any, error) {
		<-block
		return nil, nil
	})
	defer close(block)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	err := a.Call(ctx, "slow", nil, nil)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestConnParamsEncodeError(t *testing.T) {
	a, _, stop := connPair(t)
	defer stop()

	err := a.Call(context.Background(), "m", make(chan int), nil)
	require.ErrorContains(t, err, "encode params")

	err = a.Notify(context.Background(), "m", make(chan int))
	require.ErrorContains(t, err, "encode params")
}

func TestConnCallCancelledContext(t *testing.T) {
	a, b, stop := connPair(t)
	defer stop()

	block := make(chan struct{})
	b.Register("blocks", func(_ context.Context, _ json.RawMessage) (any, error) {
		<-block
		return 1, nil
	})
	defer close(block)

	ctx, cancel := context.WithCancel(context.Background())
	go cancel()
	err := a.Call(ctx, "blocks", nil, nil)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestConnConcurrentCalls(t *testing.T) {
	a, b, stop := connPair(t)
	defer stop()

	b.Register("double", func(_ context.Context, p json.RawMessage) (any, error) {
		var n int
		require.NoError(t, json.Unmarshal(p, &n))
		return n * 2, nil
	})

	var wg sync.WaitGroup
	for i := range 50 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var got int
			require.NoError(t, a.Call(context.Background(), "double", i, &got))
			assert.Equal(t, i*2, got)
		}()
	}
	wg.Wait()
}

func TestConnServeStopsOnReadError(t *testing.T) {
	ar, aw := io.Pipe()
	c := NewConn(rwc{ar, aw})

	errCh := make(chan error, 1)
	go func() { errCh <- c.Serve(context.Background()) }()
	require.NoError(t, aw.Close()) // closes the write end; reader sees EOF

	select {
	case err := <-errCh:
		assert.ErrorIs(t, err, io.EOF)
	case <-time.After(time.Second):
		t.Fatal("Serve did not return after read error")
	}
}

func TestIsResponse(t *testing.T) {
	assert.True(t, isResponse([]byte(`{"jsonrpc":"2.0","result":1,"id":1}`)))
	assert.True(t, isResponse([]byte(`{"jsonrpc":"2.0","error":{"code":-1,"message":"x"},"id":1}`)))
	assert.False(t, isResponse([]byte(`{"jsonrpc":"2.0","method":"m","id":1}`)))
	assert.False(t, isResponse([]byte(`{"jsonrpc":"2.0","method":"m"}`)))
	assert.False(t, isResponse([]byte(`not json`)))
}
