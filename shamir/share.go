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
	// SecretLen is the original secret length in bytes. It lets CombineAuto
	// reconstruct the exact secret, including any leading zero bytes that the
	// field-element representation would otherwise drop. Zero means unknown
	// (shares created before this field existed, or internal chunk shares).
	SecretLen int
}

// Binary format versions and header sizes.
const (
	// shareVersion1 lacked the secretLen field.
	shareVersion1    = 1
	shareHeaderSize1 = 1 + 2 + 2 + 2 + 2 // version + threshold + total + xLen + yLen
	shareVersion     = 2
	shareHeaderSize  = 1 + 2 + 2 + 2 + 2 + 2 // + secretLen
)

// Bytes serializes the share to a binary format.
// Format: version(1) | threshold(2) | total(2) | secretLen(2) | xLen(2) | yLen(2) | x | y
func (s *Share) Bytes() []byte {
	xBytes := s.X.Bytes()
	yBytes := s.Y.Bytes()

	buf := make([]byte, shareHeaderSize+len(xBytes)+len(yBytes))

	buf[0] = shareVersion
	binary.BigEndian.PutUint16(buf[1:3], uint16(s.Threshold))
	binary.BigEndian.PutUint16(buf[3:5], uint16(s.Total))
	binary.BigEndian.PutUint16(buf[5:7], uint16(s.SecretLen))
	binary.BigEndian.PutUint16(buf[7:9], uint16(len(xBytes)))
	binary.BigEndian.PutUint16(buf[9:11], uint16(len(yBytes)))

	copy(buf[shareHeaderSize:], xBytes)
	copy(buf[shareHeaderSize+len(xBytes):], yBytes)

	return buf
}

// String returns the share as a base64-encoded string.
func (s *Share) String() string {
	return base64.StdEncoding.EncodeToString(s.Bytes())
}

// ParseShare deserializes a share from binary format. Both the current format
// and the legacy version-1 format (without secretLen) are accepted; legacy
// shares parse with SecretLen == 0.
func ParseShare(data []byte) (*Share, error) {
	if len(data) < 1 {
		return nil, ErrInvalidShareFormat
	}

	switch data[0] {
	case shareVersion1:
		return parseShareV1(data)
	case shareVersion:
		return parseShareV2(data)
	default:
		return nil, ErrUnsupportedVersion
	}
}

func parseShareV1(data []byte) (*Share, error) {
	if len(data) < shareHeaderSize1 {
		return nil, ErrInvalidShareFormat
	}

	threshold := int(binary.BigEndian.Uint16(data[1:3]))
	total := int(binary.BigEndian.Uint16(data[3:5]))
	xLen := int(binary.BigEndian.Uint16(data[5:7]))
	yLen := int(binary.BigEndian.Uint16(data[7:9]))

	if len(data) != shareHeaderSize1+xLen+yLen {
		return nil, ErrInvalidShareFormat
	}

	return buildShare(data[shareHeaderSize1:shareHeaderSize1+xLen], data[shareHeaderSize1+xLen:], threshold, total, 0)
}

func parseShareV2(data []byte) (*Share, error) {
	if len(data) < shareHeaderSize {
		return nil, ErrInvalidShareFormat
	}

	threshold := int(binary.BigEndian.Uint16(data[1:3]))
	total := int(binary.BigEndian.Uint16(data[3:5]))
	secretLen := int(binary.BigEndian.Uint16(data[5:7]))
	xLen := int(binary.BigEndian.Uint16(data[7:9]))
	yLen := int(binary.BigEndian.Uint16(data[9:11]))

	if len(data) != shareHeaderSize+xLen+yLen {
		return nil, ErrInvalidShareFormat
	}

	return buildShare(data[shareHeaderSize:shareHeaderSize+xLen], data[shareHeaderSize+xLen:], threshold, total, secretLen)
}

func buildShare(xBytes, yBytes []byte, threshold, total, secretLen int) (*Share, error) {
	x := new(big.Int).SetBytes(xBytes)
	y := new(big.Int).SetBytes(yBytes)

	if x.Sign() == 0 {
		return nil, ErrInvalidShareX
	}

	return &Share{
		X:         x,
		Y:         y,
		Threshold: threshold,
		Total:     total,
		SecretLen: secretLen,
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
		SecretLen: s.SecretLen,
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
		s.Total == other.Total &&
		s.SecretLen == other.SecretLen
}
