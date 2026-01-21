package shamir

import (
	"encoding/base64"
	"encoding/binary"
	"errors"
	"math/big"
)

// Share represents a single share of a secret.
type Share struct {
	// X is the x-coordinate (share index), must be non-zero.
	X *big.Int
	// Y is the y-coordinate (share value).
	Y *big.Int
	// Threshold is the minimum number of shares required for reconstruction.
	Threshold int
	// Total is the total number of shares created.
	Total int
}

// shareHeader is the binary format header.
const (
	shareVersion    = 1
	shareHeaderSize = 1 + 2 + 2 + 2 + 2 // version + threshold + total + xLen + yLen
)

// Bytes serializes the share to a binary format.
// Format: version(1) | threshold(2) | total(2) | xLen(2) | yLen(2) | x | y
func (s *Share) Bytes() []byte {
	xBytes := s.X.Bytes()
	yBytes := s.Y.Bytes()

	buf := make([]byte, shareHeaderSize+len(xBytes)+len(yBytes))

	buf[0] = shareVersion
	binary.BigEndian.PutUint16(buf[1:3], uint16(s.Threshold))
	binary.BigEndian.PutUint16(buf[3:5], uint16(s.Total))
	binary.BigEndian.PutUint16(buf[5:7], uint16(len(xBytes)))
	binary.BigEndian.PutUint16(buf[7:9], uint16(len(yBytes)))

	copy(buf[shareHeaderSize:], xBytes)
	copy(buf[shareHeaderSize+len(xBytes):], yBytes)

	return buf
}

// String returns the share as a base64-encoded string.
func (s *Share) String() string {
	return base64.StdEncoding.EncodeToString(s.Bytes())
}

// ParseShare deserializes a share from binary format.
func ParseShare(data []byte) (*Share, error) {
	if len(data) < shareHeaderSize {
		return nil, ErrInvalidShareFormat
	}

	version := data[0]
	if version != shareVersion {
		return nil, ErrUnsupportedVersion
	}

	threshold := int(binary.BigEndian.Uint16(data[1:3]))
	total := int(binary.BigEndian.Uint16(data[3:5]))
	xLen := int(binary.BigEndian.Uint16(data[5:7]))
	yLen := int(binary.BigEndian.Uint16(data[7:9]))

	if len(data) != shareHeaderSize+xLen+yLen {
		return nil, ErrInvalidShareFormat
	}

	x := new(big.Int).SetBytes(data[shareHeaderSize : shareHeaderSize+xLen])
	y := new(big.Int).SetBytes(data[shareHeaderSize+xLen:])

	if x.Sign() == 0 {
		return nil, ErrInvalidShareX
	}

	return &Share{
		X:         x,
		Y:         y,
		Threshold: threshold,
		Total:     total,
	}, nil
}

// ParseShareString deserializes a share from a base64-encoded string.
func ParseShareString(s string) (*Share, error) {
	data, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return nil, errors.Join(ErrInvalidShareFormat, err)
	}
	return ParseShare(data)
}

// Clone creates a deep copy of the share.
func (s *Share) Clone() *Share {
	return &Share{
		X:         new(big.Int).Set(s.X),
		Y:         new(big.Int).Set(s.Y),
		Threshold: s.Threshold,
		Total:     s.Total,
	}
}

// Equal checks if two shares are equal.
func (s *Share) Equal(other *Share) bool {
	if s == nil || other == nil {
		return s == other
	}
	return s.X.Cmp(other.X) == 0 &&
		s.Y.Cmp(other.Y) == 0 &&
		s.Threshold == other.Threshold &&
		s.Total == other.Total
}
