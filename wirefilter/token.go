package wirefilter

// TokenType represents the type of a token in the filter language.
type TokenType uint8

const (
	TokenEOF TokenType = iota
	TokenIdent
	TokenString
	TokenInt
	TokenBool
	TokenIP

	// Comparison operators
	TokenEq    // ==
	TokenNe    // !=
	TokenAllEq // ===
	TokenAnyNe // !==
	TokenLt    // <
	TokenGt    // >
	TokenLe    // <=
	TokenGe    // >=

	// Logical operators
	TokenAnd // and, &&
	TokenOr  // or, ||
	TokenNot // not

	// Membership operators
	TokenContains // contains
	TokenMatches  // matches
	TokenIn       // in

	// Delimiters
	TokenLParen   // (
	TokenRParen   // )
	TokenLBrace   // {
	TokenRBrace   // }
	TokenLBracket // [
	TokenRBracket // ]

	// Separators
	TokenComma // ,
	TokenRange // ..
)

var tokenNames = map[TokenType]string{
	TokenEOF:      "EOF",
	TokenIdent:    "IDENT",
	TokenString:   "STRING",
	TokenInt:      "INT",
	TokenBool:     "BOOL",
	TokenIP:       "IP",
	TokenEq:       "==",
	TokenNe:       "!=",
	TokenAllEq:    "===",
	TokenAnyNe:    "!==",
	TokenLt:       "<",
	TokenGt:       ">",
	TokenLe:       "<=",
	TokenGe:       ">=",
	TokenAnd:      "&&",
	TokenOr:       "||",
	TokenNot:      "not",
	TokenContains: "contains",
	TokenMatches:  "matches",
	TokenIn:       "in",
	TokenLParen:   "(",
	TokenRParen:   ")",
	TokenLBrace:   "{",
	TokenRBrace:   "}",
	TokenLBracket: "[",
	TokenRBracket: "]",
	TokenComma:    ",",
	TokenRange:    "..",
}

// String returns the string representation of a token type.
func (t TokenType) String() string {
	if name, ok := tokenNames[t]; ok {
		return name
	}
	return "UNKNOWN"
}

// Token represents a lexical token in the filter language.
type Token struct {
	Type    TokenType
	Literal string
	Value   interface{}
}
