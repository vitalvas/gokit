package wirefilter

import (
	"fmt"
	"net"
)

// Operator precedence levels for parsing expressions.
// Precedence order (lowest to highest): OR < XOR < AND < EQUALS < COMPARE < MEMBERSHIP
const (
	_ int = iota
	LOWEST
	OR
	XOR
	AND
	EQUALS
	COMPARE
	MEMBERSHIP
)

var precedences = map[TokenType]int{
	TokenOr:             OR,
	TokenXor:            XOR,
	TokenAnd:            AND,
	TokenEq:             EQUALS,
	TokenNe:             EQUALS,
	TokenAllEq:          EQUALS,
	TokenAnyNe:          EQUALS,
	TokenLt:             COMPARE,
	TokenGt:             COMPARE,
	TokenLe:             COMPARE,
	TokenGe:             COMPARE,
	TokenContains:       MEMBERSHIP,
	TokenMatches:        MEMBERSHIP,
	TokenIn:             MEMBERSHIP,
	TokenWildcard:       MEMBERSHIP,
	TokenStrictWildcard: MEMBERSHIP,
}

// Parser parses tokens from a lexer into an abstract syntax tree.
type Parser struct {
	lexer     *Lexer
	curToken  Token
	peekToken Token
	errors    []string
}

// NewParser creates a new parser for the given lexer.
func NewParser(lexer *Lexer) *Parser {
	p := &Parser{lexer: lexer}
	p.nextToken()
	p.nextToken()
	return p
}

func (p *Parser) nextToken() {
	p.curToken = p.peekToken
	p.peekToken = p.lexer.NextToken()
}

// Errors returns the list of parsing errors encountered.
func (p *Parser) Errors() []string {
	return p.errors
}

func (p *Parser) addError(format string, args ...interface{}) {
	p.errors = append(p.errors, fmt.Sprintf(format, args...))
}

// Parse parses the input and returns an expression tree.
// Returns an error if parsing fails or if there is trailing input.
func (p *Parser) Parse() (Expression, error) {
	expr := p.parseExpression(LOWEST)

	// Check for trailing tokens (garbage after valid expression)
	if p.peekToken.Type == TokenError {
		if errMsg, ok := p.peekToken.Value.(string); ok {
			p.addError("lexer error: %s", errMsg)
		} else {
			p.addError("lexer error at: %s", p.peekToken.Literal)
		}
	} else if p.peekToken.Type != TokenEOF {
		p.addError("unexpected trailing token: %s", p.peekToken.Type)
	}

	if len(p.errors) > 0 {
		return nil, fmt.Errorf("parse errors: %v", p.errors)
	}
	return expr, nil
}

func (p *Parser) parseExpression(precedence int) Expression {
	var left Expression

	switch p.curToken.Type {
	case TokenError:
		// Propagate lexer error
		if errMsg, ok := p.curToken.Value.(string); ok {
			p.addError("lexer error: %s", errMsg)
		} else {
			p.addError("lexer error at: %s", p.curToken.Literal)
		}
		return nil
	case TokenNot:
		left = p.parseUnaryExpression()
	case TokenLParen:
		left = p.parseGroupedExpression()
	case TokenIdent:
		left = p.parseFieldExpression()
	case TokenString:
		left = p.parseLiteralExpression()
	case TokenRawString:
		left = p.parseLiteralExpression()
	case TokenInt:
		left = p.parseLiteralExpression()
	case TokenBool:
		left = p.parseLiteralExpression()
	case TokenIP:
		left = p.parseLiteralExpression()
	case TokenListRef:
		left = p.parseListRefExpression()
	default:
		p.addError("unexpected token: %s", p.curToken.Type)
		return nil
	}

	for p.peekToken.Type != TokenEOF && precedence < p.peekPrecedence() {
		p.nextToken()
		left = p.parseBinaryExpression(left)
	}

	return left
}

func (p *Parser) parseUnaryExpression() Expression {
	operator := p.curToken.Type
	p.nextToken()
	operand := p.parseExpression(LOWEST)
	return &UnaryExpr{
		Operator: operator,
		Operand:  operand,
	}
}

func (p *Parser) parseGroupedExpression() Expression {
	p.nextToken()
	expr := p.parseExpression(LOWEST)
	if p.peekToken.Type != TokenRParen {
		p.addError("expected ), got %s", p.peekToken.Type)
		return nil
	}
	p.nextToken()
	return expr
}

func (p *Parser) parseFieldExpression() Expression {
	name := p.curToken.Literal

	// Check if this is a function call (identifier followed by '(')
	if p.peekToken.Type == TokenLParen {
		expr := p.parseFunctionCallExpression(name)
		// Check for array index on function result: func()[0]
		if p.peekToken.Type == TokenLBracket {
			return p.parseIndexExpression(expr)
		}
		return expr
	}

	field := &FieldExpr{Name: name}

	if p.peekToken.Type == TokenLBracket {
		return p.parseIndexExpression(field)
	}

	return field
}

func (p *Parser) parseFunctionCallExpression(name string) Expression {
	p.nextToken() // consume '('
	p.nextToken() // move to first argument or ')'

	args := []Expression{}

	// Handle empty argument list
	if p.curToken.Type == TokenRParen {
		return &FunctionCallExpr{Name: name, Arguments: args}
	}

	// Parse first argument
	arg := p.parseExpression(LOWEST)
	args = append(args, arg)

	// Parse remaining arguments
	for p.peekToken.Type == TokenComma {
		p.nextToken() // consume ','
		p.nextToken() // move to next argument
		arg = p.parseExpression(LOWEST)
		args = append(args, arg)
	}

	if p.peekToken.Type != TokenRParen {
		p.addError("expected ), got %s", p.peekToken.Type)
		return nil
	}
	p.nextToken() // consume ')'

	return &FunctionCallExpr{Name: name, Arguments: args}
}

func (p *Parser) parseIndexExpression(object Expression) Expression {
	p.nextToken() // consume [

	p.nextToken() // move to the index expression

	// Check for array unpack [*]
	if p.curToken.Type == TokenAsterisk {
		if p.peekToken.Type != TokenRBracket {
			p.addError("expected ], got %s", p.peekToken.Type)
			return nil
		}
		p.nextToken() // consume ]
		return &UnpackExpr{Array: object}
	}

	// Validate index is a literal type (string or int)
	switch p.curToken.Type {
	case TokenString, TokenRawString, TokenInt:
		// Valid index types
	default:
		p.addError("index must be a string or integer literal, got %s", p.curToken.Type)
		return nil
	}

	index := p.parseLiteralExpression()

	if p.peekToken.Type != TokenRBracket {
		p.addError("expected ], got %s", p.peekToken.Type)
		return nil
	}
	p.nextToken() // consume ]

	expr := &IndexExpr{
		Object: object,
		Index:  index,
	}

	// Support chained index expressions like field["a"]["b"]
	if p.peekToken.Type == TokenLBracket {
		return p.parseIndexExpression(expr)
	}

	return expr
}

func (p *Parser) parseLiteralExpression() Expression {
	var value Value

	switch p.curToken.Type {
	case TokenString:
		value = StringValue(p.curToken.Literal)
	case TokenRawString:
		value = StringValue(p.curToken.Literal)
	case TokenInt:
		value = IntValue(p.curToken.Value.(int64))
	case TokenBool:
		value = BoolValue(p.curToken.Value.(bool))
	case TokenIP:
		value = IPValue{IP: p.curToken.Value.(net.IP)}
	}

	return &LiteralExpr{Value: value}
}

func (p *Parser) parseListRefExpression() Expression {
	return &ListRefExpr{Name: p.curToken.Literal}
}

func (p *Parser) parseBinaryExpression(left Expression) Expression {
	operator := p.curToken.Type
	precedence := p.curPrecedence()

	if operator == TokenIn || operator == TokenContains {
		p.nextToken()
		var right Expression
		if p.curToken.Type == TokenLBrace {
			right = p.parseArrayExpression()
		} else {
			right = p.parseExpression(precedence)
		}
		return &BinaryExpr{
			Left:     left,
			Operator: operator,
			Right:    right,
		}
	}

	p.nextToken()
	right := p.parseExpression(precedence)

	return &BinaryExpr{
		Left:     left,
		Operator: operator,
		Right:    right,
	}
}

func (p *Parser) parseArrayExpression() Expression {
	elements := []Expression{}

	p.nextToken()

	if p.curToken.Type == TokenRBrace {
		return &ArrayExpr{Elements: elements}
	}

	element := p.parseExpression(LOWEST)

	if p.peekToken.Type == TokenRange {
		p.nextToken()
		p.nextToken()
		end := p.parseExpression(LOWEST)
		element = &RangeExpr{Start: element, End: end}
	}

	elements = append(elements, element)

	for p.peekToken.Type == TokenComma {
		p.nextToken()
		p.nextToken()

		element = p.parseExpression(LOWEST)

		if p.peekToken.Type == TokenRange {
			p.nextToken()
			p.nextToken()
			end := p.parseExpression(LOWEST)
			element = &RangeExpr{Start: element, End: end}
		}

		elements = append(elements, element)
	}

	if p.peekToken.Type != TokenRBrace {
		p.addError("expected }, got %s", p.peekToken.Type)
		return nil
	}

	p.nextToken()

	return &ArrayExpr{Elements: elements}
}

func (p *Parser) curPrecedence() int {
	if p, ok := precedences[p.curToken.Type]; ok {
		return p
	}
	return LOWEST
}

func (p *Parser) peekPrecedence() int {
	if p, ok := precedences[p.peekToken.Type]; ok {
		return p
	}
	return LOWEST
}
