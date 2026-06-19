package render

import (
	"fmt"
	"math/rand/v2"
	"strings"

	"github.com/exey/fakecodegen/internal/ast"
)

var goTypes = []string{"int64", "float64", "string", "bool", "int", "uint64", "any"}
var goReturnTypes = []string{"int64", "float64", "string", "bool", "any", "error"}

// GoRenderer renders AST to Go source.
type GoRenderer struct {
	w          IndentWriter
	inFunction bool
}

func NewGoRenderer() *GoRenderer {
	return &GoRenderer{w: NewIndentWriter("\t")}
}

func randGoType() string    { return goTypes[rand.IntN(len(goTypes))] }
func randGoReturn() string  { return goReturnTypes[rand.IntN(len(goReturnTypes))] }

func (r *GoRenderer) renderExpr(expr ast.Expression) string {
	switch e := expr.(type) {
	case ast.ArithExpr:
		if e.Op == ast.Power {
			return fmt.Sprintf("(%s + %s)", r.renderExpr(e.Left), r.renderExpr(e.Right))
		}
		return fmt.Sprintf("(%s %s %s)", r.renderExpr(e.Left), goArithOp(e.Op), r.renderExpr(e.Right))
	case ast.UnaryExpr:
		switch e.Op {
		case ast.Negate:
			return fmt.Sprintf("(-%s)", r.renderExpr(e.Operand))
		default:
			return fmt.Sprintf("(^%s)", r.renderExpr(e.Operand))
		}
	case ast.BoolExpr:
		op := "&&"
		if e.Op == ast.Or {
			op = "||"
		}
		return fmt.Sprintf("(%s %s %s)", r.renderExpr(e.Left), op, r.renderExpr(e.Right))
	case ast.CondExpr:
		return fmt.Sprintf("(%s %s %s)", r.renderExpr(e.Left), renderCondOp(e.Op), r.renderExpr(e.Right))
	case ast.FuncCallExpr:
		return fmt.Sprintf("%s()", e.Name)
	case ast.VarExpr:
		return e.Name
	case ast.StringLit:
		return fmt.Sprintf("%q", e.Value)
	case ast.IntLit:
		return fmt.Sprintf("%d", e.Value)
	case ast.FloatLit:
		return fmt.Sprintf("%.4f", e.Value)
	case ast.BoolLit:
		if e.Value {
			return "true"
		}
		return "false"
	}
	return "_"
}

func (r *GoRenderer) renderBlock(block []ast.Statement, out *strings.Builder) {
	for _, stmt := range block {
		r.renderStmt(stmt, out)
	}
}

func (r *GoRenderer) renderStmt(stmt ast.Statement, out *strings.Builder) {
	switch s := stmt.(type) {
	case ast.Assignment:
		if r.inFunction {
			r.w.Line(out, fmt.Sprintf("%s := %s", s.Name, r.renderExpr(s.Value)))
		} else {
			r.w.Line(out, fmt.Sprintf("var %s %s = %s", s.Name, randGoType(), r.renderExpr(s.Value)))
		}
	case ast.IfStmt:
		r.w.Line(out, fmt.Sprintf("if %s {", r.renderExpr(s.Cond)))
		r.w.Inc()
		r.renderBlock(s.Then, out)
		r.w.Dec()
		if len(s.Else) > 0 {
			r.w.Line(out, "} else {")
			r.w.Inc()
			r.renderBlock(s.Else, out)
			r.w.Dec()
		}
		r.w.Line(out, "}")
	case ast.WhileStmt:
		r.w.Line(out, fmt.Sprintf("for %s {", r.renderExpr(s.Cond)))
		r.w.Inc()
		r.renderBlock(s.Body, out)
		r.w.Dec()
		r.w.Line(out, "}")
	case ast.ForStmt:
		r.w.Line(out, fmt.Sprintf("for %s := range %s {", s.Var, r.renderExpr(s.Range)))
		r.w.Inc()
		r.renderBlock(s.Body, out)
		r.w.Dec()
		r.w.Line(out, "}")
	case ast.FuncDef:
		ret := randGoReturn()
		r.w.Line(out, fmt.Sprintf("func %s() %s {", s.Name, ret))
		r.w.Inc()
		prev := r.inFunction
		r.inFunction = true
		r.renderBlock(s.Body, out)
		r.inFunction = prev
		r.w.Dec()
		r.w.Line(out, "}")
		r.w.Blank(out)
	case ast.ReturnStmt:
		r.w.Line(out, fmt.Sprintf("return %s", r.renderExpr(s.Value)))
	case ast.ExprStmt:
		r.w.Line(out, fmt.Sprintf("_ = %s", r.renderExpr(s.Expr)))
	case ast.CommentStmt:
		r.w.Line(out, fmt.Sprintf("// %s", s.Text))
	}
}

// RenderProgram renders a full Go source file.
// Top-level FuncDefs are emitted as package-level functions.
// All other statements are collected into an init() function.
func (r *GoRenderer) RenderProgram(program []ast.Statement) string {
	var out strings.Builder
	out.WriteString("package generated\n\n")

	var topFuncs []ast.Statement
	var initStmts []ast.Statement
	for _, stmt := range program {
		if _, ok := stmt.(ast.FuncDef); ok {
			topFuncs = append(topFuncs, stmt)
		} else {
			initStmts = append(initStmts, stmt)
		}
	}

	for _, stmt := range topFuncs {
		r.inFunction = false
		r.renderStmt(stmt, &out)
	}

	if len(initStmts) > 0 {
		out.WriteString("func init() {\n")
		r.inFunction = true
		r.w.Inc()
		for _, stmt := range initStmts {
			r.renderStmt(stmt, &out)
		}
		r.w.Dec()
		out.WriteString("}\n")
	}

	return out.String()
}

func goArithOp(op ast.ArithmeticOperator) string {
	switch op {
	case ast.Add:
		return "+"
	case ast.Subtract:
		return "-"
	case ast.Multiply:
		return "*"
	case ast.Divide:
		return "/"
	case ast.Modulo:
		return "%"
	case ast.BitwiseAnd:
		return "&"
	case ast.BitwiseOr:
		return "|"
	case ast.BitwiseXor:
		return "^"
	case ast.ShiftLeft:
		return "<<"
	case ast.ShiftRight:
		return ">>"
	}
	return "+"
}
