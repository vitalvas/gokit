package otp

import (
	"crypto/hmac"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"time"
)

// Errors returned by OCRA operations.
var (
	ErrInvalidSuite = errors.New("invalid OCRA suite")
	ErrInvalidInput = errors.New("invalid OCRA input")
)

const questionBufLen = 128 // Q is always a 128-byte field (RFC 6287 section 5.1)

// OCRASuite is a parsed RFC 6287 OCRASuite string such as
// "OCRA-1:HOTP-SHA256-8:C-QN08-PSHA1".
type OCRASuite struct {
	// Hash is the HMAC hash of the CryptoFunction.
	Hash Algorithm

	// Digits is the truncation length (4-10), or 0 for no truncation
	// (the response is then the full HMAC value, hex-encoded).
	Digits int

	// Counter reports whether the data input includes the C counter.
	Counter bool

	// QuestionFormat is the challenge format: 'A' (alphanumeric),
	// 'N' (numeric), or 'H' (hexadecimal).
	QuestionFormat byte

	// QuestionLength is the maximum challenge length declared in the suite.
	QuestionLength int

	// HasPassword reports whether the data input includes the P hash, and
	// PasswordHash is its algorithm.
	HasPassword  bool
	PasswordHash Algorithm

	// SessionLength is the S field length in bytes, 0 when absent.
	SessionLength int

	// TimeStep is the T granularity, 0 when absent.
	TimeStep time.Duration

	raw string
}

// String returns the original OCRASuite string.
func (s *OCRASuite) String() string {
	return s.raw
}

// OCRAInput carries the data inputs for an OCRA computation. Only the fields
// the suite declares are used; providing Password or Session when the suite
// does not include them is an error.
type OCRAInput struct {
	// Counter is the C value for suites with a counter.
	Counter uint64

	// Question is the challenge (or concatenated challenges for mutual
	// authentication), in the format declared by the suite.
	Question string

	// Password is the pre-hashed PIN/password for suites with P; its length
	// must match the suite's hash size.
	Password []byte

	// Session is the session information for suites with S; it is
	// zero-padded on the left to the declared length.
	Session []byte

	// Timestamp is the time for suites with T.
	Timestamp time.Time
}

// ParseOCRASuite parses an RFC 6287 OCRASuite string of the form
// <Algorithm>:<CryptoFunction>:<DataInput>.
func ParseOCRASuite(s string) (*OCRASuite, error) {
	parts := strings.Split(s, ":")
	if len(parts) != 3 || parts[0] != "OCRA-1" {
		return nil, fmt.Errorf("%w: %q", ErrInvalidSuite, s)
	}

	out := &OCRASuite{raw: s}

	if err := out.parseCryptoFunction(parts[1]); err != nil {
		return nil, err
	}

	if err := out.parseDataInput(parts[2]); err != nil {
		return nil, err
	}

	return out, nil
}

// parseCryptoFunction parses "HOTP-<hash>-<digits>".
func (s *OCRASuite) parseCryptoFunction(cf string) error {
	parts := strings.Split(cf, "-")
	if len(parts) != 3 || parts[0] != "HOTP" {
		return fmt.Errorf("%w: crypto function %q", ErrInvalidSuite, cf)
	}

	switch parts[1] {
	case "SHA1":
		s.Hash = SHA1
	case "SHA256":
		s.Hash = SHA256
	case "SHA512":
		s.Hash = SHA512
	default:
		return fmt.Errorf("%w: hash %q", ErrInvalidSuite, parts[1])
	}

	digits, err := strconv.Atoi(parts[2])
	if err != nil || (digits != 0 && (digits < 4 || digits > 10)) {
		return fmt.Errorf("%w: truncation %q", ErrInvalidSuite, parts[2])
	}
	s.Digits = digits

	return nil
}

// parseDataInput parses "[C]-QFxx[-PH][-Snnn][-TG]" with the fixed field
// order required by RFC 6287.
func (s *OCRASuite) parseDataInput(di string) error {
	tokens := strings.Split(di, "-")
	i := 0

	if i < len(tokens) && tokens[i] == "C" {
		s.Counter = true
		i++
	}

	if i >= len(tokens) || !s.parseQuestion(tokens[i]) {
		return fmt.Errorf("%w: data input %q", ErrInvalidSuite, di)
	}
	i++

	if i < len(tokens) && strings.HasPrefix(tokens[i], "P") {
		switch tokens[i] {
		case "PSHA1":
			s.PasswordHash = SHA1
		case "PSHA256":
			s.PasswordHash = SHA256
		case "PSHA512":
			s.PasswordHash = SHA512
		default:
			return fmt.Errorf("%w: password hash %q", ErrInvalidSuite, tokens[i])
		}
		s.HasPassword = true
		i++
	}

	if i < len(tokens) && strings.HasPrefix(tokens[i], "S") {
		n, err := strconv.Atoi(tokens[i][1:])
		if len(tokens[i]) != 4 || err != nil || n < 1 || n > 512 {
			return fmt.Errorf("%w: session %q", ErrInvalidSuite, tokens[i])
		}
		s.SessionLength = n
		i++
	}

	if i < len(tokens) && strings.HasPrefix(tokens[i], "T") {
		step, ok := parseTimeStep(tokens[i])
		if !ok {
			return fmt.Errorf("%w: time step %q", ErrInvalidSuite, tokens[i])
		}
		s.TimeStep = step
		i++
	}

	if i != len(tokens) {
		return fmt.Errorf("%w: data input %q", ErrInvalidSuite, di)
	}

	return nil
}

// parseQuestion parses "QFxx" where F is A, N, or H and xx is the maximum
// challenge length (4-64).
func (s *OCRASuite) parseQuestion(token string) bool {
	if len(token) < 3 || token[0] != 'Q' {
		return false
	}

	format := token[1]
	if format != 'A' && format != 'N' && format != 'H' {
		return false
	}

	n, err := strconv.Atoi(token[2:])
	if err != nil || n < 4 || n > 64 {
		return false
	}

	s.QuestionFormat = format
	s.QuestionLength = n

	return true
}

// parseTimeStep parses "TG" where G is [1-59]S, [1-59]M, or [1-48]H.
func parseTimeStep(token string) (time.Duration, bool) {
	if len(token) < 3 {
		return 0, false
	}

	n, err := strconv.Atoi(token[1 : len(token)-1])
	if err != nil {
		return 0, false
	}

	switch token[len(token)-1] {
	case 'S':
		if n >= 1 && n <= 59 {
			return time.Duration(n) * time.Second, true
		}
	case 'M':
		if n >= 1 && n <= 59 {
			return time.Duration(n) * time.Minute, true
		}
	case 'H':
		if n >= 1 && n <= 48 {
			return time.Duration(n) * time.Hour, true
		}
	}

	return 0, false
}

// OCRA computes the RFC 6287 response for the suite, base32 secret, and
// inputs. With truncation (Digits 4-10) the response is a decimal code;
// with Digits 0 it is the full HMAC value, hex-encoded.
func OCRA(suite, secret string, in OCRAInput) (string, error) {
	s, err := ParseOCRASuite(suite)
	if err != nil {
		return "", err
	}

	key, err := decodeSecret(secret)
	if err != nil {
		return "", err
	}

	msg, err := s.message(in)
	if err != nil {
		return "", err
	}

	mac := hmac.New(s.Hash.hash(), key)
	mac.Write(msg)
	sum := mac.Sum(nil)

	if s.Digits == 0 {
		return hex.EncodeToString(sum), nil
	}

	return truncate(sum, s.Digits), nil
}

// VerifyOCRA reports whether code is the correct response for the suite,
// base32 secret, and inputs. The comparison is constant-time.
func VerifyOCRA(suite, secret, code string, in OCRAInput) (bool, error) {
	want, err := OCRA(suite, secret, in)
	if err != nil {
		return false, err
	}

	return equal(want, code), nil
}

// message assembles DataInput = OCRASuite | 00 | C | Q | P | S | T.
func (s *OCRASuite) message(in OCRAInput) ([]byte, error) {
	msg := make([]byte, 0, len(s.raw)+1+8+questionBufLen+64+s.SessionLength+8)
	msg = append(msg, s.raw...)
	msg = append(msg, 0x00)

	if s.Counter {
		var c [8]byte
		binary.BigEndian.PutUint64(c[:], in.Counter)
		msg = append(msg, c[:]...)
	}

	q, err := encodeQuestion(in.Question, s.QuestionFormat)
	if err != nil {
		return nil, err
	}
	msg = append(msg, q...)

	switch {
	case s.HasPassword:
		if size := s.PasswordHash.hash()().Size(); len(in.Password) != size {
			return nil, fmt.Errorf("%w: password hash must be %d bytes", ErrInvalidInput, size)
		}
		msg = append(msg, in.Password...)
	case len(in.Password) != 0:
		return nil, fmt.Errorf("%w: suite has no password input", ErrInvalidInput)
	}

	switch {
	case s.SessionLength > 0:
		if len(in.Session) > s.SessionLength {
			return nil, fmt.Errorf("%w: session longer than %d bytes", ErrInvalidInput, s.SessionLength)
		}
		// Zero-padded on the left, matching the RFC reference implementation.
		msg = append(msg, make([]byte, s.SessionLength-len(in.Session))...)
		msg = append(msg, in.Session...)
	case len(in.Session) != 0:
		return nil, fmt.Errorf("%w: suite has no session input", ErrInvalidInput)
	}

	if s.TimeStep > 0 {
		var t [8]byte
		binary.BigEndian.PutUint64(t[:], timeCounter(in.Timestamp, uint64(s.TimeStep/time.Second), time.Time{}))
		msg = append(msg, t[:]...)
	}

	return msg, nil
}

// encodeQuestion encodes a challenge into the fixed 128-byte Q field:
// numeric challenges as a big-endian integer, alphanumeric as ASCII, and
// hexadecimal as raw bytes, all left-justified and zero-padded.
func encodeQuestion(q string, format byte) ([]byte, error) {
	if q == "" || len(q) > questionBufLen {
		return nil, fmt.Errorf("%w: question length %d", ErrInvalidInput, len(q))
	}

	var raw []byte

	switch format {
	case 'N':
		for _, c := range q {
			if c < '0' || c > '9' {
				return nil, fmt.Errorf("%w: non-numeric question", ErrInvalidInput)
			}
		}

		n, _ := new(big.Int).SetString(q, 10)
		raw = hexBytes(n.Text(16))
		if raw == nil {
			return nil, fmt.Errorf("%w: non-numeric question", ErrInvalidInput)
		}

	case 'A':
		for _, c := range q {
			isDigit := c >= '0' && c <= '9'
			isAlpha := (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z')
			if !isDigit && !isAlpha {
				return nil, fmt.Errorf("%w: non-alphanumeric question", ErrInvalidInput)
			}
		}
		raw = []byte(q)

	default: // 'H'
		raw = hexBytes(q)
		if raw == nil {
			return nil, fmt.Errorf("%w: non-hexadecimal question", ErrInvalidInput)
		}
	}

	buf := make([]byte, questionBufLen)
	copy(buf, raw)

	return buf, nil
}

// hexBytes decodes a hex string, right-padding odd-length input with a
// trailing zero nibble as the RFC reference implementation does.
// Returns nil on invalid hex.
func hexBytes(s string) []byte {
	if len(s)%2 != 0 {
		s = fmt.Sprintf("%s0", s)
	}

	raw, err := hex.DecodeString(s)
	if err != nil {
		return nil
	}

	return raw
}
