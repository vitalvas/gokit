# jsonrpc2

A JSON-RPC 2.0 implementation in Go: spec-correct message types and a transport-neutral server dispatcher.

Implements the [JSON-RPC 2.0 specification](https://www.jsonrpc.org/specification).

## Features

- **Message types**: `Request`, `Response`, and `Error` with spec-correct JSON encoding
- **Flexible id**: request id as string, number, or null, per the spec
- **Notifications**: requests without an id produce no response
- **Batch requests**: array payloads processed as a batch, with the all-notifications case returning nothing
- **Standard error codes**: parse error, invalid request, method not found, invalid params, internal error
- **Transport-neutral**: feed raw bytes, get raw bytes back; wire it to HTTP, stdio, or any stream yourself
- **Panic-safe dispatch**: a handler panic becomes an internal error instead of crashing the loop
- **Zero dependencies**: only the Go standard library

## Quick Start

```go
package main

import (
    "context"
    "encoding/json"
    "fmt"

    "github.com/vitalvas/gokit/jsonrpc2"
)

func main() {
    s := jsonrpc2.NewServer()
    s.Register("subtract", func(ctx context.Context, params json.RawMessage) (any, error) {
        var args []int
        if err := json.Unmarshal(params, &args); err != nil || len(args) != 2 {
            return nil, jsonrpc2.NewError(jsonrpc2.CodeInvalidParams, "Invalid params", nil)
        }
        return args[0] - args[1], nil
    })

    req := []byte(`{"jsonrpc":"2.0","method":"subtract","params":[42,23],"id":1}`)
    resp := s.Handle(context.Background(), req)
    fmt.Println(string(resp))
    // {"jsonrpc":"2.0","result":19,"id":1}
}
```

## Server

### NewServer / Register

```go
s := jsonrpc2.NewServer()
s.Register("method_name", handler)
```

A handler has the signature:

```go
func(ctx context.Context, params json.RawMessage) (any, error)
```

- `params` is the raw `params` member (nil when omitted). Unmarshal it into a slice for positional params or a struct for named params.
- Return a value to send as `result`.
- Return an `*jsonrpc2.Error` to send that error verbatim. Any other error becomes an internal error (`-32603`).

Registering the same method name again replaces the previous handler.

### Handle

```go
resp := s.Handle(ctx, payload)
```

`Handle` accepts a single request or a batch (a JSON array) and returns the raw JSON response payload. It returns `nil` when nothing should be sent:

- a single notification (request without an id), or
- a batch containing only notifications.

The transport layer must send nothing in those cases (for HTTP, a `204 No Content`).

## Client

The client is transport-neutral too: you supply a `Transport` that performs the
actual send (HTTP POST, stdio, a `net.Conn`), and the client handles id
assignment, encoding, and decoding.

```go
type Transport func(ctx context.Context, payload []byte) ([]byte, error)
```

### NewClient / Call / Notify

```go
c := jsonrpc2.NewClient(func(ctx context.Context, payload []byte) ([]byte, error) {
    resp, err := http.Post("http://localhost:8080/rpc", "application/json", bytes.NewReader(payload))
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    return io.ReadAll(resp.Body)
})

// Call assigns a unique id, sends, and decodes result into out (nil to discard).
var result int
if err := c.Call(ctx, "subtract", []int{42, 23}, &result); err != nil {
    // a JSON-RPC error response is returned as *jsonrpc2.Error
    var rpcErr *jsonrpc2.Error
    if errors.As(err, &rpcErr) {
        fmt.Println("rpc error:", rpcErr.Code, rpcErr.Message)
    }
}

// Notify sends a notification (no id, no response expected).
err := c.Notify(ctx, "log_event", map[string]string{"level": "info"})
```

- `params` is any JSON-marshalable value (nil to omit), positional (a slice) or named (a struct/map).
- `out` is unmarshaled from the `result` member; pass nil to discard it.
- A JSON-RPC error response surfaces as `*jsonrpc2.Error`; use `errors.As` to inspect the code.
- Each `Call` gets a unique auto-incrementing numeric id.

## Bidirectional Conn

`Server` and `Client` are one-directional. When both peers send requests *and*
answer them over a single stream, use `Conn`. It runs a read loop that
dispatches inbound requests to handlers while concurrently issuing outbound
calls and matching their responses by id. Messages are newline-delimited JSON.

```go
// stdio peer: read from stdin, write to stdout.
type stdio struct {
    io.Reader
    io.Writer
}
func (stdio) Close() error { return nil }

conn := jsonrpc2.NewConn(stdio{os.Stdin, os.Stdout})

// handlers for inbound requests
conn.Register("tools/list", func(ctx context.Context, params json.RawMessage) (any, error) {
    return []string{"search", "fetch"}, nil
})

go conn.Serve(ctx) // read loop; returns io.EOF on clean close

// outbound call (e.g. a server asking the client to sample)
var result SampleResult
if err := conn.Call(ctx, "sampling/createMessage", req, &result); err != nil {
    // ...
}

// outbound notification
conn.Notify(ctx, "notifications/progress", progress)
```

A handler may itself call back to the other peer (`conn.Call` inside a handler)
— inbound requests are dispatched in their own goroutine, so the read loop stays
free to deliver the response. Each `Call` blocks until the matching response
arrives or `ctx` is cancelled. Writes are serialized, so `Conn` is safe for
concurrent `Call`/`Notify` from many goroutines.

| Type | Role | Use |
|------|------|-----|
| `Server` | inbound only | request/response server, you own the transport |
| `Client` | outbound only | request/response client, you own the transport |
| `Conn` | both, over one stream | bidirectional peers |

## Message Types

### Request

```go
type Request struct {
    JSONRPC string          // "2.0"
    Method  string
    Params  json.RawMessage // omitted when nil
    ID      *ID             // nil = notification
}

func (r *Request) IsNotification() bool
```

### Response

```go
type Response struct {
    JSONRPC string          // "2.0"
    Result  json.RawMessage // set on success
    Error   *Error          // set on error
    ID      *ID
}
```

### ID

A JSON-RPC id is a string, a number, or null.

```go
jsonrpc2.NewStringID("abc") // "abc"
jsonrpc2.NewNumberID(42)    // 42

id.IsNull()  // true when the id is JSON null
id.String()  // stable string form for logging / map keys
```

## Errors

### Standard Codes

| Constant | Code | Meaning |
|----------|------|---------|
| `CodeParseError` | -32700 | Invalid JSON was received |
| `CodeInvalidRequest` | -32600 | The JSON is not a valid Request object |
| `CodeMethodNotFound` | -32601 | The method does not exist |
| `CodeInvalidParams` | -32602 | Invalid method parameters |
| `CodeInternalError` | -32603 | Internal JSON-RPC error |

Codes from `-32000` to `-32099` are reserved for implementation-defined server errors.

### NewError

```go
err := jsonrpc2.NewError(jsonrpc2.CodeInvalidParams, "Invalid params", nil)

// with structured data
err := jsonrpc2.NewError(-32000, "Out of range", map[string]int{"max": 100})
```

`*Error` implements the `error` interface, so handlers can return it directly.

## Batch Processing

Per the spec:

- An array payload is a batch; each element is processed independently.
- An empty array (`[]`) is a single Invalid Request response.
- A batch element that is not a valid Request object gets an Invalid Request response with a null id.
- Notifications in a batch produce no response.
- If every element is a notification, `Handle` returns `nil` (send nothing, not an empty array).
- Response order within the array is not guaranteed to match request order.

```go
batch := []byte(`[
    {"jsonrpc":"2.0","method":"sum","params":[1,2,4],"id":"1"},
    {"jsonrpc":"2.0","method":"notify","params":[7]},
    {"jsonrpc":"2.0","method":"subtract","params":[42,23],"id":"2"}
]`)
resp := s.Handle(ctx, batch)
// [{"jsonrpc":"2.0","result":7,"id":"1"},{"jsonrpc":"2.0","result":19,"id":"2"}]
```

## Transport

The server is transport-neutral. A minimal HTTP binding with `net/http`:

```go
func (s *jsonrpc2.Server) httpHandler(w http.ResponseWriter, r *http.Request) {
    body, _ := io.ReadAll(r.Body)
    resp := s.Handle(r.Context(), body)
    if resp == nil {
        w.WriteHeader(http.StatusNoContent)
        return
    }
    w.Header().Set("Content-Type", "application/json")
    w.Write(resp)
}
```

## License

This project is part of the [gokit](https://github.com/vitalvas/gokit) library.
