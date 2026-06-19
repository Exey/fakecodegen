package generator

import (
	"fmt"
	"math/rand/v2"

	"github.com/exey/fakecodegen/internal/ast"
)

var sloppyComments = []string{
	"TODO: fix this later",
	"HACK: this works somehow",
	"FIXME: no idea why this is here",
	"NOTE: do not touch this",
	"TEMP: temporary workaround",
	"XXX: refactor needed",
	"this is fine",
	"idk what this does",
	"legacy code, do not remove",
	"trust me bro",
	"works on my machine",
	"please dont delete",
	"might cause segfault lol",
	"copilot wrote this",
	"stolen from stackoverflow",
	"IMPORTANT: not actually important",
	"why does removing this break everything",
	"magic numbers ahead",
	"i wrote this at 3am",
	"REVIEW: never gonna happen",
	"TODO: add error handling",
	"TODO: add tests",
	"this variable does nothing",
	"no one knows what this function does",
	"do NOT refactor",
	"here be dragons",
}

var sloppyStrings = []string{
	"hello world", "foo", "bar", "baz", "qux", "asdf", "test", "temp",
	"data", "result", "output", "input", "value", "thing", "stuff",
	"undefined", "null", "error", "success", "placeholder", "TODO", "FIXME",
	"hack", "yolo", "bruh", "aaaaa", "12345", "password123", "admin", "root",
	"debug", "prod", "staging", "localhost", "0.0.0.0", "NaN", "infinity",
}

type randSet struct {
	set map[string]bool
	vec []string
}

func newRandSet() *randSet {
	return &randSet{set: make(map[string]bool)}
}

func (r *randSet) insert(s string) {
	if !r.set[s] {
		r.set[s] = true
		r.vec = append(r.vec, s)
	}
}

func (r *randSet) random(rng *rand.Rand) string {
	if len(r.vec) == 0 {
		return ""
	}
	return r.vec[rng.IntN(len(r.vec))]
}

// State holds all mutable generation state.
type State struct {
	maxDepth    int
	scopes      []*randSet
	functions   *randSet
	names       []string
	nameCounter int
	keywords    map[string]bool
	rng         *rand.Rand
}

var allKeywords = []string{
	// Python
	"False", "None", "True", "and", "as", "assert", "async", "await", "break", "class",
	"continue", "def", "del", "elif", "else", "except", "finally", "for", "from",
	"global", "if", "import", "in", "is", "lambda", "nonlocal", "not", "or", "pass",
	"raise", "return", "try", "while", "with", "yield",
	// JS/TS
	"var", "let", "const", "function", "new", "delete", "typeof", "instanceof", "void",
	"this", "super", "switch", "case", "default", "throw", "catch", "debugger", "export",
	"extends", "implements", "interface", "package", "private", "protected", "public",
	"static", "enum", "type", "namespace", "abstract", "declare", "readonly",
	// C++
	"auto", "bool", "char", "double", "float", "int", "long", "short", "signed",
	"unsigned", "struct", "union", "template", "typename", "virtual", "override",
	"final", "nullptr", "sizeof", "alignof", "constexpr", "noexcept", "mutable",
	"volatile", "register", "extern", "inline", "goto", "do", "using",
	// Rust
	"fn", "mut", "ref", "self", "Self", "mod", "pub", "crate", "use", "impl",
	"trait", "where", "loop", "match", "move", "box", "unsafe", "dyn",
	// Go
	"chan", "defer", "fallthrough", "go", "map", "range", "select",
	"func", "import", "package", "nil", "interface",
}

// New creates a new generator state.
func New(maxDepth int, names []string, rng *rand.Rand) *State {
	kw := make(map[string]bool, len(allKeywords))
	for _, k := range allKeywords {
		kw[k] = true
	}
	return &State{
		maxDepth:  maxDepth,
		functions: newRandSet(),
		names:     names,
		keywords:  kw,
		rng:       rng,
	}
}

func (s *State) pushScope() { s.scopes = append(s.scopes, newRandSet()) }
func (s *State) popScope()  { s.scopes = s.scopes[:len(s.scopes)-1] }

func (s *State) allVars() []string {
	var out []string
	for _, sc := range s.scopes {
		out = append(out, sc.vec...)
	}
	return out
}

func (s *State) allNames() map[string]bool {
	m := make(map[string]bool)
	for _, sc := range s.scopes {
		for _, n := range sc.vec {
			m[n] = true
		}
	}
	for _, n := range s.functions.vec {
		m[n] = true
	}
	return m
}

func (s *State) isValidIdent(name string) bool {
	if len(name) == 0 {
		return false
	}
	if s.keywords[name] {
		return false
	}
	r := []rune(name)
	if r[0] != '_' && !(r[0] >= 'a' && r[0] <= 'z') && !(r[0] >= 'A' && r[0] <= 'Z') {
		return false
	}
	for _, c := range r[1:] {
		if c != '_' && !(c >= 'a' && c <= 'z') && !(c >= 'A' && c <= 'Z') && !(c >= '0' && c <= '9') {
			return false
		}
	}
	return true
}

func (s *State) generateName() string {
	taken := s.allNames()
	for range 200 {
		candidate := s.names[s.rng.IntN(len(s.names))]
		if !taken[candidate] && s.isValidIdent(candidate) {
			return candidate
		}
	}
	for {
		s.nameCounter++
		name := fmt.Sprintf("var_%d", s.nameCounter)
		if !taken[name] {
			return name
		}
	}
}

func (s *State) randVar() string {
	vars := s.allVars()
	if len(vars) == 0 {
		return ""
	}
	return vars[s.rng.IntN(len(vars))]
}

func (s *State) randFunction() string { return s.functions.random(s.rng) }

func (s *State) randComment() string {
	return sloppyComments[s.rng.IntN(len(sloppyComments))]
}

func (s *State) randString() string {
	return sloppyStrings[s.rng.IntN(len(sloppyStrings))]
}

func (s *State) genArithOp() ast.ArithmeticOperator {
	ops := []ast.ArithmeticOperator{
		ast.Add, ast.Subtract, ast.Multiply, ast.Divide, ast.Modulo,
		ast.Power, ast.BitwiseAnd, ast.BitwiseOr, ast.BitwiseXor,
		ast.ShiftLeft, ast.ShiftRight,
	}
	return ops[s.rng.IntN(len(ops))]
}

func (s *State) genBoolOp() ast.BooleanOperator {
	if s.rng.IntN(2) == 0 {
		return ast.And
	}
	return ast.Or
}

func (s *State) genCondOp() ast.Condition {
	ops := []ast.Condition{
		ast.Equals, ast.NotEquals, ast.GreaterThan,
		ast.LessThan, ast.GreaterThanOrEqual, ast.LessThanOrEqual,
	}
	return ops[s.rng.IntN(len(ops))]
}

func (s *State) GenNumericExpr(depth int) ast.Expression {
	if depth >= s.maxDepth || s.rng.Float64() < 0.25 {
		switch s.rng.IntN(5) {
		case 0:
			if v := s.randVar(); v != "" {
				return ast.VarExpr{Name: v}
			}
			return ast.IntLit{Value: int64(s.rng.IntN(1000))}
		case 1:
			if f := s.randFunction(); f != "" {
				return ast.FuncCallExpr{Name: f}
			}
			return ast.IntLit{Value: int64(s.rng.IntN(1000))}
		case 2:
			return ast.IntLit{Value: int64(s.rng.IntN(1999)) - 999}
		case 3:
			return ast.FloatLit{Value: float64(s.rng.IntN(1999)-999) / float64(s.rng.IntN(99)+1)}
		default:
			return ast.StringLit{Value: s.randString()}
		}
	}
	switch s.rng.IntN(3) {
	case 1:
		op := ast.Negate
		if s.rng.IntN(2) == 0 {
			op = ast.BitwiseNot
		}
		return ast.UnaryExpr{Op: op, Operand: s.GenNumericExpr(depth + 1)}
	default:
		return ast.ArithExpr{
			Left:  s.GenNumericExpr(depth + 1),
			Op:    s.genArithOp(),
			Right: s.GenNumericExpr(depth + 1),
		}
	}
}

func (s *State) genCondExpr(depth int) ast.Expression {
	return ast.CondExpr{
		Left:  s.GenNumericExpr(depth + 1),
		Op:    s.genCondOp(),
		Right: s.GenNumericExpr(depth + 1),
	}
}

func (s *State) genBoolExpr(depth int) ast.Expression {
	if depth >= s.maxDepth || s.rng.Float64() < 0.4 {
		switch s.rng.IntN(3) {
		case 1:
			return ast.BoolLit{Value: s.rng.IntN(2) == 0}
		case 2:
			if f := s.randFunction(); f != "" {
				return ast.FuncCallExpr{Name: f}
			}
			return ast.BoolLit{Value: true}
		default:
			return s.genCondExpr(depth + 1)
		}
	}
	return ast.BoolExpr{
		Left:  s.genBoolExpr(depth + 1),
		Op:    s.genBoolOp(),
		Right: s.genBoolExpr(depth + 1),
	}
}

func (s *State) genExpr(depth int) ast.Expression {
	if s.rng.Float64() < 0.65 {
		return s.GenNumericExpr(depth)
	}
	return s.genBoolExpr(depth)
}

// GenStatement generates a single statement.
func (s *State) GenStatement(depth int, inFunction bool) ast.Statement {
	if depth >= s.maxDepth {
		if v := s.randVar(); v != "" {
			return ast.Assignment{Name: v, Value: s.genExpr(depth + 1)}
		}
		return ast.ExprStmt{Expr: s.genExpr(depth + 1)}
	}
	if s.rng.Float64() < 0.18 {
		return ast.CommentStmt{Text: s.randComment()}
	}

	switch s.rng.IntN(8) {
	case 0, 1:
		name := s.generateName()
		s.scopes[len(s.scopes)-1].insert(name)
		return ast.Assignment{Name: name, Value: s.genExpr(depth + 1)}
	case 2:
		if v := s.randVar(); v != "" {
			return ast.Assignment{Name: v, Value: s.genExpr(depth + 1)}
		}
		name := s.generateName()
		s.scopes[len(s.scopes)-1].insert(name)
		return ast.Assignment{Name: name, Value: s.genExpr(depth + 1)}
	case 3:
		elseBlock := []ast.Statement{}
		if s.rng.Float64() < 0.6 {
			elseBlock = s.genScopedBlock(depth+1, inFunction)
		}
		return ast.IfStmt{
			Cond: s.genBoolExpr(depth + 1),
			Then: s.genScopedBlock(depth+1, inFunction),
			Else: elseBlock,
		}
	case 4:
		return ast.WhileStmt{
			Cond: s.genBoolExpr(depth + 1),
			Body: s.genScopedBlock(depth+1, inFunction),
		}
	case 5:
		loopVar := s.generateName()
		s.scopes[len(s.scopes)-1].insert(loopVar)
		return ast.ForStmt{
			Var:   loopVar,
			Range: s.GenNumericExpr(depth + 1),
			Body:  s.genScopedBlock(depth+1, inFunction),
		}
	case 6:
		name := s.generateName()
		s.functions.insert(name)
		return ast.FuncDef{Name: name, Body: s.genFunctionBody(depth + 1)}
	case 7:
		if inFunction {
			return ast.ReturnStmt{Value: s.genExpr(depth + 1)}
		}
		fallthrough
	default:
		return ast.ExprStmt{Expr: s.genExpr(depth + 1)}
	}
}

func (s *State) genScopedBlock(depth int, inFunction bool) []ast.Statement {
	s.pushScope()
	block := s.genBlock(depth, inFunction)
	s.popScope()
	return block
}

func (s *State) genBlock(depth int, inFunction bool) []ast.Statement {
	count := s.rng.IntN(5) + 1
	block := make([]ast.Statement, count)
	for i := range block {
		block[i] = s.GenStatement(depth, inFunction)
	}
	return block
}

func (s *State) genFunctionBody(depth int) []ast.Statement {
	s.pushScope()
	body := s.genBlock(depth, true)
	if _, ok := body[len(body)-1].(ast.ReturnStmt); !ok {
		body = append(body, ast.ReturnStmt{Value: s.genExpr(depth + 1)})
	}
	s.popScope()
	return body
}

// GenerateProgram generates a full program (list of top-level statements).
func (s *State) GenerateProgram() []ast.Statement {
	s.pushScope()
	var program []ast.Statement

	fnCount := s.rng.IntN(3) + 2
	for range fnCount {
		name := s.generateName()
		s.functions.insert(name)
		program = append(program, ast.FuncDef{
			Name: name,
			Body: s.genFunctionBody(0),
		})
	}

	stmtCount := s.rng.IntN(18) + 8
	for range stmtCount {
		program = append(program, s.GenStatement(0, false))
	}

	s.popScope()
	return program
}

// FunctionNames returns all top-level function names in the program.
func FunctionNames(program []ast.Statement) []string {
	var names []string
	for _, stmt := range program {
		if fd, ok := stmt.(ast.FuncDef); ok {
			names = append(names, fd.Name)
		}
	}
	return names
}

// NewTargeted creates a State whose maxDepth is tuned so that rendering a
// program with len(declNames) top-level functions produces roughly targetLines
// lines of output.
func NewTargeted(targetLines int, declNames []string, names []string, rng *rand.Rand) *State {
	return New(estimateDepth(targetLines, len(declNames)), names, rng)
}

func estimateDepth(targetLines, declCount int) int {
	n := declCount
	if n < 1 {
		n = 3
	}
	switch lpp := targetLines / n; {
	case lpp < 25:
		return 1
	case lpp < 60:
		return 2
	case lpp < 150:
		return 3
	case lpp < 300:
		return 4
	case lpp < 600:
		return 5
	default:
		return 6
	}
}

// GenerateProgramWithDecls generates a program using declNames as the names of
// top-level functions. Unknown or keyword names are skipped; if none are valid
// the generator falls back to random names as usual.
func (s *State) GenerateProgramWithDecls(declNames []string) []ast.Statement {
	s.pushScope()
	var program []ast.Statement

	for _, name := range declNames {
		if s.isValidIdent(name) {
			s.functions.insert(name)
			program = append(program, ast.FuncDef{
				Name: name,
				Body: s.genFunctionBody(0),
			})
		}
	}

	// Fallback: no valid names provided.
	if len(program) == 0 {
		for range s.rng.IntN(3) + 2 {
			name := s.generateName()
			s.functions.insert(name)
			program = append(program, ast.FuncDef{
				Name: name,
				Body: s.genFunctionBody(0),
			})
		}
	}

	for range s.rng.IntN(10) + 5 {
		program = append(program, s.GenStatement(0, false))
	}

	s.popScope()
	return program
}
