package shamir

import (
	"encoding/base64"
	"encoding/binary"
	"errors"
	"math/big"
)

const (
	// maxChunkSize is the maximum size of each chunk (31 bytes to fit in field)
	maxChunkSize = 31

	chunkedShareVersion    = 2
	chunkedShareHeaderSize = 1 + 2 + 2 + 4 + 2 + 2 // version + threshold + total + secretLen + numChunks + xLen
)

// ChunkedShare represents a share of a secret that may be larger than the field size.
// It contains multiple Y values, one for each chunk of the original secret.
type ChunkedShare struct {
	// X is the x-coordinate (share index), must be non-zero.
	X *big.Int
	// Ys contains the y-coordinates for each chunk.
	Ys []*big.Int
	// Threshold is the minimum number of shares required for reconstruction.
	Threshold int
	// Total is the total number of shares created.
	Total int
	// SecretLen is the original secret length in bytes.
	SecretLen int
}

// SplitBytes divides a secret of any size into shares.
// The secret is split into 31-byte chunks, each processed independently.
//
// Parameters:
//   - secret: the secret data to split (any length)
//   - threshold: minimum number of shares required for reconstruction (k)
//   - total: total number of shares to generate (n)
func SplitBytes(secret []byte, threshold, total int) ([]*ChunkedShare, error) {
	if len(secret) == 0 {
		return nil, ErrEmptySecret
	}

	if threshold < 2 {
		return nil, ErrInvalidThreshold
	}

	if total < threshold {
		return nil, ErrInvalidTotal
	}

	// Split secret into chunks
	chunks := splitIntoChunks(secret, maxChunkSize)

	// Create shares for each chunk
	chunkShares := make([][]*Share, len(chunks))
	for i, chunk := range chunks {
		shares, err := Split(chunk, threshold, total)
		if err != nil {
			return nil, err
		}
		chunkShares[i] = shares
	}

	// Combine into chunked shares (one per participant)
	result := make([]*ChunkedShare, total)
	for i := range total {
		ys := make([]*big.Int, len(chunks))
		for j := range chunks {
			ys[j] = chunkShares[j][i].Y
		}

		result[i] = &ChunkedShare{
			X:         chunkShares[0][i].X,
			Ys:        ys,
			Threshold: threshold,
			Total:     total,
			SecretLen: len(secret),
		}
	}

	return result, nil
}

// CombineBytes reconstructs the secret from chunked shares.
func CombineBytes(shares []*ChunkedShare) ([]byte, error) {
	if len(shares) == 0 {
		return nil, ErrInsufficientShares
	}

	threshold := shares[0].Threshold
	if len(shares) < threshold {
		return nil, ErrInsufficientShares
	}

	numChunks := len(shares[0].Ys)
	secretLen := shares[0].SecretLen

	// Verify shares have consistent parameters
	seen := make(map[string]bool)
	for _, share := range shares {
		if share.Threshold != threshold {
			return nil, ErrInconsistentShares
		}
		if len(share.Ys) != numChunks {
			return nil, ErrInconsistentShares
		}
		if share.SecretLen != secretLen {
			return nil, ErrInconsistentShares
		}
		key := share.X.String()
		if seen[key] {
			return nil, ErrDuplicateShares
		}
		seen[key] = true
	}

	// Use only the required number of shares
	usedShares := shares[:threshold]

	// Reconstruct each chunk
	result := make([]byte, 0, secretLen)

	for chunkIdx := range numChunks {
		// Build regular shares for this chunk
		chunkShareList := make([]*Share, threshold)
		for i, share := range usedShares {
			chunkShareList[i] = &Share{
				X:         share.X,
				Y:         share.Ys[chunkIdx],
				Threshold: threshold,
				Total:     share.Total,
			}
		}

		// Determine chunk size
		chunkSize := maxChunkSize
		remaining := secretLen - len(result)
		if remaining < maxChunkSize {
			chunkSize = remaining
		}

		// Combine this chunk
		chunk, err := Combine(chunkShareList, chunkSize)
		if err != nil {
			return nil, err
		}

		result = append(result, chunk...)
	}

	return result, nil
}

// Bytes serializes the chunked share to a binary format.
// Format: version(1) | threshold(2) | total(2) | secretLen(4) | numChunks(2) | xLen(2) | x | [yLen(2) | y]...
func (s *ChunkedShare) Bytes() []byte {
	xBytes := s.X.Bytes()

	// Calculate total size
	size := chunkedShareHeaderSize + len(xBytes)
	yBytesSlice := make([][]byte, len(s.Ys))
	for i, y := range s.Ys {
		yBytesSlice[i] = y.Bytes()
		size += 2 + len(yBytesSlice[i]) // yLen + y
	}

	buf := make([]byte, size)
	offset := 0

	buf[offset] = chunkedShareVersion
	offset++

	binary.BigEndian.PutUint16(buf[offset:], uint16(s.Threshold))
	offset += 2

	binary.BigEndian.PutUint16(buf[offset:], uint16(s.Total))
	offset += 2

	binary.BigEndian.PutUint32(buf[offset:], uint32(s.SecretLen))
	offset += 4

	binary.BigEndian.PutUint16(buf[offset:], uint16(len(s.Ys)))
	offset += 2

	binary.BigEndian.PutUint16(buf[offset:], uint16(len(xBytes)))
	offset += 2

	copy(buf[offset:], xBytes)
	offset += len(xBytes)

	for _, yBytes := range yBytesSlice {
		binary.BigEndian.PutUint16(buf[offset:], uint16(len(yBytes)))
		offset += 2
		copy(buf[offset:], yBytes)
		offset += len(yBytes)
	}

	return buf
}

// String returns the chunked share as a base64-encoded string.
func (s *ChunkedShare) String() string {
	return base64.StdEncoding.EncodeToString(s.Bytes())
}

// ParseChunkedShare deserializes a chunked share from binary format.
func ParseChunkedShare(data []byte) (*ChunkedShare, error) {
	if len(data) < chunkedShareHeaderSize {
		return nil, ErrInvalidShareFormat
	}

	offset := 0

	version := data[offset]
	offset++

	if version != chunkedShareVersion {
		return nil, ErrUnsupportedVersion
	}

	threshold := int(binary.BigEndian.Uint16(data[offset:]))
	offset += 2

	total := int(binary.BigEndian.Uint16(data[offset:]))
	offset += 2

	secretLen := int(binary.BigEndian.Uint32(data[offset:]))
	offset += 4

	numChunks := int(binary.BigEndian.Uint16(data[offset:]))
	offset += 2

	xLen := int(binary.BigEndian.Uint16(data[offset:]))
	offset += 2

	if len(data) < offset+xLen {
		return nil, ErrInvalidShareFormat
	}

	x := new(big.Int).SetBytes(data[offset : offset+xLen])
	offset += xLen

	if x.Sign() == 0 {
		return nil, ErrInvalidShareX
	}

	ys := make([]*big.Int, numChunks)
	for i := range numChunks {
		if len(data) < offset+2 {
			return nil, ErrInvalidShareFormat
		}

		yLen := int(binary.BigEndian.Uint16(data[offset:]))
		offset += 2

		if len(data) < offset+yLen {
			return nil, ErrInvalidShareFormat
		}

		ys[i] = new(big.Int).SetBytes(data[offset : offset+yLen])
		offset += yLen
	}

	return &ChunkedShare{
		X:         x,
		Ys:        ys,
		Threshold: threshold,
		Total:     total,
		SecretLen: secretLen,
	}, nil
}

// ParseChunkedShareString deserializes a chunked share from a base64-encoded string.
func ParseChunkedShareString(s string) (*ChunkedShare, error) {
	data, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return nil, errors.Join(ErrInvalidShareFormat, err)
	}
	return ParseChunkedShare(data)
}

// Clone creates a deep copy of the chunked share.
func (s *ChunkedShare) Clone() *ChunkedShare {
	ys := make([]*big.Int, len(s.Ys))
	for i, y := range s.Ys {
		ys[i] = new(big.Int).Set(y)
	}

	return &ChunkedShare{
		X:         new(big.Int).Set(s.X),
		Ys:        ys,
		Threshold: s.Threshold,
		Total:     s.Total,
		SecretLen: s.SecretLen,
	}
}

// Equal checks if two chunked shares are equal.
func (s *ChunkedShare) Equal(other *ChunkedShare) bool {
	if s == nil || other == nil {
		return s == other
	}

	if s.X.Cmp(other.X) != 0 ||
		s.Threshold != other.Threshold ||
		s.Total != other.Total ||
		s.SecretLen != other.SecretLen ||
		len(s.Ys) != len(other.Ys) {
		return false
	}

	for i, y := range s.Ys {
		if y.Cmp(other.Ys[i]) != 0 {
			return false
		}
	}

	return true
}

// splitIntoChunks splits data into chunks of specified size.
func splitIntoChunks(data []byte, chunkSize int) [][]byte {
	if len(data) == 0 {
		return nil
	}

	numChunks := (len(data) + chunkSize - 1) / chunkSize
	chunks := make([][]byte, numChunks)

	for i := range numChunks {
		start := i * chunkSize
		end := min(start+chunkSize, len(data))
		chunks[i] = data[start:end]
	}

	return chunks
}
