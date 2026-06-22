package render

import (
	"fmt"
	"math/rand/v2"
	"strings"

	"github.com/exey/fakecodegen/internal/ast"
)

var kotlinTypes = []string{"Int", "Long", "Double", "String", "Boolean"}
var kotlinReturnTypes = []string{"Int", "Double", "String", "Boolean", "Unit"}
var kotlinVarKeywords = []string{"val", "var"}

// KotlinRenderer renders AST to Kotlin source.
type KotlinRenderer struct {
	w IndentWriter
}

func NewKotlinRenderer() *KotlinRenderer {
	return &KotlinRenderer{w: NewIndentWriter("    ")}
}

func randKotlinType() string   { return kotlinTypes[rand.IntN(len(kotlinTypes))] }
func randKotlinReturn() string { return kotlinReturnTypes[rand.IntN(len(kotlinReturnTypes))] }

func (r *KotlinRenderer) renderExpr(expr ast.Expression) string {
	switch e := expr.(type) {
	case ast.ArithExpr:
		if e.Op == ast.Power {
			return fmt.Sprintf("(%s + %s)", r.renderExpr(e.Left), r.renderExpr(e.Right))
		}
		return fmt.Sprintf("(%s %s %s)", r.renderExpr(e.Left), kotlinArithOp(e.Op), r.renderExpr(e.Right))
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

func (r *KotlinRenderer) renderBlock(block []ast.Statement, out *strings.Builder) {
	for _, stmt := range block {
		r.renderStmt(stmt, out)
	}
}

func (r *KotlinRenderer) renderStmt(stmt ast.Statement, out *strings.Builder) {
	switch s := stmt.(type) {
	case ast.Assignment:
		kw := kotlinVarKeywords[rand.IntN(len(kotlinVarKeywords))]
		if s.IsDecl {
			r.w.Line(out, fmt.Sprintf("%s %s: %s = %s", kw, s.Name, randKotlinType(), r.renderExpr(s.Value)))
		} else {
			r.w.Line(out, fmt.Sprintf("%s = %s", s.Name, r.renderExpr(s.Value)))
		}
	case ast.IfStmt:
		r.w.Line(out, fmt.Sprintf("if (%s) {", r.renderExpr(s.Cond)))
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
		r.w.Line(out, fmt.Sprintf("while (%s) {", r.renderExpr(s.Cond)))
		r.w.Inc()
		r.renderBlock(s.Body, out)
		r.w.Dec()
		r.w.Line(out, "}")
	case ast.ForStmt:
		r.w.Line(out, fmt.Sprintf("for (%s in 0 until %s) {", s.Var, r.renderExpr(s.Range)))
		r.w.Inc()
		r.renderBlock(s.Body, out)
		r.w.Dec()
		r.w.Line(out, "}")
	case ast.FuncDef:
		ret := randKotlinReturn()
		var params []string
		for _, p := range s.ParamNames {
			params = append(params, fmt.Sprintf("%s: %s", p, randKotlinType()))
		}
		paramStr := strings.Join(params, ", ")
		if ret == "Unit" {
			r.w.Line(out, fmt.Sprintf("fun %s(%s) {", s.Name, paramStr))
		} else {
			r.w.Line(out, fmt.Sprintf("fun %s(%s): %s {", s.Name, paramStr, ret))
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

// RenderProgram renders a full Kotlin source file.
func (r *KotlinRenderer) RenderProgram(program []ast.Statement) string {
	var out strings.Builder
	for _, stmt := range program {
		r.renderStmt(stmt, &out)
		out.WriteByte('\n')
	}
	return out.String()
}

func kotlinArithOp(op ast.ArithmeticOperator) string {
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
