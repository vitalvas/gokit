# xnet

A Go networking utility library for working with IP addresses and CIDR blocks.

## Features

- **IP Address Stripping**: Apply network masks to IP addresses
- **CIDR Containment**: Check if an IP is within CIDR blocks
- **CIDR Merging**: Combine adjacent and overlapping networks into minimal set
- **CIDR Splitting**: Divide networks into smaller subnets
- **Fast CIDR Matching**: O(log n) IP lookups using radix tree
- **IPv4 and IPv6**: Full support for both IP versions
- **Type Safety**: Compile-time type checking
- **Zero Dependencies**: Only uses Go standard library

## Installation

```bash
go get github.com/vitalvas/gokit/xnet
```

## Quick Start

```go
package main

import (
    "fmt"
    "net"
    "github.com/vitalvas/gokit/xnet"
)

func main() {
    // Check if IP is in CIDR blocks
    cidrs := []string{"10.0.0.0/8", "192.168.0.0/16"}
    ip := net.ParseIP("10.1.2.3")

    if xnet.CIDRContainsString(cidrs, ip) {
        fmt.Println("IP is in allowed range")
    }

    // Merge overlapping networks
    merged, _ := xnet.CIDRMergeString([]string{
        "10.0.0.0/24",
        "10.0.1.0/24",
        "10.0.2.0/24",
    })
    fmt.Printf("Merged: %v\n", merged)
}
```

## IP Address Stripping

Apply network masks to IP addresses to get network addresses.

### GetStripedAddress

Masks an IP address with specified IPv4 or IPv6 prefix lengths.

```go
ip := net.ParseIP("192.168.1.100")
network, err := xnet.GetStripedAddress(ip, 24, 0)
if err != nil {
    panic(err)
}
fmt.Println(network) // 192.168.1.0
```

**IPv6 Example:**
```go
ip := net.ParseIP("2001:db8:85a3::8a2e:370:7334")
network, err := xnet.GetStripedAddress(ip, 0, 64)
if err != nil {
    panic(err)
}
fmt.Println(network) // 2001:db8:85a3::
```

**Parameters:**
- `addr`: IP address to mask
- `ipv4`: IPv4 prefix length (0-32, 0 to skip)
- `ipv6`: IPv6 prefix length (0-128, 0 to skip)

**Returns:**
- Network address after applying mask
- Error if parameters are invalid

**Errors:**
- `ErrInvalidIPAddress`: nil IP address
- `ErrInvalidIPv4MaskSize`: IPv4 mask not in range 0-32
- `ErrInvalidIPv6MaskSize`: IPv6 mask not in range 0-128
- `ErrInvalidIPMaskSize`: Both masks are zero
- `ErrNonStandardIP`: IP version doesn't match specified mask

## CIDR Containment

Check if an IP address is contained within CIDR blocks.

### CIDRContains

Check if IP is in any of the provided `net.IPNet` networks.

```go
_, net1, _ := net.ParseCIDR("10.0.0.0/8")
_, net2, _ := net.ParseCIDR("172.16.0.0/12")
networks := []net.IPNet{*net1, *net2}

ip := net.ParseIP("10.1.2.3")
if xnet.CIDRContains(networks, ip) {
    fmt.Println("IP is in allowed range")
}
```

### CIDRContainsString

Check if IP is in any of the provided CIDR strings.

```go
cidrs := []string{
    "10.0.0.0/8",
    "172.16.0.0/12",
    "192.168.0.0/16",
}

ip := net.ParseIP("192.168.1.1")
if xnet.CIDRContainsString(cidrs, ip) {
    fmt.Println("IP is in private range")
}
```

**Note:** Invalid CIDR strings are silently skipped.

## CIDR Merging

Merge adjacent and overlapping CIDR blocks into the smallest possible list.

### CIDRMerge

Merge `net.IPNet` networks.

```go
_, net1, _ := net.ParseCIDR("10.0.0.0/24")
_, net2, _ := net.ParseCIDR("10.0.1.0/24")
_, net3, _ := net.ParseCIDR("10.0.2.0/25")
_, net4, _ := net.ParseCIDR("10.0.2.128/25")

networks := []net.IPNet{*net1, *net2, *net3, *net4}
merged := xnet.CIDRMerge(networks)

for _, n := range merged {
    fmt.Println(n.String())
}
// Output:
// 10.0.0.0/23
// 10.0.2.0/24
```

### CIDRMergeString

Merge CIDR strings.

```go
cidrs := []string{
    "192.168.0.0/24",
    "192.168.1.0/24",
    "192.168.2.0/24",
    "192.168.3.0/24",
}

merged, err := xnet.CIDRMergeString(cidrs)
if err != nil {
    panic(err)
}

fmt.Println(merged) // [192.168.0.0/22]
```

**Features:**
- Combines adjacent networks with same prefix length
- Removes networks contained within larger networks
- Aggregates networks into larger blocks when possible
- Processes IPv4 and IPv6 separately
- Returns optimized minimal set

**Errors:**
- `ErrInvalidCIDR`: Invalid CIDR notation

## CIDR Splitting

Split a network into smaller subnets.

### CIDRSplit

Split a `net.IPNet` into subnets of specified prefix length.

```go
_, network, _ := net.ParseCIDR("10.0.0.0/24")
subnets, err := xnet.CIDRSplit(*network, 26)
if err != nil {
    panic(err)
}

for _, subnet := range subnets {
    fmt.Println(subnet.String())
}
// Output:
// 10.0.0.0/26
// 10.0.0.64/26
// 10.0.0.128/26
// 10.0.0.192/26
```

### CIDRSplitString

Split a CIDR string into subnet strings.

```go
subnets, err := xnet.CIDRSplitString("192.168.0.0/24", 27)
if err != nil {
    panic(err)
}

fmt.Println(subnets)
// [192.168.0.0/27 192.168.0.32/27 192.168.0.64/27 192.168.0.96/27
//  192.168.0.128/27 192.168.0.160/27 192.168.0.192/27 192.168.0.224/27]
```

**IPv6 Example:**
```go
subnets, err := xnet.CIDRSplitString("2001:db8::/32", 34)
if err != nil {
    panic(err)
}
// Splits into 4 /34 subnets
```

**Parameters:**
- `network`: Network to split
- `newPrefix`: New prefix length (must be more specific than original)

**Returns:**
- List of subnets
- Error if new prefix is invalid

**Errors:**
- `ErrInvalidCIDR`: Invalid CIDR notation (string version)
- `ErrInvalidPrefixSize`: New prefix is not more specific than original

**Validation:**
- For IPv4: new prefix must be between current prefix and 32
- For IPv6: new prefix must be between current prefix and 128
- If new prefix equals current, returns original network unchanged

## Fast CIDR Matching

High-performance IP matching using a radix tree for O(log n) lookups.

### When to Use CIDRMatcher

Use `CIDRMatcher` instead of `CIDRContains` when:
- Checking many IPs against the same CIDR list
- Performance is critical
- CIDR list is large (100+ networks)

**Performance Comparison:**
- `CIDRContains`: O(n) - linear scan through all networks
- `CIDRMatcher`: O(log n) - radix tree lookup

### Creating a Matcher

**From net.IPNet:**
```go
_, net1, _ := net.ParseCIDR("10.0.0.0/8")
_, net2, _ := net.ParseCIDR("172.16.0.0/12")
_, net3, _ := net.ParseCIDR("192.168.0.0/16")

networks := []net.IPNet{*net1, *net2, *net3}
matcher := xnet.NewCIDRMatcher(networks)
```

**From CIDR strings:**
```go
cidrs := []string{
    "10.0.0.0/8",
    "172.16.0.0/12",
    "192.168.0.0/16",
}

matcher, err := xnet.NewCIDRMatcherFromStrings(cidrs)
if err != nil {
    panic(err)
}
```

### Checking IPs

```go
matcher, _ := xnet.NewCIDRMatcherFromStrings([]string{
    "10.0.0.0/8",
    "192.168.0.0/16",
})

// Check multiple IPs efficiently
ips := []string{"10.1.2.3", "8.8.8.8", "192.168.1.1"}
for _, ipStr := range ips {
    ip := net.ParseIP(ipStr)
    if matcher.Contains(ip) {
        fmt.Printf("%s is allowed\n", ipStr)
    } else {
        fmt.Printf("%s is blocked\n", ipStr)
    }
}
// Output:
// 10.1.2.3 is allowed
// 8.8.8.8 is blocked
// 192.168.1.1 is allowed
```

### Adding Networks Dynamically

```go
matcher := &xnet.CIDRMatcher{}

// Add networks one at a time
_, net1, _ := net.ParseCIDR("10.0.0.0/24")
matcher.Add(*net1)

_, net2, _ := net.ParseCIDR("10.0.1.0/24")
matcher.Add(*net2)

ip := net.ParseIP("10.0.0.50")
fmt.Println(matcher.Contains(ip)) // true
```

### Implementation Details

- Uses radix tree (trie) data structure
- Separate trees for IPv4 and IPv6
- Build time: O(n * bits) where n is number of networks
- Lookup time: O(bits) which is O(1) for fixed-size IPs
- Memory efficient: shared prefixes stored once

## Use Cases

### IP Access Control

```go
// Allow list for API access
allowedNets := []string{
    "10.0.0.0/8",      // Internal network
    "203.0.113.0/24",  // Partner network
}

matcher, _ := xnet.NewCIDRMatcherFromStrings(allowedNets)

func isAllowed(ip net.IP) bool {
    return matcher.Contains(ip)
}
```

### Network Optimization

```go
// Optimize firewall rules by merging overlapping ranges
rules := []string{
    "10.0.0.0/24",
    "10.0.1.0/24",
    "10.0.2.0/24",
    "10.0.3.0/24",
}

optimized, _ := xnet.CIDRMergeString(rules)
// Result: ["10.0.0.0/22"]
// Reduced from 4 rules to 1
```

### Subnet Planning

```go
// Divide /24 network into /28 subnets for different departments
subnets, _ := xnet.CIDRSplitString("192.168.1.0/24", 28)

departments := []string{"Engineering", "Sales", "Marketing", "HR"}
for i, dept := range departments {
    fmt.Printf("%s: %s\n", dept, subnets[i])
}
// Output:
// Engineering: 192.168.1.0/28
// Sales: 192.168.1.16/28
// Marketing: 192.168.1.32/28
// HR: 192.168.1.48/28
```

### IP Geolocation

```go
// Check if IP is in specific country ranges
countryRanges := []string{
    "1.0.0.0/24",
    "1.0.1.0/24",
    // ... more ranges
}

matcher, _ := xnet.NewCIDRMatcherFromStrings(countryRanges)

func isFromCountry(ip net.IP) bool {
    return matcher.Contains(ip)
}
```

## Error Handling

All functions return descriptive errors:

```go
// Invalid IP address
_, err := xnet.GetStripedAddress(nil, 24, 0)
// err == xnet.ErrInvalidIPAddress

// Invalid mask size
_, err = xnet.GetStripedAddress(net.ParseIP("192.168.1.1"), 33, 0)
// err == xnet.ErrInvalidIPv4MaskSize

// Invalid CIDR
_, err = xnet.CIDRMergeString([]string{"invalid"})
// err == xnet.ErrInvalidCIDR

// Invalid prefix size
_, err = xnet.CIDRSplitString("10.0.0.0/24", 16)
// err == xnet.ErrInvalidPrefixSize
```

### Available Errors

| Error | Description |
|-------|-------------|
| `ErrInvalidIPAddress` | IP address is nil |
| `ErrInvalidIPv4MaskSize` | IPv4 mask not in range 0-32 |
| `ErrInvalidIPv6MaskSize` | IPv6 mask not in range 0-128 |
| `ErrInvalidIPMaskSize` | Both IPv4 and IPv6 masks are zero |
| `ErrNonStandardIP` | IP version doesn't match specified mask |
| `ErrInvalidCIDR` | Invalid CIDR notation |
| `ErrInvalidPrefixSize` | New prefix not more specific than original |

## Performance Considerations

### CIDR Containment

For checking a few IPs:
```go
// Simple approach - good for < 10 checks
xnet.CIDRContainsString(cidrs, ip)
```

For checking many IPs:
```go
// Faster approach - build once, query many times
matcher, _ := xnet.NewCIDRMatcherFromStrings(cidrs)
for _, ip := range manyIPs {
    matcher.Contains(ip)
}
```

### CIDR Merging

Merging reduces the number of rules to maintain:
```go
// Before: 1000 firewall rules
// After: 50 merged rules (typical optimization)
merged := xnet.CIDRMerge(manyNetworks)
```

### Memory Usage

- `CIDRMatcher` uses O(n * bits) memory for n networks
- More memory than simple slice, but much faster lookups
- Trade memory for speed when checking many IPs

## License

This project is part of the [gokit](https://github.com/vitalvas/gokit) library.
