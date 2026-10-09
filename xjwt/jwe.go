package xjwt

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
)

// jweHeader is the JWE protected header (RFC 7516 Section 4).
type jweHeader struct {
	Crit json.RawMessage `json:"crit,omitempty"`
	Alg  string          `json:"alg"`
	Enc  string          `json:"enc"`
	Kid  string          `json:"kid,omitempty"`
	Epk  *JSONWebKey     `json:"epk,omitempty"`
	Zip  string          `json:"zip,omitempty"`
	P2s  string          `json:"p2s,omitempty"` // PBES2 salt input (base64url)
	P2c  int             `json:"p2c,omitempty"` // PBES2 iteration count
	IV   string          `json:"iv,omitempty"`  // AES-GCMKW key-wrap IV (base64url)
	Tag  string          `json:"tag,omitempty"` // AES-GCMKW key-wrap tag (base64url)
	Apu  string          `json:"apu,omitempty"` // ECDH-ES PartyUInfo (base64url)
	Apv  string          `json:"apv,omitempty"` // ECDH-ES PartyVInfo (base64url)
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

// DecryptOptions restricts the algorithms and password-derivation work a JWE
// may use. PBES2 always requires an explicit AllowedAlgs entry. Other supported
// algorithms are accepted when the corresponding allowlist is empty.
type DecryptOptions struct {
	AllowedAlgs []string // permitted key-management "alg" values
	AllowedEnc  []string // permitted content-encryption "enc" values
	// MaxPBES2Count caps password derivation iterations. Zero uses 600000.
	// Negative values are invalid; values above 10000000 are not permitted.
	MaxPBES2Count int
}

// Decrypt decrypts a compact JWE with the recipient's private key, returning the
// plaintext, accepting supported algorithms except PBES2. The key type depends on the
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

	if err := validateJWEHeader(header, opts); err != nil {
		return nil, err
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
	if err := validateContentCiphertext(contentCiphertext{enc: header.Enc, iv: iv, ciphertext: ciphertext, tag: tag}); err != nil {
		return nil, err
	}
	if err := validateEncryptedKey(header, encryptedKey); err != nil {
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

func validateJWEHeader(header jweHeader, opts DecryptOptions) error {
	if len(header.Crit) > 0 {
		return fmt.Errorf("xjwt: unsupported critical JWE headers")
	}
	if header.Zip != "" && header.Zip != zipDEF {
		return fmt.Errorf("xjwt: unsupported compression algorithm %q", header.Zip)
	}
	if !isJWEKeyAlg(header.Alg) && !isPBES2Alg(header.Alg) && !isGCMKWAlg(header.Alg) {
		return fmt.Errorf("xjwt: unsupported key management algorithm %q", header.Alg)
	}
	if (len(opts.AllowedAlgs) > 0 || isPBES2Alg(header.Alg)) && !algAllowed(header.Alg, opts.AllowedAlgs) {
		return fmt.Errorf("xjwt: key management algorithm %q is not in the allowed set", header.Alg)
	}
	if len(opts.AllowedEnc) > 0 && !algAllowed(header.Enc, opts.AllowedEnc) {
		return fmt.Errorf("xjwt: content encryption algorithm %q is not in the allowed set", header.Enc)
	}
	if _, err := cekLength(header.Enc); err != nil {
		return err
	}
	if opts.MaxPBES2Count < 0 || opts.MaxPBES2Count > maxPBES2Count {
		return fmt.Errorf("xjwt: invalid maximum PBES2 count")
	}
	maximum := opts.MaxPBES2Count
	if maximum == 0 {
		maximum = 600000
	}
	if isPBES2Alg(header.Alg) && (header.P2c <= 0 || header.P2c > maximum) {
		return fmt.Errorf("xjwt: PBES2 iteration count outside permitted range")
	}
	return nil
}

func validateEncryptedKey(header jweHeader, encryptedKey []byte) error {
	cekLen, err := cekLength(header.Enc)
	if err != nil {
		return err
	}
	switch {
	case header.Alg == Dir || header.Alg == ECDHES:
		if len(encryptedKey) != 0 {
			return fmt.Errorf("xjwt: direct encryption requires an empty encrypted key")
		}
	case isPBES2Alg(header.Alg) || aesKWKeyLen(header.Alg) != 0 || header.Alg == ECDHESA128 || header.Alg == ECDHESA192 || header.Alg == ECDHESA256:
		if len(encryptedKey) != cekLen+8 {
			return fmt.Errorf("xjwt: invalid wrapped CEK length")
		}
	case isGCMKWAlg(header.Alg):
		if len(encryptedKey) != cekLen {
			return fmt.Errorf("xjwt: invalid GCM wrapped CEK length")
		}
	default:
		if len(encryptedKey) == 0 {
			return fmt.Errorf("xjwt: missing encrypted key")
		}
	}
	return nil
}

// decodeOptionalB64 decodes an optional base64url header value, returning nil for
// an empty string.
func decodeOptionalB64(s string) ([]byte, error) {
	if s == "" {
		return nil, nil
	}

	return base64.RawURLEncoding.DecodeString(s)
}
