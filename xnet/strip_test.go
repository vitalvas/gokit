package xnet

import (
	"net"
	"testing"
)

func TestGetStripedAddress(t *testing.T) {
	tests := []struct {
		name    string
		addr    net.IP
		ipv4    int
		ipv6    int
		want    net.IP
		wantErr error
	}{
		{
			name:    "Valid IPv4 Address with valid mask",
			addr:    net.ParseIP("192.168.1.10"),
			ipv4:    24,
			want:    net.ParseIP("192.168.1.0"),
			wantErr: nil,
		},
		{
			name:    "Valid IPv6 Address with valid mask",
			addr:    net.ParseIP("2001:db8::1"),
			ipv6:    64,
			want:    net.ParseIP("2001:db8::"),
			wantErr: nil,
		},
		{
			name:    "Valid IPv4 Address with /18 mask",
			addr:    net.ParseIP("192.168.99.22"),
			ipv4:    18,
			want:    net.ParseIP("192.168.64.0"),
			wantErr: nil,
		},
		{
			name:    "Valid IPv6 Address with /56 mask",
			addr:    net.ParseIP("2001:db8:1234:5678:90ab:cdef:0000:0001"),
			ipv6:    56,
			want:    net.ParseIP("2001:db8:1234:5600::"),
			wantErr: nil,
		},
		{
			name:    "Nil IP Address",
			addr:    nil,
			want:    nil,
			wantErr: ErrInvalidIPAddress,
		},
		{
			name:    "Invalid IPv4 Mask Size (negative)",
			addr:    net.ParseIP("192.168.1.10"),
			ipv4:    -1,
			want:    nil,
			wantErr: ErrInvalidIPv4MaskSize,
		},
		{
			name:    "Invalid IPv4 Mask Size (too large)",
			addr:    net.ParseIP("192.168.1.10"),
			ipv4:    33,
			want:    nil,
			wantErr: ErrInvalidIPv4MaskSize,
		},
		{
			name:    "Invalid IPv6 Mask Size (negative)",
			addr:    net.ParseIP("2001:db8::1"),
			ipv6:    -1,
			want:    nil,
			wantErr: ErrInvalidIPv6MaskSize,
		},
		{
			name:    "Invalid IPv6 Mask Size (too large)",
			addr:    net.ParseIP("2001:db8::1"),
			ipv6:    129,
			want:    nil,
			wantErr: ErrInvalidIPv6MaskSize,
		},
		{
			name:    "Both IPv4 and IPv6 Mask Sizes Are Zero",
			addr:    net.ParseIP("192.168.1.10"),
			ipv4:    0,
			ipv6:    0,
			want:    nil,
			wantErr: ErrInvalidIPMaskSize,
		},
		{
			name:    "Invalid IP Address",
			addr:    net.IP{0xff, 0x00, 0x00, 0x00, 0xff},
			ipv4:    24,
			want:    nil,
			wantErr: ErrNonStandardIP,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetStripedAddress(tt.addr, tt.ipv4, tt.ipv6)

			if err != nil && tt.wantErr == nil || err == nil && tt.wantErr != nil || err != nil && err.Error() != tt.wantErr.Error() {
				t.Errorf("GetStripedAddress() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !got.Equal(tt.want) && (got != nil || tt.want != nil) {
				t.Errorf("GetStripedAddress() = %v, want %v", got, tt.want)
			}
		})
	}
}

func BenchmarkGetStripedAddress(b *testing.B) {
	b.Run("IPv4", func(b *testing.B) {
		addr := net.ParseIP("192.168.99.22")

		for b.Loop() {
			_, _ = GetStripedAddress(addr, 24, 0)
		}
	})

	b.Run("IPv6", func(b *testing.B) {
		addr := net.ParseIP("2001:db8:1234:5678:90ab:cdef:0000:0001")

		for b.Loop() {
			_, _ = GetStripedAddress(addr, 0, 64)
		}
	})
}

func FuzzGetStripedAddress(f *testing.F) {
	f.Add([]byte{192, 168, 1, 10}, 24, 64)
	f.Add([]byte{10, 0, 0, 1}, 18, 56)
	f.Add([]byte{0, 0, 0, 0}, 32, 128)
	f.Add([]byte{255, 255, 255, 255}, 8, 48)
	f.Add([]byte{0x20, 0x01, 0x0d, 0xb8, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01}, 24, 64)

	f.Fuzz(func(_ *testing.T, ipBytes []byte, ipv4Mask, ipv6Mask int) {
		var addr net.IP
		if len(ipBytes) == 4 || len(ipBytes) == 16 {
			addr = net.IP(ipBytes)
		}
		_, _ = GetStripedAddress(addr, ipv4Mask, ipv6Mask)
	})
}
