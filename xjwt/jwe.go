package xjwt

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
)

// jweHeader is the JWE protected header (RFC 7516 Section 4).
type jweHeader struct {
	Alg string      `json:"alg"`
	Enc string      `json:"enc"`
	Kid string      `json:"kid,omitempty"`
	Epk *JSONWebKey `json:"epk,omitempty"`
	Zip string      `json:"zip,omitempty"`
	P2s string      `json:"p2s,omitempty"` // PBES2 salt input (base64url)
	P2c int         `json:"p2c,omitempty"` // PBES2 iteration count
	IV  string      `json:"iv,omitempty"`  // AES-GCMKW key-wrap IV (base64url)
	Tag string      `json:"tag,omitempty"` // AES-GCMKW key-wrap tag (base64url)
	Apu string      `json:"apu,omitempty"` // ECDH-ES PartyUInfo (base64url)
	Apv string      `json:"apv,omitempty"` // ECDH-ES PartyVInfo (base64url)
}

// EncryptOptions configures Encrypt.
type EncryptOptions struct {
	// Kid is written to the protected header when non-empty.
	Kid string
	// Compress enables DEFLATE compression of the plaintext (zip="DEF").
	Compress bool
	// APU and APV are the ECDH-ES PartyUInfo/PartyVInfo agreement-party values,
	// fed into the Concat KDF and written to the header (apu/apv). Ignored for
	// non-ECDH algorithms.
	APU []byte
	APV []byte
}

// Encrypt encrypts plaintext into a compact JWE under the recipient key, using
// the key-management algorithm alg and content-encryption algorithm enc. The key
// type depends on alg: *rsa.PublicKey for RSA-OAEP(-256), []byte for A*KW, dir,
// and PBES2* (a password), and an EC public key (*ecdsa.PublicKey or
// *ecdh.PublicKey) for ECDH-ES.
func Encrypt(alg, enc string, key any, plaintext []byte, opts EncryptOptions) (string, error) {
	header := jweHeader{Alg: alg, Enc: enc, Kid: opts.Kid}

	if opts.Compress {
		compressed, err := deflate(plaintext)
		if err != nil {
			return "", err
		}
		plaintext = compressed
		header.Zip = zipDEF
	}

	var (
		cek, encryptedKey []byte
		err               error
	)

	switch {
	case isPBES2Alg(alg):
		password, ok := key.([]byte)
		if !ok {
			return "", ErrKeyTypeMismatch
		}
		cek, encryptedKey, err = pbes2EncryptCEK(alg, enc, password, &header)
	case isGCMKWAlg(alg):
		cek, encryptedKey, err = gcmkwEncryptCEK(alg, enc, key, &header)
	case isJWEKeyAlg(alg):
		var epk *JSONWebKey
		cek, encryptedKey, epk, err = encryptCEK(alg, enc, key, opts.APU, opts.APV)
		header.Epk = epk
		if len(opts.APU) > 0 {
			header.Apu = base64.RawURLEncoding.EncodeToString(opts.APU)
		}
		if len(opts.APV) > 0 {
			header.Apv = base64.RawURLEncoding.EncodeToString(opts.APV)
		}
	default:
		return "", fmt.Errorf("xjwt: unsupported key management algorithm %q", alg)
	}

	if err != nil {
		return "", err
	}

	headerJSON, err := json.Marshal(header)
	if err != nil {
		return "", err
	}

	enc64 := base64.RawURLEncoding
	protected := enc64.EncodeToString(headerJSON)

	// The protected header (ASCII) is the AAD for content encryption.
	iv, ciphertext, tag, err := contentEncrypt(enc, cek, plaintext, []byte(protected))
	if err != nil {
		return "", err
	}

	return strings.Join([]string{
		protected,
		enc64.EncodeToString(encryptedKey),
		enc64.EncodeToString(iv),
		enc64.EncodeToString(ciphertext),
		enc64.EncodeToString(tag),
	}, "."), nil
}

// DecryptOptions restricts which algorithms a JWE may use. Empty slices mean
// "accept any supported algorithm"; a non-empty slice is an allowlist that the
// token's "alg"/"enc" must be a member of. Pinning these is defense-in-depth
// against a sender downgrading to a weaker algorithm.
type DecryptOptions struct {
	AllowedAlgs []string // permitted key-management "alg" values
	AllowedEnc  []string // permitted content-encryption "enc" values
}

// Decrypt decrypts a compact JWE with the recipient's private key, returning the
// plaintext, accepting any supported algorithm. The key type depends on the
// token's alg: *rsa.PrivateKey, []byte, or an EC private key. To restrict the
// accepted algorithms, use DecryptWithOptions.
func Decrypt(jwe string, key any) ([]byte, error) {
	return DecryptWithOptions(jwe, key, DecryptOptions{})
}

// DecryptWithOptions is Decrypt with an optional allowlist of accepted "alg" and
// "enc" algorithms.
func DecryptWithOptions(jwe string, key any, opts DecryptOptions) ([]byte, error) {
	parts, ok := splitCompact(jwe, 5)
	if !ok {
		return nil, fmt.Errorf("xjwt: compact JWE must have five segments")
	}

	enc64 := base64.RawURLEncoding

	headerBytes, err := enc64.DecodeString(parts[0])
	if err != nil {
		return nil, fmt.Errorf("xjwt: decoding JWE header: %w", err)
	}

	var header jweHeader
	if err := json.Unmarshal(headerBytes, &header); err != nil {
		return nil, fmt.Errorf("xjwt: parsing JWE header: %w", err)
	}

	if !isJWEKeyAlg(header.Alg) && !isPBES2Alg(header.Alg) && !isGCMKWAlg(header.Alg) {
		return nil, fmt.Errorf("xjwt: unsupported key management algorithm %q", header.Alg)
	}

	if len(opts.AllowedAlgs) > 0 && !algAllowed(header.Alg, opts.AllowedAlgs) {
		return nil, fmt.Errorf("xjwt: key management algorithm %q is not in the allowed set", header.Alg)
	}

	if len(opts.AllowedEnc) > 0 && !algAllowed(header.Enc, opts.AllowedEnc) {
		return nil, fmt.Errorf("xjwt: content encryption algorithm %q is not in the allowed set", header.Enc)
	}

	encryptedKey, err := enc64.DecodeString(parts[1])
	if err != nil {
		return nil, err
	}

	iv, err := enc64.DecodeString(parts[2])
	if err != nil {
		return nil, err
	}

	ciphertext, err := enc64.DecodeString(parts[3])
	if err != nil {
		return nil, err
	}

	tag, err := enc64.DecodeString(parts[4])
	if err != nil {
		return nil, err
	}

	var cek []byte
	switch {
	case isPBES2Alg(header.Alg):
		password, ok := key.([]byte)
		if !ok {
			return nil, ErrKeyTypeMismatch
		}
		cek, err = pbes2DecryptCEK(header.Alg, password, encryptedKey, &header)
	case isGCMKWAlg(header.Alg):
		cek, err = gcmkwDecryptCEK(header.Alg, key, encryptedKey, &header)
	default:
		var apu, apv []byte
		if apu, err = decodeOptionalB64(header.Apu); err != nil {
			return nil, fmt.Errorf("xjwt: decoding apu: %w", err)
		}
		if apv, err = decodeOptionalB64(header.Apv); err != nil {
			return nil, fmt.Errorf("xjwt: decoding apv: %w", err)
		}
		cek, err = decryptCEK(cekDecryptInput{
			alg:          header.Alg,
			enc:          header.Enc,
			key:          key,
			encryptedKey: encryptedKey,
			epk:          header.Epk,
			apu:          apu,
			apv:          apv,
		})
	}

	if err != nil {
		return nil, err
	}

	plaintext, err := contentDecrypt(contentCiphertext{
		enc:        header.Enc,
		cek:        cek,
		iv:         iv,
		ciphertext: ciphertext,
		tag:        tag,
		aad:        []byte(parts[0]),
	})
	if err != nil {
		return nil, err
	}

	switch header.Zip {
	case "":
		return plaintext, nil
	case zipDEF:
		return inflate(plaintext)
	default:
		return nil, fmt.Errorf("xjwt: unsupported compression algorithm %q", header.Zip)
	}
}

// decodeOptionalB64 decodes an optional base64url header value, returning nil for
// an empty string.
func decodeOptionalB64(s string) ([]byte, error) {
	if s == "" {
		return nil, nil
	}

	return base64.RawURLEncoding.DecodeString(s)
}
