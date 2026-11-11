package xflags

import (
	"fmt"
	"strconv"
	"strings"
)

// ByteSize is a byte count that parses an optional IEC unit suffix when used as
// an option destination. Supported suffixes are B, KiB, MiB, GiB, and TiB, each
// a power of 1024; a bare number is interpreted as bytes. It implements
// encoding.TextUnmarshaler and encoding.TextMarshaler, so it works as an xflags
// option type and round-trips through text.
type ByteSize uint64

// IEC byte-size units, as powers of 1024.
const (
	Byte     ByteSize = 1
	Kibibyte          = Byte << 10
	Mebibyte          = Kibibyte << 10
	Gibibyte          = Mebibyte << 10
	Tebibyte          = Gibibyte << 10
)

// byteSizeUnits lists suffixes from longest to shortest so matching is
// unambiguous: "KiB" is tried before the single-character "B".
var byteSizeUnits = []struct {
	suffix string
	mul    uint64
}{
	{"TiB", uint64(Tebibyte)},
	{"GiB", uint64(Gibibyte)},
	{"MiB", uint64(Mebibyte)},
	{"KiB", uint64(Kibibyte)},
	{"B", uint64(Byte)},
}

// ParseByteSize parses a byte size with an optional IEC unit suffix: B, KiB,
// MiB, GiB, or TiB. A bare number is interpreted as bytes. It reports an error
// for an empty, non-numeric, or overflowing input.
func ParseByteSize(s string) (ByteSize, error) {
	trimmed := strings.TrimSpace(s)
	if trimmed == "" {
		return 0, fmt.Errorf("empty size")
	}

	num, mul := trimmed, uint64(Byte)
	for _, u := range byteSizeUnits {
		if rest, ok := strings.CutSuffix(trimmed, u.suffix); ok {
			num, mul = strings.TrimSpace(rest), u.mul
			break
		}
	}

	value, err := strconv.ParseUint(num, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid size %q", s)
	}
	if mul != 1 && value > uint64(^ByteSize(0))/mul {
		return 0, fmt.Errorf("size overflow %q", s)
	}
	return ByteSize(value * mul), nil
}

// UnmarshalText parses a byte size, implementing encoding.TextUnmarshaler.
func (b *ByteSize) UnmarshalText(text []byte) error {
	v, err := ParseByteSize(string(text))
	if err != nil {
		return err
	}
	*b = v
	return nil
}

// MarshalText renders the byte size using the largest IEC unit that divides it
// evenly, implementing encoding.TextMarshaler. A value of zero renders as "0B".
func (b ByteSize) MarshalText() ([]byte, error) {
	return []byte(b.String()), nil
}

// String renders the byte size using the largest IEC unit that divides it
// evenly. A value of zero renders as "0B".
func (b ByteSize) String() string {
	if b == 0 {
		return "0B"
	}
	for _, u := range byteSizeUnits {
		if uint64(b)%u.mul == 0 {
			return fmt.Sprintf("%d%s", uint64(b)/u.mul, u.suffix)
		}
	}
	return fmt.Sprintf("%dB", uint64(b))
}
