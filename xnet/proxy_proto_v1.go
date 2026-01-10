package xnet

import (
	"bufio"
	"bytes"
	"net"
	"strconv"
	"unsafe"
)

const (
	proxyV1Prefix    = "PROXY "
	proxyV1MaxLength = 107
)

var proxyV1PrefixBytes = []byte(proxyV1Prefix)

func isProxyV1(data []byte) bool {
	return bytes.HasPrefix(data, proxyV1PrefixBytes)
}

func parseProxyV1(reader *bufio.Reader) (*ProxyHeader, error) {
	line, err := reader.ReadSlice('\n')
	if err != nil {
		return nil, ErrProxyProtoInvalid
	}

	if len(line) > proxyV1MaxLength {
		return nil, ErrProxyProtoInvalid
	}

	if len(line) < 2 || line[len(line)-2] != '\r' {
		return nil, ErrProxyProtoInvalid
	}

	line = line[:len(line)-2]

	if !bytes.HasPrefix(line, proxyV1PrefixBytes) {
		return nil, ErrProxyProtoInvalid
	}

	line = line[len(proxyV1Prefix):]

	parts := splitProxyV1Fields(line)
	if len(parts) < 1 {
		return nil, ErrProxyProtoInvalid
	}

	if bytes.Equal(parts[0], []byte("UNKNOWN")) {
		return &ProxyHeader{}, nil
	}

	if len(parts) != 5 {
		return nil, ErrProxyProtoInvalid
	}

	isTCP4 := bytes.Equal(parts[0], []byte("TCP4"))
	isTCP6 := bytes.Equal(parts[0], []byte("TCP6"))

	if !isTCP4 && !isTCP6 {
		return nil, ErrProxyProtoInvalid
	}

	srcIP := net.ParseIP(unsafeString(parts[1]))
	if srcIP == nil {
		return nil, ErrProxyProtoInvalid
	}

	dstIP := net.ParseIP(unsafeString(parts[2]))
	if dstIP == nil {
		return nil, ErrProxyProtoInvalid
	}

	srcPort, err := strconv.Atoi(unsafeString(parts[3]))
	if err != nil || srcPort < 0 || srcPort > 65535 {
		return nil, ErrProxyProtoInvalid
	}

	dstPort, err := strconv.Atoi(unsafeString(parts[4]))
	if err != nil || dstPort < 0 || dstPort > 65535 {
		return nil, ErrProxyProtoInvalid
	}

	if isTCP4 {
		if srcIP.To4() == nil || dstIP.To4() == nil {
			return nil, ErrProxyProtoInvalid
		}
	} else {
		if srcIP.To4() != nil || dstIP.To4() != nil {
			return nil, ErrProxyProtoInvalid
		}
	}

	return &ProxyHeader{
		SourceAddr: &net.TCPAddr{IP: srcIP, Port: srcPort},
		DestAddr:   &net.TCPAddr{IP: dstIP, Port: dstPort},
	}, nil
}

// splitProxyV1Fields splits bytes by spaces without allocating slices for each field.
// Returns up to 5 fields (the maximum needed for PROXY v1 protocol).
func splitProxyV1Fields(data []byte) [][]byte {
	var fields [5][]byte
	fieldCount := 0
	start := 0

	for i := 0; i < len(data) && fieldCount < 5; i++ {
		if data[i] == ' ' {
			if i > start {
				fields[fieldCount] = data[start:i]
				fieldCount++
			}
			start = i + 1
		}
	}

	if start < len(data) && fieldCount < 5 {
		fields[fieldCount] = data[start:]
		fieldCount++
	}

	return fields[:fieldCount]
}

// unsafeString converts bytes to string without allocation.
// The string is only valid as long as the bytes are not modified.
func unsafeString(b []byte) string {
	if len(b) == 0 {
		return ""
	}
	return unsafe.String(&b[0], len(b))
}
