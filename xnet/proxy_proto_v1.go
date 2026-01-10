package xnet

import (
	"bufio"
	"bytes"
	"net"
	"strconv"
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
	line, err := reader.ReadString('\n')
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

	if !bytes.HasPrefix([]byte(line), proxyV1PrefixBytes) {
		return nil, ErrProxyProtoInvalid
	}

	line = line[len(proxyV1Prefix):]

	parts := bytes.Fields([]byte(line))
	if len(parts) < 1 {
		return nil, ErrProxyProtoInvalid
	}

	proto := string(parts[0])

	if proto == "UNKNOWN" {
		return &ProxyHeader{}, nil
	}

	if len(parts) != 5 {
		return nil, ErrProxyProtoInvalid
	}

	if proto != "TCP4" && proto != "TCP6" {
		return nil, ErrProxyProtoInvalid
	}

	srcIP := net.ParseIP(string(parts[1]))
	if srcIP == nil {
		return nil, ErrProxyProtoInvalid
	}

	dstIP := net.ParseIP(string(parts[2]))
	if dstIP == nil {
		return nil, ErrProxyProtoInvalid
	}

	srcPort, err := strconv.Atoi(string(parts[3]))
	if err != nil || srcPort < 0 || srcPort > 65535 {
		return nil, ErrProxyProtoInvalid
	}

	dstPort, err := strconv.Atoi(string(parts[4]))
	if err != nil || dstPort < 0 || dstPort > 65535 {
		return nil, ErrProxyProtoInvalid
	}

	if proto == "TCP4" {
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
