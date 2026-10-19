package jsonrpc2

import (
	"bytes"
	"context"
	"encoding/json"
)

// Handler handles a single JSON-RPC method call. params is the raw params
// member (nil if omitted). A returned error of type *Error is sent verbatim;
// any other error is reported as an Internal error (-32603).
type Handler func(ctx context.Context, params json.RawMessage) (any, error)

// Server dispatches JSON-RPC 2.0 requests to registered method handlers. It is
// transport-neutral: feed it a raw request payload and it returns the raw
// response payload (nil when nothing should be sent, per the spec).
type Server struct {
	methods map[string]Handler
}

// NewServer returns a Server with no registered methods.
func NewServer() *Server {
	return &Server{methods: make(map[string]Handler)}
}

// Register associates a handler with a method name, replacing any existing one.
func (s *Server) Register(method string, h Handler) {
	s.methods[method] = h
}

// Handle processes a single request or a batch and returns the raw JSON
// response payload. It returns nil when there is nothing to send: a single
// notification, or a batch containing only notifications.
func (s *Server) Handle(ctx context.Context, payload []byte) []byte {
	trimmed := bytes.TrimSpace(payload)

	// A batch is a JSON array; anything else is treated as a single request.
	if len(trimmed) > 0 && trimmed[0] == '[' {
		return s.handleBatch(ctx, trimmed)
	}

	resp := s.handleSingle(ctx, trimmed)
	if resp == nil {
		return nil
	}
	out, err := json.Marshal(resp)
	if err != nil {
		return mustMarshal(errorResponse(nil, NewError(CodeInternalError, "Internal error", nil)))
	}
	return out
}

func (s *Server) handleBatch(ctx context.Context, payload []byte) []byte {
	var raws []json.RawMessage
	if err := json.Unmarshal(payload, &raws); err != nil {
		return mustMarshal(errorResponse(nil, NewError(CodeParseError, "Parse error", nil)))
	}
	if len(raws) == 0 {
		return mustMarshal(errorResponse(nil, NewError(CodeInvalidRequest, "Invalid Request", nil)))
	}

	responses := make([]*Response, 0, len(raws))
	for _, raw := range raws {
		if resp := s.handleSingle(ctx, raw); resp != nil {
			responses = append(responses, resp)
		}
	}
	if len(responses) == 0 {
		return nil
	}
	return mustMarshal(responses)
}

// handleSingle processes one request object. It returns nil when no response is
// due (a valid notification).
func (s *Server) handleSingle(ctx context.Context, raw []byte) *Response {
	var req Request
	if err := json.Unmarshal(raw, &req); err != nil {
		if _, ok := err.(*json.SyntaxError); ok {
			return errorResponse(nil, NewError(CodeParseError, "Parse error", nil))
		}
		return errorResponse(nil, NewError(CodeInvalidRequest, "Invalid Request", nil))
	}

	if req.JSONRPC != Version || req.Method == "" {
		return errorResponse(req.ID, NewError(CodeInvalidRequest, "Invalid Request", nil))
	}

	h, ok := s.methods[req.Method]
	if !ok {
		if req.IsNotification() {
			return nil
		}
		return errorResponse(req.ID, NewError(CodeMethodNotFound, "Method not found", nil))
	}

	result, err := call(ctx, h, req.Params)
	if req.IsNotification() {
		return nil
	}
	if err != nil {
		return errorResponse(req.ID, toError(err))
	}
	return resultResponse(req.ID, result)
}

// call invokes a handler, converting a panic into an Internal error so one
// faulty handler cannot crash the dispatch loop.
func call(ctx context.Context, h Handler, params json.RawMessage) (result any, err error) {
	defer func() {
		if r := recover(); r != nil {
			result = nil
			err = NewError(CodeInternalError, "Internal error", nil)
		}
	}()
	return h(ctx, params)
}

func toError(err error) *Error {
	if e, ok := err.(*Error); ok {
		return e
	}
	return NewError(CodeInternalError, "Internal error", nil)
}

func errorResponse(id *ID, e *Error) *Response {
	return &Response{JSONRPC: Version, Error: e, ID: id}
}

func resultResponse(id *ID, result any) *Response {
	raw, err := json.Marshal(result)
	if err != nil {
		return errorResponse(id, NewError(CodeInternalError, "Internal error", nil))
	}
	return &Response{JSONRPC: Version, Result: raw, ID: id}
}

func mustMarshal(v any) []byte {
	out, err := json.Marshal(v)
	if err != nil {
		return []byte(`{"jsonrpc":"2.0","error":{"code":-32603,"message":"Internal error"},"id":null}`)
	}
	return out
}
