package wirefilter

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"unicode"
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

// NextToken returns the next token from the input.
func (l *Lexer) NextToken() Token {
	l.skipWhitespace()

	var tok Token

	switch l.ch {
	case 0:
		tok = Token{Type: TokenEOF}
	case '=':
		if l.peekChar() == '=' {
			l.readChar()
			if l.peekChar() == '=' {
				l.readChar()
				tok = Token{Type: TokenAllEq, Literal: "==="}
			} else {
				tok = Token{Type: TokenEq, Literal: "=="}
			}
		}
	case '!':
		if l.peekChar() == '=' {
			l.readChar()
			if l.peekChar() == '=' {
				l.readChar()
				tok = Token{Type: TokenAnyNe, Literal: "!=="}
			} else {
				tok = Token{Type: TokenNe, Literal: "!="}
			}
		}
	case '<':
		if l.peekChar() == '=' {
			l.readChar()
			tok = Token{Type: TokenLe, Literal: "<="}
		} else {
			tok = Token{Type: TokenLt, Literal: "<"}
		}
	case '>':
		if l.peekChar() == '=' {
			l.readChar()
			tok = Token{Type: TokenGe, Literal: ">="}
		} else {
			tok = Token{Type: TokenGt, Literal: ">"}
		}
	case '&':
		if l.peekChar() == '&' {
			l.readChar()
			tok = Token{Type: TokenAnd, Literal: "&&"}
		}
	case '|':
		if l.peekChar() == '|' {
			l.readChar()
			tok = Token{Type: TokenOr, Literal: "||"}
		}
	case '(':
		tok = Token{Type: TokenLParen, Literal: "("}
	case ')':
		tok = Token{Type: TokenRParen, Literal: ")"}
	case '{':
		tok = Token{Type: TokenLBrace, Literal: "{"}
	case '}':
		tok = Token{Type: TokenRBrace, Literal: "}"}
	case ',':
		tok = Token{Type: TokenComma, Literal: ","}
	case '.':
		if l.peekChar() == '.' {
			l.readChar()
			tok = Token{Type: TokenRange, Literal: ".."}
		}
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
	var result strings.Builder
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

func isLetter(ch byte) bool {
	return unicode.IsLetter(rune(ch))
}

func isDigit(ch byte) bool {
	return unicode.IsDigit(rune(ch))
}

func (l *Lexer) readIdentifierToken() Token {
	literal := l.readIdentifier()
	tok := Token{Literal: literal}

	switch strings.ToLower(literal) {
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
	case "true", "false":
		tok.Type = TokenBool
		tok.Value = strings.ToLower(literal) == "true"
	default:
		if ip := net.ParseIP(literal); ip != nil {
			tok.Type = TokenIP
			tok.Value = ip
		} else {
			tok.Type = TokenIdent
			tok.Value = literal
		}
	}
	return tok
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
