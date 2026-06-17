package xnet

import (
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newUDPPair creates a wrapped server packet conn and a raw client conn dialed
// to it. Both are registered for cleanup.
func newUDPPair(t *testing.T, config ProxyProtoUDPConfig) (*ProxyProtoPacketConn, *net.UDPConn) {
	t.Helper()

	server, err := ProxyProtoListenPacket("udp", "127.0.0.1:0", config)
	require.NoError(t, err)
	t.Cleanup(func() { server.Close() })

	serverAddr := server.LocalAddr().(*net.UDPAddr)
	client, err := net.DialUDP("udp", nil, serverAddr)
	require.NoError(t, err)
	t.Cleanup(func() { client.Close() })

	return server.(*ProxyProtoPacketConn), client
}

func TestProxyProtoListenPacket(t *testing.T) {
	t.Run("creates packet conn successfully", func(t *testing.T) {
		conn, err := ProxyProtoListenPacket("udp", "127.0.0.1:0", ProxyProtoUDPConfig{})
		require.NoError(t, err)
		defer conn.Close()

		assert.NotNil(t, conn.LocalAddr())
		assert.Implements(t, (*net.PacketConn)(nil), conn)
	})

	t.Run("returns error for invalid network", func(t *testing.T) {
		_, err := ProxyProtoListenPacket("tcp", "127.0.0.1:0", ProxyProtoUDPConfig{})
		assert.Error(t, err)
	})
}

func TestProxyProtoPacketConnStrict(t *testing.T) {
	t.Run("reads payload and original source from header", func(t *testing.T) {
		server, client := newUDPPair(t, ProxyProtoUDPConfig{})

		header := buildProxyV2Header(proxyV2CmdProxy, proxyV2FamilyUDPv4,
			net.ParseIP("203.0.113.7").To4(),
			net.ParseIP("198.51.100.9").To4(),
			40000, 53)

		_, err := client.Write(append(header, []byte("query")...))
		require.NoError(t, err)

		require.NoError(t, server.SetReadDeadline(time.Now().Add(2*time.Second)))

		buf := make([]byte, 64)
		n, header2, sender, err := server.ReadFromProxy(buf)
		require.NoError(t, err)

		assert.Equal(t, "query", string(buf[:n]))
		require.NotNil(t, header2)
		assert.Equal(t, ProxyTransportUDP, header2.Transport)
		assert.Equal(t, "203.0.113.7:40000", sender.String())
		assert.Equal(t, "udp", sender.Network())
	})

	t.Run("local command falls back to sender address", func(t *testing.T) {
		server, client := newUDPPair(t, ProxyProtoUDPConfig{})

		header := buildProxyV2Header(proxyV2CmdLocal, proxyV2FamilyUDPv4,
			net.ParseIP("203.0.113.7").To4(),
			net.ParseIP("198.51.100.9").To4(),
			40000, 53)

		_, err := client.Write(append(header, []byte("local")...))
		require.NoError(t, err)

		require.NoError(t, server.SetReadDeadline(time.Now().Add(2*time.Second)))

		buf := make([]byte, 64)
		n, hdr, sender, err := server.ReadFromProxy(buf)
		require.NoError(t, err)

		assert.Equal(t, "local", string(buf[:n]))
		require.NotNil(t, hdr)
		assert.Nil(t, hdr.SourceAddr)
		// No address in the header, so the real datagram sender is returned.
		assert.Contains(t, sender.String(), "127.0.0.1:")
	})

	t.Run("rejects datagram without proxy header", func(t *testing.T) {
		server, client := newUDPPair(t, ProxyProtoUDPConfig{})

		_, err := client.Write([]byte("plain"))
		require.NoError(t, err)

		require.NoError(t, server.SetReadDeadline(time.Now().Add(2*time.Second)))

		buf := make([]byte, 64)
		_, _, err = server.ReadFrom(buf)
		assert.ErrorIs(t, err, ErrProxyProtoUnknownProto)
	})

	t.Run("rejects malformed proxy header", func(t *testing.T) {
		server, client := newUDPPair(t, ProxyProtoUDPConfig{})

		header := buildProxyV2Header(proxyV2CmdProxy, proxyV2FamilyUDPv4,
			net.ParseIP("203.0.113.7").To4(),
			net.ParseIP("198.51.100.9").To4(),
			40000, 53)
		// Corrupt the declared address length so it exceeds the datagram.
		header[15] = 0xFF

		_, err := client.Write(header)
		require.NoError(t, err)

		require.NoError(t, server.SetReadDeadline(time.Now().Add(2*time.Second)))

		buf := make([]byte, 64)
		_, _, err = server.ReadFrom(buf)
		assert.ErrorIs(t, err, ErrProxyProtoInvalid)
	})
}

func TestProxyProtoPacketConnAuto(t *testing.T) {
	t.Run("accepts regular datagram", func(t *testing.T) {
		server, client := newUDPPair(t, ProxyProtoUDPConfig{Mode: ProxyProtoModeAuto})

		_, err := client.Write([]byte("plain datagram"))
		require.NoError(t, err)

		require.NoError(t, server.SetReadDeadline(time.Now().Add(2*time.Second)))

		buf := make([]byte, 64)
		n, header, sender, err := server.ReadFromProxy(buf)
		require.NoError(t, err)

		assert.Equal(t, "plain datagram", string(buf[:n]))
		assert.Nil(t, header)
		assert.Contains(t, sender.String(), "127.0.0.1:")
	})

	t.Run("accepts proxy datagram", func(t *testing.T) {
		server, client := newUDPPair(t, ProxyProtoUDPConfig{Mode: ProxyProtoModeAuto})

		header := buildProxyV2Header(proxyV2CmdProxy, proxyV2FamilyUDPv6,
			net.ParseIP("2001:db8::1"),
			net.ParseIP("2001:db8::2"),
			5555, 4444)

		_, err := client.Write(append(header, []byte("v6 payload")...))
		require.NoError(t, err)

		require.NoError(t, server.SetReadDeadline(time.Now().Add(2*time.Second)))

		buf := make([]byte, 64)
		n, _, sender, err := server.ReadFromProxy(buf)
		require.NoError(t, err)

		assert.Equal(t, "v6 payload", string(buf[:n]))
		assert.Equal(t, "[2001:db8::1]:5555", sender.String())
	})
}

func TestProxyProtoPacketConnTrustedProxies(t *testing.T) {
	t.Run("strict rejects untrusted source", func(t *testing.T) {
		_, untrusted, _ := net.ParseCIDR("10.0.0.0/8")
		server, client := newUDPPair(t, ProxyProtoUDPConfig{
			TrustedProxies: []net.IPNet{*untrusted},
		})

		header := buildProxyV2Header(proxyV2CmdProxy, proxyV2FamilyUDPv4,
			net.ParseIP("203.0.113.7").To4(),
			net.ParseIP("198.51.100.9").To4(),
			40000, 53)

		_, err := client.Write(append(header, []byte("payload")...))
		require.NoError(t, err)

		require.NoError(t, server.SetReadDeadline(time.Now().Add(2*time.Second)))

		buf := make([]byte, 64)
		_, _, err = server.ReadFrom(buf)
		assert.ErrorIs(t, err, ErrProxyProtoUntrusted)
	})

	t.Run("auto delivers untrusted source as regular", func(t *testing.T) {
		_, untrusted, _ := net.ParseCIDR("10.0.0.0/8")
		server, client := newUDPPair(t, ProxyProtoUDPConfig{
			Mode:           ProxyProtoModeAuto,
			TrustedProxies: []net.IPNet{*untrusted},
		})

		// Even though it looks like a PROXY datagram, the loopback source is
		// untrusted so the bytes are delivered verbatim.
		header := buildProxyV2Header(proxyV2CmdProxy, proxyV2FamilyUDPv4,
			net.ParseIP("203.0.113.7").To4(),
			net.ParseIP("198.51.100.9").To4(),
			40000, 53)
		payload := make([]byte, 0, len(header)+3)
		payload = append(payload, header...)
		payload = append(payload, []byte("raw")...)

		_, err := client.Write(payload)
		require.NoError(t, err)

		require.NoError(t, server.SetReadDeadline(time.Now().Add(2*time.Second)))

		buf := make([]byte, 128)
		n, hdr, _, err := server.ReadFromProxy(buf)
		require.NoError(t, err)
		assert.Nil(t, hdr)
		assert.Equal(t, payload, buf[:n])
	})
}

func TestProxyProtoPacketConnWrite(t *testing.T) {
	t.Run("write routes reply back to proxy", func(t *testing.T) {
		_, trusted, _ := net.ParseCIDR("127.0.0.0/8")
		server, client := newUDPPair(t, ProxyProtoUDPConfig{
			TrustedProxies: []net.IPNet{*trusted},
		})

		clientAddr := net.UDPAddr{IP: net.ParseIP("203.0.113.7"), Port: 40000}
		header := buildProxyV2Header(proxyV2CmdProxy, proxyV2FamilyUDPv4,
			clientAddr.IP.To4(),
			net.ParseIP("198.51.100.9").To4(),
			clientAddr.Port, 53)

		_, err := client.Write(append(header, []byte("ping")...))
		require.NoError(t, err)

		require.NoError(t, server.SetReadDeadline(time.Now().Add(2*time.Second)))
		buf := make([]byte, 64)
		_, src, err := server.ReadFrom(buf)
		require.NoError(t, err)
		assert.Equal(t, clientAddr.String(), src.String())

		// Reply addressed to the original client must arrive back at the proxy.
		_, err = server.WriteTo([]byte("pong"), src)
		require.NoError(t, err)

		require.NoError(t, client.SetReadDeadline(time.Now().Add(2*time.Second)))
		reply := make([]byte, 64)
		n, err := client.Read(reply)
		require.NoError(t, err)
		assert.Equal(t, "pong", string(reply[:n]))
	})

	t.Run("write to unknown address sends directly", func(t *testing.T) {
		server, _ := newUDPPair(t, ProxyProtoUDPConfig{})

		peer, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1")})
		require.NoError(t, err)
		defer peer.Close()

		_, err = server.WriteTo([]byte("direct"), peer.LocalAddr())
		require.NoError(t, err)

		require.NoError(t, peer.SetReadDeadline(time.Now().Add(2*time.Second)))
		buf := make([]byte, 64)
		n, _, err := peer.ReadFrom(buf)
		require.NoError(t, err)
		assert.Equal(t, "direct", string(buf[:n]))
	})

	t.Run("WriteToProxy emits parseable header", func(t *testing.T) {
		server, _ := newUDPPair(t, ProxyProtoUDPConfig{})

		backend, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1")})
		require.NoError(t, err)
		defer backend.Close()

		src := &net.UDPAddr{IP: net.ParseIP("203.0.113.7"), Port: 40000}
		_, err = server.WriteToProxy([]byte("forwarded"), src, backend.LocalAddr())
		require.NoError(t, err)

		require.NoError(t, backend.SetReadDeadline(time.Now().Add(2*time.Second)))
		buf := make([]byte, 128)
		n, _, err := backend.ReadFrom(buf)
		require.NoError(t, err)

		header, offset, err := parseProxyV2Datagram(buf[:n])
		require.NoError(t, err)
		assert.Equal(t, ProxyTransportUDP, header.Transport)
		assert.Equal(t, src.String(), header.SourceAddr.String())
		assert.Equal(t, "forwarded", string(buf[offset:n]))
	})
}

func TestProxyProtoPacketConnDeadlines(t *testing.T) {
	server, _ := newUDPPair(t, ProxyProtoUDPConfig{})

	assert.NoError(t, server.SetDeadline(time.Now().Add(time.Second)))
	assert.NoError(t, server.SetReadDeadline(time.Now().Add(time.Second)))
	assert.NoError(t, server.SetWriteDeadline(time.Now().Add(time.Second)))
}

func TestParseProxyV2Datagram(t *testing.T) {
	tests := []struct {
		name      string
		build     func() []byte
		wantErr   error
		wantTrans ProxyTransport
		wantSrc   string
		wantVPCE  string
		wantOff   int
	}{
		{
			name: "udp4 with payload",
			build: func() []byte {
				h := buildProxyV2Header(proxyV2CmdProxy, proxyV2FamilyUDPv4,
					net.ParseIP("10.0.0.1").To4(), net.ParseIP("10.0.0.2").To4(), 1, 2)
				return append(h, []byte("body")...)
			},
			wantTrans: ProxyTransportUDP,
			wantSrc:   "10.0.0.1:1",
			wantOff:   proxyV2HeaderLen + proxyV2IPv4AddrLen,
		},
		{
			name: "udp4 with AWS VPCE TLV",
			build: func() []byte {
				return buildProxyV2HeaderWithTLVs(proxyV2CmdProxy, proxyV2FamilyUDPv4,
					net.ParseIP("10.0.0.1").To4(), net.ParseIP("10.0.0.2").To4(), 1, 2,
					awsVPCEndpointTLV("vpce-udp"))
			},
			wantTrans: ProxyTransportUDP,
			wantSrc:   "10.0.0.1:1",
			wantVPCE:  "vpce-udp",
			wantOff:   proxyV2HeaderLen + proxyV2IPv4AddrLen + proxyV2TLVHeaderLen + 1 + len("vpce-udp"),
		},
		{
			name: "local command",
			build: func() []byte {
				return buildProxyV2Header(proxyV2CmdLocal, proxyV2FamilyUDPv4,
					net.ParseIP("10.0.0.1").To4(), net.ParseIP("10.0.0.2").To4(), 1, 2)
			},
			wantTrans: ProxyTransportUnspec,
			wantOff:   proxyV2HeaderLen + proxyV2IPv4AddrLen,
		},
		{
			name: "too short",
			build: func() []byte {
				return proxyV2Signature[:8]
			},
			wantErr: ErrProxyProtoInvalid,
		},
		{
			name: "bad signature",
			build: func() []byte {
				b := make([]byte, proxyV2HeaderLen)
				copy(b, []byte("not-a-signature!"))
				return b
			},
			wantErr: ErrProxyProtoInvalid,
		},
		{
			name: "addr length exceeds datagram",
			build: func() []byte {
				b := make([]byte, proxyV2HeaderLen)
				copy(b, proxyV2Signature)
				b[12] = proxyV2CmdProxy
				b[13] = proxyV2FamilyUDPv4
				b[14] = 0x00
				b[15] = 0xFF
				return b
			},
			wantErr: ErrProxyProtoInvalid,
		},
		{
			name: "invalid command",
			build: func() []byte {
				b := make([]byte, proxyV2HeaderLen)
				copy(b, proxyV2Signature)
				b[12] = 0xFF
				return b
			},
			wantErr: ErrProxyProtoInvalid,
		},
		{
			name: "malformed TLV",
			build: func() []byte {
				// Valid UDP4 addresses followed by a TLV whose declared
				// length runs past the block.
				badTLV := []byte{PP2TypeAWS, 0x00, 0x10, 0x01}
				return buildProxyV2HeaderWithTLVs(proxyV2CmdProxy, proxyV2FamilyUDPv4,
					net.ParseIP("10.0.0.1").To4(), net.ParseIP("10.0.0.2").To4(), 1, 2, badTLV)
			},
			wantErr: ErrProxyProtoInvalid,
		},
		{
			name: "address block shorter than family size",
			build: func() []byte {
				b := make([]byte, proxyV2HeaderLen+8)
				copy(b, proxyV2Signature)
				b[12] = proxyV2CmdProxy
				b[13] = proxyV2FamilyUDPv4
				b[15] = 0x08
				return b
			},
			wantErr: ErrProxyProtoInvalid,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			header, offset, err := parseProxyV2Datagram(tt.build())

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, header)
			assert.Equal(t, tt.wantTrans, header.Transport)
			assert.Equal(t, tt.wantOff, offset)

			if tt.wantSrc != "" {
				assert.Equal(t, tt.wantSrc, header.SourceAddr.String())
			}

			if tt.wantVPCE != "" {
				id, ok := header.VPCEndpointID()
				assert.True(t, ok)
				assert.Equal(t, tt.wantVPCE, id)
			}
		})
	}
}

func TestEncodeProxyV2(t *testing.T) {
	t.Run("udp4 round trip", func(t *testing.T) {
		src := &net.UDPAddr{IP: net.ParseIP("192.0.2.1"), Port: 1111}
		dst := &net.UDPAddr{IP: net.ParseIP("192.0.2.2"), Port: 2222}

		encoded := encodeProxyV2(src, dst)
		header, offset, err := parseProxyV2Datagram(encoded)
		require.NoError(t, err)
		assert.Equal(t, ProxyTransportUDP, header.Transport)
		assert.Equal(t, src.String(), header.SourceAddr.String())
		assert.Equal(t, dst.String(), header.DestAddr.String())
		assert.Equal(t, len(encoded), offset)
	})

	t.Run("udp6 round trip", func(t *testing.T) {
		src := &net.UDPAddr{IP: net.ParseIP("2001:db8::1"), Port: 1111}
		dst := &net.UDPAddr{IP: net.ParseIP("2001:db8::2"), Port: 2222}

		encoded := encodeProxyV2(src, dst)
		header, _, err := parseProxyV2Datagram(encoded)
		require.NoError(t, err)
		assert.Equal(t, ProxyTransportUDP, header.Transport)
		assert.Equal(t, src.String(), header.SourceAddr.String())
		assert.Equal(t, dst.String(), header.DestAddr.String())
	})

	t.Run("unsupported address yields local header", func(t *testing.T) {
		src := &net.TCPAddr{IP: net.ParseIP("192.0.2.1"), Port: 1111}
		dst := &net.TCPAddr{IP: net.ParseIP("192.0.2.2"), Port: 2222}

		encoded := encodeProxyV2(src, dst)
		header, _, err := parseProxyV2Datagram(encoded)
		require.NoError(t, err)
		assert.Nil(t, header.SourceAddr)
		assert.Equal(t, ProxyTransportUnspec, header.Transport)
	})
}

func BenchmarkProxyProtoPacketConnReadFrom(b *testing.B) {
	server, err := ProxyProtoListenPacket("udp", "127.0.0.1:0", ProxyProtoUDPConfig{})
	require.NoError(b, err)
	defer server.Close()

	client, err := net.DialUDP("udp", nil, server.LocalAddr().(*net.UDPAddr))
	require.NoError(b, err)
	defer client.Close()

	datagram := buildProxyV2HeaderWithTLVs(proxyV2CmdProxy, proxyV2FamilyUDPv4,
		net.ParseIP("203.0.113.7").To4(),
		net.ParseIP("198.51.100.9").To4(),
		40000, 53,
		awsVPCEndpointTLV("vpce-08d2bf15fac5001c9"))
	datagram = append(datagram, []byte("payload")...)

	buf := make([]byte, 1500)

	for b.Loop() {
		if _, err := client.Write(datagram); err != nil {
			b.Fatal(err)
		}

		if _, _, err := server.ReadFrom(buf); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkProxyProtoPacketConnWriteToProxy(b *testing.B) {
	raw, err := net.ListenPacket("udp", "127.0.0.1:0")
	require.NoError(b, err)
	defer raw.Close()

	server := NewProxyProtoPacketConn(raw, ProxyProtoUDPConfig{}).(*ProxyProtoPacketConn)

	backend, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1")})
	require.NoError(b, err)
	defer backend.Close()

	src := &net.UDPAddr{IP: net.ParseIP("203.0.113.7"), Port: 40000}
	dst := backend.LocalAddr()
	payload := []byte("payload")
	buf := make([]byte, 1500)

	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			if _, _, err := backend.ReadFrom(buf); err != nil {
				return
			}
		}
	}()

	for b.Loop() {
		if _, err := server.WriteToProxy(payload, src, dst); err != nil {
			b.Fatal(err)
		}
	}

	backend.Close()
	<-done
}
