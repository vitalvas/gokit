package xjwt

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
)

// Header is the JOSE header (RFC 7515 Section 4) read and written by this package.
type Header struct {
	Alg  string   `json:"alg"`
	Typ  string   `json:"typ,omitempty"`
	Kid  string   `json:"kid,omitempty"`
	Crit []string `json:"crit,omitempty"`
	B64  *bool    `json:"b64,omitempty"`
}

// KeyResolver returns the verification key for a token's header. Returning an
// error rejects the token. Implementations select a key by kid and alg and
// must return the public/secret key matching the header algorithm.
type KeyResolver func(h Header) (any, error)

// Sign produces a compact JWS (header.payload.signature) over the JSON-encoded
// claims, signed with key under alg, with the JOSE typ header set to "JWT". kid,
// when non-empty, is written to the header. "none" is rejected.
func Sign(alg, kid string, claims any, key any) (string, error) {
	return SignWithType(alg, "JWT", kid, claims, key)
}

// SignWithType is Sign with an explicit JOSE typ header, e.g. "at+jwt" for an
// RFC 9068 access token or "logout+jwt" for a back-channel logout token. An
// empty typ omits the header.
func SignWithType(alg, typ, kid string, claims any, key any) (string, error) {
	if !IsSupportedAlg(alg) {
		return "", fmt.Errorf("xjwt: unsupported signing algorithm %q", alg)
	}

	header := Header{Alg: alg, Typ: typ, Kid: kid}

	headerJSON, err := json.Marshal(header)
	if err != nil {
		return "", err
	}

	payloadJSON, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}

	enc := base64.RawURLEncoding
	signingInput := fmt.Sprintf("%s.%s", enc.EncodeToString(headerJSON), enc.EncodeToString(payloadJSON))

	sig, err := signPayload(alg, signingInput, key)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s.%s", signingInput, enc.EncodeToString(sig)), nil
}

// Verify checks a compact JWS, returning the raw decoded payload bytes. The
// token's algorithm must be in allowedAlgs (a non-empty allowlist) and must
// not be "none". The resolver supplies the verification key for the header.
func Verify(token string, resolve KeyResolver, allowedAlgs []string) ([]byte, error) {
	header, signingInput, payloadSeg, sig, err := splitToken(token)
	if err != nil {
		return nil, err
	}

	if header.Alg == "" || strings.EqualFold(header.Alg, "none") {
		return nil, fmt.Errorf("xjwt: algorithm %q is not permitted", header.Alg)
	}

	if err := validateVerificationHeader(header); err != nil {
		return nil, err
	}

	if !algAllowed(header.Alg, allowedAlgs) {
		return nil, fmt.Errorf("xjwt: algorithm %q is not in the allowed set", header.Alg)
	}

	if !IsSupportedAlg(header.Alg) {
		return nil, fmt.Errorf("xjwt: unsupported signing algorithm %q", header.Alg)
	}

	key, err := resolve(header)
	if err != nil {
		return nil, err
	}

	if err := verifyPayload(header.Alg, signingInput, sig, key); err != nil {
		return nil, err
	}

	payload, err := decodeSegment(payloadSeg)
	if err != nil {
		return nil, err
	}

	return payload, nil
}

// DecodeHeader returns the JOSE header of a compact JWS without verifying it.
// Use only to select a key or inspect typ before verification.
func DecodeHeader(token string) (Header, error) {
	parts, ok := splitCompact(token, 3)
	if !ok {
		return Header{}, errMalformedToken
	}

	headerBytes, err := decodeSegment(parts[0])
	if err != nil {
		return Header{}, fmt.Errorf("xjwt: decoding header: %w", err)
	}

	var header Header
	if err := json.Unmarshal(headerBytes, &header); err != nil {
		return Header{}, fmt.Errorf("xjwt: parsing header: %w", err)
	}

	return header, nil
}

// DecodePayloadUnverified returns the raw payload bytes of a compact JWS
// WITHOUT verifying the signature. Callers must not trust the result for
// authorization decisions.
func DecodePayloadUnverified(token string) ([]byte, error) {
	parts, ok := splitCompact(token, 3)
	if !ok {
		return nil, errMalformedToken
	}

	return decodeSegment(parts[1])
}

var errMalformedToken = fmt.Errorf("xjwt: token must have three segments")

// splitCompact splits a compact JOSE token into exactly n dot-separated
// segments, returning false if the segment count differs. It counts separators
// first and rejects a wrong count before allocating, so a token padded with a
// flood of '.' characters cannot force an O(n) slice allocation
// (CWE-405 amplification; cf. golang-jwt GHSA-mh63-6h87-95cp).
func splitCompact(token string, n int) ([]string, bool) {
	if strings.Count(token, ".") != n-1 {
		return nil, false
	}

	parts := make([]string, 0, n)
	for range n - 1 {
		i := strings.IndexByte(token, '.')
		parts = append(parts, token[:i])
		token = token[i+1:]
	}

	return append(parts, token), true
}

// splitToken parses a compact JWS, returning the header, the signing input
// (header.payload), the raw payload segment, and the decoded signature.
func splitToken(token string) (h Header, signingInput, payloadSeg string, sig []byte, err error) {
	parts, ok := splitCompact(token, 3)
	if !ok {
		return Header{}, "", "", nil, errMalformedToken
	}

	headerBytes, err := decodeSegment(parts[0])
	if err != nil {
		return Header{}, "", "", nil, fmt.Errorf("xjwt: decoding header: %w", err)
	}

	var header Header
	if err := json.Unmarshal(headerBytes, &header); err != nil {
		return Header{}, "", "", nil, fmt.Errorf("xjwt: parsing header: %w", err)
	}

	sig, err = decodeSegment(parts[2])
	if err != nil {
		return Header{}, "", "", nil, fmt.Errorf("xjwt: decoding signature: %w", err)
	}

	return header, fmt.Sprintf("%s.%s", parts[0], parts[1]), parts[1], sig, nil
}

// decodeSegment decodes a base64url segment, rejecting padding and non-url
// alphabets (strict RawURLEncoding).
func decodeSegment(seg string) ([]byte, error) {
	return base64.RawURLEncoding.DecodeString(seg)
}

func algAllowed(alg string, allowed []string) bool {
	for _, a := range allowed {
		if a == alg {
			return true
		}
	}

	return false
}

func validateVerificationHeader(header Header) error {
	if len(header.Crit) > 0 {
		return fmt.Errorf("xjwt: unsupported critical headers")
	}

	if header.B64 != nil && !*header.B64 {
		return fmt.Errorf("xjwt: unencoded payloads are not supported")
	}

	return nil
}
