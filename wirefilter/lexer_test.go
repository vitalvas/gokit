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
		input := "== != === !== < > <= >= && || and or not"
		lexer := NewLexer(input)

		tests := []TokenType{
			TokenEq, TokenNe, TokenAllEq, TokenAnyNe, TokenLt, TokenGt, TokenLe, TokenGe,
			TokenAnd, TokenOr, TokenAnd, TokenOr, TokenNot, TokenEOF,
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
}
