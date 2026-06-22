package render

import (
	"fmt"
	"math/rand/v2"
	"strings"

	"github.com/exey/fakecodegen/internal/ast"
)

var swiftTypes = []string{"Int", "Int64", "Double", "String", "Bool"}
var swiftReturnTypes = []string{"Int", "Double", "String", "Bool", "Void"}
var swiftVarKeywords = []string{"let", "var"}

// SwiftRenderer renders AST to Swift source.
type SwiftRenderer struct {
	w IndentWriter
}

func NewSwiftRenderer() *SwiftRenderer {
	return &SwiftRenderer{w: NewIndentWriter("    ")}
}

func randSwiftType() string   { return swiftTypes[rand.IntN(len(swiftTypes))] }
func randSwiftReturn() string { return swiftReturnTypes[rand.IntN(len(swiftReturnTypes))] }

func (r *SwiftRenderer) renderExpr(expr ast.Expression) string {
	switch e := expr.(type) {
	case ast.ArithExpr:
		if e.Op == ast.Power {
			return fmt.Sprintf("(%s + %s)", r.renderExpr(e.Left), r.renderExpr(e.Right))
		}
		return fmt.Sprintf("(%s %s %s)", r.renderExpr(e.Left), swiftArithOp(e.Op), r.renderExpr(e.Right))
	case ast.UnaryExpr:
		switch e.Op {
		case ast.Negate:
			return fmt.Sprintf("(-%s)", r.renderExpr(e.Operand))
		default:
			return fmt.Sprintf("(~%s)", r.renderExpr(e.Operand))
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

func (r *SwiftRenderer) renderBlock(block []ast.Statement, out *strings.Builder) {
	for _, stmt := range block {
		r.renderStmt(stmt, out)
	}
}

func (r *SwiftRenderer) renderStmt(stmt ast.Statement, out *strings.Builder) {
	switch s := stmt.(type) {
	case ast.Assignment:
		kw := swiftVarKeywords[rand.IntN(len(swiftVarKeywords))]
		if s.IsDecl {
			r.w.Line(out, fmt.Sprintf("%s %s: %s = %s", kw, s.Name, randSwiftType(), r.renderExpr(s.Value)))
		} else {
			r.w.Line(out, fmt.Sprintf("%s = %s", s.Name, r.renderExpr(s.Value)))
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
		r.w.Line(out, fmt.Sprintf("while %s {", r.renderExpr(s.Cond)))
		r.w.Inc()
		r.renderBlock(s.Body, out)
		r.w.Dec()
		r.w.Line(out, "}")
	case ast.ForStmt:
		r.w.Line(out, fmt.Sprintf("for %s in 0..<%s {", s.Var, r.renderExpr(s.Range)))
		r.w.Inc()
		r.renderBlock(s.Body, out)
		r.w.Dec()
		r.w.Line(out, "}")
	case ast.FuncDef:
		ret := randSwiftReturn()
		var params []string
		for _, p := range s.ParamNames {
			params = append(params, fmt.Sprintf("%s: %s", p, randSwiftType()))
		}
		paramStr := strings.Join(params, ", ")
		if ret == "Void" {
			r.w.Line(out, fmt.Sprintf("func %s(%s) {", s.Name, paramStr))
		} else {
			r.w.Line(out, fmt.Sprintf("func %s(%s) -> %s {", s.Name, paramStr, ret))
		}
		r.w.Inc()
		r.renderBlock(s.Body, out)
		r.w.Dec()
		r.w.Line(out, "}")
		r.w.Blank(out)
	case ast.ReturnStmt:
		r.w.Line(out, fmt.Sprintf("return %s", r.renderExpr(s.Value)))
	case ast.ExprStmt:
		r.w.Line(out, fmt.Sprintf("%s", r.renderExpr(s.Expr)))
	case ast.CommentStmt:
		r.w.Line(out, fmt.Sprintf("// %s", s.Text))
	}
}

// RenderProgram renders a full Swift source file.
func (r *SwiftRenderer) RenderProgram(program []ast.Statement) string {
	var out strings.Builder
	for _, stmt := range program {
		r.renderStmt(stmt, &out)
		out.WriteByte('\n')
	}
	return out.String()
}

func swiftArithOp(op ast.ArithmeticOperator) string {
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
