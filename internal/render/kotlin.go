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

var kotlinTypeRemap = map[string]string{
	"int":             "Int",
	"int64":           "Long",
	"uint32":          "UInt",
	"string":          "String",
	"bool":            "Boolean",
	"float64":         "Double",
	"[]byte":          "ByteArray",
	"time.Time":       "Long",
	"error":           "Exception",
	"any":             "Any",
	"context.Context": "Any",
	"(string, error)": "Pair<String, Exception?>",
	"(int, error)":    "Pair<Int, Exception?>",
}

func kotlinMapType(t string) string {
	if v, ok := kotlinTypeRemap[t]; ok {
		return v
	}
	return t
}

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
		retType := randKotlinReturn()
		if s.ReturnType != "" {
			retType = kotlinMapType(s.ReturnType)
		}
		var params []string
		if len(s.TypedParams) > 0 {
			for _, tp := range s.TypedParams {
				params = append(params, fmt.Sprintf("%s: %s", tp.Name, kotlinMapType(tp.Type)))
			}
		} else {
			for _, p := range s.ParamNames {
				params = append(params, fmt.Sprintf("%s: %s", p, randKotlinType()))
			}
		}
		paramStr := strings.Join(params, ", ")
		if retType == "Unit" || retType == "" {
			r.w.Line(out, fmt.Sprintf("fun %s(%s) {", s.Name, paramStr))
		} else {
			r.w.Line(out, fmt.Sprintf("fun %s(%s): %s {", s.Name, paramStr, retType))
		}
		r.w.Inc()
		r.renderBlock(s.Body, out)
		r.w.Dec()
		r.w.Line(out, "}")
		r.w.Blank(out)
	case ast.ReturnStmt:
		r.w.Line(out, fmt.Sprintf("return %s", r.renderExpr(s.Value)))
	case ast.ExprStmt:
		r.w.Line(out, r.renderExpr(s.Expr))
	case ast.CommentStmt:
		r.w.Line(out, fmt.Sprintf("// %s", s.Text))
	case ast.SwitchStmt:
		r.w.Line(out, fmt.Sprintf("when (%s) {", s.Tag))
		r.w.Inc()
		for _, c := range s.Cases {
			r.w.Line(out, fmt.Sprintf("%d -> {", c.Value))
			r.w.Inc()
			r.renderBlock(c.Body, out)
			r.w.Dec()
			r.w.Line(out, "}")
		}
		if len(s.Default) > 0 {
			r.w.Line(out, "else -> {")
			r.w.Inc()
			r.renderBlock(s.Default, out)
			r.w.Dec()
			r.w.Line(out, "}")
		}
		r.w.Dec()
		r.w.Line(out, "}")
	case ast.DeferStmt:
		r.w.Line(out, fmt.Sprintf("Runtime.getRuntime().addShutdownHook(Thread { println(%q) })", s.Text))
	case ast.HelperCallStmt:
		args := strings.Join(s.Args, ", ")
		r.w.Line(out, fmt.Sprintf("val %s = %s(%s)", s.Result, s.Name, args))
	case ast.StructDecl:
		var fields []string
		for _, f := range s.Fields {
			fields = append(fields, fmt.Sprintf("var %s: %s", lowerFirst(f.Name), kotlinMapType(f.Type)))
		}
		r.w.Line(out, fmt.Sprintf("data class %s(%s)", s.Name, strings.Join(fields, ", ")))
		r.w.Blank(out)
	case ast.InterfaceDecl:
		r.w.Line(out, fmt.Sprintf("interface %s {", s.Name))
		r.w.Inc()
		for _, m := range s.Methods {
			var params []string
			for i, pt := range m.ParamTypes {
				params = append(params, fmt.Sprintf("arg%d: %s", i, kotlinMapType(pt)))
			}
			retType := kotlinMapType(m.ReturnType)
			if retType == "Unit" || retType == "" {
				r.w.Line(out, fmt.Sprintf("fun %s(%s)", lowerFirst(m.Name), strings.Join(params, ", ")))
			} else {
				r.w.Line(out, fmt.Sprintf("fun %s(%s): %s", lowerFirst(m.Name), strings.Join(params, ", "), retType))
			}
		}
		r.w.Dec()
		r.w.Line(out, "}")
		r.w.Blank(out)
	}
}

// RenderProgram renders a full Kotlin source file.
func (r *KotlinRenderer) RenderProgram(program []ast.Statement) string {
	var out strings.Builder

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
		out.WriteString("fun setup() {\n")
		r.w.Inc()
		for _, stmt := range setupStmts {
			r.renderStmt(stmt, &out)
		}
		r.w.Dec()
		out.WriteString("}\n")
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
