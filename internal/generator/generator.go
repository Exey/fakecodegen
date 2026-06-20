package generator

import (
	"fmt"
	"math/rand/v2"

	"github.com/exey/fakecodegen/internal/ast"
)

var sloppyComments = []string{
	"HACK: this works somehow",
	"NOTE: do not touch this",
	"TEMP: temporary workaround",
	"XXX: refactor needed",
	"this is fine",
	"idk what this does",
	"legacy code, do not remove",
	"trust me bro",
	"works on my machine",
	"please dont delete",
	"copilot wrote this",
	"stolen from stackoverflow",
	"IMPORTANT: not actually important",
	"why does removing this break everything",
	"magic numbers ahead",
	"i wrote this at 3am",
	"REVIEW: never gonna happen",
	"this variable does nothing",
	"no one knows what this function does",
	"do NOT refactor",
	"here be dragons",
	"TODO: revisit before next release",
	"off by one? probably fine",
	"not sure this handles all edge cases",
	"ask alice before changing this",
}

var sloppyStrings = []string{
	"hello world", "foo", "bar", "baz", "qux", "asdf", "test", "temp",
	"data", "result", "output", "input", "value", "thing", "stuff",
	"undefined", "null", "error", "success", "placeholder", "pending", "unknown",
	"hack", "yolo", "bruh", "aaaaa", "12345", "password123", "admin", "root",
	"debug", "prod", "staging", "localhost", "0.0.0.0", "NaN", "infinity",
}

// smallInts are realistic-looking constants; favour these over arbitrary large values.
var smallInts = []int64{0, 1, 2, 3, 4, 8, 10, 16, 32, 64, 100, 128, 256, 512, 1024}

// ── Domain system ─────────────────────────────────────────────────────────────

// domain describes a problem domain used for naming variables and helpers.
type domain struct {
	name      string
	vars      []string // realistic variable names
	helpers   []string // realistic helper function names
	subject   string   // short description for log messages
	structFld []string // realistic struct field names
}

var domainPool = []domain{
	{
		name:      "payment",
		vars:      []string{"customerID", "invoiceNumber", "totalAmount", "dueDate", "taxRate", "discount", "balance", "creditLimit"},
		helpers:   []string{"applyDiscount", "validatePayment", "processTx", "calculateTax"},
		subject:   "payment processing",
		structFld: []string{"Amount", "Currency", "Status", "InvoiceID", "CustomerID", "DueDate", "TaxRate", "Discount"},
	},
	{
		name:      "user",
		vars:      []string{"userID", "roleID", "sessionCount", "loginAttempts", "permissionMask", "accountAge", "tier", "score"},
		helpers:   []string{"validateUser", "fetchProfile", "updateSession", "checkPermission"},
		subject:   "user management",
		structFld: []string{"ID", "Email", "Role", "CreatedAt", "LastLogin", "SessionToken", "Permissions", "IsActive"},
	},
	{
		name:      "product",
		vars:      []string{"productID", "stockLevel", "price", "quantity", "categoryID", "weight", "reorderPoint", "unitCost"},
		helpers:   []string{"checkInventory", "applyMarkup", "updateCatalog", "reserveStock"},
		subject:   "product catalog",
		structFld: []string{"SKU", "Name", "Price", "Stock", "CategoryID", "Weight", "Description", "IsActive"},
	},
	{
		name:      "order",
		vars:      []string{"orderID", "quantity", "discount", "subtotal", "taxRate", "shippingCost", "itemCount", "priority"},
		helpers:   []string{"calculateTotal", "validateOrder", "fulfillOrder", "cancelOrder"},
		subject:   "order lifecycle",
		structFld: []string{"ID", "CustomerID", "Status", "Total", "ItemCount", "ShippingAddr", "PlacedAt", "Priority"},
	},
	{
		name:      "auth",
		vars:      []string{"userID", "sessionID", "tokenExpiry", "failedAttempts", "lockDuration", "secretLen", "audience", "version"},
		helpers:   []string{"verifyToken", "refreshClaims", "revokeSession", "issueToken"},
		subject:   "authentication",
		structFld: []string{"Token", "Claims", "ExpiresAt", "Issuer", "Audience", "Algorithm", "KeyID", "SessionID"},
	},
	{
		name:      "report",
		vars:      []string{"reportID", "pageSize", "offset", "totalRows", "filterCount", "timeRange", "groupBy", "limit"},
		helpers:   []string{"generateReport", "exportData", "aggregateMetrics", "buildQuery"},
		subject:   "report generation",
		structFld: []string{"ID", "Title", "StartDate", "EndDate", "Filters", "Format", "RowCount", "Status"},
	},
	{
		name:      "cache",
		vars:      []string{"keyHash", "ttl", "maxSize", "hitCount", "missCount", "evictions", "shards", "loadFactor"},
		helpers:   []string{"lookup", "evict", "refresh", "warmup"},
		subject:   "cache management",
		structFld: []string{"Key", "Value", "TTL", "HitCount", "MissCount", "Evictions", "Size", "MaxSize"},
	},
	{
		name:      "notification",
		vars:      []string{"userID", "channelID", "retries", "priority", "batchSize", "delay", "timeout", "maxQueue"},
		helpers:   []string{"sendNotification", "scheduleDelivery", "markDelivered", "retryFailed"},
		subject:   "notification delivery",
		structFld: []string{"ID", "UserID", "Channel", "Title", "Body", "SentAt", "Status", "Retries"},
	},
}

var goStructFieldTypes = []string{
	"int", "int64", "uint32", "string", "bool", "float64",
	"[]byte", "time.Time", "error",
}

var goInterfaceReturnTypes = []string{
	"error", "(string, error)", "(int, error)", "bool", "string", "int64",
}

var goInterfaceParamTypes = [][]string{
	{"context.Context"},
	{"context.Context", "int"},
	{"context.Context", "string"},
	{"int"},
	{"string", "int"},
	{"context.Context", "int", "string"},
}

// ── randSet ───────────────────────────────────────────────────────────────────

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

// ── State ─────────────────────────────────────────────────────────────────────

// State holds all mutable generation state.
type State struct {
	maxDepth    int
	scopes      []*randSet
	// closures tracks no-param inner functions that can be safely called
	// without arguments.  Top-level functions (which may have params) are
	// deliberately NOT added here to avoid "too few arguments" compile errors.
	closures    *randSet
	names       []string
	nameCounter int
	keywords    map[string]bool
	// generated tracks every identifier ever produced across all scopes so that
	// names freed when a scope is popped are never reused.
	generated map[string]bool
	rng       *rand.Rand

	// Go-specific: stubs that need to be declared at program level.
	// Maps helper name → parameter count (all int-typed).
	stubs map[string]int
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
	// Go special function names
	"init", "main",
	// Go built-in identifiers — not keywords but shadow built-in types/funcs
	"error", "string", "bool", "byte", "rune", "any",
	// Common stdlib package names used as imports — must not be shadowed
	"fmt", "log", "context", "time", "sync", "io", "os", "errors", "strings", "strconv",
	"int", "int8", "int16", "int32", "int64",
	"uint", "uint8", "uint16", "uint32", "uint64", "uintptr",
	"float32", "float64", "complex64", "complex128",
	"make", "len", "cap", "append", "copy", "delete", "close",
	"panic", "recover", "print", "println", "new", "real", "imag",
	"true", "false", "iota",
}

// New creates a new generator state with a fresh generated-name set.
func New(maxDepth int, names []string, rng *rand.Rand) *State {
	return newState(maxDepth, make(map[string]bool), names, rng)
}

// NewShared creates a new per-file generator state that shares a generated-name
// set with other State instances to prevent duplicate top-level identifiers in
// the same Go package.
func NewShared(maxDepth int, shared map[string]bool, names []string, rng *rand.Rand) *State {
	return newState(maxDepth, shared, names, rng)
}

func newState(maxDepth int, generated map[string]bool, names []string, rng *rand.Rand) *State {
	kw := make(map[string]bool, len(allKeywords))
	for _, k := range allKeywords {
		kw[k] = true
	}
	return &State{
		maxDepth:  maxDepth,
		closures:  newRandSet(),
		names:     names,
		keywords:  kw,
		generated: generated,
		rng:       rng,
		stubs:     make(map[string]int),
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

func (s *State) insertVar(name string) {
	s.scopes[len(s.scopes)-1].insert(name)
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

// generateName returns a unique identifier but does NOT insert it into any scope.
func (s *State) generateName() string {
	for range 200 {
		candidate := s.names[s.rng.IntN(len(s.names))]
		if !s.generated[candidate] && s.isValidIdent(candidate) {
			s.generated[candidate] = true
			return candidate
		}
	}
	for {
		s.nameCounter++
		name := fmt.Sprintf("var_%d", s.nameCounter)
		if !s.generated[name] {
			s.generated[name] = true
			return name
		}
	}
}

// generateVar creates a fresh variable name and inserts it into the current scope.
// Must only be called AFTER the initialiser expression is generated so that the
// new variable cannot appear in its own initialiser.
func (s *State) generateVar() string {
	name := s.generateName()
	s.insertVar(name)
	return name
}

// generateDomainVar picks a name from domain.vars, registers and scopes it.
// Falls back to generateVar if all domain names are taken.
func (s *State) generateDomainVar(d domain) string {
	for _, candidate := range d.vars {
		if !s.generated[candidate] && s.isValidIdent(candidate) {
			s.generated[candidate] = true
			s.insertVar(candidate)
			return candidate
		}
	}
	return s.generateVar()
}

func (s *State) randVar() string {
	vars := s.allVars()
	if len(vars) == 0 {
		return ""
	}
	return vars[s.rng.IntN(len(vars))]
}

// randClosure returns a callable no-param closure name (safe for call-sites).
func (s *State) randClosure() string { return s.closures.random(s.rng) }

func (s *State) randComment() string {
	return sloppyComments[s.rng.IntN(len(sloppyComments))]
}

func (s *State) randString() string {
	return sloppyStrings[s.rng.IntN(len(sloppyStrings))]
}

func (s *State) randSmallInt() int64 {
	return smallInts[s.rng.IntN(len(smallInts))]
}

func (s *State) genArithOp() ast.ArithmeticOperator {
	ops := []ast.ArithmeticOperator{
		ast.Add, ast.Subtract, ast.Multiply,
		ast.BitwiseAnd, ast.BitwiseOr,
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
	// Use only == and != to avoid ordering comparisons on non-int types.
	if s.rng.IntN(2) == 0 {
		return ast.Equals
	}
	return ast.NotEquals
}

// GenIntExpr generates an expression guaranteed to evaluate to an integer.
// It never uses FuncCallExpr or non-integer literals, making it safe for
// for-range, arithmetic, and variable initialisation in typed languages.
func (s *State) GenIntExpr(depth int) ast.Expression {
	if depth >= s.maxDepth || s.rng.Float64() < 0.4 {
		if v := s.randVar(); v != "" {
			return ast.VarExpr{Name: v}
		}
		return ast.IntLit{Value: s.randSmallInt()}
	}
	op := []ast.ArithmeticOperator{ast.Add, ast.Subtract, ast.Multiply}[s.rng.IntN(3)]
	return ast.ArithExpr{
		Left:  s.GenIntExpr(depth + 1),
		Op:    op,
		Right: s.GenIntExpr(depth + 1),
	}
}

// GenNumericExpr generates a general expression (may be string at leaf level).
// Arithmetic sub-expressions always use GenIntExpr to avoid type mismatches.
func (s *State) GenNumericExpr(depth int) ast.Expression {
	if depth >= s.maxDepth || s.rng.Float64() < 0.3 {
		switch s.rng.IntN(5) {
		case 0, 1:
			if v := s.randVar(); v != "" {
				return ast.VarExpr{Name: v}
			}
			return ast.IntLit{Value: s.randSmallInt()}
		case 2:
			// Only call no-param closures to avoid missing-argument errors.
			if f := s.randClosure(); f != "" {
				return ast.FuncCallExpr{Name: f}
			}
			return ast.IntLit{Value: s.randSmallInt()}
		case 3:
			return ast.IntLit{Value: s.randSmallInt()}
		default:
			return ast.StringLit{Value: s.randString()}
		}
	}
	if s.rng.IntN(5) == 0 {
		op := ast.Negate
		if s.rng.IntN(2) == 0 {
			op = ast.BitwiseNot
		}
		return ast.UnaryExpr{Op: op, Operand: s.GenIntExpr(depth + 1)}
	}
	return ast.ArithExpr{
		Left:  s.GenIntExpr(depth + 1),
		Op:    s.genArithOp(),
		Right: s.GenIntExpr(depth + 1),
	}
}

func (s *State) genCondExpr(depth int) ast.Expression {
	return ast.CondExpr{
		Left:  s.GenIntExpr(depth + 1),
		Op:    s.genCondOp(),
		Right: s.GenIntExpr(depth + 1),
	}
}

func (s *State) genBoolExpr(depth int) ast.Expression {
	if depth >= s.maxDepth || s.rng.Float64() < 0.4 {
		if s.rng.IntN(3) == 1 {
			return ast.BoolLit{Value: s.rng.IntN(2) == 0}
		}
		// Use a comparison (always type-safe) rather than a bare function call.
		return s.genCondExpr(depth + 1)
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

// genParams creates 0–3 parameter names and inserts them into the current scope.
func (s *State) genParams() []string {
	count := s.rng.IntN(4) // 0–3 params
	params := make([]string, 0, count)
	for range count {
		name := s.generateVar()
		params = append(params, name)
	}
	return params
}

// genDomainParams creates 1–3 domain-named int parameters.
func (s *State) genDomainParams(d domain) []string {
	count := s.rng.IntN(3) + 1
	params := make([]string, 0, count)
	used := 0
	for _, candidate := range d.vars {
		if used >= count {
			break
		}
		if !s.generated[candidate] && s.isValidIdent(candidate) {
			s.generated[candidate] = true
			s.insertVar(candidate)
			params = append(params, candidate)
			used++
		}
	}
	// Fallback for any remaining slots.
	for used < count {
		params = append(params, s.generateVar())
		used++
	}
	return params
}

// GenStatement generates a single statement.
func (s *State) GenStatement(depth int, inFunction bool) ast.Statement {
	if depth >= s.maxDepth {
		if v := s.randVar(); v != "" {
			// Always use an integer expression so the assignment is type-safe.
			return ast.Assignment{Name: v, Value: s.GenIntExpr(depth + 1), IsDecl: false}
		}
		return ast.ExprStmt{Expr: s.GenIntExpr(depth + 1)}
	}
	if s.rng.Float64() < 0.04 {
		return ast.CommentStmt{Text: s.randComment()}
	}

	switch s.rng.IntN(8) {
	case 0, 1:
		// Generate the VALUE before the variable so the new name cannot
		// appear in its own initialiser expression.
		val := s.GenIntExpr(depth + 1)
		name := s.generateVar()
		return ast.Assignment{Name: name, Value: val, IsDecl: true}
	case 2:
		// Reassign an existing variable with a safe integer expression.
		if v := s.randVar(); v != "" {
			return ast.Assignment{Name: v, Value: s.GenIntExpr(depth + 1), IsDecl: false}
		}
		val := s.GenIntExpr(depth + 1)
		name := s.generateVar()
		return ast.Assignment{Name: name, Value: val, IsDecl: true}
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
		// Generate range BEFORE the loop variable so the variable cannot
		// appear in its own range expression.
		// The loop variable is inserted into the BODY scope only so it
		// cannot be referenced in subsequent statements of the parent block
		// (in Go, range variables go out of scope after the for statement).
		rangeExpr := s.GenIntExpr(depth + 1)
		loopVarName := s.generateName() // unique name, no scope insert yet
		s.pushScope()
		s.insertVar(loopVarName) // visible inside body only
		body := s.genBlock(depth+1, inFunction)
		s.popScope()
		return ast.ForStmt{Var: loopVarName, Range: rangeExpr, Body: body}
	case 6:
		// Inner closures have no params; register in s.closures so call-sites
		// can invoke them as name() without arguments.
		// Only generate closures when inside a function body — at package level
		// a FuncDef would be lifted to a top-level function by the renderer but
		// its body was generated with init-scope vars, causing "undefined" errors.
		if inFunction {
			name := s.generateName()
			s.closures.insert(name)
			return ast.FuncDef{Name: name, Body: s.genFunctionBody(depth + 1)}
		}
		fallthrough
	case 7:
		if inFunction {
			return ast.ReturnStmt{Value: s.GenIntExpr(depth + 1)}
		}
		fallthrough
	default:
		return ast.ExprStmt{Expr: s.GenNumericExpr(depth + 1)}
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
	block := make([]ast.Statement, 0, count)
	for range count {
		stmt := s.GenStatement(depth, inFunction)
		block = append(block, stmt)
		// Stop after a return so no unreachable statements follow it.
		if _, ok := stmt.(ast.ReturnStmt); ok {
			break
		}
	}
	return block
}

func (s *State) genFunctionBody(depth int) []ast.Statement {
	s.pushScope()
	body := s.genBlock(depth, true)
	if _, ok := body[len(body)-1].(ast.ReturnStmt); !ok {
		body = append(body, ast.ReturnStmt{Value: s.GenIntExpr(depth + 1)})
	}
	s.popScope()
	return body
}

// genFunctionWithParams generates a top-level FuncDef with typed parameters.
// The function name is NOT registered in s.closures to avoid call-sites trying
// to call it without passing the required arguments.
//
// s.closures is isolated per top-level function so that closures defined inside
// one function cannot be called from a sibling function (they'd be out of scope
// in the rendered Go code).
func (s *State) genFunctionWithParams(depth int, name string) ast.FuncDef {
	prevClosures := s.closures
	s.closures = newRandSet() // fresh, isolated closure set for this function
	s.pushScope()
	params := s.genParams()
	body := s.genBlock(depth, true)
	if _, ok := body[len(body)-1].(ast.ReturnStmt); !ok {
		body = append(body, ast.ReturnStmt{Value: s.GenIntExpr(depth + 1)})
	}
	s.popScope()
	s.closures = prevClosures // restore (avoids cross-function closure leakage)
	return ast.FuncDef{Name: name, ParamNames: params, Body: body}
}

// ── Go-specific domain-aware generation ───────────────────────────────────────

// pickDomain returns a random domain.
func (s *State) pickDomain() domain {
	return domainPool[s.rng.IntN(len(domainPool))]
}

// genDeferStmt generates a defer log.Println("...") statement.
func (s *State) genDeferStmt(d domain) ast.DeferStmt {
	messages := []string{
		d.subject + " finished",
		d.subject + " cleanup done",
		"completed " + d.subject,
		"exiting " + d.name + " handler",
	}
	return ast.DeferStmt{Text: messages[s.rng.IntN(len(messages))]}
}

// genSwitchStmt generates a switch on the first available in-scope variable.
// Returns nil if no variables are in scope.
func (s *State) genSwitchStmt(depth int, inFunction bool) *ast.SwitchStmt {
	tag := s.randVar()
	if tag == "" {
		return nil
	}
	caseCount := s.rng.IntN(2) + 1 // 1-2 explicit cases + default
	cases := make([]ast.SwitchCase, caseCount)
	usedVals := make(map[int64]bool)
	for i := range cases {
		var v int64
		for {
			v = int64(s.rng.IntN(10))
			if !usedVals[v] {
				break
			}
		}
		usedVals[v] = true
		cases[i] = ast.SwitchCase{
			Value: v,
			Body:  s.genScopedBlock(depth+1, inFunction),
		}
	}
	def := s.genScopedBlock(depth+1, inFunction)
	// Ensure the default block ends with a return so the compiler is satisfied.
	if inFunction {
		if _, ok := def[len(def)-1].(ast.ReturnStmt); !ok {
			def = append(def, ast.ReturnStmt{Value: s.GenIntExpr(depth + 1)})
		}
	}
	return &ast.SwitchStmt{Tag: tag, Cases: cases, Default: def}
}

// generateDomainFuncName picks an available domain helper name, or falls back
// to a unique generated name.  Marks the chosen name in s.generated so it
// can never be reused by another file in the same package.
func (s *State) generateDomainFuncName(d domain) string {
	for _, h := range d.helpers {
		if !s.generated[h] && s.isValidIdent(h) {
			s.generated[h] = true
			return h
		}
	}
	return s.generateName()
}

// genHelperCallStmt picks a domain helper and calls it with in-scope int vars.
// The helper name is guaranteed unique across files via generateDomainFuncName.
func (s *State) genHelperCallStmt(d domain) *ast.HelperCallStmt {
	vars := s.allVars()
	argCount := s.rng.IntN(2) + 1 // 1-2 args
	if len(vars) < argCount {
		argCount = len(vars)
	}
	if argCount == 0 {
		return nil
	}

	helperName := s.generateDomainFuncName(d)
	s.stubs[helperName] = argCount

	args := make([]string, argCount)
	perm := s.rng.Perm(len(vars))
	for i := range argCount {
		args[i] = vars[perm[i]]
	}

	result := s.generateVar()
	return &ast.HelperCallStmt{Name: helperName, Args: args, Result: result}
}

// goReturnTypes for domain-aware functions (realistic set).
var goReturnTypes = []string{"int64", "string", "bool", "error"}

// genDomainFunction generates a domain-aware top-level function.
func (s *State) genDomainFunction(depth int, name string) ast.FuncDef {
	d := s.pickDomain()
	retType := goReturnTypes[s.rng.IntN(len(goReturnTypes))]

	prevClosures := s.closures
	s.closures = newRandSet()
	s.pushScope()

	params := s.genDomainParams(d)

	var body []ast.Statement

	// 40% chance: open with a defer statement.
	if s.rng.Float64() < 0.4 {
		body = append(body, s.genDeferStmt(d))
	}

	// ~4% chance: a plausible comment.
	if s.rng.Float64() < 0.04 {
		body = append(body, ast.CommentStmt{Text: s.randComment()})
	}

	// 2-4 domain-named local variable declarations with compound expressions.
	localCount := s.rng.IntN(3) + 2
	for range localCount {
		// Generate value first (so the new var can't reference itself).
		val := s.GenIntExpr(depth + 1)
		varName := s.generateDomainVar(d)
		body = append(body, ast.Assignment{Name: varName, Value: val, IsDecl: true})
	}

	// Combine two vars into an "interim" result for pseudo-meaningful computation.
	if vars := s.allVars(); len(vars) >= 2 {
		left := ast.VarExpr{Name: vars[s.rng.IntN(len(vars))]}
		right := ast.VarExpr{Name: vars[s.rng.IntN(len(vars))]}
		interimVal := ast.ArithExpr{Left: left, Op: ast.Add, Right: right}
		interimName := s.generateVar()
		body = append(body, ast.Assignment{Name: interimName, Value: interimVal, IsDecl: true})
	}

	// 50% chance: generate a helper call.
	if s.rng.Float64() < 0.5 {
		if hc := s.genHelperCallStmt(d); hc != nil {
			body = append(body, *hc)
		}
	}

	// 40% chance: generate a switch statement.
	if s.rng.Float64() < 0.4 {
		if sw := s.genSwitchStmt(depth+1, true); sw != nil {
			body = append(body, *sw)
		}
	}

	// 30% chance: generate a nested if/else block.
	if s.rng.Float64() < 0.3 {
		body = append(body, ast.IfStmt{
			Cond: s.genBoolExpr(depth + 1),
			Then: s.genScopedBlock(depth+1, true),
			Else: s.genScopedBlock(depth+1, true),
		})
	}

	// Final return always present — handles any non-returning switch case arms.
	body = append(body, ast.ReturnStmt{Value: s.GenIntExpr(depth + 1)})

	s.popScope()
	s.closures = prevClosures
	return ast.FuncDef{
		Name:       name,
		ParamNames: params,
		ReturnType: retType,
		Body:       body,
	}
}

// genStub generates a simple int→int helper stub function.
// ReturnType is always "int" so the caller's result variable is int-typed
// and safe in arithmetic expressions.
func (s *State) genStub(name string, argCount int) ast.FuncDef {
	s.pushScope()
	params := make([]string, argCount)
	for i := range argCount {
		p := fmt.Sprintf("p%d", i)
		params[i] = p
		s.generated[p] = true
		s.insertVar(p)
	}
	var body []ast.Statement
	if len(params) >= 2 {
		body = append(body, ast.ReturnStmt{
			Value: ast.ArithExpr{
				Left:  ast.VarExpr{Name: params[0]},
				Op:    ast.Add,
				Right: ast.VarExpr{Name: params[1]},
			},
		})
	} else if len(params) == 1 {
		body = append(body, ast.ReturnStmt{Value: ast.VarExpr{Name: params[0]}})
	} else {
		body = append(body, ast.ReturnStmt{Value: ast.IntLit{Value: 0}})
	}
	s.popScope()
	return ast.FuncDef{Name: name, ParamNames: params, ReturnType: "int", Body: body}
}

// ── Struct / Interface generation ─────────────────────────────────────────────

// genStructDecl generates a named struct type with 2-6 fields.
func (s *State) genStructDecl() ast.StructDecl {
	name := s.generateName()
	// Capitalise first letter so the struct is exported.
	runes := []rune(name)
	if runes[0] >= 'a' && runes[0] <= 'z' {
		runes[0] = runes[0] - 'a' + 'A'
	}
	name = string(runes)
	s.generated[name] = true

	d := s.pickDomain()
	fieldCount := s.rng.IntN(5) + 2
	fields := make([]ast.StructField, 0, fieldCount)
	usedFields := map[string]bool{}
	for _, candidate := range d.structFld {
		if len(fields) >= fieldCount {
			break
		}
		if !usedFields[candidate] {
			usedFields[candidate] = true
			typ := goStructFieldTypes[s.rng.IntN(len(goStructFieldTypes))]
			fields = append(fields, ast.StructField{Name: candidate, Type: typ})
		}
	}
	return ast.StructDecl{Name: name, Fields: fields}
}

// genInterfaceDecl generates a named interface type with 2-5 methods.
func (s *State) genInterfaceDecl() ast.InterfaceDecl {
	name := s.generateName()
	runes := []rune(name)
	if runes[0] >= 'a' && runes[0] <= 'z' {
		runes[0] = runes[0] - 'a' + 'A'
	}
	name = string(runes)
	s.generated[name] = true

	methodCount := s.rng.IntN(4) + 2
	methods := make([]ast.InterfaceMethod, methodCount)
	for i := range methods {
		mName := s.generateName()
		// Capitalise.
		mr := []rune(mName)
		if mr[0] >= 'a' && mr[0] <= 'z' {
			mr[0] = mr[0] - 'a' + 'A'
		}
		paramTypes := goInterfaceParamTypes[s.rng.IntN(len(goInterfaceParamTypes))]
		retType := goInterfaceReturnTypes[s.rng.IntN(len(goInterfaceReturnTypes))]
		methods[i] = ast.InterfaceMethod{
			Name:       string(mr),
			ParamTypes: paramTypes,
			ReturnType: retType,
		}
	}
	return ast.InterfaceDecl{Name: name, Methods: methods}
}

// ── Program generation ────────────────────────────────────────────────────────

// GenerateProgram generates a full program (list of top-level statements).
func (s *State) GenerateProgram() []ast.Statement {
	s.pushScope()
	var program []ast.Statement

	fnCount := s.rng.IntN(3) + 2
	for range fnCount {
		name := s.generateName()
		// Top-level functions are NOT added to s.closures because they have
		// params; call-sites would need to supply arguments.
		program = append(program, s.genFunctionWithParams(0, name))
	}

	stmtCount := s.rng.IntN(18) + 8
	for range stmtCount {
		program = append(program, s.GenStatement(0, false))
	}

	s.popScope()
	return program
}

// GenerateGoProgram generates a Go-idiomatic program with struct/interface
// declarations, domain-aware functions, and helper stubs.
func (s *State) GenerateGoProgram() []ast.Statement {
	s.pushScope()
	var program []ast.Statement

	// 1-2 interface declarations.
	ifaceCount := s.rng.IntN(2) + 1
	for range ifaceCount {
		program = append(program, s.genInterfaceDecl())
	}

	// 1-3 struct declarations.
	structCount := s.rng.IntN(3) + 1
	for range structCount {
		program = append(program, s.genStructDecl())
	}

	// 2-4 domain-aware functions.
	fnCount := s.rng.IntN(3) + 2
	for range fnCount {
		name := s.generateName()
		program = append(program, s.genDomainFunction(0, name))
	}

	// Emit stubs for any helper functions that were referenced.
	// Names are already unique (registered in s.generated by generateDomainFuncName).
	for stubName, argCount := range s.stubs {
		program = append(program, s.genStub(stubName, argCount))
	}

	// Init-scope statements.
	stmtCount := s.rng.IntN(10) + 5
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

// AllDeclsByType returns all top-level declarations grouped by kind:
// "func", "struct", "interface".
func AllDeclsByType(program []ast.Statement) map[string][]string {
	out := map[string][]string{}
	for _, stmt := range program {
		switch v := stmt.(type) {
		case ast.FuncDef:
			out["func"] = append(out["func"], v.Name)
		case ast.StructDecl:
			out["struct"] = append(out["struct"], v.Name)
		case ast.InterfaceDecl:
			out["interface"] = append(out["interface"], v.Name)
		}
	}
	return out
}

// NewTargeted creates a State whose maxDepth is tuned so that rendering a
// program with len(declNames) top-level functions produces roughly targetLines
// lines of output.
func NewTargeted(targetLines int, declNames []string, names []string, rng *rand.Rand) *State {
	return New(estimateDepth(targetLines, len(declNames)), names, rng)
}

// NewTargetedShared is like NewTargeted but shares a generated-name set.
func NewTargetedShared(targetLines int, declNames []string, shared map[string]bool, names []string, rng *rand.Rand) *State {
	return NewShared(estimateDepth(targetLines, len(declNames)), shared, names, rng)
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
	return s.GenerateProgramWithDeclsAndHints(declNames, nil)
}

// GenerateProgramWithDeclsAndHints is like GenerateProgramWithDecls but accepts
// a per-function line-count hint map (funcName → targetLines).  Functions whose
// names appear in the map are generated at a depth tuned to approximate the
// target line count, making the output match the longest-function profile of the
// original project.
func (s *State) GenerateProgramWithDeclsAndHints(declNames []string, funcLineHints map[string]int) []ast.Statement {
	s.pushScope()
	var program []ast.Statement

	for _, name := range declNames {
		if s.isValidIdent(name) && !s.generated[name] {
			s.generated[name] = true
			if hint, ok := funcLineHints[name]; ok && hint > 0 {
				// Temporarily raise maxDepth so this function hits ~hint lines.
				saved := s.maxDepth
				s.maxDepth = estimateDepth(hint, 1)
				program = append(program, s.genFunctionWithParams(0, name))
				s.maxDepth = saved
			} else {
				program = append(program, s.genFunctionWithParams(0, name))
			}
		}
	}

	// Fallback: no valid names provided.
	if len(program) == 0 {
		for range s.rng.IntN(3) + 2 {
			name := s.generateName()
			program = append(program, s.genFunctionWithParams(0, name))
		}
	}

	for range s.rng.IntN(10) + 5 {
		program = append(program, s.GenStatement(0, false))
	}

	s.popScope()
	return program
}

// GenerateGoProgramWithDecls is like GenerateProgramWithDeclsAndHints but uses
// domain-aware generation and includes struct/interface declarations.
func (s *State) GenerateGoProgramWithDecls(declNames []string, funcLineHints map[string]int) []ast.Statement {
	s.pushScope()
	var program []ast.Statement

	// Leading type declarations.
	ifaceCount := s.rng.IntN(2) + 1
	for range ifaceCount {
		program = append(program, s.genInterfaceDecl())
	}
	structCount := s.rng.IntN(2) + 1
	for range structCount {
		program = append(program, s.genStructDecl())
	}

	for _, name := range declNames {
		if s.isValidIdent(name) && !s.generated[name] {
			s.generated[name] = true
			if hint, ok := funcLineHints[name]; ok && hint > 0 {
				saved := s.maxDepth
				s.maxDepth = estimateDepth(hint, 1)
				program = append(program, s.genDomainFunction(0, name))
				s.maxDepth = saved
			} else {
				program = append(program, s.genDomainFunction(0, name))
			}
		}
	}

	// Fallback.
	if len(program) == ifaceCount+structCount {
		for range s.rng.IntN(3) + 2 {
			name := s.generateName()
			program = append(program, s.genDomainFunction(0, name))
		}
	}

	// Stubs for any referenced helpers.
	for stubName, argCount := range s.stubs {
		if !s.generated[stubName] {
			s.generated[stubName] = true
			program = append(program, s.genStub(stubName, argCount))
		}
	}

	for range s.rng.IntN(10) + 5 {
		program = append(program, s.GenStatement(0, false))
	}

	s.popScope()
	return program
}
