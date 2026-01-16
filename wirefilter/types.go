package wirefilter

import (
	"fmt"
	"net"
	"regexp"
	"strings"
)

// Type represents the data type of a value in the filter system.
type Type uint8

const (
	TypeString Type = iota
	TypeInt
	TypeBool
	TypeIP
	TypeBytes
	TypeArray
)

// Value is the interface that all value types must implement.
type Value interface {
	Type() Type
	Equal(other Value) bool
	String() string
	IsTruthy() bool
}

// StringValue represents a string value.
type StringValue string

func (s StringValue) Type() Type     { return TypeString }
func (s StringValue) String() string { return string(s) }
func (s StringValue) IsTruthy() bool { return true }
func (s StringValue) Equal(v Value) bool {
	if v.Type() != TypeString {
		return false
	}
	return string(s) == string(v.(StringValue))
}

// IntValue represents an integer value.
type IntValue int64

func (i IntValue) Type() Type     { return TypeInt }
func (i IntValue) String() string { return fmt.Sprintf("%d", i) }
func (i IntValue) IsTruthy() bool { return true }
func (i IntValue) Equal(v Value) bool {
	if v.Type() != TypeInt {
		return false
	}
	return int64(i) == int64(v.(IntValue))
}

// BoolValue represents a boolean value.
type BoolValue bool

func (b BoolValue) Type() Type     { return TypeBool }
func (b BoolValue) String() string { return fmt.Sprintf("%t", b) }
func (b BoolValue) IsTruthy() bool { return bool(b) }
func (b BoolValue) Equal(v Value) bool {
	if v.Type() != TypeBool {
		return false
	}
	return bool(b) == bool(v.(BoolValue))
}

// IPValue represents an IP address value (IPv4 or IPv6).
type IPValue struct {
	IP net.IP
}

func (ip IPValue) Type() Type     { return TypeIP }
func (ip IPValue) String() string { return ip.IP.String() }
func (ip IPValue) IsTruthy() bool { return true }
func (ip IPValue) Equal(v Value) bool {
	if v.Type() != TypeIP {
		return false
	}
	return ip.IP.Equal(v.(IPValue).IP)
}

// BytesValue represents a byte array value.
type BytesValue []byte

func (b BytesValue) Type() Type     { return TypeBytes }
func (b BytesValue) String() string { return string(b) }
func (b BytesValue) IsTruthy() bool { return true }
func (b BytesValue) Equal(v Value) bool {
	if v.Type() != TypeBytes {
		return false
	}
	other := v.(BytesValue)
	if len(b) != len(other) {
		return false
	}
	for i := range b {
		if b[i] != other[i] {
			return false
		}
	}
	return true
}

// ArrayValue represents an array of values.
type ArrayValue []Value

func (a ArrayValue) Type() Type     { return TypeArray }
func (a ArrayValue) IsTruthy() bool { return true }
func (a ArrayValue) String() string {
	parts := make([]string, len(a))
	for i, v := range a {
		parts[i] = v.String()
	}
	return "[" + strings.Join(parts, ", ") + "]"
}
func (a ArrayValue) Equal(v Value) bool {
	if v.Type() != TypeArray {
		return false
	}
	other := v.(ArrayValue)
	if len(a) != len(other) {
		return false
	}
	for i := range a {
		if !a[i].Equal(other[i]) {
			return false
		}
	}
	return true
}

// Contains checks if the array contains the specified value.
func (a ArrayValue) Contains(v Value) bool {
	for _, item := range a {
		if item.Equal(v) {
			return true
		}
	}
	return false
}

// IPInCIDR checks if an IP address is within the specified CIDR range.
func IPInCIDR(ip net.IP, cidr string) (bool, error) {
	_, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return false, err
	}
	return ipNet.Contains(ip), nil
}

// IsIPv6 checks if an IP address is IPv6.
func IsIPv6(ip net.IP) bool {
	return ip.To4() == nil && ip.To16() != nil
}

// IsIPv4 checks if an IP address is IPv4.
func IsIPv4(ip net.IP) bool {
	return ip.To4() != nil
}

// MatchesRegex checks if a value matches the specified regular expression pattern.
func MatchesRegex(value string, pattern string) (bool, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return false, err
	}
	return re.MatchString(value), nil
}

// ContainsString checks if haystack contains needle as a substring.
func ContainsString(haystack, needle string) bool {
	return strings.Contains(haystack, needle)
}
