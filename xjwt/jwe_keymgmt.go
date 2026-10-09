package xjwt

import (
	"crypto/ecdh"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1" //nolint:gosec // RSA-OAEP (not -256) mandates SHA-1 per RFC 7518 Section 4.2
	"crypto/sha256"
	"crypto/sha512"
	"fmt"
	"hash"
)

// isJWEKeyAlg reports whether alg is a supported JWE key-management algorithm.
func isJWEKeyAlg(alg string) bool {
	switch alg {
	case RSAOAEP, RSAOAEP256, RSAOAEP384, RSAOAEP512,
		A128KW, A192KW, A256KW, Dir,
		ECDHES, ECDHESA128, ECDHESA192, ECDHESA256:
		return true
	default:
		return false
	}
}

// encryptCEK produces the content-encryption key (cek) and its encrypted form
// (encryptedKey) for the given key-management algorithm and recipient key. For
// "dir" the cek is the supplied symmetric key and encryptedKey is empty. For
// ECDH-ES (direct) the cek is derived and encryptedKey is empty. When an
// ephemeral key is generated (ECDH-ES*), its public JWK is returned in epk for
// the recipient header.
func encryptCEK(alg, enc string, key any, apu, apv []byte) (cek, encryptedKey []byte, epk *JSONWebKey, err error) {
	cekLen, err := cekLength(enc)
	if err != nil {
		return nil, nil, nil, err
	}

	switch alg {
	case RSAOAEP, RSAOAEP256, RSAOAEP384, RSAOAEP512:
		pub, ok := key.(*rsa.PublicKey)
		if !ok {
			return nil, nil, nil, ErrKeyTypeMismatch
		}
		cek = make([]byte, cekLen)
		if _, err = rand.Read(cek); err != nil {
			return nil, nil, nil, err
		}
		encryptedKey, err = rsa.EncryptOAEP(oaepHash(alg), rand.Reader, pub, cek, nil)

		return cek, encryptedKey, nil, err

	case A128KW, A192KW, A256KW:
		kek, ok := key.([]byte)
		if !ok {
			return nil, nil, nil, ErrKeyTypeMismatch
		}
		cek = make([]byte, cekLen)
		if _, err = rand.Read(cek); err != nil {
			return nil, nil, nil, err
		}
		encryptedKey, err = aesKeyWrap(kek, cek)

		return cek, encryptedKey, nil, err

	case Dir:
		kek, ok := key.([]byte)
		if !ok {
			return nil, nil, nil, ErrKeyTypeMismatch
		}
		if len(kek) != cekLen {
			return nil, nil, nil, fmt.Errorf("xjwt: dir key must be %d bytes for %s", cekLen, enc)
		}

		return kek, nil, nil, nil

	case ECDHES, ECDHESA128, ECDHESA192, ECDHESA256:
		return encryptCEKECDH(ecdhEncryptInput{
			alg:    alg,
			enc:    enc,
			key:    key,
			cekLen: cekLen,
			apu:    apu,
			apv:    apv,
		})

	default:
		return nil, nil, nil, fmt.Errorf("xjwt: unsupported key management algorithm %q", alg)
	}
}

// cekDecryptInput bundles the inputs needed to recover a content-encryption key,
// so the dispatch and ECDH helpers stay within the project's parameter limit.
type cekDecryptInput struct {
	alg          string
	enc          string
	key          any
	encryptedKey []byte
	epk          *JSONWebKey // ECDH-ES ephemeral public key
	apu, apv     []byte      // ECDH-ES agreement-party info
}

// decryptCEK recovers the content-encryption key from in.encryptedKey (and, for
// ECDH-ES, the ephemeral public key in.epk) using the recipient's private key.
func decryptCEK(in cekDecryptInput) ([]byte, error) {
	cekLen, err := cekLength(in.enc)
	if err != nil {
		return nil, err
	}

	switch in.alg {
	case RSAOAEP, RSAOAEP256, RSAOAEP384, RSAOAEP512:
		priv, ok := in.key.(*rsa.PrivateKey)
		if !ok {
			return nil, ErrKeyTypeMismatch
		}

		return rsa.DecryptOAEP(oaepHash(in.alg), rand.Reader, priv, in.encryptedKey, nil)

	case A128KW, A192KW, A256KW:
		kek, ok := in.key.([]byte)
		if !ok {
			return nil, ErrKeyTypeMismatch
		}

		return aesKeyUnwrap(kek, in.encryptedKey)

	case Dir:
		kek, ok := in.key.([]byte)
		if !ok {
			return nil, ErrKeyTypeMismatch
		}
		if len(kek) != cekLen {
			return nil, fmt.Errorf("xjwt: dir key must be %d bytes for %s", cekLen, in.enc)
		}

		return kek, nil

	case ECDHES, ECDHESA128, ECDHESA192, ECDHESA256:
		return decryptCEKECDH(in, cekLen)

	default:
		return nil, fmt.Errorf("xjwt: unsupported key management algorithm %q", in.alg)
	}
}

func oaepHash(alg string) hash.Hash {
	switch alg {
	case RSAOAEP256:
		return sha256.New()
	case RSAOAEP384:
		return sha512.New384()
	case RSAOAEP512:
		return sha512.New()
	default:
		return sha1.New() // RSA-OAEP uses SHA-1 (RFC 7518 Section 4.2)
	}
}

// ecdhEncryptInput bundles the inputs for ECDH-ES key agreement on the sender
// side (keeps the parameter count within the project limit).
type ecdhEncryptInput struct {
	alg      string
	enc      string
	key      any
	cekLen   int
	apu, apv []byte
}

// encryptCEKECDH performs ECDH-ES (direct or with AES-KW) key agreement, drawing
// an ephemeral key on the sender side.
func encryptCEKECDH(in ecdhEncryptInput) (cek, encryptedKey []byte, epk *JSONWebKey, err error) {
	pub, ok := in.key.(*ecdh.PublicKey)
	if !ok {
		// Accept an *ecdsa.PublicKey by converting it.
		conv, cerr := toECDHPublic(in.key)
		if cerr != nil {
			return nil, nil, nil, ErrKeyTypeMismatch
		}
		pub = conv
	}

	eph, err := pub.Curve().GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil, nil, err
	}

	shared, err := eph.ECDH(pub)
	if err != nil {
		return nil, nil, nil, err
	}

	epkJWK, err := ecdhPublicJWK(eph.PublicKey())
	if err != nil {
		return nil, nil, nil, err
	}

	// For direct agreement the derived key IS the CEK (sized to enc). For the
	// AES-KW variants it is a KEK sized to the wrap algorithm.
	derived, kwKeyLen, direct := ecdhDerivedSpec(in.alg, in.enc, in.cekLen)
	derivedKey := concatKDF(shared, kwKeyLen, []byte(derived), in.apu, in.apv)

	if direct {
		return derivedKey, nil, epkJWK, nil
	}

	cek = make([]byte, in.cekLen)
	if _, err = rand.Read(cek); err != nil {
		return nil, nil, nil, err
	}
	encryptedKey, err = aesKeyWrap(derivedKey, cek)

	return cek, encryptedKey, epkJWK, err
}

func decryptCEKECDH(in cekDecryptInput, cekLen int) ([]byte, error) {
	if in.epk == nil {
		return nil, fmt.Errorf("xjwt: ECDH-ES missing ephemeral public key (epk)")
	}

	priv, err := toECDHPrivate(in.key)
	if err != nil {
		return nil, ErrKeyTypeMismatch
	}

	ephPub, err := ecdhPublicFromJWK(in.epk)
	if err != nil {
		return nil, err
	}

	shared, err := priv.ECDH(ephPub)
	if err != nil {
		return nil, err
	}

	derived, kwKeyLen, direct := ecdhDerivedSpec(in.alg, in.enc, cekLen)
	derivedKey := concatKDF(shared, kwKeyLen, []byte(derived), in.apu, in.apv)

	if direct {
		return derivedKey, nil
	}

	return aesKeyUnwrap(derivedKey, in.encryptedKey)
}

// ecdhDerivedSpec returns the algorithm id fed to the Concat KDF, the derived
// key length, and whether this is direct agreement (derived key == CEK).
func ecdhDerivedSpec(alg, enc string, cekLen int) (algID string, keyLen int, direct bool) {
	switch alg {
	case ECDHES:
		return enc, cekLen, true // algID = enc, key sized to the content enc
	case ECDHESA128:
		return alg, 16, false
	case ECDHESA192:
		return alg, 24, false
	case ECDHESA256:
		return alg, 32, false
	default:
		return alg, cekLen, true
	}
}
