# xcmd

Package xcmd provides utilities for command execution and goroutine management.

## Installation

```bash
go get github.com/vitalvas/gokit/xcmd
```

## Features

### ErrGroup

A goroutine group manager that cancels all goroutines on the first error. Unlike `golang.org/x/sync/errgroup`, this implementation immediately cancels the context when the first error occurs, allowing other goroutines to shut down gracefully.

**Usage:**

```go
package main

import (
    "context"
    "errors"
    "fmt"
    "time"

    "github.com/vitalvas/gokit/xcmd"
)

func main() {
    group, ctx := xcmd.ErrGroup(context.Background())

    group.Go(func(ctx context.Context) error {
        select {
        case <-ctx.Done():
            fmt.Println("Task 1 cancelled")
            return ctx.Err()
        case <-time.After(5 * time.Second):
            return nil
        }
    })

    group.Go(func(ctx context.Context) error {
        return errors.New("task 2 failed")
    })

    if err := group.Wait(); err != nil {
        fmt.Printf("Error: %v\n", err)
    }
}
```

**Key differences from errgroup:**

- Context is automatically cancelled on first error
- Context is passed to each goroutine function
- Other goroutines can monitor `ctx.Done()` for immediate shutdown

### PeriodicRun

Execute a function periodically with context support.

**Usage:**

```go
err := xcmd.PeriodicRun(ctx, func(ctx context.Context) error {
    // Your periodic task
    return nil
}, 1*time.Minute)
```

### PeriodicRunWithSignal

Execute a function periodically or when a signal is received.

**Usage:**

```go
err := xcmd.PeriodicRunWithSignal(
    ctx,
    func(ctx context.Context) error {
        // Your task
        return nil
    },
    1*time.Minute,
    syscall.SIGUSR1,
)
```

### WaitInterrupted

Block until a signal is received or context is cancelled.

**Usage:**

```go
err := xcmd.WaitInterrupted(ctx, syscall.SIGINT, syscall.SIGTERM)
```

## License

This package is part of the gokit project.
