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

// proxyV2TLVHeaderLen is the size of a TLV type (1) plus length (2) prefix.
const proxyV2TLVHeaderLen = 3

func isProxyV2(data []byte) bool {
	if len(data) < proxyV2SignatureLen {
		return false
	}

	return bytes.Equal(data[:proxyV2SignatureLen], proxyV2Signature)
}

func parseProxyV2(reader *bufio.Reader) (*ProxyHeader, error) {
	var headerBuf [proxyV2HeaderLen]byte

	if _, err := io.ReadFull(reader, headerBuf[:]); err != nil {
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

	if addrLen == 0 {
		return &ProxyHeader{}, nil
	}

	// addrLen is a uint16, so the allocation is bounded to at most 64 KiB.
	block := make([]byte, addrLen)
	if _, err := io.ReadFull(reader, block); err != nil {
		return nil, ErrProxyProtoInvalid
	}

	if verCmd == proxyV2CmdLocal {
		return &ProxyHeader{}, nil
	}

	return parseProxyV2Block(family, block)
}

// parseProxyV2Block parses the address+TLV block that follows the 16-byte PROXY
// v2 header for a PROXY command.
func parseProxyV2Block(family byte, block []byte) (*ProxyHeader, error) {
	addrSize, ok := proxyV2AddrSize(family)
	if !ok {
		// Unspec, Unix sockets and unknown families carry no addresses that
		// this package decodes; the block is ignored.
		return &ProxyHeader{}, nil
	}

	if len(block) < addrSize {
		return nil, ErrProxyProtoInvalid
	}

	header := parseProxyV2Addresses(family, block[:addrSize])

	tlvs, err := parseProxyV2TLVs(block[addrSize:])
	if err != nil {
		return nil, err
	}

	header.TLVs = tlvs

	return header, nil
}

// proxyV2AddrSize returns the fixed address block size for an address family
// that carries IP addresses, and whether the family is such a family.
func proxyV2AddrSize(family byte) (int, bool) {
	switch family {
	case proxyV2FamilyTCPv4, proxyV2FamilyUDPv4:
		return proxyV2IPv4AddrLen, true
	case proxyV2FamilyTCPv6, proxyV2FamilyUDPv6:
		return proxyV2IPv6AddrLen, true
	default:
		return 0, false
	}
}

// parseProxyV2TLVs parses a sequence of Type-Length-Value vectors. It returns
// nil when data is empty.
func parseProxyV2TLVs(data []byte) ([]ProxyTLV, error) {
	if len(data) == 0 {
		return nil, nil
	}

	var tlvs []ProxyTLV

	for len(data) > 0 {
		if len(data) < proxyV2TLVHeaderLen {
			return nil, ErrProxyProtoInvalid
		}

		typ := data[0]
		length := int(binary.BigEndian.Uint16(data[1:3]))

		end := proxyV2TLVHeaderLen + length
		if end > len(data) {
			return nil, ErrProxyProtoInvalid
		}

		value := make([]byte, length)
		copy(value, data[proxyV2TLVHeaderLen:end])

		tlvs = append(tlvs, ProxyTLV{Type: typ, Value: value})
		data = data[end:]
	}

	return tlvs, nil
}

// parseProxyV2Addresses decodes the source and destination addresses from an
// address block sized for the given family. The caller is responsible for
// passing an IPv4 or IPv6 family with a correctly sized block.
func parseProxyV2Addresses(family byte, data []byte) *ProxyHeader {
	if family == proxyV2FamilyTCPv4 || family == proxyV2FamilyUDPv4 {
		srcIP := net.IP(data[0:4])
		dstIP := net.IP(data[4:8])
		srcPort := int(binary.BigEndian.Uint16(data[8:10]))
		dstPort := int(binary.BigEndian.Uint16(data[10:12]))

		return buildAddrHeader(family, srcIP, dstIP, srcPort, dstPort)
	}

	srcIP := net.IP(data[0:16])
	dstIP := net.IP(data[16:32])
	srcPort := int(binary.BigEndian.Uint16(data[32:34]))
	dstPort := int(binary.BigEndian.Uint16(data[34:36]))

	return buildAddrHeader(family, srcIP, dstIP, srcPort, dstPort)
}

// parseProxyV2Datagram parses a PROXY protocol v2 header from the start of a
// single datagram. Unlike parseProxyV2, it operates on a self-contained byte
// slice and returns the offset at which the payload begins so the caller can
// recover the datagram body that follows the header.
func parseProxyV2Datagram(data []byte) (*ProxyHeader, int, error) {
	if len(data) < proxyV2HeaderLen {
		return nil, 0, ErrProxyProtoInvalid
	}

	if !bytes.Equal(data[:proxyV2SignatureLen], proxyV2Signature) {
		return nil, 0, ErrProxyProtoInvalid
	}

	verCmd := data[12]
	family := data[13]
	addrLen := int(binary.BigEndian.Uint16(data[14:16]))

	if verCmd != proxyV2CmdLocal && verCmd != proxyV2CmdProxy {
		return nil, 0, ErrProxyProtoInvalid
	}

	end := proxyV2HeaderLen + addrLen
	if end > len(data) {
		return nil, 0, ErrProxyProtoInvalid
	}

	if verCmd == proxyV2CmdLocal || addrLen == 0 {
		return &ProxyHeader{}, end, nil
	}

	header, err := parseProxyV2Block(family, data[proxyV2HeaderLen:end])
	if err != nil {
		return nil, 0, err
	}

	return header, end, nil
}

// encodeProxyV2 encodes a PROXY protocol v2 PROXY-command header for the given
// source and destination addresses. Only *net.UDPAddr and *net.TCPAddr are
// supported; any other address type yields a LOCAL header with no addresses.
func encodeProxyV2(src, dst net.Addr) []byte {
	srcUDP, dstUDP, family, ok := proxyV2UDPPair(src, dst)
	if !ok {
		header := make([]byte, proxyV2HeaderLen)
		copy(header, proxyV2Signature)
		header[12] = proxyV2CmdLocal
		header[13] = proxyV2FamilyUnspec
		return header
	}

	var addrLen int
	if family == proxyV2FamilyUDPv4 {
		addrLen = proxyV2IPv4AddrLen
	} else {
		addrLen = proxyV2IPv6AddrLen
	}

	header := make([]byte, proxyV2HeaderLen+addrLen)
	copy(header, proxyV2Signature)
	header[12] = proxyV2CmdProxy
	header[13] = family
	binary.BigEndian.PutUint16(header[14:16], uint16(addrLen))

	body := header[proxyV2HeaderLen:]
	if addrLen == proxyV2IPv4AddrLen {
		copy(body[0:4], srcUDP.IP.To4())
		copy(body[4:8], dstUDP.IP.To4())
		binary.BigEndian.PutUint16(body[8:10], uint16(srcUDP.Port))
		binary.BigEndian.PutUint16(body[10:12], uint16(dstUDP.Port))
	} else {
		copy(body[0:16], srcUDP.IP.To16())
		copy(body[16:32], dstUDP.IP.To16())
		binary.BigEndian.PutUint16(body[32:34], uint16(srcUDP.Port))
		binary.BigEndian.PutUint16(body[34:36], uint16(dstUDP.Port))
	}

	return header
}

// proxyV2UDPPair validates that src and dst are a usable UDP address pair and
// returns them together with the matching v2 address family. It returns
// ok=false when either address is not a UDP address with an IP.
func proxyV2UDPPair(src, dst net.Addr) (srcUDP, dstUDP *net.UDPAddr, family byte, ok bool) {
	srcUDP, srcOK := src.(*net.UDPAddr)
	dstUDP, dstOK := dst.(*net.UDPAddr)
	if !srcOK || !dstOK || srcUDP.IP == nil || dstUDP.IP == nil {
		return nil, nil, 0, false
	}

	if srcUDP.IP.To4() != nil && dstUDP.IP.To4() != nil {
		family = proxyV2FamilyUDPv4
	} else {
		family = proxyV2FamilyUDPv6
	}

	return srcUDP, dstUDP, family, true
}

// buildAddrHeader builds a ProxyHeader with addresses of the concrete type that
// matches the transport protocol declared by the address family.
func buildAddrHeader(family byte, srcIP, dstIP net.IP, srcPort, dstPort int) *ProxyHeader {
	if family == proxyV2FamilyUDPv4 || family == proxyV2FamilyUDPv6 {
		return &ProxyHeader{
			SourceAddr: &net.UDPAddr{IP: srcIP, Port: srcPort},
			DestAddr:   &net.UDPAddr{IP: dstIP, Port: dstPort},
			Transport:  ProxyTransportUDP,
		}
	}

	return &ProxyHeader{
		SourceAddr: &net.TCPAddr{IP: srcIP, Port: srcPort},
		DestAddr:   &net.TCPAddr{IP: dstIP, Port: dstPort},
		Transport:  ProxyTransportTCP,
	}
}
