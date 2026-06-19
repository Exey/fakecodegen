package render

import (
	"strings"

	"github.com/exey/fakecodegen/internal/ast"
)

// Renderer renders a program to source code.
type Renderer interface {
	RenderProgram(program []ast.Statement) string
}

// IndentWriter manages indentation for output.
type IndentWriter struct {
	indent    int
	indentStr string
}

func NewIndentWriter(indentStr string) IndentWriter {
	return IndentWriter{indentStr: indentStr}
}

func (w *IndentWriter) Line(out *strings.Builder, line string) {
	for range w.indent {
		out.WriteString(w.indentStr)
	}
	out.WriteString(line)
	out.WriteByte('\n')
}

func (w *IndentWriter) Blank(out *strings.Builder) { out.WriteByte('\n') }
func (w *IndentWriter) Inc()                        { w.indent++ }
func (w *IndentWriter) Dec()                        { w.indent-- }

// RenderSourceFile dispatches to the right renderer by extension.
// filePath is the relative path of the output file and is used by the Go
// renderer to derive the package name from the directory component.
// For unrecognised extensions it falls back to the Go renderer.
func RenderSourceFile(program []ast.Statement, ext string, filePath string) (string, error) {
	switch ext {
	case "go":
		r := NewGoRenderer(filePath)
		return r.RenderProgram(program), nil
	case "py":
		r := NewPythonRenderer()
		return r.RenderProgram(program), nil
	case "rs":
		r := NewRustRenderer()
		return r.RenderProgram(program), nil
	case "js":
		r := NewJavaScriptRenderer()
		return r.RenderProgram(program), nil
	case "ts":
		r := NewTypeScriptRenderer()
		return r.RenderProgram(program), nil
	default:
		// Unknown extension: fall back to Go-like brace syntax.
		r := NewGoRenderer(filePath)
		return r.RenderProgram(program), nil
	}
}

// shared helpers

func renderCondOp(op ast.Condition) string {
	switch op {
	case ast.Equals:
		return "=="
	case ast.NotEquals:
		return "!="
	case ast.GreaterThan:
		return ">"
	case ast.LessThan:
		return "<"
	case ast.GreaterThanOrEqual:
		return ">="
	case ast.LessThanOrEqual:
		return "<="
	}
	return "=="
}
