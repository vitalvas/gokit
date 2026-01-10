package xnet

import (
	"bufio"
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsProxyV1(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		expected bool
	}{
		{
			name:     "valid v1 prefix",
			data:     []byte("PROXY TCP4 192.168.1.1"),
			expected: true,
		},
		{
			name:     "invalid prefix",
			data:     []byte("HTTP/1.1 200 OK"),
			expected: false,
		},
		{
			name:     "empty data",
			data:     []byte{},
			expected: false,
		},
		{
			name:     "v2 signature",
			data:     proxyV2Signature,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isProxyV1(tt.data)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseProxyV1(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantErr    error
		wantSrcIP  string
		wantSrcP   int
		wantDstIP  string
		wantDstP   int
		wantNilHdr bool
	}{
		{
			name:      "valid TCP4",
			input:     "PROXY TCP4 192.168.1.1 192.168.1.2 12345 80\r\n",
			wantSrcIP: "192.168.1.1",
			wantSrcP:  12345,
			wantDstIP: "192.168.1.2",
			wantDstP:  80,
		},
		{
			name:      "valid TCP6",
			input:     "PROXY TCP6 2001:db8::1 2001:db8::2 12345 443\r\n",
			wantSrcIP: "2001:db8::1",
			wantSrcP:  12345,
			wantDstIP: "2001:db8::2",
			wantDstP:  443,
		},
		{
			name:       "unknown protocol",
			input:      "PROXY UNKNOWN\r\n",
			wantNilHdr: true,
		},
		{
			name:    "missing CRLF",
			input:   "PROXY TCP4 192.168.1.1 192.168.1.2 12345 80\n",
			wantErr: ErrProxyProtoInvalid,
		},
		{
			name:    "invalid protocol",
			input:   "PROXY UDP4 192.168.1.1 192.168.1.2 12345 80\r\n",
			wantErr: ErrProxyProtoInvalid,
		},
		{
			name:    "invalid source IP",
			input:   "PROXY TCP4 invalid 192.168.1.2 12345 80\r\n",
			wantErr: ErrProxyProtoInvalid,
		},
		{
			name:    "invalid destination IP",
			input:   "PROXY TCP4 192.168.1.1 invalid 12345 80\r\n",
			wantErr: ErrProxyProtoInvalid,
		},
		{
			name:    "invalid source port",
			input:   "PROXY TCP4 192.168.1.1 192.168.1.2 invalid 80\r\n",
			wantErr: ErrProxyProtoInvalid,
		},
		{
			name:    "invalid destination port",
			input:   "PROXY TCP4 192.168.1.1 192.168.1.2 12345 invalid\r\n",
			wantErr: ErrProxyProtoInvalid,
		},
		{
			name:    "port out of range",
			input:   "PROXY TCP4 192.168.1.1 192.168.1.2 12345 99999\r\n",
			wantErr: ErrProxyProtoInvalid,
		},
		{
			name:    "TCP4 with IPv6 addresses",
			input:   "PROXY TCP4 2001:db8::1 2001:db8::2 12345 80\r\n",
			wantErr: ErrProxyProtoInvalid,
		},
		{
			name:    "TCP6 with IPv4 addresses",
			input:   "PROXY TCP6 192.168.1.1 192.168.1.2 12345 80\r\n",
			wantErr: ErrProxyProtoInvalid,
		},
		{
			name:    "missing parts",
			input:   "PROXY TCP4 192.168.1.1\r\n",
			wantErr: ErrProxyProtoInvalid,
		},
		{
			name:    "empty line",
			input:   "\r\n",
			wantErr: ErrProxyProtoInvalid,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := bufio.NewReader(bytes.NewReader([]byte(tt.input)))
			header, err := parseProxyV1(reader)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, header)

			if tt.wantNilHdr {
				assert.Nil(t, header.SourceAddr)
				assert.Nil(t, header.DestAddr)
				return
			}

			require.NotNil(t, header.SourceAddr)
			require.NotNil(t, header.DestAddr)
			assert.Equal(t, tt.wantSrcIP, header.SourceAddr.IP.String())
			assert.Equal(t, tt.wantSrcP, header.SourceAddr.Port)
			assert.Equal(t, tt.wantDstIP, header.DestAddr.IP.String())
			assert.Equal(t, tt.wantDstP, header.DestAddr.Port)
		})
	}
}

func BenchmarkIsProxyV1(b *testing.B) {
	data := []byte("PROXY TCP4 192.168.1.1 192.168.1.2 12345 80\r\n")

	for b.Loop() {
		isProxyV1(data)
	}
}

func BenchmarkParseProxyV1(b *testing.B) {
	b.Run("TCP4", func(b *testing.B) {
		data := []byte("PROXY TCP4 192.168.1.1 192.168.1.2 12345 80\r\n")

		for b.Loop() {
			reader := bufio.NewReader(bytes.NewReader(data))
			_, _ = parseProxyV1(reader)
		}
	})

	b.Run("TCP6", func(b *testing.B) {
		data := []byte("PROXY TCP6 2001:db8::1 2001:db8::2 12345 443\r\n")

		for b.Loop() {
			reader := bufio.NewReader(bytes.NewReader(data))
			_, _ = parseProxyV1(reader)
		}
	})

	b.Run("UNKNOWN", func(b *testing.B) {
		data := []byte("PROXY UNKNOWN\r\n")

		for b.Loop() {
			reader := bufio.NewReader(bytes.NewReader(data))
			_, _ = parseProxyV1(reader)
		}
	})
}

func FuzzParseProxyV1(f *testing.F) {
	f.Add([]byte("PROXY TCP4 192.168.1.1 192.168.1.2 12345 80\r\n"))
	f.Add([]byte("PROXY TCP6 2001:db8::1 2001:db8::2 12345 443\r\n"))
	f.Add([]byte("PROXY UNKNOWN\r\n"))
	f.Add([]byte("PROXY TCP4 10.0.0.1 10.0.0.2 1 65535\r\n"))
	f.Add([]byte("PROXY TCP4 0.0.0.0 255.255.255.255 0 0\r\n"))
	f.Add([]byte(""))
	f.Add([]byte("PROXY"))
	f.Add([]byte("PROXY "))
	f.Add([]byte("PROXY TCP4"))
	f.Add([]byte("GET / HTTP/1.1\r\n"))

	f.Fuzz(func(_ *testing.T, data []byte) {
		reader := bufio.NewReader(bytes.NewReader(data))
		_, _ = parseProxyV1(reader)
	})
}
