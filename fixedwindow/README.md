# fixedwindow

A fixed window counter rate limiter in Go with per-key lockout, configurable cleanup, and oldest-window eviction.

## Features

- **Fixed window counter**: Tracks event counts per key within fixed time windows
- **Per-key lockout**: Keys exceeding their limit are locked out for the remainder of the window
- **Independent windows**: Each key's window starts on first access, not wall-clock aligned
- **Max key eviction**: Oldest-window eviction when max tracked keys is reached
- **Configurable cleanup**: Optional background goroutine for expired entry removal
- **Thread-safe**: All operations protected by mutex
- **O(1) operations**: Map lookup + counter increment per operation
- **Zero dependencies**: Only uses Go standard library

## What is Fixed Window Counter?

The fixed window counter algorithm divides time into fixed-duration windows and counts events per key within each window. When a key's count exceeds a configured limit, all subsequent requests are denied until the window resets.

**Key Properties:**

- **O(1) per operation**: Map lookup + counter increment
- **O(n) memory**: Where n = number of tracked keys
- **Hard lockout**: Once exceeded, all requests denied until window reset
- **Per-key windows**: Each key has its own independent window

**Use Cases:** Daily message quotas, API request quotas, account-level rate limiting with lockout behavior.

**References:**

- Gmail sending limits (fixed 24h window with lockout)
- Cloudflare Rate Limiting (fixed window counters)
- RFC 6585 (429 Too Many Requests)

## Installation

```bash
go get github.com/vitalvas/gokit/fixedwindow
```

## Quick Start

```go
package main

import (
    "fmt"
    "time"

    "github.com/vitalvas/gokit/fixedwindow"
)

func main() {
    // Create counter: 1-hour window, max 10000 keys, cleanup every 5 minutes
    c := fixedwindow.New(time.Hour, 10000, 5*time.Minute)
    defer c.Stop()

    // Check if request is allowed (consumes one unit)
    if c.Allow("user-123", 100) {
        fmt.Println("Request allowed")
    } else {
        fmt.Println("Rate limit exceeded")
    }

    // Check current count without consuming
    fmt.Printf("Current count: %d\n", c.Count("user-123"))

    // Check if locked out
    if c.IsLockedOut("user-123") {
        fmt.Println("User is locked out")
    }
}
```

## Creating a Counter

### New

Create a new fixed window counter with specified parameters.

```go
// 1-hour window, max 10000 keys, cleanup every 5 minutes
c := fixedwindow.New(time.Hour, 10000, 5*time.Minute)
defer c.Stop()

// 24-hour window, max 100000 keys, no background cleanup (lazy only)
c := fixedwindow.New(24*time.Hour, 100000, 0)
defer c.Stop()

// 1-minute window, max 1000 keys, cleanup every 10 seconds
c := fixedwindow.New(time.Minute, 1000, 10*time.Second)
defer c.Stop()
```

**Parameters:**

- `window`: Duration of each fixed window (zero/negative defaults to 1 minute)
- `maxKeys`: Maximum number of tracked keys (zero/negative defaults to 1000)
- `cleanupInterval`: Background cleanup interval; zero or negative disables background cleanup

**Cleanup Modes:**

| Mode | cleanupInterval | Behavior |
|------|----------------|----------|
| Lazy only | `0` | Expired entries removed on access or eviction |
| Background | `> 0` | Background goroutine removes expired entries periodically |

**Guidelines:**

- Set `maxKeys` large enough that eviction rarely occurs in production
- Set `cleanupInterval` to a fraction of the window duration (e.g., window/10)
- Always call `Stop()` when done to release the background goroutine

### Stop

Stop the background cleanup goroutine. Safe to call multiple times.

```go
c := fixedwindow.New(time.Hour, 10000, 5*time.Minute)

// When done
c.Stop()

// Safe to call again
c.Stop()
```

## Checking Rate Limits

### Allow

Check whether a key is within its rate limit and consume one unit atomically. This is the primary method for rate limiting.

```go
c := fixedwindow.New(time.Hour, 10000, 0)
defer c.Stop()

// Returns true if within limit
if c.Allow("user-123", 100) {
    // Process request
} else {
    // Reject request (429 Too Many Requests)
}
```

**Behavior:**

1. Increments the counter by 1 (always, even after lockout)
2. If the count exceeds the limit, sets lockout flag
3. Returns `true` if not locked out, `false` if locked out

**Important:** `Allow` always consumes a unit. For read-only checks, use `IsLockedOut` or `Count`.

### IsLockedOut

Check if a key is currently locked out (read-only, does not consume).

```go
c := fixedwindow.New(time.Hour, 10000, 0)
defer c.Stop()

if c.IsLockedOut("user-123") {
    fmt.Println("User is locked out until window resets")
}
```

**Returns:**

- `true`: Key has exceeded its limit and is locked out
- `false`: Key is not tracked, window expired, or within limit

## Counting Events

### Add

Increment the counter for a key by a specific delta. Returns the new count. The counter is always incremented regardless of lockout status (useful for monitoring over-limit usage).

```go
c := fixedwindow.New(time.Hour, 10000, 0)
defer c.Stop()

// Increment by 1
count := c.Add("user-123", 1)
fmt.Printf("Count: %d\n", count)

// Increment by batch size
count = c.Add("user-123", 50)
fmt.Printf("Count after batch: %d\n", count)
```

**Important:** `Add` does **not** trigger lockout. Only `Allow` sets the lockout flag. This means you can use `Add` to track usage without enforcing limits.

**Zero/negative delta:** Returns current count without modifying.

### Count

Get the current counter value for a key (read-only).

```go
c := fixedwindow.New(time.Hour, 10000, 0)
defer c.Stop()

c.Add("user-123", 5)

count := c.Count("user-123")
fmt.Printf("Count: %d\n", count)
// Output: Count: 5
```

**Returns:**

- Current count if key is tracked and window is active
- `0` if key is not tracked or window has expired

**Side effect:** Expired entries are lazily removed on access.

## Managing State

### Reset

Clear all tracked keys and their counters.

```go
c := fixedwindow.New(time.Hour, 10000, 0)
defer c.Stop()

c.Add("key1", 5)
c.Add("key2", 10)

c.Reset()

fmt.Printf("Count: %d\n", c.Count("key1"))
// Output: Count: 0
```

### Len

Get the number of currently tracked keys.

```go
c := fixedwindow.New(time.Hour, 10000, 0)
defer c.Stop()

c.Add("key1", 1)
c.Add("key2", 1)

fmt.Printf("Tracked keys: %d\n", c.Len())
// Output: Tracked keys: 2
```

**Note:** May include expired entries that have not yet been cleaned up.

## Eviction Behavior

When the number of tracked keys reaches `maxKeys`, the counter must evict to make room for new keys.

**Eviction strategy:**

1. First, remove all expired entries
2. If still at capacity, evict the key with the oldest (earliest) window end time

```go
c := fixedwindow.New(time.Minute, 3, 0)
defer c.Stop()

c.Add("key1", 1)
c.Add("key2", 1)
c.Add("key3", 1)

// At capacity (3 keys). Adding key4 evicts oldest window.
c.Add("key4", 1)

fmt.Printf("key1 count: %d\n", c.Count("key1"))
// Output: key1 count: 0 (evicted)
```

**Sizing guidelines:**

| Use Case | Suggested maxKeys |
|----------|-------------------|
| Per-user API limits | Number of active users |
| Per-IP rate limiting | Number of unique client IPs |
| Per-endpoint limiting | Number of endpoints |

## Use Cases

### API Request Quota

Rate limit API requests per user with hourly quotas.

```go
limiter := fixedwindow.New(time.Hour, 100000, 5*time.Minute)
defer limiter.Stop()

func handleRequest(w http.ResponseWriter, r *http.Request) {
    userID := getUserID(r)

    if !limiter.Allow(userID, 1000) {
        http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
        return
    }

    // Process request...
}
```

### Daily Email Sending Limit

Enforce daily email sending quotas (similar to Gmail limits).

```go
emailLimiter := fixedwindow.New(24*time.Hour, 50000, time.Hour)
defer emailLimiter.Stop()

func sendEmail(senderID string, msg *Email) error {
    if !emailLimiter.Allow(senderID, 500) {
        return fmt.Errorf("daily sending limit exceeded, try again tomorrow")
    }

    return deliverEmail(msg)
}

func getEmailsRemaining(senderID string) int {
    used := emailLimiter.Count(senderID)
    remaining := 500 - used
    if remaining < 0 {
        return 0
    }
    return remaining
}
```

### Account-Level Rate Limiting

Track and enforce multiple limits per account.

```go
type RateLimiter struct {
    minutely *fixedwindow.Counter
    hourly   *fixedwindow.Counter
    daily    *fixedwindow.Counter
}

func NewRateLimiter(maxAccounts int) *RateLimiter {
    return &RateLimiter{
        minutely: fixedwindow.New(time.Minute, maxAccounts, 10*time.Second),
        hourly:   fixedwindow.New(time.Hour, maxAccounts, time.Minute),
        daily:    fixedwindow.New(24*time.Hour, maxAccounts, 10*time.Minute),
    }
}

func (rl *RateLimiter) Allow(accountID string) bool {
    // Check all windows - most restrictive wins
    if !rl.minutely.Allow(accountID, 60) {
        return false
    }
    if !rl.hourly.Allow(accountID, 1000) {
        return false
    }
    if !rl.daily.Allow(accountID, 10000) {
        return false
    }
    return true
}

func (rl *RateLimiter) Stop() {
    rl.minutely.Stop()
    rl.hourly.Stop()
    rl.daily.Stop()
}
```

### Login Attempt Limiting

Lock out accounts after too many failed login attempts.

```go
loginLimiter := fixedwindow.New(15*time.Minute, 100000, time.Minute)
defer loginLimiter.Stop()

func handleLogin(username, password string) error {
    if loginLimiter.IsLockedOut(username) {
        return fmt.Errorf("account temporarily locked, try again later")
    }

    if !authenticate(username, password) {
        loginLimiter.Allow(username, 5) // 5 attempts per 15 minutes
        return fmt.Errorf("invalid credentials")
    }

    return nil
}
```

### Monitoring Over-Limit Usage

Use `Add` to track actual usage even when locked out.

```go
limiter := fixedwindow.New(time.Hour, 10000, 5*time.Minute)
defer limiter.Stop()

func processRequest(userID string) bool {
    allowed := limiter.Allow(userID, 100)
    if !allowed {
        // Track how far over the limit users go
        overage := limiter.Count(userID) - 100
        metrics.RecordOverage(userID, overage)
    }
    return allowed
}
```

## Performance Characteristics

### Time Complexity

| Operation | Complexity | Description |
|-----------|------------|-------------|
| `Allow` | O(1) | Map lookup + counter increment |
| `Add` | O(1) | Map lookup + counter increment |
| `Count` | O(1) | Map lookup |
| `IsLockedOut` | O(1) | Map lookup |
| `Reset` | O(1) | Replace map |
| `Len` | O(1) | Map length |
| Eviction | O(n) | Scan all entries (only on capacity) |

Where n = number of tracked keys

### Space Complexity

**Memory usage:** O(n) where n = number of tracked keys

**Per-key overhead:** ~80 bytes (entry struct + map overhead)

| Keys | Memory |
|------|--------|
| 1,000 | ~80 KB |
| 10,000 | ~800 KB |
| 100,000 | ~8 MB |
| 1,000,000 | ~80 MB |

## Thread Safety

All operations are thread-safe and protected by mutex:

```go
c := fixedwindow.New(time.Hour, 10000, 0)
defer c.Stop()

// Safe to call from multiple goroutines
go func() {
    for range requests {
        c.Allow("user-1", 100)
    }
}()

go func() {
    for range requests {
        c.Allow("user-2", 100)
    }
}()

go func() {
    ticker := time.NewTicker(time.Second)
    for range ticker.C {
        fmt.Printf("User-1 count: %d\n", c.Count("user-1"))
    }
}()
```

## Comparison with Other Rate Limiting Algorithms

| Algorithm | Burst Handling | Memory | Smoothness | Lockout |
|-----------|---------------|--------|------------|---------|
| Fixed Window | Allows burst at boundary | O(n) | Low | Hard lockout |
| Sliding Window Log | No boundary burst | O(n*r) | High | Soft |
| Sliding Window Counter | Reduced boundary burst | O(n) | Medium | Soft |
| Token Bucket | Controlled burst | O(n) | High | Soft |
| Leaky Bucket | No burst | O(n) | Very high | Soft |

**When to Use Fixed Window Counter:**

- Need simple, predictable quotas (daily/hourly limits)
- Hard lockout is desired behavior (not gradual degradation)
- Low overhead is important (O(1) per operation)
- Boundary burst is acceptable

**When NOT to Use:**

- Need smooth rate limiting (use token bucket or leaky bucket)
- Need sub-second precision (use sliding window)
- Boundary burst is unacceptable (use sliding window log)

## Best Practices

### Size maxKeys Appropriately

```go
// Too small: frequent eviction, losing tracking data
c := fixedwindow.New(time.Hour, 10, 0)

// Good: sized for expected number of unique keys
c := fixedwindow.New(time.Hour, 100000, 0)
```

**Rule of thumb:** Set `maxKeys` to 2x the expected number of unique keys within a window.

### Use Background Cleanup for Long Windows

```go
// 24-hour window: use background cleanup to free memory
c := fixedwindow.New(24*time.Hour, 100000, time.Hour)
defer c.Stop()

// 1-minute window: lazy cleanup is sufficient
c := fixedwindow.New(time.Minute, 10000, 0)
defer c.Stop()
```

### Always Call Stop

```go
c := fixedwindow.New(time.Hour, 10000, 5*time.Minute)

// Always stop when done to prevent goroutine leaks
defer c.Stop()
```

### Separate Counters for Different Limits

```go
// Different windows for different resources
apiLimiter := fixedwindow.New(time.Hour, 100000, 5*time.Minute)
emailLimiter := fixedwindow.New(24*time.Hour, 50000, time.Hour)
loginLimiter := fixedwindow.New(15*time.Minute, 100000, time.Minute)

defer apiLimiter.Stop()
defer emailLimiter.Stop()
defer loginLimiter.Stop()
```

## License

This project is part of the [gokit](https://github.com/vitalvas/gokit) library.
