package xnet

import (
	"io"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProxyProtoListener(t *testing.T) {
	t.Run("accept with v1 header", func(t *testing.T) {
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		require.NoError(t, err)
		defer listener.Close()

		proxyListener := NewProxyProtoListener(listener, ProxyProtoConfig{
			HeaderTimeout: 5 * time.Second,
		})

		go func() {
			conn, err := net.Dial("tcp", listener.Addr().String())
			require.NoError(t, err)
			defer conn.Close()

			_, err = conn.Write([]byte("PROXY TCP4 10.0.0.1 10.0.0.2 54321 8080\r\n"))
			require.NoError(t, err)

			_, err = conn.Write([]byte("hello"))
			require.NoError(t, err)
		}()

		conn, err := proxyListener.Accept()
		require.NoError(t, err)
		defer conn.Close()

		assert.Equal(t, "10.0.0.1:54321", conn.RemoteAddr().String())
		assert.Equal(t, "10.0.0.2:8080", conn.LocalAddr().String())

		buf := make([]byte, 5)
		_, err = io.ReadFull(conn, buf)
		require.NoError(t, err)
		assert.Equal(t, "hello", string(buf))
	})

	t.Run("accept with v2 header", func(t *testing.T) {
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		require.NoError(t, err)
		defer listener.Close()

		proxyListener := NewProxyProtoListener(listener, ProxyProtoConfig{
			HeaderTimeout: 5 * time.Second,
		})

		go func() {
			conn, err := net.Dial("tcp", listener.Addr().String())
			require.NoError(t, err)
			defer conn.Close()

			header := buildProxyV2Header(proxyV2CmdProxy, proxyV2FamilyTCPv4,
				net.ParseIP("172.16.0.1").To4(),
				net.ParseIP("172.16.0.2").To4(),
				11111, 9090)
			_, err = conn.Write(header)
			require.NoError(t, err)

			_, err = conn.Write([]byte("world"))
			require.NoError(t, err)
		}()

		conn, err := proxyListener.Accept()
		require.NoError(t, err)
		defer conn.Close()

		assert.Equal(t, "172.16.0.1:11111", conn.RemoteAddr().String())
		assert.Equal(t, "172.16.0.2:9090", conn.LocalAddr().String())

		buf := make([]byte, 5)
		_, err = io.ReadFull(conn, buf)
		require.NoError(t, err)
		assert.Equal(t, "world", string(buf))
	})

	t.Run("reject untrusted proxy", func(t *testing.T) {
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		require.NoError(t, err)
		defer listener.Close()

		_, trustedNet, _ := net.ParseCIDR("10.0.0.0/8")
		proxyListener := NewProxyProtoListener(listener, ProxyProtoConfig{
			HeaderTimeout:  5 * time.Second,
			TrustedProxies: []net.IPNet{*trustedNet},
		})

		go func() {
			conn, err := net.Dial("tcp", listener.Addr().String())
			if err != nil {
				return
			}
			defer conn.Close()
			time.Sleep(100 * time.Millisecond)
		}()

		_, err = proxyListener.Accept()
		assert.ErrorIs(t, err, ErrProxyProtoUntrusted)
	})

	t.Run("reject invalid header", func(t *testing.T) {
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		require.NoError(t, err)
		defer listener.Close()

		proxyListener := NewProxyProtoListener(listener, ProxyProtoConfig{
			HeaderTimeout: 5 * time.Second,
		})

		go func() {
			conn, err := net.Dial("tcp", listener.Addr().String())
			require.NoError(t, err)
			defer conn.Close()

			_, err = conn.Write([]byte("GET / HTTP/1.1\r\n"))
			require.NoError(t, err)
		}()

		_, err = proxyListener.Accept()
		assert.Error(t, err)
	})

	t.Run("close listener", func(t *testing.T) {
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		require.NoError(t, err)

		proxyListener := NewProxyProtoListener(listener, ProxyProtoConfig{})

		addr := proxyListener.Addr()
		assert.NotNil(t, addr)

		err = proxyListener.Close()
		assert.NoError(t, err)
	})
}

func TestProxyProtoConn(t *testing.T) {
	t.Run("real remote addr", func(t *testing.T) {
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		require.NoError(t, err)
		defer listener.Close()

		proxyListener := NewProxyProtoListener(listener, ProxyProtoConfig{
			HeaderTimeout: 5 * time.Second,
		})

		go func() {
			conn, err := net.Dial("tcp", listener.Addr().String())
			require.NoError(t, err)
			defer conn.Close()

			_, err = conn.Write([]byte("PROXY TCP4 10.0.0.1 10.0.0.2 54321 8080\r\n"))
			require.NoError(t, err)
		}()

		conn, err := proxyListener.Accept()
		require.NoError(t, err)
		defer conn.Close()

		proxyConn, ok := conn.(*ProxyProtoConn)
		require.True(t, ok)

		assert.Equal(t, "10.0.0.1:54321", proxyConn.RemoteAddr().String())
		assert.Contains(t, proxyConn.RealRemoteAddr().String(), "127.0.0.1:")

		header := proxyConn.ProxyHeader()
		require.NotNil(t, header)
		assert.Equal(t, "10.0.0.1", header.SourceAddr.IP.String())
	})
}

func TestProxyProtoListenerAutoMode(t *testing.T) {
	t.Run("auto mode accepts regular connection", func(t *testing.T) {
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		require.NoError(t, err)
		defer listener.Close()

		proxyListener := NewProxyProtoListener(listener, ProxyProtoConfig{
			Mode:          ProxyProtoModeAuto,
			HeaderTimeout: 5 * time.Second,
		})

		go func() {
			conn, err := net.Dial("tcp", listener.Addr().String())
			require.NoError(t, err)
			defer conn.Close()

			_, err = conn.Write([]byte("hello world"))
			require.NoError(t, err)
		}()

		conn, err := proxyListener.Accept()
		require.NoError(t, err)
		defer conn.Close()

		assert.Contains(t, conn.RemoteAddr().String(), "127.0.0.1:")

		buf := make([]byte, 11)
		_, err = io.ReadFull(conn, buf)
		require.NoError(t, err)
		assert.Equal(t, "hello world", string(buf))

		proxyConn, ok := conn.(*ProxyProtoConn)
		require.True(t, ok)
		assert.Nil(t, proxyConn.ProxyHeader())
	})

	t.Run("auto mode accepts proxy header", func(t *testing.T) {
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		require.NoError(t, err)
		defer listener.Close()

		proxyListener := NewProxyProtoListener(listener, ProxyProtoConfig{
			Mode:          ProxyProtoModeAuto,
			HeaderTimeout: 5 * time.Second,
		})

		go func() {
			conn, err := net.Dial("tcp", listener.Addr().String())
			require.NoError(t, err)
			defer conn.Close()

			_, err = conn.Write([]byte("PROXY TCP4 10.0.0.1 10.0.0.2 54321 8080\r\n"))
			require.NoError(t, err)

			_, err = conn.Write([]byte("data"))
			require.NoError(t, err)
		}()

		conn, err := proxyListener.Accept()
		require.NoError(t, err)
		defer conn.Close()

		assert.Equal(t, "10.0.0.1:54321", conn.RemoteAddr().String())

		buf := make([]byte, 4)
		_, err = io.ReadFull(conn, buf)
		require.NoError(t, err)
		assert.Equal(t, "data", string(buf))
	})

	t.Run("auto mode with trusted proxies accepts untrusted as regular", func(t *testing.T) {
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		require.NoError(t, err)
		defer listener.Close()

		_, trustedNet, _ := net.ParseCIDR("10.0.0.0/8")
		proxyListener := NewProxyProtoListener(listener, ProxyProtoConfig{
			Mode:           ProxyProtoModeAuto,
			HeaderTimeout:  5 * time.Second,
			TrustedProxies: []net.IPNet{*trustedNet},
		})

		go func() {
			conn, err := net.Dial("tcp", listener.Addr().String())
			require.NoError(t, err)
			defer conn.Close()

			_, err = conn.Write([]byte("regular data"))
			require.NoError(t, err)
		}()

		conn, err := proxyListener.Accept()
		require.NoError(t, err)
		defer conn.Close()

		assert.Contains(t, conn.RemoteAddr().String(), "127.0.0.1:")

		buf := make([]byte, 12)
		_, err = io.ReadFull(conn, buf)
		require.NoError(t, err)
		assert.Equal(t, "regular data", string(buf))
	})

	t.Run("strict mode rejects regular connection", func(t *testing.T) {
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		require.NoError(t, err)
		defer listener.Close()

		proxyListener := NewProxyProtoListener(listener, ProxyProtoConfig{
			Mode:          ProxyProtoModeStrict,
			HeaderTimeout: 5 * time.Second,
		})

		go func() {
			conn, err := net.Dial("tcp", listener.Addr().String())
			require.NoError(t, err)
			defer conn.Close()

			_, err = conn.Write([]byte("GET / HTTP/1.1\r\n"))
			require.NoError(t, err)
		}()

		_, err = proxyListener.Accept()
		assert.Error(t, err)
	})
}

func TestProxyProtoListen(t *testing.T) {
	t.Run("creates listener successfully", func(t *testing.T) {
		listener, err := ProxyProtoListen("tcp", "127.0.0.1:0", ProxyProtoConfig{
			HeaderTimeout: 5 * time.Second,
		})
		require.NoError(t, err)
		defer listener.Close()

		assert.NotNil(t, listener)
		assert.NotNil(t, listener.Addr())
	})

	t.Run("returns error for invalid address", func(t *testing.T) {
		_, err := ProxyProtoListen("tcp", "invalid:address:format", ProxyProtoConfig{})
		assert.Error(t, err)
	})

	t.Run("accepts connection with v1 header", func(t *testing.T) {
		listener, err := ProxyProtoListen("tcp", "127.0.0.1:0", ProxyProtoConfig{
			Mode:          ProxyProtoModeStrict,
			HeaderTimeout: 5 * time.Second,
		})
		require.NoError(t, err)
		defer listener.Close()

		go func() {
			conn, err := net.Dial("tcp", listener.Addr().String())
			require.NoError(t, err)
			defer conn.Close()

			_, err = conn.Write([]byte("PROXY TCP4 10.0.0.1 10.0.0.2 54321 8080\r\n"))
			require.NoError(t, err)

			_, err = conn.Write([]byte("hello"))
			require.NoError(t, err)
		}()

		conn, err := listener.Accept()
		require.NoError(t, err)
		defer conn.Close()

		assert.Equal(t, "10.0.0.1:54321", conn.RemoteAddr().String())

		buf := make([]byte, 5)
		_, err = io.ReadFull(conn, buf)
		require.NoError(t, err)
		assert.Equal(t, "hello", string(buf))
	})

	t.Run("accepts connection with v2 header", func(t *testing.T) {
		listener, err := ProxyProtoListen("tcp", "127.0.0.1:0", ProxyProtoConfig{
			Mode:          ProxyProtoModeStrict,
			HeaderTimeout: 5 * time.Second,
		})
		require.NoError(t, err)
		defer listener.Close()

		go func() {
			conn, err := net.Dial("tcp", listener.Addr().String())
			require.NoError(t, err)
			defer conn.Close()

			header := buildProxyV2Header(proxyV2CmdProxy, proxyV2FamilyTCPv4,
				net.ParseIP("172.16.0.1").To4(),
				net.ParseIP("172.16.0.2").To4(),
				11111, 9090)
			_, err = conn.Write(header)
			require.NoError(t, err)

			_, err = conn.Write([]byte("world"))
			require.NoError(t, err)
		}()

		conn, err := listener.Accept()
		require.NoError(t, err)
		defer conn.Close()

		assert.Equal(t, "172.16.0.1:11111", conn.RemoteAddr().String())

		buf := make([]byte, 5)
		_, err = io.ReadFull(conn, buf)
		require.NoError(t, err)
		assert.Equal(t, "world", string(buf))
	})

	t.Run("auto mode accepts regular connection", func(t *testing.T) {
		listener, err := ProxyProtoListen("tcp", "127.0.0.1:0", ProxyProtoConfig{
			Mode:          ProxyProtoModeAuto,
			HeaderTimeout: 5 * time.Second,
		})
		require.NoError(t, err)
		defer listener.Close()

		go func() {
			conn, err := net.Dial("tcp", listener.Addr().String())
			require.NoError(t, err)
			defer conn.Close()

			_, err = conn.Write([]byte("plain data"))
			require.NoError(t, err)
		}()

		conn, err := listener.Accept()
		require.NoError(t, err)
		defer conn.Close()

		assert.Contains(t, conn.RemoteAddr().String(), "127.0.0.1:")

		buf := make([]byte, 10)
		_, err = io.ReadFull(conn, buf)
		require.NoError(t, err)
		assert.Equal(t, "plain data", string(buf))
	})

	t.Run("returns net.Listener interface", func(t *testing.T) {
		listener, err := ProxyProtoListen("tcp", "127.0.0.1:0", ProxyProtoConfig{})
		require.NoError(t, err)
		defer listener.Close()

		assert.Implements(t, (*net.Listener)(nil), listener)
	})

	t.Run("with trusted proxies in auto mode", func(t *testing.T) {
		_, trustedNet, _ := net.ParseCIDR("127.0.0.0/8")
		listener, err := ProxyProtoListen("tcp", "127.0.0.1:0", ProxyProtoConfig{
			Mode:           ProxyProtoModeAuto,
			HeaderTimeout:  5 * time.Second,
			TrustedProxies: []net.IPNet{*trustedNet},
		})
		require.NoError(t, err)
		defer listener.Close()

		go func() {
			conn, err := net.Dial("tcp", listener.Addr().String())
			require.NoError(t, err)
			defer conn.Close()

			_, err = conn.Write([]byte("PROXY TCP4 192.168.1.1 192.168.1.2 12345 80\r\n"))
			require.NoError(t, err)

			_, err = conn.Write([]byte("data"))
			require.NoError(t, err)
		}()

		conn, err := listener.Accept()
		require.NoError(t, err)
		defer conn.Close()

		assert.Equal(t, "192.168.1.1:12345", conn.RemoteAddr().String())
	})
}

func TestNewProxyProtoListenerReturnsInterface(t *testing.T) {
	t.Run("returns net.Listener interface", func(t *testing.T) {
		rawListener, err := net.Listen("tcp", "127.0.0.1:0")
		require.NoError(t, err)
		defer rawListener.Close()

		listener := NewProxyProtoListener(rawListener, ProxyProtoConfig{})

		assert.Implements(t, (*net.Listener)(nil), listener)
	})

	t.Run("works as drop-in replacement", func(t *testing.T) {
		rawListener, err := net.Listen("tcp", "127.0.0.1:0")
		require.NoError(t, err)

		listener := NewProxyProtoListener(rawListener, ProxyProtoConfig{
			Mode:          ProxyProtoModeAuto,
			HeaderTimeout: 5 * time.Second,
		})
		defer listener.Close()

		go func() {
			conn, err := net.Dial("tcp", listener.Addr().String())
			require.NoError(t, err)
			defer conn.Close()

			_, err = conn.Write([]byte("test message"))
			require.NoError(t, err)
		}()

		conn, err := listener.Accept()
		require.NoError(t, err)
		defer conn.Close()

		buf := make([]byte, 12)
		_, err = io.ReadFull(conn, buf)
		require.NoError(t, err)
		assert.Equal(t, "test message", string(buf))
	})
}

func TestProxyProtoConnInterface(t *testing.T) {
	t.Run("implements net.Conn", func(t *testing.T) {
		listener, err := ProxyProtoListen("tcp", "127.0.0.1:0", ProxyProtoConfig{
			Mode:          ProxyProtoModeAuto,
			HeaderTimeout: 5 * time.Second,
		})
		require.NoError(t, err)
		defer listener.Close()

		go func() {
			conn, err := net.Dial("tcp", listener.Addr().String())
			require.NoError(t, err)
			defer conn.Close()

			_, err = conn.Write([]byte("data"))
			require.NoError(t, err)
		}()

		conn, err := listener.Accept()
		require.NoError(t, err)
		defer conn.Close()

		assert.Implements(t, (*net.Conn)(nil), conn)
	})

	t.Run("SetDeadline works", func(t *testing.T) {
		listener, err := ProxyProtoListen("tcp", "127.0.0.1:0", ProxyProtoConfig{
			Mode:          ProxyProtoModeAuto,
			HeaderTimeout: 5 * time.Second,
		})
		require.NoError(t, err)
		defer listener.Close()

		go func() {
			conn, err := net.Dial("tcp", listener.Addr().String())
			require.NoError(t, err)
			defer conn.Close()
			time.Sleep(100 * time.Millisecond)
		}()

		conn, err := listener.Accept()
		require.NoError(t, err)
		defer conn.Close()

		err = conn.SetDeadline(time.Now().Add(time.Second))
		assert.NoError(t, err)

		err = conn.SetReadDeadline(time.Now().Add(time.Second))
		assert.NoError(t, err)

		err = conn.SetWriteDeadline(time.Now().Add(time.Second))
		assert.NoError(t, err)
	})

	t.Run("Write works", func(t *testing.T) {
		listener, err := ProxyProtoListen("tcp", "127.0.0.1:0", ProxyProtoConfig{
			Mode:          ProxyProtoModeAuto,
			HeaderTimeout: 5 * time.Second,
		})
		require.NoError(t, err)
		defer listener.Close()

		done := make(chan struct{})
		go func() {
			defer close(done)
			conn, err := net.Dial("tcp", listener.Addr().String())
			require.NoError(t, err)
			defer conn.Close()

			buf := make([]byte, 5)
			_, err = io.ReadFull(conn, buf)
			require.NoError(t, err)
			assert.Equal(t, "hello", string(buf))
		}()

		conn, err := listener.Accept()
		require.NoError(t, err)
		defer conn.Close()

		n, err := conn.Write([]byte("hello"))
		assert.NoError(t, err)
		assert.Equal(t, 5, n)

		<-done
	})

	t.Run("Network and addresses", func(t *testing.T) {
		listener, err := ProxyProtoListen("tcp", "127.0.0.1:0", ProxyProtoConfig{
			Mode:          ProxyProtoModeStrict,
			HeaderTimeout: 5 * time.Second,
		})
		require.NoError(t, err)
		defer listener.Close()

		go func() {
			conn, err := net.Dial("tcp", listener.Addr().String())
			require.NoError(t, err)
			defer conn.Close()

			_, err = conn.Write([]byte("PROXY TCP4 10.0.0.1 10.0.0.2 54321 8080\r\n"))
			require.NoError(t, err)
		}()

		conn, err := listener.Accept()
		require.NoError(t, err)
		defer conn.Close()

		assert.Equal(t, "tcp", conn.RemoteAddr().Network())
		assert.Equal(t, "tcp", conn.LocalAddr().Network())
		assert.Equal(t, "10.0.0.1:54321", conn.RemoteAddr().String())
		assert.Equal(t, "10.0.0.2:8080", conn.LocalAddr().String())
	})
}
