package xnet

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"io"
	"net"
)

const (
	proxyV2SignatureLen = 12
	proxyV2HeaderLen    = 16
)

var proxyV2Signature = []byte{0x0D, 0x0A, 0x0D, 0x0A, 0x00, 0x0D, 0x0A, 0x51, 0x55, 0x49, 0x54, 0x0A}

const (
	proxyV2CmdLocal = 0x20
	proxyV2CmdProxy = 0x21
)

const (
	proxyV2FamilyUnspec     = 0x00
	proxyV2FamilyTCPv4      = 0x11
	proxyV2FamilyUDPv4      = 0x12
	proxyV2FamilyTCPv6      = 0x21
	proxyV2FamilyUDPv6      = 0x22
	proxyV2FamilyUnixStream = 0x31
	proxyV2FamilyUnixDgram  = 0x32
)

const (
	proxyV2IPv4AddrLen = 12
	proxyV2IPv6AddrLen = 36
)

func isProxyV2(data []byte) bool {
	if len(data) < proxyV2SignatureLen {
		return false
	}

	return bytes.Equal(data[:proxyV2SignatureLen], proxyV2Signature)
}

func parseProxyV2(reader *bufio.Reader) (*ProxyHeader, error) {
	headerBuf := make([]byte, proxyV2HeaderLen)

	if _, err := io.ReadFull(reader, headerBuf); err != nil {
		return nil, ErrProxyProtoInvalid
	}

	if !bytes.Equal(headerBuf[:proxyV2SignatureLen], proxyV2Signature) {
		return nil, ErrProxyProtoInvalid
	}

	verCmd := headerBuf[12]
	family := headerBuf[13]
	addrLen := binary.BigEndian.Uint16(headerBuf[14:16])

	if verCmd != proxyV2CmdLocal && verCmd != proxyV2CmdProxy {
		return nil, ErrProxyProtoInvalid
	}

	if addrLen > 0 {
		addrBuf := make([]byte, addrLen)
		if _, err := io.ReadFull(reader, addrBuf); err != nil {
			return nil, ErrProxyProtoInvalid
		}

		if verCmd == proxyV2CmdLocal {
			return &ProxyHeader{}, nil
		}

		return parseProxyV2Addresses(family, addrBuf)
	}

	return &ProxyHeader{}, nil
}

func parseProxyV2Addresses(family byte, data []byte) (*ProxyHeader, error) {
	switch family {
	case proxyV2FamilyUnspec:
		return &ProxyHeader{}, nil

	case proxyV2FamilyTCPv4, proxyV2FamilyUDPv4:
		if len(data) < proxyV2IPv4AddrLen {
			return nil, ErrProxyProtoInvalid
		}

		srcIP := net.IP(data[0:4])
		dstIP := net.IP(data[4:8])
		srcPort := int(binary.BigEndian.Uint16(data[8:10]))
		dstPort := int(binary.BigEndian.Uint16(data[10:12]))

		return &ProxyHeader{
			SourceAddr: &net.TCPAddr{IP: srcIP, Port: srcPort},
			DestAddr:   &net.TCPAddr{IP: dstIP, Port: dstPort},
		}, nil

	case proxyV2FamilyTCPv6, proxyV2FamilyUDPv6:
		if len(data) < proxyV2IPv6AddrLen {
			return nil, ErrProxyProtoInvalid
		}

		srcIP := net.IP(data[0:16])
		dstIP := net.IP(data[16:32])
		srcPort := int(binary.BigEndian.Uint16(data[32:34]))
		dstPort := int(binary.BigEndian.Uint16(data[34:36]))

		return &ProxyHeader{
			SourceAddr: &net.TCPAddr{IP: srcIP, Port: srcPort},
			DestAddr:   &net.TCPAddr{IP: dstIP, Port: dstPort},
		}, nil

	case proxyV2FamilyUnixStream, proxyV2FamilyUnixDgram:
		return &ProxyHeader{}, nil

	default:
		return nil, ErrProxyProtoInvalid
	}
}
