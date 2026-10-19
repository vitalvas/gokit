package jsonrpc2

import (
	"context"
	"encoding/json"
	"fmt"
	"sync/atomic"
)

// Transport sends an encoded request payload and returns the raw response
// payload. It is supplied by the caller and performs the actual I/O (HTTP POST,
// stdio, a net.Conn, etc.). For a notification the returned bytes are ignored.
type Transport func(ctx context.Context, payload []byte) ([]byte, error)

// Client issues JSON-RPC 2.0 calls over a caller-supplied Transport. It
// assigns a unique numeric id to each non-notification request.
type Client struct {
	transport Transport
	seq       atomic.Uint64
}

// NewClient returns a Client that sends requests through transport.
func NewClient(transport Transport) *Client {
	return &Client{transport: transport}
}

// Call invokes method with params (nil to omit) and decodes the result into
// out (nil to discard). A JSON-RPC error response is returned as *Error.
func (c *Client) Call(ctx context.Context, method string, params, out any) error {
	rawParams, err := encodeParams(params)
	if err != nil {
		return err
	}

	id := NewNumberID(float64(c.seq.Add(1)))
	reqBytes, err := json.Marshal(Request{JSONRPC: Version, Method: method, Params: rawParams, ID: id})
	if err != nil {
		return fmt.Errorf("jsonrpc2: encode request: %w", err)
	}

	respBytes, err := c.transport(ctx, reqBytes)
	if err != nil {
		return fmt.Errorf("jsonrpc2: transport: %w", err)
	}

	var resp Response
	if err := json.Unmarshal(respBytes, &resp); err != nil {
		return fmt.Errorf("jsonrpc2: decode response: %w", err)
	}
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

// Notify sends a notification (a request without an id). The server sends no
// response, so any transport reply is ignored.
func (c *Client) Notify(ctx context.Context, method string, params any) error {
	rawParams, err := encodeParams(params)
	if err != nil {
		return err
	}

	reqBytes, err := json.Marshal(Request{JSONRPC: Version, Method: method, Params: rawParams})
	if err != nil {
		return fmt.Errorf("jsonrpc2: encode notification: %w", err)
	}

	if _, err := c.transport(ctx, reqBytes); err != nil {
		return fmt.Errorf("jsonrpc2: transport: %w", err)
	}
	return nil
}

func encodeParams(params any) (json.RawMessage, error) {
	if params == nil {
		return nil, nil
	}
	raw, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("jsonrpc2: encode params: %w", err)
	}
	return raw, nil
}
