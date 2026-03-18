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
- Array index access: `tags[0]`
- Array unpack operations: `tags[*] == "value"` (ANY semantics)
- Raw strings: `r"..."` (no escape processing)
- Custom lists: `$list_name` for external list references
- Lookup tables: `$table_name[field]` for key-value lookups
- Negated membership: `not in`, `not contains`
- Built-in functions: `lower()`, `upper()`, `len()`, `starts_with()`, and more
- Field-to-field comparisons
- IP/CIDR matching for IPv4 and IPv6
- Regular expression matching
- Schema validation for field references and operator-type compatibility
- Expression complexity limits (max depth and node count)
- Binary serialization for pre-compiled filter storage and fast loading

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

### Raw Strings

Raw strings use the `r"..."` syntax and do not process escape sequences.
Useful for regex patterns and file paths:

```go
// Regular string (escape sequences processed)
path matches "^C:\\Users\\.*"               // backslashes need escaping

// Raw string (no escape processing)
path matches r"^C:\Users\.*"                // backslashes preserved as-is
email matches r"^\w+@\w+\.\w+$"             // cleaner regex patterns
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
- `"www.example.com" strict wildcard "*.Example.com"` - false

### Combining Conditions

```go
http.host == "example.com" and http.status == 200
http.host == "example.com" && http.status == 200   // && is alias for and
http.status == 404 or http.status == 500
http.status == 404 || http.status == 500           // || is alias for or
not (http.status >= 500)
! http.secure                                      // ! is alias for not
http.secure xor http.authenticated                 // XOR: exactly one true
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

- Any field that exists is truthy (including zero values and empty strings)
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

### Array Index Access

Access individual elements of an array by index (0-based):

```go
tags[0] == "first"                          // first element
tags[1] == "second"                         // second element
ports[0] > 1000                             // comparison on array element
```

Out-of-bounds or negative indices return no match (false).

### Array Unpack

Apply operations to all array elements with `[*]` syntax (ANY semantics):

```go
tags[*] == "admin"                          // true if ANY tag equals "admin"
tags[*] contains "test"                     // true if ANY tag contains "test"
tags[*] matches "^prod.*"                   // true if ANY tag matches pattern
ports[*] > 1000                             // true if ANY port > 1000
hosts[*] wildcard "*.example.com"           // true if ANY host matches
roles[*] in {"admin", "superuser"}          // true if ANY role is in the set
```

Example:

```go
tags := ["user", "admin", "guest"]

tags[*] == "admin"                          // true (admin matches)
tags[*] == "root"                           // false (no match)
tags[*] contains "min"                      // true (admin contains "min")
```

### Custom Lists

Reference external lists defined at runtime with `$list_name` syntax:

```go
role in $admin_roles         // check if role is in the admin_roles list
ip.src in $blocked_ips       // check if IP is in the blocked list
http.host in $allowed_hosts  // check if host is allowed
```

Lists are defined in the execution context (see API Reference below).

### Lookup Tables

Named key-value tables for dynamic lookups using `$table_name[field]` syntax:

```go
// Scalar lookup: returns a single value
$geo_table[ip.src] == "US"
$rate_limits[user.role] >= 100

// Array lookup: returns an array for use with in/contains
user.role in $allowed_roles[department]
ip.src in $blocked_nets[region]
ip.src not in $allowed_nets[zone]
```

Tables support different value types per key:

```go
// String values
ctx.SetTable("geo", map[string]string{"10.0.0.1": "US", "8.8.8.8": "DE"})

// Mixed value types
ctx.SetTableValues("limits", map[string]wirefilter.Value{
    "admin": wirefilter.IntValue(1000),
    "user":  wirefilter.IntValue(100),
})

// String arrays per key
ctx.SetTableList("roles", map[string][]string{
    "eng":   {"dev", "sre", "lead"},
    "sales": {"account", "manager"},
})

// IP/CIDR arrays per key
ctx.SetTableIPList("nets", map[string][]string{
    "office": {"10.0.0.0/8"},
    "vpn":    {"172.16.0.0/12"},
})
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

### Controlling Function Availability

You can control which functions are available in filter expressions
using allowlist or blocklist modes.

#### Blocklist Mode (Default)

All functions are allowed by default. Disable specific functions:

```go
schema := wirefilter.NewSchema().
    AddField("name", wirefilter.TypeString).
    DisableFunctions("lower", "upper", "concat")

// This will fail: lower is disabled
_, err := wirefilter.Compile(`lower(name) == "test"`, schema)
// Error: function not allowed: lower
```

#### Allowlist Mode

Only explicitly enabled functions are allowed:

```go
schema := wirefilter.NewSchema().
    AddField("name", wirefilter.TypeString).
    SetFunctionMode(wirefilter.FunctionModeAllowlist).
    EnableFunctions("lower", "upper", "len", "starts_with")

// This works: lower is enabled
_, err := wirefilter.Compile(`lower(name) == "test"`, schema)

// This fails: concat is not enabled
_, err = wirefilter.Compile(`concat(name, "!") == "test!"`, schema)
// Error: function not allowed: concat
```

Function names are case-insensitive.
Disabling "lower" also disables "LOWER" and "Lower".

### Type Validation

When a schema is provided, the compiler validates that operators are compatible
with field types. This catches errors at compile time rather than runtime:

```go
schema := wirefilter.NewSchema().
    AddField("status", wirefilter.TypeInt).
    AddField("name", wirefilter.TypeString).
    AddField("ip", wirefilter.TypeIP)

// Valid: > works on Int
_, err := wirefilter.Compile(`status > 400`, schema)

// Invalid: > does not work on String
_, err = wirefilter.Compile(`name > "test"`, schema)
// Error: operator > is not valid for field type String

// Invalid: contains does not work on IP
_, err = wirefilter.Compile(`ip contains "10"`, schema)
// Error: operator contains is not valid for field type IP
```

Type validation rules:

| Type | Valid Operators |
|------|----------------|
| String | `==`, `!=`, `contains`, `matches`, `in`, `wildcard`, `strict wildcard` |
| Int | `==`, `!=`, `<`, `>`, `<=`, `>=`, `in` |
| Bool | `==`, `!=` |
| IP | `==`, `!=`, `in` |
| Array | `==`, `!=`, `===`, `!==`, `contains`, `in` |
| Map | `==`, `!=` |
| Bytes | `==`, `!=`, `contains` |

Expressions using `[*]` (unpack), `["key"]` (index), or functions skip type
validation since the resulting element type is not known at compile time.

### Expression Complexity Limits

Prevent overly complex or deeply nested expressions that could cause excessive
resource consumption:

```go
schema := wirefilter.NewSchema().
    AddField("x", wirefilter.TypeInt).
    SetMaxDepth(32).   // max AST nesting depth
    SetMaxNodes(1000)  // max total AST nodes

// Deeply nested expressions are rejected
_, err := wirefilter.Compile(deeplyNestedExpr, schema)
// Error: expression exceeds maximum depth of 32

// Extremely large expressions are rejected
_, err = wirefilter.Compile(hugeExpr, schema)
// Error: expression exceeds maximum node count of 1000
```

Both limits default to zero (unlimited). Set them on the schema to enable.

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

#### Setting Array Fields

For array fields with string values:

```go
ctx := wirefilter.NewExecutionContext().
    SetArrayField("tags", []string{"admin", "user", "guest"})
```

For array fields with integer values:

```go
ctx := wirefilter.NewExecutionContext().
    SetIntArrayField("ports", []int64{80, 443, 8080})
```

#### Setting Custom Lists

Custom lists are referenced in expressions with `$list_name` syntax:

```go
ctx := wirefilter.NewExecutionContext().
    SetStringField("role", "admin").
    SetList("admin_roles", []string{"admin", "superuser", "root"})

// Expression: role in $admin_roles
```

For IP address lists:

```go
ctx := wirefilter.NewExecutionContext().
    SetIPField("ip.src", "192.168.1.100").
    SetIPList("blocked_ips", []string{"10.0.0.1", "192.168.1.100", "172.16.0.0/12"})

// Expression: ip.src in $blocked_ips
// Expression: ip.src not in $blocked_ips
```

### Expression Hash

Compiled filters expose a canonical hash for deduplication. The hash ignores
whitespace differences and operator aliases (`and` vs `&&`, `or` vs `||`,
`not` vs `!`, `xor` vs `^^`, `matches` vs `~`):

```go
f1, _ := wirefilter.Compile(`name == "test" and status >= 400`, schema)
f2, _ := wirefilter.Compile(`name   ==   "test"  &&  status  >=  400`, schema)

f1.Hash() == f2.Hash() // true - semantically identical
```

This is useful for deduplicating rules in large rule sets.

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
| `TypeMap` | Map of string keys to values | `{"key": "value"}` |

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
| `in` | Value in array, IP in CIDR, array ANY match | `port in {80, 443}` |
| `not in` | Negated membership | `ip not in {10.0.0.0/8}` |
| `contains` | Substring or array ALL match | `path contains "/api"` |
| `not contains` | Negated containment | `path not contains "/admin"` |
| `matches`, `~` | Regex match | `ua matches "^Mozilla"` |

### Wildcard Operators

| Operator | Description | Example |
|----------|-------------|---------|
| `wildcard` | Glob match (case-insensitive) | `host wildcard "*.ex.com"` |
| `strict wildcard` | Glob (case-sensitive) | `host strict wildcard "*.a"` |

Wildcard patterns support:

- `*` matches any sequence of characters (including empty)
- `?` matches any single character

### Arithmetic Operators

| Operator | Description | Example |
|----------|-------------|---------|
| `+` | Addition | `x + 1` |
| `-` | Subtraction | `x - 5` |
| `*` | Multiplication | `x * 2` |
| `/` | Division (nil on zero) | `x / 3` |
| `%` | Modulo (nil on zero) | `x % 2` |

Arithmetic works on Int and Float types. Mixed Int/Float produces Float results.
Standard precedence: `*`, `/`, `%` bind tighter than `+`, `-`.

### Array Operators

| Operator | Description | Example |
|----------|-------------|---------|
| `===` | All elements equal | `tags === "prod"` |
| `!==` | Any element not equal | `tags !== "test"` |

## Functions

Wirefilter provides built-in functions for transforming and inspecting values.

### String Functions

| Function | Description | Example |
|----------|-------------|---------|
| `lower(String)` | Convert to lowercase | `lower(http.host) == "example.com"` |
| `upper(String)` | Convert to uppercase | `upper(method) == "GET"` |
| `len(String)` | String length in bytes | `len(path) > 100` |
| `starts_with(String, String)` | Check prefix | `starts_with(path, "/api/")` |
| `ends_with(String, String)` | Check suffix | `ends_with(file, ".json")` |
| `substring(String, Int [, Int])` | Extract substring | `substring(s, 0, 4)` |
| `concat(String...)` | Concatenate strings | `concat(scheme, "://", host)` |
| `split(String, String)` | Split into array | `split(header, ",")[0]` |
| `url_decode(String)` | URL decode | `url_decode(query)` |
| `trim(String)` | Trim whitespace from both ends | `trim(name)` |
| `trim_left(String)` | Trim whitespace from left | `trim_left(name)` |
| `trim_right(String)` | Trim whitespace from right | `trim_right(name)` |
| `replace(String, String, String)` | Replace all occurrences | `replace(path, "/", "-")` |
| `regex_replace(String, String, String)` | Replace regex matches | `regex_replace(s, "[0-9]+", "X")` |
| `regex_extract(String, String)` | Extract first regex match | `regex_extract(path, "/v([0-9]+)/")` |
| `contains_word(String, String)` | Word boundary match | `contains_word(msg, "admin")` |

### Array Functions

| Function | Description | Example |
|----------|-------------|---------|
| `len(Array)` | Array element count | `len(tags) > 0` |
| `count(Array)` | Count truthy elements | `count(tags) >= 3` |
| `any(expression)` | Any element matches | `any(tags[*] == "admin")` |
| `all(expression)` | All elements match | `all(ports[*] > 0)` |
| `has_value(Array, Value)` | Array contains value | `has_value(tags, "admin")` |
| `join(Array, String)` | Join array elements | `join(tags, ",")` |
| `contains_any(Array, Array)` | Any element from second in first | `contains_any(tags, required)` |
| `contains_all(Array, Array)` | All elements from second in first | `contains_all(tags, required)` |
| `intersection(Array, Array)` | Elements in both arrays | `intersection(a, b)` |
| `union(Array, Array)` | Combined unique elements | `union(a, b)` |
| `difference(Array, Array)` | Elements in first not in second | `difference(a, b)` |

### Map Functions

| Function | Description | Example |
|----------|-------------|---------|
| `len(Map)` | Map key count | `len(headers) > 0` |
| `has_key(Map, String)` | Check key exists | `has_key(headers, "Auth")` |

### IP Functions

| Function | Description | Example |
|----------|-------------|---------|
| `cidr(IP, Int)` | Apply CIDR mask for IPv4 | `cidr(ip, 24)` |
| `cidr6(IP, Int)` | Apply CIDR mask for IPv6 | `cidr6(ip, 64)` |
| `is_ipv4(IP)` | Check if IPv4 | `is_ipv4(ip)` |
| `is_ipv6(IP)` | Check if IPv6 | `is_ipv6(ip)` |
| `is_loopback(IP)` | Check if loopback | `is_loopback(ip)` |

### Math Functions

| Function | Description | Example |
|----------|-------------|---------|
| `abs(Int\|Float)` | Absolute value | `abs(x) > 5` |
| `ceil(Float)` | Ceiling (round up) | `ceil(x) == 4` |
| `floor(Float)` | Floor (round down) | `floor(x) == 3` |
| `round(Float)` | Round to nearest | `round(x) == 4` |

### Utility Functions

| Function | Description | Example |
|----------|-------------|---------|
| `coalesce(Value...)` | First non-nil value | `coalesce(a, b, "default")` |

### Function Examples

```go
// Case-insensitive comparison
lower(http.host) == "example.com"

// Check path prefix
starts_with(http.path, "/api/v1/")

// Check file extension
ends_with(request.file, ".pdf")

// URL decode and search
url_decode(http.query) contains "admin"

// Check if any tag matches
any(tags[*] contains "prod")

// Check if all ports are valid
all(ports[*] > 0 and ports[*] < 65536)

// Build URL from parts
concat(scheme, "://", host, path) == "https://api.example.com/users"

// Parse CSV header
split(header, ",")[0] == "value1"

// Check map key exists
has_key(request.headers, "X-Auth-Token")

// Apply /24 CIDR mask to IPv4
cidr(ip.src, 24) == "192.168.1.0"

// Apply /64 CIDR mask for IPv6 networks
cidr6(ip.src, 64) == "2001:db8::"
```

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
expr := `http.host strict wildcard "*.Example.com"`
filterStrict, _ := wirefilter.Compile(expr, schema)

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

### Array Index and Unpack Operations

```go
schema := wirefilter.NewSchema().
    AddField("tags", wirefilter.TypeArray).
    AddField("ports", wirefilter.TypeArray)

// Access specific array element
filter1, _ := wirefilter.Compile(`tags[0] == "primary"`, schema)

// Check if ANY element matches (unpack)
filter2, _ := wirefilter.Compile(`tags[*] contains "admin"`, schema)

// Check if ANY port is in a dangerous range
filter3, _ := wirefilter.Compile(`ports[*] > 1000 and ports[*] < 2000`, schema)

ctx := wirefilter.NewExecutionContext().
    SetArrayField("tags", []string{"primary", "admin-role", "active"}).
    SetIntArrayField("ports", []int64{80, 443, 1500})

matched1, _ := filter1.Execute(ctx) // true (tags[0] == "primary")
matched2, _ := filter2.Execute(ctx) // true (admin-role contains "admin")
matched3, _ := filter3.Execute(ctx) // true (1500 is between 1000 and 2000)
```

### Custom Lists for Dynamic Filtering

```go
schema := wirefilter.NewSchema().
    AddField("user.role", wirefilter.TypeString).
    AddField("ip.src", wirefilter.TypeIP)

// Filter using custom lists
expression := `user.role in $privileged_roles and not (ip.src in $blocked_ips)`

filter, _ := wirefilter.Compile(expression, schema)

// Lists can be updated at runtime without recompiling the filter
ctx := wirefilter.NewExecutionContext().
    SetStringField("user.role", "admin").
    SetIPField("ip.src", "10.0.0.50").
    SetList("privileged_roles", []string{"admin", "superuser", "operator"}).
    SetIPList("blocked_ips", []string{"192.168.1.1", "10.0.0.100"})

matched, _ := filter.Execute(ctx) // true (admin is privileged, IP not blocked)
```

### Raw Strings for Complex Patterns

```go
schema := wirefilter.NewSchema().
    AddField("file.path", wirefilter.TypeString).
    AddField("log.message", wirefilter.TypeString)

// Raw strings make regex patterns cleaner
expr1 := `file.path matches r"^C:\Windows\System32\.*\.dll$"`
filter1, _ := wirefilter.Compile(expr1, schema)
expr2 := `log.message matches r"error code: \d{4}"`
filter2, _ := wirefilter.Compile(expr2, schema)

ctx := wirefilter.NewExecutionContext().
    SetStringField("file.path", `C:\Windows\System32\kernel32.dll`).
    SetStringField("log.message", "error code: 1234")

matched1, _ := filter1.Execute(ctx) // true
matched2, _ := filter2.Execute(ctx) // true
```

## Binary Serialization

Compiled filters can be serialized to bytes and restored later without re-parsing.
This is useful for pre-compiling large rule sets and loading them from external
storage (databases, caches, files) at runtime.

### Saving a Compiled Filter

```go
filter, err := wirefilter.Compile(`ip.src not in $blocked and status < 500`, schema)
if err != nil {
    log.Fatal(err)
}

data, err := filter.MarshalBinary()
if err != nil {
    log.Fatal(err)
}

// Store data in Redis, database, file, etc.
```

### Loading a Compiled Filter

```go
// Load data from external storage
filter := &wirefilter.Filter{}
if err := filter.UnmarshalBinary(data); err != nil {
    log.Fatal(err)
}

// Use the filter directly - no re-parsing needed
result, err := filter.Execute(ctx)
```

The `Filter` type implements the standard `encoding.BinaryMarshaler` and
`encoding.BinaryUnmarshaler` interfaces.

Regex and CIDR caches are rebuilt lazily on first use after deserialization.
Schema validation is not re-applied; the filter is assumed to have been
validated at compile time.

## Performance

The filter engine is designed for high performance:

- Filters are compiled once and can be executed multiple times
- Schema validation happens at compile time, not runtime
- Efficient AST-based evaluation
- No runtime reflection
- Binary deserialization is ~2x faster than compiling from string

For optimal performance, compile filters once and reuse them across multiple
executions. For large rule sets, pre-compile and store the binary representation
for fast loading.

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
