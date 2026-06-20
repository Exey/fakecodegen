package render

import (
	"regexp"
	"strings"

	"github.com/exey/fakecodegen/internal/ast"
)

var funcDeclRe = regexp.MustCompile(`^func\s+(\w+)\s*\(`)

// ExtractFuncLines scans rendered Go source and returns a map of
// function name → line count (from the func declaration to its closing brace).
func ExtractFuncLines(source string) map[string]int {
	result := make(map[string]int)
	lines := strings.Split(source, "\n")
	var cur string
	var start int
	depth := 0
	for i, line := range lines {
		if m := funcDeclRe.FindStringSubmatch(line); m != nil && depth == 0 {
			cur = m[1]
			start = i
		}
		for _, ch := range line {
			if ch == '{' {
				depth++
			} else if ch == '}' {
				depth--
			}
		}
		if cur != "" && depth == 0 {
			result[cur] = i - start + 1
			cur = ""
		}
	}
	return result
}

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
