package xnet

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsProxyV2(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		expected bool
	}{
		{
			name:     "valid v2 signature",
			data:     proxyV2Signature,
			expected: true,
		},
		{
			name:     "v1 prefix",
			data:     []byte("PROXY TCP4"),
			expected: false,
		},
		{
			name:     "short data",
			data:     proxyV2Signature[:5],
			expected: false,
		},
		{
			name:     "empty data",
			data:     []byte{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isProxyV2(tt.data)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseProxyV2(t *testing.T) {
	t.Run("valid TCP4 proxy command", func(t *testing.T) {
		header := buildProxyV2Header(proxyV2CmdProxy, proxyV2FamilyTCPv4,
			net.ParseIP("192.168.1.1").To4(),
			net.ParseIP("192.168.1.2").To4(),
			12345, 80)

		reader := bufio.NewReader(bytes.NewReader(header))
		result, err := parseProxyV2(reader)

		require.NoError(t, err)
		require.NotNil(t, result)
		require.NotNil(t, result.SourceAddr)
		assert.Equal(t, "192.168.1.1", result.SourceAddr.IP.String())
		assert.Equal(t, 12345, result.SourceAddr.Port)
		assert.Equal(t, "192.168.1.2", result.DestAddr.IP.String())
		assert.Equal(t, 80, result.DestAddr.Port)
	})

	t.Run("valid TCP6 proxy command", func(t *testing.T) {
		header := buildProxyV2Header(proxyV2CmdProxy, proxyV2FamilyTCPv6,
			net.ParseIP("2001:db8::1"),
			net.ParseIP("2001:db8::2"),
			12345, 443)

		reader := bufio.NewReader(bytes.NewReader(header))
		result, err := parseProxyV2(reader)

		require.NoError(t, err)
		require.NotNil(t, result)
		require.NotNil(t, result.SourceAddr)
		assert.Equal(t, "2001:db8::1", result.SourceAddr.IP.String())
		assert.Equal(t, 12345, result.SourceAddr.Port)
		assert.Equal(t, "2001:db8::2", result.DestAddr.IP.String())
		assert.Equal(t, 443, result.DestAddr.Port)
	})

	t.Run("local command", func(t *testing.T) {
		header := buildProxyV2Header(proxyV2CmdLocal, proxyV2FamilyTCPv4,
			net.ParseIP("192.168.1.1").To4(),
			net.ParseIP("192.168.1.2").To4(),
			12345, 80)

		reader := bufio.NewReader(bytes.NewReader(header))
		result, err := parseProxyV2(reader)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Nil(t, result.SourceAddr)
		assert.Nil(t, result.DestAddr)
	})

	t.Run("unspec family", func(t *testing.T) {
		header := make([]byte, proxyV2HeaderLen)
		copy(header, proxyV2Signature)
		header[12] = proxyV2CmdProxy
		header[13] = proxyV2FamilyUnspec
		binary.BigEndian.PutUint16(header[14:16], 0)

		reader := bufio.NewReader(bytes.NewReader(header))
		result, err := parseProxyV2(reader)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Nil(t, result.SourceAddr)
	})

	t.Run("invalid signature", func(t *testing.T) {
		header := make([]byte, proxyV2HeaderLen)
		copy(header, []byte("INVALIDHEADER"))

		reader := bufio.NewReader(bytes.NewReader(header))
		_, err := parseProxyV2(reader)

		assert.ErrorIs(t, err, ErrProxyProtoInvalid)
	})

	t.Run("invalid command", func(t *testing.T) {
		header := make([]byte, proxyV2HeaderLen)
		copy(header, proxyV2Signature)
		header[12] = 0xFF
		header[13] = proxyV2FamilyTCPv4

		reader := bufio.NewReader(bytes.NewReader(header))
		_, err := parseProxyV2(reader)

		assert.ErrorIs(t, err, ErrProxyProtoInvalid)
	})

	t.Run("truncated header", func(t *testing.T) {
		header := proxyV2Signature[:8]
		reader := bufio.NewReader(bytes.NewReader(header))
		_, err := parseProxyV2(reader)

		assert.ErrorIs(t, err, ErrProxyProtoInvalid)
	})

	t.Run("truncated address data", func(t *testing.T) {
		header := make([]byte, proxyV2HeaderLen)
		copy(header, proxyV2Signature)
		header[12] = proxyV2CmdProxy
		header[13] = proxyV2FamilyTCPv4
		binary.BigEndian.PutUint16(header[14:16], 100)

		reader := bufio.NewReader(bytes.NewReader(header))
		_, err := parseProxyV2(reader)

		assert.ErrorIs(t, err, ErrProxyProtoInvalid)
	})
}

func buildProxyV2Header(cmd, family byte, srcIP, dstIP net.IP, srcPort, dstPort int) []byte {
	var addrLen uint16
	var addrData []byte

	switch family {
	case proxyV2FamilyTCPv4, proxyV2FamilyUDPv4:
		addrLen = proxyV2IPv4AddrLen
		addrData = make([]byte, addrLen)
		copy(addrData[0:4], srcIP.To4())
		copy(addrData[4:8], dstIP.To4())
		binary.BigEndian.PutUint16(addrData[8:10], uint16(srcPort))
		binary.BigEndian.PutUint16(addrData[10:12], uint16(dstPort))

	case proxyV2FamilyTCPv6, proxyV2FamilyUDPv6:
		addrLen = proxyV2IPv6AddrLen
		addrData = make([]byte, addrLen)
		copy(addrData[0:16], srcIP.To16())
		copy(addrData[16:32], dstIP.To16())
		binary.BigEndian.PutUint16(addrData[32:34], uint16(srcPort))
		binary.BigEndian.PutUint16(addrData[34:36], uint16(dstPort))
	}

	header := make([]byte, proxyV2HeaderLen+int(addrLen))
	copy(header, proxyV2Signature)
	header[12] = cmd
	header[13] = family
	binary.BigEndian.PutUint16(header[14:16], addrLen)
	copy(header[proxyV2HeaderLen:], addrData)

	return header
}
