package render

import (
	"fmt"
	"math/rand/v2"
	"strings"

	"github.com/exey/fakecodegen/internal/ast"
)

var rustTypes = []string{"i32", "i64", "u32", "u64", "f64", "f32", "isize", "usize"}
var rustReturnTypes = []string{"i32", "i64", "f64", "bool", "String", "()"}

var rustTypeRemap = map[string]string{
	"int":             "i32",
	"int64":           "i64",
	"uint32":          "u32",
	"string":          "String",
	"bool":            "bool",
	"float64":         "f64",
	"[]byte":          "Vec<u8>",
	"time.Time":       "u64",
	"error":           "Result<(), String>",
	"any":             "Box<dyn std::any::Any>",
	"context.Context": "i32",
	"(string, error)": "(String, Result<(), String>)",
	"(int, error)":    "(i64, Result<(), String>)",
}

func rustMapType(t string) string {
	if v, ok := rustTypeRemap[t]; ok {
		return v
	}
	return t
}

func camelToSnake(s string) string {
	var b strings.Builder
	for i, r := range s {
		if r >= 'A' && r <= 'Z' {
			if i > 0 {
				b.WriteByte('_')
			}
			b.WriteRune(r - 'A' + 'a')
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// RustRenderer renders AST to Rust source.
type RustRenderer struct {
	w IndentWriter
}

func NewRustRenderer() *RustRenderer {
	return &RustRenderer{w: NewIndentWriter("    ")}
}

func randRustType() string   { return rustTypes[rand.IntN(len(rustTypes))] }
func randRustReturn() string { return rustReturnTypes[rand.IntN(len(rustReturnTypes))] }

func (r *RustRenderer) renderExpr(expr ast.Expression) string {
	switch e := expr.(type) {
	case ast.ArithExpr:
		if e.Op == ast.Power {
			return fmt.Sprintf("(%s + %s)", r.renderExpr(e.Left), r.renderExpr(e.Right))
		}
		return fmt.Sprintf("(%s %s %s)", r.renderExpr(e.Left), rustArithOp(e.Op), r.renderExpr(e.Right))
	case ast.UnaryExpr:
		switch e.Op {
		case ast.Negate:
			return fmt.Sprintf("(-%s)", r.renderExpr(e.Operand))
		default:
			return fmt.Sprintf("(!%s)", r.renderExpr(e.Operand))
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

func (r *RustRenderer) renderBlock(block []ast.Statement, out *strings.Builder) {
	for _, stmt := range block {
		r.renderStmt(stmt, out)
	}
}

func (r *RustRenderer) renderStmt(stmt ast.Statement, out *strings.Builder) {
	switch s := stmt.(type) {
	case ast.Assignment:
		r.w.Line(out, fmt.Sprintf("let mut %s: %s = %s;", s.Name, randRustType(), r.renderExpr(s.Value)))
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
		r.w.Line(out, fmt.Sprintf("for %s in 0..%s {", s.Var, r.renderExpr(s.Range)))
		r.w.Inc()
		r.renderBlock(s.Body, out)
		r.w.Dec()
		r.w.Line(out, "}")
	case ast.FuncDef:
		retType := randRustReturn()
		if s.ReturnType != "" {
			retType = rustMapType(s.ReturnType)
		}
		var params []string
		if len(s.TypedParams) > 0 {
			for _, tp := range s.TypedParams {
				params = append(params, fmt.Sprintf("%s: %s", tp.Name, rustMapType(tp.Type)))
			}
		} else {
			for _, p := range s.ParamNames {
				params = append(params, fmt.Sprintf("%s: %s", p, randRustType()))
			}
		}
		paramStr := strings.Join(params, ", ")
		if retType == "()" {
			r.w.Line(out, fmt.Sprintf("fn %s(%s) {", s.Name, paramStr))
		} else {
			r.w.Line(out, fmt.Sprintf("fn %s(%s) -> %s {", s.Name, paramStr, retType))
		}
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
	case ast.SwitchStmt:
		r.w.Line(out, fmt.Sprintf("match %s {", s.Tag))
		r.w.Inc()
		for _, c := range s.Cases {
			r.w.Line(out, fmt.Sprintf("%d => {", c.Value))
			r.w.Inc()
			r.renderBlock(c.Body, out)
			r.w.Dec()
			r.w.Line(out, "}")
		}
		if len(s.Default) > 0 {
			r.w.Line(out, "_ => {")
			r.w.Inc()
			r.renderBlock(s.Default, out)
			r.w.Dec()
			r.w.Line(out, "}")
		}
		r.w.Dec()
		r.w.Line(out, "}")
	case ast.DeferStmt:
		r.w.Line(out, fmt.Sprintf("println!(\"{}\", %q);", s.Text))
	case ast.HelperCallStmt:
		args := strings.Join(s.Args, ", ")
		r.w.Line(out, fmt.Sprintf("let mut %s = %s(%s);", s.Result, s.Name, args))
	case ast.StructDecl:
		r.w.Line(out, "#[derive(Debug, Clone)]")
		r.w.Line(out, fmt.Sprintf("struct %s {", s.Name))
		r.w.Inc()
		for _, f := range s.Fields {
			r.w.Line(out, fmt.Sprintf("%s: %s,", camelToSnake(f.Name), rustMapType(f.Type)))
		}
		r.w.Dec()
		r.w.Line(out, "}")
		r.w.Blank(out)
	case ast.InterfaceDecl:
		r.w.Line(out, fmt.Sprintf("trait %s {", s.Name))
		r.w.Inc()
		for _, m := range s.Methods {
			params := make([]string, 0, len(m.ParamTypes)+1)
			params = append(params, "&self")
			for i, pt := range m.ParamTypes {
				params = append(params, fmt.Sprintf("arg%d: %s", i, rustMapType(pt)))
			}
			retType := rustMapType(m.ReturnType)
			if retType == "()" || retType == "" {
				r.w.Line(out, fmt.Sprintf("fn %s(%s);", m.Name, strings.Join(params, ", ")))
			} else {
				r.w.Line(out, fmt.Sprintf("fn %s(%s) -> %s;", m.Name, strings.Join(params, ", "), retType))
			}
		}
		r.w.Dec()
		r.w.Line(out, "}")
		r.w.Blank(out)
	}
}

// RenderProgram renders a full Rust source file.
func (r *RustRenderer) RenderProgram(program []ast.Statement) string {
	var out strings.Builder
	out.WriteString("#![allow(unused, dead_code, unreachable_code, non_snake_case)]\n\n")

	var topDecls []ast.Statement
	var setupStmts []ast.Statement
	for _, stmt := range program {
		switch stmt.(type) {
		case ast.FuncDef, ast.StructDecl, ast.InterfaceDecl:
			topDecls = append(topDecls, stmt)
		default:
			setupStmts = append(setupStmts, stmt)
		}
	}

	for _, stmt := range topDecls {
		r.renderStmt(stmt, &out)
		out.WriteByte('\n')
	}

	if len(setupStmts) > 0 {
		out.WriteString("fn setup() {\n")
		r.w.Inc()
		for _, stmt := range setupStmts {
			r.renderStmt(stmt, &out)
		}
		r.w.Dec()
		out.WriteString("}\n")
	}

	return out.String()
}

func rustArithOp(op ast.ArithmeticOperator) string {
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
