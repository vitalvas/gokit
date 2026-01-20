# Wirefilter

Wirefilter is a filtering expression language and execution engine for Go.
It allows you to compile and evaluate filter expressions against runtime data,
inspired by Cloudflare's Wirefilter.

## Features

- Logical operators: `and`, `or`, `not`, `xor`, `&&`, `||`, `!`, `^^`
- Comparison operators: `==`, `!=`, `<`, `>`, `<=`, `>=`
- Array operators: `===` (all equal), `!==` (any not equal)
- Membership operators: `in`, `contains`, `matches` (`~`)
- Wildcard matching: `wildcard` (case-insensitive), `strict wildcard` (case-sensitive)
- Field presence/absence checking
- Range expressions: `{1..10}`
- Multiple data types: string, int, bool, IP, bytes, arrays, maps
- Map field access with bracket notation
- Field-to-field comparisons
- IP/CIDR matching for IPv4 and IPv6
- Regular expression matching
- Schema validation for field references

## Installation

```bash
go get github.com/vitalvas/gokit/wirefilter
```

## Quick Start

```go
package main

import (
    "fmt"
    "log"

    "github.com/vitalvas/gokit/wirefilter"
)

func main() {
    schema := wirefilter.NewSchema().
        AddField("http.host", wirefilter.TypeString).
        AddField("http.status", wirefilter.TypeInt)

    filter, err := wirefilter.Compile(
        `http.host == "example.com" and http.status >= 400`, schema)
    if err != nil {
        log.Fatal(err)
    }

    ctx := wirefilter.NewExecutionContext().
        SetStringField("http.host", "example.com").
        SetIntField("http.status", 500)

    result, err := filter.Execute(ctx)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println(result)
}
```

## Language Syntax

### Basic Comparisons

```go
http.status == 200
http.status != 404
http.status > 400
http.status >= 500
```

### String Operations

```go
http.host == "example.com"
http.path contains "/api"
http.user_agent matches "^Mozilla.*"
http.user_agent ~ "^Mozilla.*"              // ~ is alias for matches
```

### Wildcard Matching

Glob-style pattern matching with `*` (any chars) and `?` (single char):

```go
http.host wildcard "*.example.com"          // case-insensitive
http.host wildcard "api?.example.com"       // ? matches single char
http.host strict wildcard "*.Example.com"   // case-sensitive
```

Examples:
- `"www.example.com" wildcard "*.example.com"` - true
- `"WWW.EXAMPLE.COM" wildcard "*.example.com"` - true (case-insensitive)
- `"WWW.Example.com" strict wildcard "*.Example.com"` - true
- `"www.example.com" strict wildcard "*.Example.com"` - false (case-sensitive)

### Combining Conditions

```go
http.host == "example.com" and http.status == 200
http.host == "example.com" && http.status == 200   // && is alias for and
http.status == 404 or http.status == 500
http.status == 404 || http.status == 500           // || is alias for or
not (http.status >= 500)
! http.secure                                      // ! is alias for not
http.secure xor http.authenticated                 // XOR: true if exactly one is true
http.secure ^^ http.authenticated                  // ^^ is alias for xor
```

### Field-to-Field Comparisons

Compare two fields directly:

```go
user.login == device.owner
user.age >= minimum.age
request.region == server.region
```

### Map Field Access

Access values in map fields using bracket notation:

```go
user.attributes["region"] == "us-west"
config["timeout"] == 30
user.attributes["role"] == device.settings["required_role"]
```

### Field Presence Checking

Check if a field is present (has been set):

```go
http.host                    // true if http.host is set
not http.error               // true if http.error is not set
http.host and not http.error // true if host is set and error is not set
```

Presence checking uses existence-based truthiness:
- Any field that exists is considered truthy (including zero values and empty strings)
- Missing fields are considered falsy
- For boolean fields, the actual boolean value is used

### IP and CIDR Matching

```go
ip.src == 192.168.1.1
ip.src in "192.168.0.0/16"
ip.src in "2001:db8::/32"
```

### Array Membership

```go
http.status in {200, 201, 204}
port in {80, 443, 8080}
```

### Array-to-Array Operations

Check if an array field has any or all elements from a set:

```go
// OR logic: true if ANY element from user.groups is in the set
user.groups in {"guest", "admin"}

// AND logic: true if ALL elements from the set exist in user.groups
user.groups contains {"guest", "admin"}
```

Example:
```go
groups := ["admin", "guest", "user"]

groups in {"guest", "test"}       // true  (guest matches)
groups in {"foo", "bar"}          // false (no match)
groups contains {"guest", "user"} // true  (both exist in groups)
groups contains {"guest", "test"} // false (test is missing)
```

### Range Expressions

```go
port in {80..100, 443, 8000..9000}
http.status in {200..299}
```

### Array Comparison

```go
tags === "production"
tags !== "deprecated"
```

## API Reference

### Creating a Schema

Define the fields that can be used in filter expressions:

#### Method 1: Using method chaining

```go
schema := wirefilter.NewSchema().
    AddField("http.host", wirefilter.TypeString).
    AddField("http.status", wirefilter.TypeInt).
    AddField("http.secure", wirefilter.TypeBool).
    AddField("ip.src", wirefilter.TypeIP)
```

#### Method 2: Using a fields map

```go
fields := map[string]wirefilter.Type{
    "http.host":   wirefilter.TypeString,
    "http.status": wirefilter.TypeInt,
    "http.secure": wirefilter.TypeBool,
    "ip.src":      wirefilter.TypeIP,
}

schema := wirefilter.NewSchema(fields)
```

#### Method 3: Using multiple field maps (merged)

```go
httpFields := map[string]wirefilter.Type{
    "http.host":   wirefilter.TypeString,
    "http.status": wirefilter.TypeInt,
}

networkFields := map[string]wirefilter.Type{
    "ip.src": wirefilter.TypeIP,
    "ip.dst": wirefilter.TypeIP,
}

schema := wirefilter.NewSchema(httpFields, networkFields)
```

### Compiling a Filter

Parse and validate a filter expression:

```go
filter, err := wirefilter.Compile(expression, schema)
if err != nil {
    log.Fatal(err)
}
```

If `schema` is `nil`, field validation is skipped.

### Execution Context

Set runtime values for evaluation:

```go
ctx := wirefilter.NewExecutionContext().
    SetStringField("http.host", "example.com").
    SetIntField("http.status", 200).
    SetBoolField("http.secure", true).
    SetIPField("ip.src", "192.168.1.1")
```

#### Setting Map Fields

For map fields with string values:

```go
ctx := wirefilter.NewExecutionContext().
    SetMapField("user.attributes", map[string]string{
        "region": "us-west",
        "role":   "admin",
    })
```

For map fields with mixed value types:

```go
ctx := wirefilter.NewExecutionContext().
    SetMapFieldValues("config", map[string]wirefilter.Value{
        "timeout": wirefilter.IntValue(30),
        "host":    wirefilter.StringValue("localhost"),
        "enabled": wirefilter.BoolValue(true),
    })
```

### Executing a Filter

Evaluate the filter against the context:

```go
result, err := filter.Execute(ctx)
if err != nil {
    log.Fatal(err)
}

if result {
    fmt.Println("Filter matched")
}
```

## Data Types

| Type | Description | Example |
|------|-------------|---------|
| `TypeString` | String values | `"example.com"` |
| `TypeInt` | Integer values | `200`, `-5` |
| `TypeBool` | Boolean values | `true`, `false` |
| `TypeIP` | IP addresses (IPv4/IPv6) | `192.168.1.1`, `2001:db8::1` |
| `TypeBytes` | Byte arrays | `[]byte("data")` |
| `TypeArray` | Arrays of values | `{1, 2, 3}` |
| `TypeMap` | Map of string keys to values | `map[string]string{"key": "value"}` |

## Operators

### Comparison Operators

| Operator | Description | Example |
|----------|-------------|---------|
| `==` | Equal | `status == 200` |
| `!=` | Not equal | `status != 404` |
| `<` | Less than | `status < 400` |
| `>` | Greater than | `status > 300` |
| `<=` | Less than or equal | `status <= 299` |
| `>=` | Greater than or equal | `status >= 500` |

### Logical Operators

| Operator | Description | Example |
|----------|-------------|---------|
| `and`, `&&` | Logical AND | `a and b`, `a && b` |
| `or`, `\|\|` | Logical OR | `a or b`, `a \|\| b` |
| `xor`, `^^` | Logical XOR (exclusive OR) | `a xor b`, `a ^^ b` |
| `not`, `!` | Logical NOT | `not a`, `! a` |

### Membership Operators

| Operator | Description | Example |
|----------|-------------|---------|
| `in` | Value in array, IP in CIDR, or array ANY match | `port in {80, 443}` |
| `contains` | String contains substring, or array ALL match | `path contains "/api"` |
| `matches`, `~` | Regex match | `ua matches "^Mozilla"`, `ua ~ "^Mozilla"` |

### Wildcard Operators

| Operator | Description | Example |
|----------|-------------|---------|
| `wildcard` | Glob pattern match (case-insensitive) | `host wildcard "*.example.com"` |
| `strict wildcard` | Glob pattern match (case-sensitive) | `host strict wildcard "*.Example.com"` |

Wildcard patterns support:
- `*` matches any sequence of characters (including empty)
- `?` matches any single character

### Array Operators

| Operator | Description | Example |
|----------|-------------|---------|
| `===` | All elements equal | `tags === "prod"` |
| `!==` | Any element not equal | `tags !== "test"` |

## Advanced Examples

### HTTP Request Filtering

```go
schema := wirefilter.NewSchema().
    AddField("http.method", wirefilter.TypeString).
    AddField("http.host", wirefilter.TypeString).
    AddField("http.path", wirefilter.TypeString).
    AddField("http.status", wirefilter.TypeInt)

expression := `
    http.method == "GET" and
    http.host == "api.example.com" and
    http.path contains "/v1/" and
    http.status >= 200 and http.status < 300
`

filter, _ := wirefilter.Compile(expression, schema)

ctx := wirefilter.NewExecutionContext().
    SetStringField("http.method", "GET").
    SetStringField("http.host", "api.example.com").
    SetStringField("http.path", "/v1/users").
    SetIntField("http.status", 200)

matched, _ := filter.Execute(ctx)
```

### Network Traffic Filtering

```go
schema := wirefilter.NewSchema().
    AddField("ip.src", wirefilter.TypeIP).
    AddField("ip.dst", wirefilter.TypeIP).
    AddField("port.dst", wirefilter.TypeInt).
    AddField("protocol", wirefilter.TypeString)

expression := `
    ip.src in "10.0.0.0/8" and
    port.dst in {80, 443, 8080..8090} and
    protocol == "tcp"
`

filter, _ := wirefilter.Compile(expression, schema)

ctx := wirefilter.NewExecutionContext().
    SetIPField("ip.src", "10.1.2.3").
    SetIPField("ip.dst", "192.168.1.1").
    SetIntField("port.dst", 443).
    SetStringField("protocol", "tcp")

matched, _ := filter.Execute(ctx)
```

### Tag-based Filtering

```go
schema := wirefilter.NewSchema().
    AddField("tags", wirefilter.TypeArray).
    AddField("environment", wirefilter.TypeString)

expression := `
    environment == "production" and
    tags === "critical"
`

filter, _ := wirefilter.Compile(expression, schema)

tags := wirefilter.ArrayValue{
    wirefilter.StringValue("critical"),
    wirefilter.StringValue("monitored"),
}

ctx := wirefilter.NewExecutionContext().
    SetField("tags", tags).
    SetStringField("environment", "production")

matched, _ := filter.Execute(ctx)
```

### Field-to-Field and Map Access

```go
schema := wirefilter.NewSchema().
    AddField("user.attributes", wirefilter.TypeMap).
    AddField("device.vars", wirefilter.TypeMap).
    AddField("user.login", wirefilter.TypeString).
    AddField("device.owner", wirefilter.TypeString)

// Compare map values from different fields
expression := `
    user.attributes["region"] == device.vars["region"] and
    user.login == device.owner
`

filter, _ := wirefilter.Compile(expression, schema)

ctx := wirefilter.NewExecutionContext().
    SetMapField("user.attributes", map[string]string{"region": "us-west"}).
    SetMapField("device.vars", map[string]string{"region": "us-west"}).
    SetStringField("user.login", "john").
    SetStringField("device.owner", "john")

matched, _ := filter.Execute(ctx) // true
```

### Wildcard Host Matching

```go
schema := wirefilter.NewSchema().
    AddField("http.host", wirefilter.TypeString)

// Case-insensitive wildcard matching
filter, _ := wirefilter.Compile(`http.host wildcard "*.example.com"`, schema)

ctx := wirefilter.NewExecutionContext().
    SetStringField("http.host", "API.EXAMPLE.COM")

matched, _ := filter.Execute(ctx) // true (case-insensitive)

// Case-sensitive matching
filterStrict, _ := wirefilter.Compile(`http.host strict wildcard "*.Example.com"`, schema)

ctx2 := wirefilter.NewExecutionContext().
    SetStringField("http.host", "api.Example.com")

matched2, _ := filterStrict.Execute(ctx2) // true
```

### XOR Logic for Mutual Exclusion

```go
schema := wirefilter.NewSchema().
    AddField("user.is_admin", wirefilter.TypeBool).
    AddField("user.is_guest", wirefilter.TypeBool)

// XOR: user must be either admin or guest, but not both
filter, _ := wirefilter.Compile(`user.is_admin xor user.is_guest`, schema)

ctx := wirefilter.NewExecutionContext().
    SetBoolField("user.is_admin", true).
    SetBoolField("user.is_guest", false)

matched, _ := filter.Execute(ctx) // true
```

## Performance

The filter engine is designed for high performance:

- Filters are compiled once and can be executed multiple times
- Schema validation happens at compile time, not runtime
- Efficient AST-based evaluation
- No runtime reflection

For optimal performance, compile filters once and reuse them across multiple executions.

## Error Handling

The library returns errors for:

- Malformed filter expressions
- Unknown field references (when schema is provided)
- Invalid regex patterns
- Type mismatches during evaluation

Always check returned errors:

```go
filter, err := wirefilter.Compile(expression, schema)
if err != nil {
    log.Printf("Compilation error: %v", err)
    return
}

result, err := filter.Execute(ctx)
if err != nil {
    log.Printf("Execution error: %v", err)
    return
}
```

## License

This package is part of the gokit library.
