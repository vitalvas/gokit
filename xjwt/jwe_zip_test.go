package xjwt

import (
	"bytes"
	"compress/flate"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInflateExceedsLimit(t *testing.T) {
	// Build a DEFLATE stream that decompresses to > maxDecompressedLen.
	var buf bytes.Buffer
	w, err := flate.NewWriter(&buf, flate.BestCompression)
	require.NoError(t, err)
	_, err = w.Write(bytes.Repeat([]byte{0}, maxDecompressedLen+1024))
	require.NoError(t, err)
	require.NoError(t, w.Close())

	_, err = inflate(buf.Bytes())
	require.Error(t, err)
}

func TestDeflateInflateErrors(t *testing.T) {
	// inflate must reject data that is not valid DEFLATE.
	_, err := inflate([]byte("not deflate data at all"))
	require.Error(t, err)
}
