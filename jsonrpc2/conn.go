package jsonrpc2

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
)

// Conn is a bidirectional JSON-RPC 2.0 peer over a single stream: it dispatches
// inbound requests to registered handlers while concurrently issuing outbound
// calls and matching their responses by id.
//
// Messages are newline-delimited JSON. Serve must be running for Call to
// receive responses.
type Conn struct {
	rw   io.ReadWriteCloser
	srv  *Server
	enc  *json.Encoder
	dec  *json.Decoder
	wmu  sync.Mutex
	seq  atomic.Uint64
	pend sync.Map // id string -> chan *Response
}

// NewConn returns a Conn over rw. Register handlers with Register, then run
// Serve to process inbound messages.
func NewConn(rw io.ReadWriteCloser) *Conn {
	return &Conn{
		rw:  rw,
		srv: NewServer(),
		enc: json.NewEncoder(rw),
		dec: json.NewDecoder(rw),
	}
}

// Register associates a handler with an inbound method name.
func (c *Conn) Register(method string, h Handler) {
	c.srv.Register(method, h)
}

// Serve reads messages until the stream closes or ctx is cancelled. Inbound
// requests are dispatched to handlers and their responses written back;
// inbound responses are delivered to the matching pending Call. It returns the
// read error that stopped the loop (io.EOF on a clean close).
func (c *Conn) Serve(ctx context.Context) error {
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		var raw json.RawMessage
		if err := c.dec.Decode(&raw); err != nil {
			return err
		}
		c.route(ctx, raw)
	}
}

// route dispatches a single decoded message. Responses are delivered inline to
// the matching pending Call. Requests are dispatched in their own goroutine so
// the read loop stays free to deliver responses to outbound calls a handler
// itself makes (bidirectional re-entrancy).
func (c *Conn) route(ctx context.Context, raw json.RawMessage) {
	if isResponse(raw) {
		var resp Response
		if err := json.Unmarshal(raw, &resp); err != nil {
			return
		}
		if ch, ok := c.pend.LoadAndDelete(resp.ID.String()); ok {
			ch.(chan *Response) <- &resp
		}
		return
	}

	go func() {
		resp := c.srv.handleSingle(ctx, raw)
		if resp == nil {
			return
		}
		c.wmu.Lock()
		_ = c.enc.Encode(resp)
		c.wmu.Unlock()
	}()
}

// Call sends a request, blocks until the matching response arrives, and decodes
// result into out (nil to discard). A JSON-RPC error response is returned as
// *Error. It unblocks if ctx is cancelled.
func (c *Conn) Call(ctx context.Context, method string, params, out any) error {
	rawParams, err := encodeParams(params)
	if err != nil {
		return err
	}

	id := NewNumberID(float64(c.seq.Add(1)))
	ch := make(chan *Response, 1)
	c.pend.Store(id.String(), ch)

	c.wmu.Lock()
	err = c.enc.Encode(Request{JSONRPC: Version, Method: method, Params: rawParams, ID: id})
	c.wmu.Unlock()
	if err != nil {
		c.pend.Delete(id.String())
		return fmt.Errorf("jsonrpc2: write request: %w", err)
	}

	select {
	case <-ctx.Done():
		c.pend.Delete(id.String())
		return ctx.Err()
	case resp := <-ch:
		if resp.Error != nil {
			return resp.Error
		}
		if out != nil && len(resp.Result) > 0 {
			if err := json.Unmarshal(resp.Result, out); err != nil {
				return fmt.Errorf("jsonrpc2: decode result: %w", err)
			}
		}
		return nil
	}
}

// Notify sends a notification (no id, no response expected).
func (c *Conn) Notify(_ context.Context, method string, params any) error {
	rawParams, err := encodeParams(params)
	if err != nil {
		return err
	}

	c.wmu.Lock()
	err = c.enc.Encode(Request{JSONRPC: Version, Method: method, Params: rawParams})
	c.wmu.Unlock()
	if err != nil {
		return fmt.Errorf("jsonrpc2: write notification: %w", err)
	}
	return nil
}

// Close closes the underlying stream, which causes Serve to return.
func (c *Conn) Close() error {
	return c.rw.Close()
}

// isResponse reports whether a message is a response (carries result or error)
// rather than a request or notification (carries method).
func isResponse(raw json.RawMessage) bool {
	var probe struct {
		Method *string         `json:"method"`
		Result json.RawMessage `json:"result"`
		Error  json.RawMessage `json:"error"`
	}
	if err := json.Unmarshal(raw, &probe); err != nil {
		return false
	}
	return probe.Method == nil && (probe.Result != nil || probe.Error != nil)
}
