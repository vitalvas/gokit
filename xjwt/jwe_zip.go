package xjwt

import (
	"bytes"
	"compress/flate"
	"fmt"
	"io"
)

// zipDEF is the only JWE compression algorithm defined by RFC 7516 (DEFLATE).
const zipDEF = "DEF"

// maxDecompressedLen bounds the output of JWE decompression to prevent a
// decompression-bomb denial of service. 10 MiB is far larger than any realistic
// token payload.
const maxDecompressedLen = 10 << 20

// deflate compresses data using raw DEFLATE (RFC 1951), as required for the JWE
// "zip":"DEF" parameter.
func deflate(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	w, err := flate.NewWriter(&buf, flate.DefaultCompression)
	if err != nil {
		return nil, err
	}

	if _, err := w.Write(data); err != nil {
		return nil, err
	}

	if err := w.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// inflate reverses deflate, rejecting output larger than maxDecompressedLen.
func inflate(data []byte) ([]byte, error) {
	r := flate.NewReader(bytes.NewReader(data))
	defer r.Close()

	// Read one byte past the limit to detect overflow.
	out, err := io.ReadAll(io.LimitReader(r, maxDecompressedLen+1))
	if err != nil {
		return nil, err
	}

	if len(out) > maxDecompressedLen {
		return nil, fmt.Errorf("xjwt: decompressed payload exceeds %d bytes", maxDecompressedLen)
	}

	return out, nil
}
