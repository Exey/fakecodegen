package ast

type ArithmeticOperator int

const (
	Add ArithmeticOperator = iota
	Subtract
	Multiply
	Divide
	Modulo
	Power
	BitwiseAnd
	BitwiseOr
	BitwiseXor
	ShiftLeft
	ShiftRight
)

type BooleanOperator int

const (
	And BooleanOperator = iota
	Or
)

type UnaryOperator int

const (
	Negate UnaryOperator = iota
	BitwiseNot
)

type Condition int

const (
	Equals Condition = iota
	NotEquals
	GreaterThan
	LessThan
	GreaterThanOrEqual
	LessThanOrEqual
)

// Expression is the expression node interface.
type Expression interface{ isExpression() }

type ArithExpr struct {
	Left  Expression
	Op    ArithmeticOperator
	Right Expression
}

type UnaryExpr struct {
	Op      UnaryOperator
	Operand Expression
}

type BoolExpr struct {
	Left  Expression
	Op    BooleanOperator
	Right Expression
}

type CondExpr struct {
	Left  Expression
	Op    Condition
	Right Expression
}

type FuncCallExpr struct{ Name string }
type VarExpr struct{ Name string }
type StringLit struct{ Value string }
type IntLit struct{ Value int64 }
type FloatLit struct{ Value float64 }
type BoolLit struct{ Value bool }

func (ArithExpr) isExpression()    {}
func (UnaryExpr) isExpression()    {}
func (BoolExpr) isExpression()     {}
func (CondExpr) isExpression()     {}
func (FuncCallExpr) isExpression() {}
func (VarExpr) isExpression()      {}
func (StringLit) isExpression()    {}
func (IntLit) isExpression()       {}
func (FloatLit) isExpression()     {}
func (BoolLit) isExpression()      {}

// Statement is the statement node interface.
type Statement interface{ isStatement() }

type Assignment struct {
	Name   string
	Value  Expression
	IsDecl bool // true = new variable declaration (:= / let / var); false = reassignment (=)
}

type IfStmt struct {
	Cond Expression
	Then []Statement
	Else []Statement
}

type WhileStmt struct {
	Cond Expression
	Body []Statement
}

type ForStmt struct {
	Var   string
	Range Expression
	Body  []Statement
}

type FuncDef struct {
	Name       string
	ParamNames []string
	Body       []Statement
}

type ReturnStmt struct{ Value Expression }
type ExprStmt struct{ Expr Expression }
type CommentStmt struct{ Text string }

func (Assignment) isStatement()  {}
func (IfStmt) isStatement()      {}
func (WhileStmt) isStatement()   {}
func (ForStmt) isStatement()     {}
func (FuncDef) isStatement()     {}
func (ReturnStmt) isStatement()  {}
func (ExprStmt) isStatement()    {}
func (CommentStmt) isStatement() {}
