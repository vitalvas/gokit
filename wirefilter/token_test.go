package wirefilter

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTokenTypeString(t *testing.T) {
	t.Run("known token types", func(t *testing.T) {
		tests := []struct {
			tokenType TokenType
			expected  string
		}{
			{TokenEOF, "EOF"},
			{TokenIdent, "IDENT"},
			{TokenString, "STRING"},
			{TokenInt, "INT"},
			{TokenBool, "BOOL"},
			{TokenIP, "IP"},
			{TokenEq, "=="},
			{TokenNe, "!="},
			{TokenAllEq, "==="},
			{TokenAnyNe, "!=="},
			{TokenLt, "<"},
			{TokenGt, ">"},
			{TokenLe, "<="},
			{TokenGe, ">="},
			{TokenAnd, "&&"},
			{TokenOr, "||"},
			{TokenNot, "not"},
			{TokenContains, "contains"},
			{TokenMatches, "matches"},
			{TokenIn, "in"},
			{TokenLParen, "("},
			{TokenRParen, ")"},
			{TokenLBrace, "{"},
			{TokenRBrace, "}"},
			{TokenComma, ","},
			{TokenRange, ".."},
		}

		for _, tt := range tests {
			assert.Equal(t, tt.expected, tt.tokenType.String())
		}
	})

	t.Run("unknown token type", func(t *testing.T) {
		unknownToken := TokenType(255)
		assert.Equal(t, "UNKNOWN", unknownToken.String())
	})
}
