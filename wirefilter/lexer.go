package wirefilter

import (
	"fmt"
	"net"
	"strconv"
	"strings"
)

// Lexer tokenizes filter expression strings into tokens.
type Lexer struct {
	input string
	pos   int
	ch    byte
}

// NewLexer creates a new lexer for the given input string.
func NewLexer(input string) *Lexer {
	l := &Lexer{input: input}
	l.readChar()
	return l
}

func (l *Lexer) readChar() {
	if l.pos >= len(l.input) {
		l.ch = 0
	} else {
		l.ch = l.input[l.pos]
	}
	l.pos++
}

func (l *Lexer) peekChar() byte {
	if l.pos >= len(l.input) {
		return 0
	}
	return l.input[l.pos]
}

func (l *Lexer) skipWhitespace() {
	for l.ch == ' ' || l.ch == '\t' || l.ch == '\n' || l.ch == '\r' {
		l.readChar()
	}
}

// readOperatorToken handles multi-character operators.
func (l *Lexer) readOperatorToken() (Token, bool) {
	switch l.ch {
	case '=':
		if l.peekChar() == '=' {
			l.readChar()
			if l.peekChar() == '=' {
				l.readChar()
				return Token{Type: TokenAllEq, Literal: "==="}, true
			}
			return Token{Type: TokenEq, Literal: "=="}, true
		}
	case '!':
		if l.peekChar() == '=' {
			l.readChar()
			if l.peekChar() == '=' {
				l.readChar()
				return Token{Type: TokenAnyNe, Literal: "!=="}, true
			}
			return Token{Type: TokenNe, Literal: "!="}, true
		}
		return Token{Type: TokenNot, Literal: "!"}, true
	case '<':
		if l.peekChar() == '=' {
			l.readChar()
			return Token{Type: TokenLe, Literal: "<="}, true
		}
		return Token{Type: TokenLt, Literal: "<"}, true
	case '>':
		if l.peekChar() == '=' {
			l.readChar()
			return Token{Type: TokenGe, Literal: ">="}, true
		}
		return Token{Type: TokenGt, Literal: ">"}, true
	case '&':
		if l.peekChar() == '&' {
			l.readChar()
			return Token{Type: TokenAnd, Literal: "&&"}, true
		}
	case '|':
		if l.peekChar() == '|' {
			l.readChar()
			return Token{Type: TokenOr, Literal: "||"}, true
		}
	case '^':
		if l.peekChar() == '^' {
			l.readChar()
			return Token{Type: TokenXor, Literal: "^^"}, true
		}
	case '.':
		if l.peekChar() == '.' {
			l.readChar()
			return Token{Type: TokenRange, Literal: ".."}, true
		}
	}
	return Token{}, false
}

// NextToken returns the next token from the input.
func (l *Lexer) NextToken() Token {
	l.skipWhitespace()

	if tok, ok := l.readOperatorToken(); ok {
		l.readChar()
		return tok
	}

	var tok Token

	switch l.ch {
	case 0:
		tok = Token{Type: TokenEOF}
	case '~':
		tok = Token{Type: TokenMatches, Literal: "~"}
	case '(':
		tok = Token{Type: TokenLParen, Literal: "("}
	case ')':
		tok = Token{Type: TokenRParen, Literal: ")"}
	case '{':
		tok = Token{Type: TokenLBrace, Literal: "{"}
	case '}':
		tok = Token{Type: TokenRBrace, Literal: "}"}
	case '[':
		tok = Token{Type: TokenLBracket, Literal: "["}
	case ']':
		tok = Token{Type: TokenRBracket, Literal: "]"}
	case ',':
		tok = Token{Type: TokenComma, Literal: ","}
	case '"':
		tok.Type = TokenString
		tok.Literal = l.readString()
		tok.Value = tok.Literal
	default:
		switch {
		case isLetter(l.ch):
			return l.readIdentifierToken()
		case isDigit(l.ch) || (l.ch == '-' && isDigit(l.peekChar())):
			return l.readNumberToken()
		default:
			tok = Token{Type: TokenEOF, Literal: string(l.ch)}
		}
	}

	l.readChar()
	return tok
}

func (l *Lexer) readString() string {
	l.readChar()
	start := l.pos - 1

	// Fast path: check if string has no escape sequences
	hasEscape := false
	for l.ch != '"' && l.ch != 0 {
		if l.ch == '\\' {
			hasEscape = true
			break
		}
		l.readChar()
	}

	// If no escapes, return substring directly (zero allocation)
	if !hasEscape {
		return l.input[start : l.pos-1]
	}

	// Slow path: handle escape sequences
	var result strings.Builder
	result.Grow(l.pos - start + 16) // Pre-allocate with estimate

	// Copy what we've already scanned
	result.WriteString(l.input[start : l.pos-1])

	// Continue scanning with escape handling
	for l.ch != '"' && l.ch != 0 {
		if l.ch == '\\' {
			l.readChar()
			switch l.ch {
			case 'n':
				result.WriteByte('\n')
			case 't':
				result.WriteByte('\t')
			case 'r':
				result.WriteByte('\r')
			case '\\':
				result.WriteByte('\\')
			case '"':
				result.WriteByte('"')
			default:
				result.WriteByte(l.ch)
			}
		} else {
			result.WriteByte(l.ch)
		}
		l.readChar()
	}
	return result.String()
}

func (l *Lexer) readIdentifier() string {
	start := l.pos - 1
	for isLetter(l.ch) || isDigit(l.ch) || l.ch == '.' || l.ch == '_' || l.ch == '-' || l.ch == ':' || l.ch == '/' {
		l.readChar()
	}
	return l.input[start : l.pos-1]
}

func (l *Lexer) readNumber() string {
	start := l.pos - 1
	if l.ch == '-' {
		l.readChar()
	}
	for isDigit(l.ch) {
		l.readChar()
	}
	return l.input[start : l.pos-1]
}

// isLetter checks if the byte is an ASCII letter (fast path for common case).
func isLetter(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z')
}

// isDigit checks if the byte is an ASCII digit (fast path for common case).
func isDigit(ch byte) bool {
	return ch >= '0' && ch <= '9'
}

func (l *Lexer) readIdentifierToken() Token {
	literal := l.readIdentifier()
	tok := Token{Literal: literal}

	// Fast case-insensitive keyword matching
	lower := strings.ToLower(literal)
	switch lower {
	case "and":
		tok.Type = TokenAnd
	case "or":
		tok.Type = TokenOr
	case "not":
		tok.Type = TokenNot
	case "contains":
		tok.Type = TokenContains
	case "matches":
		tok.Type = TokenMatches
	case "in":
		tok.Type = TokenIn
	case "xor":
		tok.Type = TokenXor
	case "wildcard":
		tok.Type = TokenWildcard
	case "strict":
		// Look ahead for "wildcard" to form "strict wildcard"
		// Save position for potential rollback
		savedPos := l.pos
		savedCh := l.ch
		l.skipWhitespace()
		if l.ch != 0 && isLetter(l.ch) {
			startPos := l.pos - 1
			for isLetter(l.ch) || isDigit(l.ch) || l.ch == '.' || l.ch == '_' || l.ch == '-' || l.ch == ':' || l.ch == '/' {
				l.readChar()
			}
			nextLiteral := l.input[startPos : l.pos-1]
			if strings.ToLower(nextLiteral) == "wildcard" {
				tok.Type = TokenStrictWildcard
				tok.Literal = "strict wildcard"
				return tok
			}
			// Not "wildcard", restore position and treat "strict" as identifier
			l.pos = savedPos
			l.ch = savedCh
		} else {
			// No following identifier, restore position
			l.pos = savedPos
			l.ch = savedCh
		}
		tok.Type = TokenIdent
		tok.Value = literal
	case "true":
		tok.Type = TokenBool
		tok.Value = true
	case "false":
		tok.Type = TokenBool
		tok.Value = false
	default:
		// Only try to parse as IP if it looks like one (starts with digit or contains colon for IPv6)
		if looksLikeIP(literal) {
			if ip := net.ParseIP(literal); ip != nil {
				tok.Type = TokenIP
				tok.Value = ip
				return tok
			}
		}
		tok.Type = TokenIdent
		tok.Value = literal
	}
	return tok
}

// looksLikeIP returns true if the literal might be an IP address.
// This is a fast heuristic to avoid calling net.ParseIP on every identifier.
func looksLikeIP(s string) bool {
	if len(s) == 0 {
		return false
	}
	// IPv4 starts with a digit, IPv6 can start with a digit or letter followed by colons
	firstChar := s[0]
	if firstChar >= '0' && firstChar <= '9' {
		// Could be IPv4 or IPv6 starting with digit
		return true
	}
	// Check for IPv6 with :: or hex prefix
	for i := 0; i < len(s); i++ {
		if s[i] == ':' {
			return true
		}
	}
	return false
}

func (l *Lexer) readNumberToken() Token {
	literal := l.readNumber()
	val, _ := strconv.ParseInt(literal, 10, 64)
	return Token{
		Type:    TokenInt,
		Literal: literal,
		Value:   val,
	}
}

// Error creates a formatted error with the current lexer position.
func (l *Lexer) Error(format string, args ...interface{}) error {
	return fmt.Errorf("lexer error at position %d: %s", l.pos, fmt.Sprintf(format, args...))
}
