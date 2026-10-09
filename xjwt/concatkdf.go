package xjwt

import (
	"crypto/sha256"
	"encoding/binary"
)

// concatKDF implements the Concat KDF (NIST SP 800-56A single-step KDF with
// SHA-256) as used by ECDH-ES in RFC 7518 Section 4.6. z is the raw ECDH shared
// secret; algID/partyU/partyV/suppPubInfo form the OtherInfo. keyLen is the
// desired output length in bytes.
//
// For JWE, suppPubInfo is the key length in bits as a 32-bit big-endian integer,
// and partyU/partyV default to empty when apu/apv are absent.
func concatKDF(z []byte, keyLen int, algID, partyU, partyV []byte) []byte {
	keyBits := uint32(keyLen * 8)
	var suppPubInfo [4]byte
	binary.BigEndian.PutUint32(suppPubInfo[:], keyBits)

	out := make([]byte, 0, keyLen)
	var counter uint32 = 1

	for len(out) < keyLen {
		h := sha256.New()

		var counterBytes [4]byte
		binary.BigEndian.PutUint32(counterBytes[:], counter)
		h.Write(counterBytes[:])

		h.Write(z)
		writeLengthPrefixed(h, algID)
		writeLengthPrefixed(h, partyU)
		writeLengthPrefixed(h, partyV)
		h.Write(suppPubInfo[:])

		out = append(out, h.Sum(nil)...)
		counter++
	}

	return out[:keyLen]
}

// writeLengthPrefixed writes a 32-bit big-endian length followed by data, the
// "Datalen || Data" framing used by the Concat KDF OtherInfo fields.
func writeLengthPrefixed(h interface{ Write([]byte) (int, error) }, data []byte) {
	var l [4]byte
	binary.BigEndian.PutUint32(l[:], uint32(len(data)))
	_, _ = h.Write(l[:])
	_, _ = h.Write(data)
}
