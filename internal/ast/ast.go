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

// TypedParam is a function parameter with an explicit type.
type TypedParam struct {
	Name string
	Type string // "int", "int64", "string", "float64", "bool"
}

// FuncDef is a function definition node.
// If TypedParams is non-empty it overrides ParamNames for the rendered signature.
// If ReturnType is set the renderer uses it instead of picking randomly.
type FuncDef struct {
	Name        string
	ParamNames  []string     // legacy all-int params
	TypedParams []TypedParam // domain-aware typed params (overrides ParamNames when non-empty)
	ReturnType  string       // explicit return type (renderer picks if empty)
	Body        []Statement
}

type ReturnStmt struct{ Value Expression }
type ExprStmt struct{ Expr Expression }
type CommentStmt struct{ Text string }

// SwitchStmt is an integer switch statement.
type SwitchStmt struct {
	Tag     string       // variable name being switched on
	Cases   []SwitchCase
	Default []Statement
}

// SwitchCase is one arm of a SwitchStmt.
type SwitchCase struct {
	Value int64
	Body  []Statement
}

// DeferStmt renders as `defer log.Println("text")`.
type DeferStmt struct {
	Text string
}

// HelperCallStmt renders as `result := helperName(arg1, arg2)`.
// The helper function must be declared elsewhere in the package.
type HelperCallStmt struct {
	Name   string   // helper function name
	Args   []string // int-typed variable names in scope
	Result string   // new local variable for the return value
}

// StructDecl is a top-level struct type declaration.
type StructDecl struct {
	Name   string
	Fields []StructField
}

// StructField is one field inside a StructDecl.
type StructField struct {
	Name string
	Type string
}

// InterfaceDecl is a top-level interface type declaration.
type InterfaceDecl struct {
	Name    string
	Methods []InterfaceMethod
}

// InterfaceMethod is one method signature inside an InterfaceDecl.
type InterfaceMethod struct {
	Name       string
	ParamTypes []string
	ReturnType string
}

func (Assignment) isStatement()      {}
func (IfStmt) isStatement()          {}
func (WhileStmt) isStatement()       {}
func (ForStmt) isStatement()         {}
func (FuncDef) isStatement()         {}
func (ReturnStmt) isStatement()      {}
func (ExprStmt) isStatement()        {}
func (CommentStmt) isStatement()     {}
func (SwitchStmt) isStatement()      {}
func (DeferStmt) isStatement()       {}
func (HelperCallStmt) isStatement()  {}
func (StructDecl) isStatement()      {}
func (InterfaceDecl) isStatement()   {}
