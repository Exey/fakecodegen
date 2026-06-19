package render

import (
	"fmt"
	"math/rand/v2"
	"strings"

	"github.com/exey/fakecodegen/internal/ast"
)

var tsTypes = []string{"number", "string", "boolean", "any", "unknown", "void", "never", "object"}
var tsVarKeywords = []string{"let", "const"}

// TypeScriptRenderer renders AST to TypeScript source.
type TypeScriptRenderer struct {
	w IndentWriter
}

func NewTypeScriptRenderer() *TypeScriptRenderer {
	return &TypeScriptRenderer{w: NewIndentWriter("    ")}
}

func randTSType() string { return tsTypes[rand.IntN(len(tsTypes))] }

func (r *TypeScriptRenderer) renderExpr(expr ast.Expression) string {
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

func (r *TypeScriptRenderer) renderBlock(block []ast.Statement, out *strings.Builder) {
	for _, stmt := range block {
		r.renderStmt(stmt, out)
	}
}

func (r *TypeScriptRenderer) renderStmt(stmt ast.Statement, out *strings.Builder) {
	switch s := stmt.(type) {
	case ast.Assignment:
		kw := tsVarKeywords[rand.IntN(len(tsVarKeywords))]
		r.w.Line(out, fmt.Sprintf("%s %s: %s = %s;", kw, s.Name, randTSType(), r.renderExpr(s.Value)))
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
		r.w.Line(out, fmt.Sprintf("for (let %s: number = 0; %s < %s; %s++) {",
			s.Var, s.Var, r.renderExpr(s.Range), s.Var))
		r.w.Inc()
		r.renderBlock(s.Body, out)
		r.w.Dec()
		r.w.Line(out, "}")
	case ast.FuncDef:
		var params []string
		for _, p := range s.ParamNames {
			params = append(params, fmt.Sprintf("%s: %s", p, randTSType()))
		}
		paramStr := strings.Join(params, ", ")
		r.w.Line(out, fmt.Sprintf("function %s(%s): %s {", s.Name, paramStr, randTSType()))
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

// RenderProgram renders a full TypeScript source file.
func (r *TypeScriptRenderer) RenderProgram(program []ast.Statement) string {
	var out strings.Builder
	for _, stmt := range program {
		r.renderStmt(stmt, &out)
		out.WriteByte('\n')
	}
	return out.String()
}
