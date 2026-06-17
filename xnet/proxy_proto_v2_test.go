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
		assert.Equal(t, ProxyTransportTCP, result.Transport)
		assert.IsType(t, &net.TCPAddr{}, result.SourceAddr)
		assert.Equal(t, "192.168.1.1:12345", result.SourceAddr.String())
		assert.Equal(t, "192.168.1.2:80", result.DestAddr.String())
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
		assert.Equal(t, ProxyTransportTCP, result.Transport)
		assert.IsType(t, &net.TCPAddr{}, result.SourceAddr)
		assert.Equal(t, "[2001:db8::1]:12345", result.SourceAddr.String())
		assert.Equal(t, "[2001:db8::2]:443", result.DestAddr.String())
	})

	t.Run("valid UDP4 proxy command", func(t *testing.T) {
		header := buildProxyV2Header(proxyV2CmdProxy, proxyV2FamilyUDPv4,
			net.ParseIP("192.168.1.1").To4(),
			net.ParseIP("192.168.1.2").To4(),
			12345, 53)

		reader := bufio.NewReader(bytes.NewReader(header))
		result, err := parseProxyV2(reader)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, ProxyTransportUDP, result.Transport)
		require.IsType(t, &net.UDPAddr{}, result.SourceAddr)
		assert.Equal(t, "udp", result.SourceAddr.Network())
		assert.Equal(t, "192.168.1.1:12345", result.SourceAddr.String())
		assert.Equal(t, "192.168.1.2:53", result.DestAddr.String())
	})

	t.Run("valid UDP6 proxy command", func(t *testing.T) {
		header := buildProxyV2Header(proxyV2CmdProxy, proxyV2FamilyUDPv6,
			net.ParseIP("2001:db8::1"),
			net.ParseIP("2001:db8::2"),
			12345, 53)

		reader := bufio.NewReader(bytes.NewReader(header))
		result, err := parseProxyV2(reader)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, ProxyTransportUDP, result.Transport)
		require.IsType(t, &net.UDPAddr{}, result.SourceAddr)
		assert.Equal(t, "[2001:db8::1]:12345", result.SourceAddr.String())
		assert.Equal(t, "[2001:db8::2]:53", result.DestAddr.String())
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

	t.Run("parses AWS VPC endpoint TLV", func(t *testing.T) {
		header := buildProxyV2HeaderWithTLVs(proxyV2CmdProxy, proxyV2FamilyTCPv4,
			net.ParseIP("192.168.1.1").To4(),
			net.ParseIP("192.168.1.2").To4(),
			12345, 80,
			awsVPCEndpointTLV("vpce-08d2bf15fac5001c9"))

		reader := bufio.NewReader(bytes.NewReader(header))
		result, err := parseProxyV2(reader)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "192.168.1.1:12345", result.SourceAddr.String())

		require.Len(t, result.TLVs, 1)
		assert.Equal(t, PP2TypeAWS, result.TLVs[0].Type)

		id, ok := result.VPCEndpointID()
		assert.True(t, ok)
		assert.Equal(t, "vpce-08d2bf15fac5001c9", id)
	})

	t.Run("parses multiple TLVs preserving order", func(t *testing.T) {
		tlvs := append(encodeTLV(0x04, []byte("noop")), awsVPCEndpointTLV("vpce-1234")...)
		header := buildProxyV2HeaderWithTLVs(proxyV2CmdProxy, proxyV2FamilyUDPv6,
			net.ParseIP("2001:db8::1"),
			net.ParseIP("2001:db8::2"),
			5353, 53, tlvs)

		reader := bufio.NewReader(bytes.NewReader(header))
		result, err := parseProxyV2(reader)

		require.NoError(t, err)
		require.Len(t, result.TLVs, 2)
		assert.Equal(t, byte(0x04), result.TLVs[0].Type)
		assert.Equal(t, "noop", string(result.TLVs[0].Value))
		assert.Equal(t, PP2TypeAWS, result.TLVs[1].Type)

		id, ok := result.VPCEndpointID()
		assert.True(t, ok)
		assert.Equal(t, "vpce-1234", id)
	})

	t.Run("unix family with trailing bytes yields empty header", func(t *testing.T) {
		// Unix sockets carry a 216-byte address block; here we send a short
		// non-zero block to exercise the family without parseable addresses.
		header := buildRawProxyV2Header(proxyV2FamilyUnixStream, 8)

		reader := bufio.NewReader(bytes.NewReader(header))
		result, err := parseProxyV2(reader)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Nil(t, result.SourceAddr)
		assert.Nil(t, result.TLVs)
	})

	t.Run("rejects address block shorter than family size", func(t *testing.T) {
		// TCP4 needs 12 address octets; declare only 8.
		header := buildRawProxyV2Header(proxyV2FamilyTCPv4, 8)

		reader := bufio.NewReader(bytes.NewReader(header))
		_, err := parseProxyV2(reader)

		assert.ErrorIs(t, err, ErrProxyProtoInvalid)
	})

	t.Run("rejects truncated TLV", func(t *testing.T) {
		// Declared TLV length exceeds the bytes that follow.
		badTLV := []byte{PP2TypeAWS, 0x00, 0x10, 0x01}
		header := buildProxyV2HeaderWithTLVs(proxyV2CmdProxy, proxyV2FamilyTCPv4,
			net.ParseIP("192.168.1.1").To4(),
			net.ParseIP("192.168.1.2").To4(),
			12345, 80, badTLV)

		reader := bufio.NewReader(bytes.NewReader(header))
		_, err := parseProxyV2(reader)

		assert.ErrorIs(t, err, ErrProxyProtoInvalid)
	})
}

func TestProxyHeaderTLV(t *testing.T) {
	t.Run("TLV lookup", func(t *testing.T) {
		h := &ProxyHeader{TLVs: []ProxyTLV{
			{Type: 0x02, Value: []byte("authority")},
			{Type: PP2TypeAWS, Value: []byte{PP2SubtypeAWSVPCEID, 'v', 'p', 'c', 'e'}},
		}}

		value, ok := h.TLV(0x02)
		assert.True(t, ok)
		assert.Equal(t, "authority", string(value))

		_, ok = h.TLV(0xFF)
		assert.False(t, ok)
	})

	t.Run("VPCEndpointID", func(t *testing.T) {
		tests := []struct {
			name  string
			tlvs  []ProxyTLV
			want  string
			found bool
		}{
			{
				name:  "valid",
				tlvs:  []ProxyTLV{{Type: PP2TypeAWS, Value: []byte{PP2SubtypeAWSVPCEID, 'v', 'p', 'c', 'e', '-', '1'}}},
				want:  "vpce-1",
				found: true,
			},
			{
				name:  "wrong subtype",
				tlvs:  []ProxyTLV{{Type: PP2TypeAWS, Value: []byte{0x02, 'x'}}},
				found: false,
			},
			{
				name:  "empty value",
				tlvs:  []ProxyTLV{{Type: PP2TypeAWS, Value: []byte{}}},
				found: false,
			},
			{
				name:  "no AWS TLV",
				tlvs:  []ProxyTLV{{Type: 0x01, Value: []byte("x")}},
				found: false,
			},
			{
				name:  "subtype only, empty id",
				tlvs:  []ProxyTLV{{Type: PP2TypeAWS, Value: []byte{PP2SubtypeAWSVPCEID}}},
				want:  "",
				found: true,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				h := &ProxyHeader{TLVs: tt.tlvs}
				id, ok := h.VPCEndpointID()
				assert.Equal(t, tt.found, ok)
				assert.Equal(t, tt.want, id)
			})
		}
	})
}

func TestParseProxyV2TLVs(t *testing.T) {
	t.Run("empty input", func(t *testing.T) {
		tlvs, err := parseProxyV2TLVs(nil)
		require.NoError(t, err)
		assert.Nil(t, tlvs)
	})

	t.Run("single zero-length TLV", func(t *testing.T) {
		tlvs, err := parseProxyV2TLVs([]byte{0x04, 0x00, 0x00})
		require.NoError(t, err)
		require.Len(t, tlvs, 1)
		assert.Equal(t, byte(0x04), tlvs[0].Type)
		assert.Empty(t, tlvs[0].Value)
	})

	t.Run("incomplete header", func(t *testing.T) {
		_, err := parseProxyV2TLVs([]byte{0x04, 0x00})
		assert.ErrorIs(t, err, ErrProxyProtoInvalid)
	})

	t.Run("length exceeds buffer", func(t *testing.T) {
		_, err := parseProxyV2TLVs([]byte{0x04, 0x00, 0x05, 'a', 'b'})
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

// buildRawProxyV2Header builds a PROXY v2 PROXY-command header for an arbitrary
// family with a zero-filled address block of the given length.
func buildRawProxyV2Header(family byte, bodyLen int) []byte {
	header := make([]byte, proxyV2HeaderLen+bodyLen)
	copy(header, proxyV2Signature)
	header[12] = proxyV2CmdProxy
	header[13] = family
	binary.BigEndian.PutUint16(header[14:16], uint16(bodyLen))

	return header
}

// encodeTLV encodes a single Type-Length-Value vector.
func encodeTLV(typ byte, value []byte) []byte {
	tlv := make([]byte, proxyV2TLVHeaderLen+len(value))
	tlv[0] = typ
	binary.BigEndian.PutUint16(tlv[1:3], uint16(len(value)))
	copy(tlv[proxyV2TLVHeaderLen:], value)

	return tlv
}

// buildProxyV2HeaderWithTLVs builds a PROXY v2 header and appends the given raw
// TLV bytes, fixing up the declared address length to include them.
func buildProxyV2HeaderWithTLVs(cmd, family byte, srcIP, dstIP net.IP, srcPort, dstPort int, tlvs []byte) []byte {
	header := buildProxyV2Header(cmd, family, srcIP, dstIP, srcPort, dstPort)
	header = append(header, tlvs...)

	addrLen := len(header) - proxyV2HeaderLen
	binary.BigEndian.PutUint16(header[14:16], uint16(addrLen))

	return header
}

// awsVPCEndpointTLV builds a PP2_TYPE_AWS TLV carrying the given VPC endpoint ID.
func awsVPCEndpointTLV(id string) []byte {
	value := append([]byte{PP2SubtypeAWSVPCEID}, []byte(id)...)
	return encodeTLV(PP2TypeAWS, value)
}

func BenchmarkIsProxyV2(b *testing.B) {
	data := buildProxyV2Header(proxyV2CmdProxy, proxyV2FamilyTCPv4,
		net.ParseIP("192.168.1.1").To4(),
		net.ParseIP("192.168.1.2").To4(),
		12345, 80)

	for b.Loop() {
		isProxyV2(data)
	}
}

func BenchmarkParseProxyV2(b *testing.B) {
	b.Run("TCP4", func(b *testing.B) {
		data := buildProxyV2Header(proxyV2CmdProxy, proxyV2FamilyTCPv4,
			net.ParseIP("192.168.1.1").To4(),
			net.ParseIP("192.168.1.2").To4(),
			12345, 80)

		for b.Loop() {
			reader := bufio.NewReader(bytes.NewReader(data))
			_, _ = parseProxyV2(reader)
		}
	})

	b.Run("TCP6", func(b *testing.B) {
		data := buildProxyV2Header(proxyV2CmdProxy, proxyV2FamilyTCPv6,
			net.ParseIP("2001:db8::1"),
			net.ParseIP("2001:db8::2"),
			12345, 443)

		for b.Loop() {
			reader := bufio.NewReader(bytes.NewReader(data))
			_, _ = parseProxyV2(reader)
		}
	})

	b.Run("LOCAL", func(b *testing.B) {
		data := buildProxyV2Header(proxyV2CmdLocal, proxyV2FamilyTCPv4,
			net.ParseIP("192.168.1.1").To4(),
			net.ParseIP("192.168.1.2").To4(),
			12345, 80)

		for b.Loop() {
			reader := bufio.NewReader(bytes.NewReader(data))
			_, _ = parseProxyV2(reader)
		}
	})

	b.Run("TCP4_AWS_TLV", func(b *testing.B) {
		data := buildProxyV2HeaderWithTLVs(proxyV2CmdProxy, proxyV2FamilyTCPv4,
			net.ParseIP("192.168.1.1").To4(),
			net.ParseIP("192.168.1.2").To4(),
			12345, 80,
			awsVPCEndpointTLV("vpce-08d2bf15fac5001c9"))

		for b.Loop() {
			reader := bufio.NewReader(bytes.NewReader(data))
			_, _ = parseProxyV2(reader)
		}
	})
}

func BenchmarkParseProxyV2Datagram(b *testing.B) {
	b.Run("UDP4", func(b *testing.B) {
		data := buildProxyV2Header(proxyV2CmdProxy, proxyV2FamilyUDPv4,
			net.ParseIP("192.168.1.1").To4(),
			net.ParseIP("192.168.1.2").To4(),
			12345, 53)

		for b.Loop() {
			_, _, _ = parseProxyV2Datagram(data)
		}
	})

	b.Run("UDP4_AWS_TLV", func(b *testing.B) {
		data := buildProxyV2HeaderWithTLVs(proxyV2CmdProxy, proxyV2FamilyUDPv4,
			net.ParseIP("192.168.1.1").To4(),
			net.ParseIP("192.168.1.2").To4(),
			12345, 53,
			awsVPCEndpointTLV("vpce-08d2bf15fac5001c9"))

		for b.Loop() {
			_, _, _ = parseProxyV2Datagram(data)
		}
	})
}

func BenchmarkParseProxyV2TLVs(b *testing.B) {
	b.Run("single", func(b *testing.B) {
		data := awsVPCEndpointTLV("vpce-08d2bf15fac5001c9")

		for b.Loop() {
			_, _ = parseProxyV2TLVs(data)
		}
	})

	b.Run("multiple", func(b *testing.B) {
		data := append(encodeTLV(0x04, []byte("noop")), awsVPCEndpointTLV("vpce-08d2bf15fac5001c9")...)
		data = append(data, encodeTLV(0x03, []byte{0xDE, 0xAD, 0xBE, 0xEF})...)

		for b.Loop() {
			_, _ = parseProxyV2TLVs(data)
		}
	})
}

func BenchmarkEncodeProxyV2(b *testing.B) {
	b.Run("UDP4", func(b *testing.B) {
		src := &net.UDPAddr{IP: net.ParseIP("192.0.2.1"), Port: 1111}
		dst := &net.UDPAddr{IP: net.ParseIP("192.0.2.2"), Port: 2222}

		for b.Loop() {
			_ = encodeProxyV2(src, dst)
		}
	})

	b.Run("UDP6", func(b *testing.B) {
		src := &net.UDPAddr{IP: net.ParseIP("2001:db8::1"), Port: 1111}
		dst := &net.UDPAddr{IP: net.ParseIP("2001:db8::2"), Port: 2222}

		for b.Loop() {
			_ = encodeProxyV2(src, dst)
		}
	})
}

func BenchmarkVPCEndpointID(b *testing.B) {
	header := &ProxyHeader{TLVs: []ProxyTLV{
		{Type: 0x03, Value: []byte{0xDE, 0xAD, 0xBE, 0xEF}},
		{Type: PP2TypeAWS, Value: append([]byte{PP2SubtypeAWSVPCEID}, []byte("vpce-08d2bf15fac5001c9")...)},
	}}

	for b.Loop() {
		_, _ = header.VPCEndpointID()
	}
}

func FuzzParseProxyV2(f *testing.F) {
	f.Add(buildProxyV2Header(proxyV2CmdProxy, proxyV2FamilyTCPv4,
		net.ParseIP("192.168.1.1").To4(),
		net.ParseIP("192.168.1.2").To4(),
		12345, 80))
	f.Add(buildProxyV2Header(proxyV2CmdProxy, proxyV2FamilyTCPv6,
		net.ParseIP("2001:db8::1"),
		net.ParseIP("2001:db8::2"),
		12345, 443))
	f.Add(buildProxyV2Header(proxyV2CmdLocal, proxyV2FamilyTCPv4,
		net.ParseIP("10.0.0.1").To4(),
		net.ParseIP("10.0.0.2").To4(),
		1, 65535))
	f.Add(buildProxyV2HeaderWithTLVs(proxyV2CmdProxy, proxyV2FamilyTCPv4,
		net.ParseIP("192.168.1.1").To4(),
		net.ParseIP("192.168.1.2").To4(),
		12345, 80,
		awsVPCEndpointTLV("vpce-08d2bf15fac5001c9")))
	f.Add([]byte{})
	f.Add(proxyV2Signature)
	f.Add([]byte("PROXY TCP4 192.168.1.1"))

	f.Fuzz(func(_ *testing.T, data []byte) {
		reader := bufio.NewReader(bytes.NewReader(data))
		_, _ = parseProxyV2(reader)
	})
}

func FuzzParseProxyV2Datagram(f *testing.F) {
	f.Add(buildProxyV2Header(proxyV2CmdProxy, proxyV2FamilyUDPv4,
		net.ParseIP("10.0.0.1").To4(),
		net.ParseIP("10.0.0.2").To4(),
		1, 53))
	f.Add(buildProxyV2Header(proxyV2CmdProxy, proxyV2FamilyUDPv6,
		net.ParseIP("2001:db8::1"),
		net.ParseIP("2001:db8::2"),
		1, 53))
	f.Add(buildProxyV2HeaderWithTLVs(proxyV2CmdProxy, proxyV2FamilyUDPv4,
		net.ParseIP("10.0.0.1").To4(),
		net.ParseIP("10.0.0.2").To4(),
		1, 53,
		awsVPCEndpointTLV("vpce-08d2bf15fac5001c9")))
	f.Add(append(buildProxyV2Header(proxyV2CmdProxy, proxyV2FamilyUDPv4,
		net.ParseIP("10.0.0.1").To4(),
		net.ParseIP("10.0.0.2").To4(),
		1, 53), []byte("datagram payload")...))
	f.Add([]byte{})
	f.Add(proxyV2Signature)

	f.Fuzz(func(t *testing.T, data []byte) {
		header, offset, err := parseProxyV2Datagram(data)
		if err != nil {
			return
		}

		// On success the payload offset must lie within the datagram so the
		// caller can always recover the body slice safely.
		if offset < 0 || offset > len(data) {
			t.Fatalf("offset %d out of range for %d-byte datagram", offset, len(data))
		}

		require.NotNil(t, header)
	})
}

func FuzzParseProxyV2TLVs(f *testing.F) {
	f.Add(awsVPCEndpointTLV("vpce-08d2bf15fac5001c9"))
	f.Add(encodeTLV(0x04, []byte("noop")))
	f.Add(append(encodeTLV(0x04, []byte("noop")), awsVPCEndpointTLV("vpce-1")...))
	f.Add([]byte{0x04, 0x00, 0x00})
	f.Add([]byte{})
	f.Add([]byte{0xEA, 0xFF, 0xFF})

	f.Fuzz(func(t *testing.T, data []byte) {
		tlvs, err := parseProxyV2TLVs(data)
		if err != nil {
			return
		}

		// A successful parse must consume the input exactly: the sum of the
		// TLV headers and values equals the input length.
		total := 0
		for _, tlv := range tlvs {
			total += proxyV2TLVHeaderLen + len(tlv.Value)
		}

		if total != len(data) {
			t.Fatalf("parsed TLVs span %d bytes, input was %d", total, len(data))
		}
	})
}
