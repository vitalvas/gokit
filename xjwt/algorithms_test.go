package xjwt

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAlgorithmConstants(t *testing.T) {
	cases := map[string]string{
		RS256:      "RS256",
		ES256:      "ES256",
		EdDSA:      "EdDSA",
		A128GCM:    "A128GCM",
		RSAOAEP256: "RSA-OAEP-256",
		ECDHES:     "ECDH-ES",
		Dir:        "dir",
	}
	for got, want := range cases {
		assert.Equal(t, want, got)
	}
}
