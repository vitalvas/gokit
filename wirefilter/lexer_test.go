package wirefilter

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func BenchmarkLexer(b *testing.B) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "simple expression",
			input: `http.host == "example.com"`,
		},
		{
			name:  "complex expression",
			input: `http.host == "example.com" and http.status >= 400 or http.path contains "/api"`,
		},
		{
			name:  "array expression",
			input: `http.status in {200, 201, 204, 301, 302, 304}`,
		},
		{
			name:  "range expression",
			input: `port in {80..100, 443, 8000..9000}`,
		},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				lexer := NewLexer(tt.input)
				for {
					tok := lexer.NextToken()
					if tok.Type == TokenEOF {
						break
					}
				}
			}
		})
	}
}

func FuzzLexer(f *testing.F) {
	f.Add(`http.host == "example.com"`)
	f.Add(`http.status >= 400`)
	f.Add(`http.host == "example.com" and http.status >= 400`)
	f.Add(`(http.host == "test.com" or http.path contains "/api") and http.status < 500`)
	f.Add(`http.status in {200, 201, 204, 301, 302, 304}`)
	f.Add(`port in {80..100, 443, 8000..9000}`)
	f.Add(`ip.src in "192.168.0.0/16"`)
	f.Add(`http.path matches "^/api/v[0-9]+/"`)
	f.Add(`not http.host == "blocked.com"`)
	f.Add(`true and false`)
	f.Add(`""`)
	f.Add(`"string with \"escape\""`)
	f.Add(`field === "value"`)
	f.Add(`field !== "value"`)

	f.Fuzz(func(_ *testing.T, input string) {
		lexer := NewLexer(input)
		for {
			tok := lexer.NextToken()
			if tok.Type == TokenEOF {
				break
			}
		}
	})
}

func TestLexer(t *testing.T) {
	t.Run("operators", func(t *testing.T) {
		input := "== != === !== < > <= >= && || and or not ^^ xor ~ !"
		lexer := NewLexer(input)

		tests := []TokenType{
			TokenEq, TokenNe, TokenAllEq, TokenAnyNe, TokenLt, TokenGt, TokenLe, TokenGe,
			TokenAnd, TokenOr, TokenAnd, TokenOr, TokenNot, TokenXor, TokenXor, TokenMatches, TokenNot, TokenEOF,
		}

		for _, expected := range tests {
			tok := lexer.NextToken()
			assert.Equal(t, expected, tok.Type)
		}
	})

	t.Run("keywords", func(t *testing.T) {
		input := "contains matches in"
		lexer := NewLexer(input)

		tests := []TokenType{TokenContains, TokenMatches, TokenIn, TokenEOF}

		for _, expected := range tests {
			tok := lexer.NextToken()
			assert.Equal(t, expected, tok.Type)
		}
	})

	t.Run("literals", func(t *testing.T) {
		input := `"test string" 42 -10 true false`
		lexer := NewLexer(input)

		tok := lexer.NextToken()
		assert.Equal(t, TokenString, tok.Type)
		assert.Equal(t, "test string", tok.Literal)

		tok = lexer.NextToken()
		assert.Equal(t, TokenInt, tok.Type)
		assert.Equal(t, int64(42), tok.Value)

		tok = lexer.NextToken()
		assert.Equal(t, TokenInt, tok.Type)
		assert.Equal(t, int64(-10), tok.Value)

		tok = lexer.NextToken()
		assert.Equal(t, TokenBool, tok.Type)
		assert.Equal(t, true, tok.Value)

		tok = lexer.NextToken()
		assert.Equal(t, TokenBool, tok.Type)
		assert.Equal(t, false, tok.Value)
	})

	t.Run("identifiers", func(t *testing.T) {
		input := "http.method user.name field_name"
		lexer := NewLexer(input)

		tests := []string{"http.method", "user.name", "field_name"}

		for _, expected := range tests {
			tok := lexer.NextToken()
			assert.Equal(t, TokenIdent, tok.Type)
			assert.Equal(t, expected, tok.Literal)
		}
	})

	t.Run("delimiters", func(t *testing.T) {
		input := "( ) { } ,"
		lexer := NewLexer(input)

		tests := []TokenType{
			TokenLParen, TokenRParen, TokenLBrace, TokenRBrace, TokenComma, TokenEOF,
		}

		for _, expected := range tests {
			tok := lexer.NextToken()
			assert.Equal(t, expected, tok.Type)
		}
	})

	t.Run("complex expression", func(t *testing.T) {
		input := `http.method == "GET" && port in {80, 443}`
		lexer := NewLexer(input)

		tok := lexer.NextToken()
		assert.Equal(t, TokenIdent, tok.Type)
		assert.Equal(t, "http.method", tok.Literal)

		tok = lexer.NextToken()
		assert.Equal(t, TokenEq, tok.Type)

		tok = lexer.NextToken()
		assert.Equal(t, TokenString, tok.Type)
		assert.Equal(t, "GET", tok.Literal)

		tok = lexer.NextToken()
		assert.Equal(t, TokenAnd, tok.Type)

		tok = lexer.NextToken()
		assert.Equal(t, TokenIdent, tok.Type)
		assert.Equal(t, "port", tok.Literal)

		tok = lexer.NextToken()
		assert.Equal(t, TokenIn, tok.Type)

		tok = lexer.NextToken()
		assert.Equal(t, TokenLBrace, tok.Type)

		tok = lexer.NextToken()
		assert.Equal(t, TokenInt, tok.Type)
		assert.Equal(t, int64(80), tok.Value)

		tok = lexer.NextToken()
		assert.Equal(t, TokenComma, tok.Type)

		tok = lexer.NextToken()
		assert.Equal(t, TokenInt, tok.Type)
		assert.Equal(t, int64(443), tok.Value)

		tok = lexer.NextToken()
		assert.Equal(t, TokenRBrace, tok.Type)
	})

	t.Run("string escape sequences", func(t *testing.T) {
		input := `"hello\nworld\t\r\\\"test"`
		lexer := NewLexer(input)

		tok := lexer.NextToken()
		assert.Equal(t, TokenString, tok.Type)
		assert.Equal(t, "hello\nworld\t\r\\\"test", tok.Literal)
	})

	t.Run("string with unknown escape", func(t *testing.T) {
		input := `"test\xvalue"`
		lexer := NewLexer(input)

		tok := lexer.NextToken()
		assert.Equal(t, TokenString, tok.Type)
		assert.Equal(t, "testxvalue", tok.Literal)
	})

	t.Run("unterminated string", func(t *testing.T) {
		input := `"unterminated`
		lexer := NewLexer(input)

		tok := lexer.NextToken()
		assert.Equal(t, TokenString, tok.Type)
	})

	t.Run("identifier with colon not ip", func(t *testing.T) {
		input := "field:value"
		lexer := NewLexer(input)

		tok := lexer.NextToken()
		assert.Equal(t, TokenIdent, tok.Type)
		assert.Equal(t, "field:value", tok.Literal)
	})

	t.Run("identifier that looks like ip but isnt", func(t *testing.T) {
		input := "abc:def:ghi"
		lexer := NewLexer(input)

		tok := lexer.NextToken()
		assert.Equal(t, TokenIdent, tok.Type)
	})

	t.Run("range token", func(t *testing.T) {
		input := "1..10"
		lexer := NewLexer(input)

		tok := lexer.NextToken()
		assert.Equal(t, TokenInt, tok.Type)
		assert.Equal(t, int64(1), tok.Value)

		tok = lexer.NextToken()
		assert.Equal(t, TokenRange, tok.Type)

		tok = lexer.NextToken()
		assert.Equal(t, TokenInt, tok.Type)
		assert.Equal(t, int64(10), tok.Value)
	})

	t.Run("error method", func(t *testing.T) {
		lexer := NewLexer("test")
		err := lexer.Error("test error %s", "message")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "lexer error")
		assert.Contains(t, err.Error(), "test error message")
	})

	t.Run("single dot not range", func(t *testing.T) {
		input := "field.name"
		lexer := NewLexer(input)

		tok := lexer.NextToken()
		assert.Equal(t, TokenIdent, tok.Type)
		assert.Equal(t, "field.name", tok.Literal)
	})

	t.Run("looksLikeIP empty string", func(t *testing.T) {
		input := `field == ""`
		lexer := NewLexer(input)

		tok := lexer.NextToken()
		assert.Equal(t, TokenIdent, tok.Type)
	})

	t.Run("peek char at end", func(t *testing.T) {
		input := "a"
		lexer := NewLexer(input)
		lexer.NextToken()
		tok := lexer.NextToken()
		assert.Equal(t, TokenEOF, tok.Type)
	})

	t.Run("single ampersand", func(t *testing.T) {
		input := "a & b"
		lexer := NewLexer(input)

		tok := lexer.NextToken()
		assert.Equal(t, TokenIdent, tok.Type)

		tok = lexer.NextToken()
		assert.Equal(t, TokenEOF, tok.Type)
	})

	t.Run("single pipe", func(t *testing.T) {
		input := "a | b"
		lexer := NewLexer(input)

		tok := lexer.NextToken()
		assert.Equal(t, TokenIdent, tok.Type)

		tok = lexer.NextToken()
		assert.Equal(t, TokenEOF, tok.Type)
	})

	t.Run("single equals", func(t *testing.T) {
		input := "a = b"
		lexer := NewLexer(input)

		tok := lexer.NextToken()
		assert.Equal(t, TokenIdent, tok.Type)

		tok = lexer.NextToken()
		assert.Equal(t, TokenEOF, tok.Type)
	})

	t.Run("single exclamation as not operator", func(t *testing.T) {
		input := "a ! b"
		lexer := NewLexer(input)

		tok := lexer.NextToken()
		assert.Equal(t, TokenIdent, tok.Type)

		tok = lexer.NextToken()
		assert.Equal(t, TokenNot, tok.Type)
		assert.Equal(t, "!", tok.Literal)

		tok = lexer.NextToken()
		assert.Equal(t, TokenIdent, tok.Type)
	})

	t.Run("looksLikeIP with empty string check", func(t *testing.T) {
		result := looksLikeIP("")
		assert.False(t, result)
	})

	t.Run("looksLikeIP with letter only", func(t *testing.T) {
		result := looksLikeIP("hostname")
		assert.False(t, result)
	})

	t.Run("looksLikeIP with digit start", func(t *testing.T) {
		result := looksLikeIP("192")
		assert.True(t, result)
	})

	t.Run("looksLikeIP with colon", func(t *testing.T) {
		result := looksLikeIP("abc:def")
		assert.True(t, result)
	})

	t.Run("single dot", func(t *testing.T) {
		input := "."
		lexer := NewLexer(input)

		tok := lexer.NextToken()
		assert.Equal(t, TokenEOF, tok.Type)
	})

	t.Run("identifier with underscore", func(t *testing.T) {
		input := "field_name"
		lexer := NewLexer(input)

		tok := lexer.NextToken()
		assert.Equal(t, TokenIdent, tok.Type)
		assert.Equal(t, "field_name", tok.Literal)
	})

	t.Run("identifier with hyphen", func(t *testing.T) {
		input := "field-name"
		lexer := NewLexer(input)

		tok := lexer.NextToken()
		assert.Equal(t, TokenIdent, tok.Type)
		assert.Equal(t, "field-name", tok.Literal)
	})

	t.Run("identifier with slash", func(t *testing.T) {
		input := "path/name"
		lexer := NewLexer(input)

		tok := lexer.NextToken()
		assert.Equal(t, TokenIdent, tok.Type)
		assert.Equal(t, "path/name", tok.Literal)
	})

	t.Run("uppercase keywords", func(t *testing.T) {
		input := "AND OR NOT CONTAINS MATCHES IN TRUE FALSE"
		lexer := NewLexer(input)

		tests := []TokenType{TokenAnd, TokenOr, TokenNot, TokenContains, TokenMatches, TokenIn, TokenBool, TokenBool, TokenEOF}

		for _, expected := range tests {
			tok := lexer.NextToken()
			assert.Equal(t, expected, tok.Type)
		}
	})

	t.Run("tilde as matches alias", func(t *testing.T) {
		input := `field ~ "pattern"`
		lexer := NewLexer(input)

		tok := lexer.NextToken()
		assert.Equal(t, TokenIdent, tok.Type)

		tok = lexer.NextToken()
		assert.Equal(t, TokenMatches, tok.Type)
		assert.Equal(t, "~", tok.Literal)

		tok = lexer.NextToken()
		assert.Equal(t, TokenString, tok.Type)
	})

	t.Run("xor operator symbol", func(t *testing.T) {
		input := "a ^^ b"
		lexer := NewLexer(input)

		tok := lexer.NextToken()
		assert.Equal(t, TokenIdent, tok.Type)

		tok = lexer.NextToken()
		assert.Equal(t, TokenXor, tok.Type)
		assert.Equal(t, "^^", tok.Literal)

		tok = lexer.NextToken()
		assert.Equal(t, TokenIdent, tok.Type)
	})

	t.Run("xor operator keyword", func(t *testing.T) {
		input := "a xor b"
		lexer := NewLexer(input)

		tok := lexer.NextToken()
		assert.Equal(t, TokenIdent, tok.Type)

		tok = lexer.NextToken()
		assert.Equal(t, TokenXor, tok.Type)

		tok = lexer.NextToken()
		assert.Equal(t, TokenIdent, tok.Type)
	})

	t.Run("wildcard operator", func(t *testing.T) {
		input := `field wildcard "*.example.com"`
		lexer := NewLexer(input)

		tok := lexer.NextToken()
		assert.Equal(t, TokenIdent, tok.Type)

		tok = lexer.NextToken()
		assert.Equal(t, TokenWildcard, tok.Type)

		tok = lexer.NextToken()
		assert.Equal(t, TokenString, tok.Type)
	})

	t.Run("strict wildcard operator", func(t *testing.T) {
		input := `field strict wildcard "*.Example.com"`
		lexer := NewLexer(input)

		tok := lexer.NextToken()
		assert.Equal(t, TokenIdent, tok.Type)
		assert.Equal(t, "field", tok.Literal)

		tok = lexer.NextToken()
		assert.Equal(t, TokenStrictWildcard, tok.Type)
		assert.Equal(t, "strict wildcard", tok.Literal)

		tok = lexer.NextToken()
		assert.Equal(t, TokenString, tok.Type)
	})

	t.Run("strict alone is identifier", func(t *testing.T) {
		input := "strict"
		lexer := NewLexer(input)

		tok := lexer.NextToken()
		assert.Equal(t, TokenIdent, tok.Type)
		assert.Equal(t, "strict", tok.Literal)
	})

	t.Run("strict followed by non-wildcard is identifier", func(t *testing.T) {
		input := "strict other"
		lexer := NewLexer(input)

		tok := lexer.NextToken()
		assert.Equal(t, TokenIdent, tok.Type)
		assert.Equal(t, "strict", tok.Literal)

		tok = lexer.NextToken()
		assert.Equal(t, TokenIdent, tok.Type)
		assert.Equal(t, "other", tok.Literal)
	})

	t.Run("single caret", func(t *testing.T) {
		input := "a ^ b"
		lexer := NewLexer(input)

		tok := lexer.NextToken()
		assert.Equal(t, TokenIdent, tok.Type)

		tok = lexer.NextToken()
		assert.Equal(t, TokenEOF, tok.Type)
	})

	t.Run("uppercase wildcard keywords", func(t *testing.T) {
		input := "WILDCARD XOR"
		lexer := NewLexer(input)

		tok := lexer.NextToken()
		assert.Equal(t, TokenWildcard, tok.Type)

		tok = lexer.NextToken()
		assert.Equal(t, TokenXor, tok.Type)
	})

	t.Run("strict wildcard case insensitive", func(t *testing.T) {
		input := "STRICT WILDCARD"
		lexer := NewLexer(input)

		tok := lexer.NextToken()
		assert.Equal(t, TokenStrictWildcard, tok.Type)
	})

	t.Run("raw string basic", func(t *testing.T) {
		input := `r"path\to\file"`
		lexer := NewLexer(input)

		tok := lexer.NextToken()
		assert.Equal(t, TokenRawString, tok.Type)
		assert.Equal(t, `path\to\file`, tok.Literal)
		assert.Equal(t, `path\to\file`, tok.Value)
	})

	t.Run("raw string with regex", func(t *testing.T) {
		input := `r"^\d+\.\d+\.\d+\.\d+$"`
		lexer := NewLexer(input)

		tok := lexer.NextToken()
		assert.Equal(t, TokenRawString, tok.Type)
		assert.Equal(t, `^\d+\.\d+\.\d+\.\d+$`, tok.Literal)
	})

	t.Run("raw string empty", func(t *testing.T) {
		input := `r""`
		lexer := NewLexer(input)

		tok := lexer.NextToken()
		assert.Equal(t, TokenRawString, tok.Type)
		assert.Equal(t, "", tok.Literal)
	})

	t.Run("asterisk token", func(t *testing.T) {
		input := "*"
		lexer := NewLexer(input)

		tok := lexer.NextToken()
		assert.Equal(t, TokenAsterisk, tok.Type)
		assert.Equal(t, "*", tok.Literal)
	})

	t.Run("list reference basic", func(t *testing.T) {
		input := "$blocked_ips"
		lexer := NewLexer(input)

		tok := lexer.NextToken()
		assert.Equal(t, TokenListRef, tok.Type)
		assert.Equal(t, "blocked_ips", tok.Literal)
		assert.Equal(t, "blocked_ips", tok.Value)
	})

	t.Run("list reference with hyphen", func(t *testing.T) {
		input := "$admin-roles"
		lexer := NewLexer(input)

		tok := lexer.NextToken()
		assert.Equal(t, TokenListRef, tok.Type)
		assert.Equal(t, "admin-roles", tok.Literal)
	})

	t.Run("array unpack syntax", func(t *testing.T) {
		input := "tags[*]"
		lexer := NewLexer(input)

		tok := lexer.NextToken()
		assert.Equal(t, TokenIdent, tok.Type)
		assert.Equal(t, "tags", tok.Literal)

		tok = lexer.NextToken()
		assert.Equal(t, TokenLBracket, tok.Type)

		tok = lexer.NextToken()
		assert.Equal(t, TokenAsterisk, tok.Type)

		tok = lexer.NextToken()
		assert.Equal(t, TokenRBracket, tok.Type)
	})

	t.Run("array index syntax", func(t *testing.T) {
		input := "tags[0]"
		lexer := NewLexer(input)

		tok := lexer.NextToken()
		assert.Equal(t, TokenIdent, tok.Type)
		assert.Equal(t, "tags", tok.Literal)

		tok = lexer.NextToken()
		assert.Equal(t, TokenLBracket, tok.Type)

		tok = lexer.NextToken()
		assert.Equal(t, TokenInt, tok.Type)
		assert.Equal(t, int64(0), tok.Value)

		tok = lexer.NextToken()
		assert.Equal(t, TokenRBracket, tok.Type)
	})
}
