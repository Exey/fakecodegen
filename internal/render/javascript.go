package render

import (
	"fmt"
	"math/rand/v2"
	"strings"

	"github.com/exey/fakecodegen/internal/ast"
)

var jsKeywords = []string{"var", "let", "const"}

// JavaScriptRenderer renders AST to JavaScript source.
type JavaScriptRenderer struct {
	w IndentWriter
}

func NewJavaScriptRenderer() *JavaScriptRenderer {
	return &JavaScriptRenderer{w: NewIndentWriter("    ")}
}

func (r *JavaScriptRenderer) renderExpr(expr ast.Expression) string {
	switch e := expr.(type) {
	case ast.ArithExpr:
		return fmt.Sprintf("(%s %s %s)", r.renderExpr(e.Left), jsArithOp(e.Op), r.renderExpr(e.Right))
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
	return "undefined"
}

func (r *JavaScriptRenderer) renderBlock(block []ast.Statement, out *strings.Builder) {
	for _, stmt := range block {
		r.renderStmt(stmt, out)
	}
}

func (r *JavaScriptRenderer) renderStmt(stmt ast.Statement, out *strings.Builder) {
	switch s := stmt.(type) {
	case ast.Assignment:
		kw := jsKeywords[rand.IntN(len(jsKeywords))]
		r.w.Line(out, fmt.Sprintf("%s %s = %s;", kw, s.Name, r.renderExpr(s.Value)))
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
		r.w.Line(out, fmt.Sprintf("for (let %s = 0; %s < %s; %s++) {",
			s.Var, s.Var, r.renderExpr(s.Range), s.Var))
		r.w.Inc()
		r.renderBlock(s.Body, out)
		r.w.Dec()
		r.w.Line(out, "}")
	case ast.FuncDef:
		params := strings.Join(s.ParamNames, ", ")
		r.w.Line(out, fmt.Sprintf("function %s(%s) {", s.Name, params))
		r.w.Inc()
		r.renderBlock(s.Body, out)
		r.w.Dec()
		r.w.Line(out, "}")
		r.w.Blank(out)
	case ast.ReturnStmt:
		r.w.Line(out, fmt.Sprintf("return %s;", r.renderExpr(s.Value)))
	case ast.ExprStmt:
		r.w.Line(out, fmt.Sprintf("%s;", r.renderExpr(s.Expr)))
	case ast.CommentStmt:
		r.w.Line(out, fmt.Sprintf("// %s", s.Text))
	}
}

// RenderProgram renders a full JavaScript source file.
func (r *JavaScriptRenderer) RenderProgram(program []ast.Statement) string {
	var out strings.Builder
	for _, stmt := range program {
		r.renderStmt(stmt, &out)
		out.WriteByte('\n')
	}
	return out.String()
}

func jsArithOp(op ast.ArithmeticOperator) string {
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
	case ast.Power:
		return "**"
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
