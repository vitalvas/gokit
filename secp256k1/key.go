package secp256k1

import (
	"crypto"
	"crypto/rand"
	"io"
	"math/big"
)

// SEC1 point encoding lengths and prefixes.
const (
	// pubKeyUncompressedLen is 0x04 || X(32) || Y(32).
	pubKeyUncompressedLen = 1 + 2*coordinateLen
	// pubKeyCompressedLen is 0x02|0x03 || X(32).
	pubKeyCompressedLen = 1 + coordinateLen

	prefixUncompressed   = 0x04
	prefixCompressedEven = 0x02
	prefixCompressedOdd  = 0x03
)

// PublicKey is an secp256k1 public key in affine coordinates.
type PublicKey struct {
	X *big.Int
	Y *big.Int
}

// PrivateKey is an secp256k1 private key with its derived public key.
type PrivateKey struct {
	D   *big.Int
	Pub PublicKey
}

// ParsePubKey decodes a SEC1 public key in either uncompressed form
// (0x04 || X || Y) or compressed form (0x02|0x03 || X) and verifies the point
// lies on the curve.
func ParsePubKey(serialized []byte) (*PublicKey, error) {
	if len(serialized) == 0 {
		return nil, errBadKey("empty public key")
	}

	switch serialized[0] {
	case prefixUncompressed:
		return parseUncompressed(serialized)
	case prefixCompressedEven, prefixCompressedOdd:
		return parseCompressed(serialized)
	default:
		return nil, errBadKey("unsupported public key prefix 0x%02x", serialized[0])
	}
}

func parseUncompressed(serialized []byte) (*PublicKey, error) {
	if len(serialized) != pubKeyUncompressedLen {
		return nil, errBadKey("uncompressed public key must be %d bytes, got %d", pubKeyUncompressedLen, len(serialized))
	}

	x := new(big.Int).SetBytes(serialized[1 : 1+coordinateLen])
	y := new(big.Int).SetBytes(serialized[1+coordinateLen:])

	if !isOnCurve(x, y) {
		return nil, errBadKey("public key point is not on the curve")
	}

	return &PublicKey{X: x, Y: y}, nil
}

func parseCompressed(serialized []byte) (*PublicKey, error) {
	if len(serialized) != pubKeyCompressedLen {
		return nil, errBadKey("compressed public key must be %d bytes, got %d", pubKeyCompressedLen, len(serialized))
	}

	x := new(big.Int).SetBytes(serialized[1:])

	y, err := decompressY(x, serialized[0] == prefixCompressedOdd)
	if err != nil {
		return nil, err
	}

	return &PublicKey{X: x, Y: y}, nil
}

// NewPublicKey builds a public key from raw 32-byte big-endian coordinates and
// validates that the point is on the curve.
func NewPublicKey(xBytes, yBytes []byte) (*PublicKey, error) {
	if len(xBytes) > coordinateLen || len(yBytes) > coordinateLen {
		return nil, errBadKey("coordinate longer than %d bytes", coordinateLen)
	}

	x := new(big.Int).SetBytes(xBytes)
	y := new(big.Int).SetBytes(yBytes)

	if !isOnCurve(x, y) {
		return nil, errBadKey("public key point is not on the curve")
	}

	return &PublicKey{X: x, Y: y}, nil
}

// IsValid reports whether the point lies on the curve and within the field. It
// is useful for public keys built directly as a struct literal rather than via
// ParsePubKey or NewPublicKey, which validate on construction.
func (p *PublicKey) IsValid() bool {
	return p != nil && p.X != nil && p.Y != nil && isOnCurve(p.X, p.Y)
}

// Equal reports whether p and other represent the same public key. It satisfies
// the informal crypto.PublicKey Equal contract (as implemented by the stdlib
// key types), accepting the concrete *PublicKey type.
func (p *PublicKey) Equal(other crypto.PublicKey) bool {
	o, ok := other.(*PublicKey)
	if !ok || p == nil || o == nil || p.X == nil || o.X == nil {
		return false
	}

	return p.X.Cmp(o.X) == 0 && p.Y.Cmp(o.Y) == 0
}

// Equal reports whether p and other are the same private key.
func (p *PrivateKey) Equal(other crypto.PrivateKey) bool {
	o, ok := other.(*PrivateKey)
	if !ok || p == nil || o == nil || p.D == nil || o.D == nil {
		return false
	}

	return p.D.Cmp(o.D) == 0
}

// SerializeUncompressed returns the uncompressed SEC1 encoding 0x04 || X || Y.
func (p *PublicKey) SerializeUncompressed() []byte {
	out := make([]byte, pubKeyUncompressedLen)
	out[0] = prefixUncompressed
	copy(out[1:1+coordinateLen], fixedBytes(p.X))
	copy(out[1+coordinateLen:], fixedBytes(p.Y))

	return out
}

// SerializeCompressed returns the compressed SEC1 encoding 0x02|0x03 || X,
// where the prefix encodes the parity of Y.
func (p *PublicKey) SerializeCompressed() []byte {
	out := make([]byte, pubKeyCompressedLen)
	if p.Y.Bit(0) == 1 {
		out[0] = prefixCompressedOdd
	} else {
		out[0] = prefixCompressedEven
	}

	copy(out[1:], fixedBytes(p.X))

	return out
}

// PrivKeyFromBytes builds a private key from a big-endian scalar and derives the
// public key. The scalar must be in [1, N-1].
func PrivKeyFromBytes(d []byte) (*PrivateKey, error) {
	k := new(big.Int).SetBytes(d)
	if k.Sign() == 0 || k.Cmp(orderN) >= 0 {
		return nil, errBadKey("private key scalar out of range [1, N-1]")
	}

	x, y := scalarBaseMult(k)

	return &PrivateKey{D: k, Pub: PublicKey{X: x, Y: y}}, nil
}

// Serialize returns the private key as a fixed 32-byte big-endian scalar.
func (p *PrivateKey) Serialize() []byte {
	return fixedBytes(p.D)
}

// PubKey returns the public key derived from the private key.
func (p *PrivateKey) PubKey() *PublicKey {
	return &p.Pub
}

// GeneratePrivateKey creates a private key using crypto/rand.
func GeneratePrivateKey() (*PrivateKey, error) {
	return GeneratePrivateKeyFromRand(rand.Reader)
}

// GeneratePrivateKeyFromRand creates a private key, drawing the scalar from the
// given reader by rejection sampling into [1, N-1].
func GeneratePrivateKeyFromRand(r io.Reader) (*PrivateKey, error) {
	buf := make([]byte, coordinateLen)

	for {
		if _, err := io.ReadFull(r, buf); err != nil {
			return nil, err
		}

		k := new(big.Int).SetBytes(buf)
		if k.Sign() != 0 && k.Cmp(orderN) < 0 {
			x, y := scalarBaseMult(k)

			return &PrivateKey{D: k, Pub: PublicKey{X: x, Y: y}}, nil
		}
	}
}
