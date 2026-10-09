package xjwt

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
)

// SignInput is one signature to produce in a JSON-serialized JWS: an algorithm,
// a key, and an optional kid written to the protected header.
type SignInput struct {
	Alg string
	Kid string
	Key any
}

// jwsSignature is one entry in the "signatures" array of a general JSON JWS.
type jwsSignature struct {
	Protected string `json:"protected"`
	Signature string `json:"signature"`
}

// jwsGeneral is the general JSON serialization (RFC 7515 Section 7.2.1).
type jwsGeneral struct {
	Payload    *string        `json:"payload,omitempty"`
	Signatures []jwsSignature `json:"signatures"`
}

// jwsFlattened is the flattened JSON serialization (RFC 7515 Section 7.2.2).
type jwsFlattened struct {
	Payload   *string `json:"payload,omitempty"`
	Protected string  `json:"protected"`
	Signature string  `json:"signature"`
}

// SignJSON signs payload with one or more keys and returns the general JSON
// serialization. With exactly one input, SignFlattenedJSON produces the more
// compact flattened form instead.
func SignJSON(payload []byte, inputs ...SignInput) ([]byte, error) {
	if len(inputs) == 0 {
		return nil, fmt.Errorf("xjwt: at least one signing input is required")
	}

	enc := base64.RawURLEncoding
	encodedPayload := enc.EncodeToString(payload)

	sigs := make([]jwsSignature, 0, len(inputs))
	for _, in := range inputs {
		sig, err := signOne(in, encodedPayload)
		if err != nil {
			return nil, err
		}
		sigs = append(sigs, sig)
	}

	return json.Marshal(jwsGeneral{Payload: &encodedPayload, Signatures: sigs})
}

// SignFlattenedJSON signs payload with a single key and returns the flattened
// JSON serialization.
func SignFlattenedJSON(payload []byte, in SignInput) ([]byte, error) {
	enc := base64.RawURLEncoding
	encodedPayload := enc.EncodeToString(payload)

	sig, err := signOne(in, encodedPayload)
	if err != nil {
		return nil, err
	}

	return json.Marshal(jwsFlattened{
		Payload:   &encodedPayload,
		Protected: sig.Protected,
		Signature: sig.Signature,
	})
}

// SignDetachedJSON is like SignFlattenedJSON but omits the payload from the
// output (RFC 7515 Appendix F). The same payload must be supplied to
// VerifyDetachedJSON.
func SignDetachedJSON(payload []byte, in SignInput) ([]byte, error) {
	enc := base64.RawURLEncoding
	encodedPayload := enc.EncodeToString(payload)

	sig, err := signOne(in, encodedPayload)
	if err != nil {
		return nil, err
	}

	return json.Marshal(jwsFlattened{Protected: sig.Protected, Signature: sig.Signature})
}

func signOne(in SignInput, encodedPayload string) (jwsSignature, error) {
	if !IsSupportedAlg(in.Alg) {
		return jwsSignature{}, fmt.Errorf("xjwt: unsupported signing algorithm %q", in.Alg)
	}

	enc := base64.RawURLEncoding
	headerJSON, err := json.Marshal(Header{Alg: in.Alg, Kid: in.Kid})
	if err != nil {
		return jwsSignature{}, err
	}

	protected := enc.EncodeToString(headerJSON)
	signingInput := fmt.Sprintf("%s.%s", protected, encodedPayload)

	raw, err := signPayload(in.Alg, signingInput, in.Key)
	if err != nil {
		return jwsSignature{}, err
	}

	return jwsSignature{Protected: protected, Signature: enc.EncodeToString(raw)}, nil
}

// VerifyJSON verifies a JSON-serialized JWS (general or flattened) and returns
// the payload. At least one signature must verify under the resolver and
// allowlist; each candidate algorithm is still subject to the allowlist and the
// "none" prohibition.
func VerifyJSON(data []byte, resolve KeyResolver, allowedAlgs []string) ([]byte, error) {
	return verifyJSON(data, nil, false, resolve, allowedAlgs)
}

// VerifyDetachedJSON verifies a detached JSON JWS against an externally supplied
// payload.
func VerifyDetachedJSON(data, payload []byte, resolve KeyResolver, allowedAlgs []string) ([]byte, error) {
	return verifyJSON(data, payload, true, resolve, allowedAlgs)
}

func verifyJSON(data, detachedPayload []byte, detached bool, resolve KeyResolver, allowedAlgs []string) ([]byte, error) {
	payloadSeg, sigs, err := parseJSONJWS(data)
	if err != nil {
		return nil, err
	}

	enc := base64.RawURLEncoding

	// Determine the encoded payload: from the document, or from the detached
	// payload supplied by the caller.
	var encodedPayload string
	switch {
	case detached:
		if payloadSeg != nil {
			return nil, fmt.Errorf("xjwt: detached verify given a document that embeds a payload")
		}
		encodedPayload = enc.EncodeToString(detachedPayload)
	case payloadSeg != nil:
		encodedPayload = *payloadSeg
	default:
		return nil, fmt.Errorf("xjwt: JWS has no payload and none supplied")
	}

	payload, err := enc.DecodeString(encodedPayload)
	if err != nil {
		return nil, fmt.Errorf("xjwt: decoding payload: %w", err)
	}

	for _, s := range sigs {
		if verifyJSONSignature(s, encodedPayload, resolve, allowedAlgs) {
			return payload, nil
		}
	}

	return nil, ErrTokenSignatureInvalid
}

func verifyJSONSignature(s jwsSignature, encodedPayload string, resolve KeyResolver, allowedAlgs []string) bool {
	headerBytes, err := base64.RawURLEncoding.DecodeString(s.Protected)
	if err != nil {
		return false
	}

	var header Header
	if err := json.Unmarshal(headerBytes, &header); err != nil {
		return false
	}

	if header.Alg == "" || strings.EqualFold(header.Alg, "none") || !algAllowed(header.Alg, allowedAlgs) || !IsSupportedAlg(header.Alg) {
		return false
	}

	if err := validateVerificationHeader(header); err != nil {
		return false
	}

	sig, err := base64.RawURLEncoding.DecodeString(s.Signature)
	if err != nil {
		return false
	}

	key, err := resolve(header)
	if err != nil {
		return false
	}

	signingInput := fmt.Sprintf("%s.%s", s.Protected, encodedPayload)

	return verifyPayload(header.Alg, signingInput, sig, key) == nil
}

// parseJSONJWS accepts both the general and flattened serializations and returns
// the (possibly empty) encoded payload and the list of signatures.
func parseJSONJWS(data []byte) (payload *string, sigs []jwsSignature, err error) {
	var fields struct {
		Payload json.RawMessage `json:"payload"`
	}
	if err := json.Unmarshal(data, &fields); err != nil {
		return nil, nil, fmt.Errorf("xjwt: parsing JSON JWS: %w", err)
	}
	if fields.Payload != nil {
		var value string
		if string(fields.Payload) == "null" {
			return nil, nil, fmt.Errorf("xjwt: payload must be a string")
		}
		if err := json.Unmarshal(fields.Payload, &value); err != nil {
			return nil, nil, fmt.Errorf("xjwt: payload must be a string: %w", err)
		}
	}
	var general jwsGeneral
	if err := json.Unmarshal(data, &general); err == nil && len(general.Signatures) > 0 {
		return general.Payload, general.Signatures, nil
	}

	var flat jwsFlattened
	if err := json.Unmarshal(data, &flat); err != nil {
		return nil, nil, fmt.Errorf("xjwt: parsing JSON JWS: %w", err)
	}

	if flat.Signature == "" || flat.Protected == "" {
		return nil, nil, fmt.Errorf("xjwt: JSON JWS has no signatures")
	}

	return flat.Payload, []jwsSignature{{Protected: flat.Protected, Signature: flat.Signature}}, nil
}
