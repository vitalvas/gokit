package wirefilter

// Node is the base interface for all AST nodes.
type Node interface {
	node()
}

// Expression represents an expression in the AST.
type Expression interface {
	Node
	expression()
}

// BinaryExpr represents a binary expression (e.g., left == right, left and right).
type BinaryExpr struct {
	Left     Expression
	Operator TokenType
	Right    Expression
}

func (b *BinaryExpr) node()       {}
func (b *BinaryExpr) expression() {}

// UnaryExpr represents a unary expression (e.g., not expr).
type UnaryExpr struct {
	Operator TokenType
	Operand  Expression
}

func (u *UnaryExpr) node()       {}
func (u *UnaryExpr) expression() {}

// FieldExpr represents a field reference (e.g., http.host).
type FieldExpr struct {
	Name string
}

func (f *FieldExpr) node()       {}
func (f *FieldExpr) expression() {}

// LiteralExpr represents a literal value (e.g., "example.com", 42, true).
type LiteralExpr struct {
	Value Value
}

func (l *LiteralExpr) node()       {}
func (l *LiteralExpr) expression() {}

// ArrayExpr represents an array literal (e.g., {1, 2, 3}).
type ArrayExpr struct {
	Elements []Expression
}

func (a *ArrayExpr) node()       {}
func (a *ArrayExpr) expression() {}

// RangeExpr represents a range expression (e.g., 1..10).
type RangeExpr struct {
	Start Expression
	End   Expression
}

func (r *RangeExpr) node()       {}
func (r *RangeExpr) expression() {}

// IndexExpr represents an index expression for map access (e.g., user.attributes["region"]).
type IndexExpr struct {
	Object Expression // The object being indexed (typically a FieldExpr)
	Index  Expression // The index key (typically a LiteralExpr with string value)
}

func (i *IndexExpr) node()       {}
func (i *IndexExpr) expression() {}
