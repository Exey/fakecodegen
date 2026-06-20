// Package prompt generates an archscope-style AI context document for the
// set of fake source files produced by fakecodegen.
package prompt

import (
	"fmt"
	"math/rand/v2"
	"path/filepath"
	"sort"
	"strings"

	"github.com/exey/fakecodegen/internal/parser"
)

// FileInfo describes one generated file.
type FileInfo struct {
	Path       string              // relative path inside the output folder
	Lines      int                 // total line count
	Decls      []string            // top-level declaration names (legacy, funcs only)
	TypedDecls map[string][]string // decls by kind: "func", "struct", "interface"
	FuncLines  map[string]int      // function name → rendered line count (Go only)
}

// GenerateResult is the output of Generate.
type GenerateResult struct {
	Doc            string
	CommitMessages []string
	Tags           []string
}

// Config holds the parameters for prompt generation.
type Config struct {
	Lang         string               // "go" | "py" | etc.
	FolderName   string               // base name of the output folder (used as module name)
	Files        []FileInfo           // generated files, in any order
	Contributors []parser.Contributor // if empty, fake contributors are generated
	Rng          *rand.Rand
}

var goTechStack = []string{
	"net/http", "encoding/json", "context", "sync", "io", "os", "log/slog",
	"database/sql", "crypto/tls", "time", "math/rand", "bufio", "strings",
	"Gin", "Echo", "Chi", "Fiber", "GORM", "sqlx", "Cobra", "Viper",
	"Prometheus", "OpenTelemetry", "Zap", "Zerolog", "testify", "gRPC",
	"Protocol Buffers", "Redis", "PostgreSQL", "MongoDB", "Kafka", "NATS",
}

var pyTechStack = []string{
	"asyncio", "dataclasses", "typing", "pathlib", "json", "os", "sys",
	"logging", "datetime", "collections", "functools", "itertools",
	"FastAPI", "Flask", "Django", "SQLAlchemy", "Pydantic", "Celery",
	"pytest", "aiohttp", "httpx", "Redis", "PostgreSQL", "MongoDB",
	"NumPy", "pandas", "Prometheus", "OpenTelemetry", "Kafka", "boto3",
}

var goFakeAuthors = []string{
	"Alice Chen", "Bob Kim", "Carlos Ruiz", "Diana Lee", "Ethan Park",
	"Fiona Wang", "Grace Liu", "Henry Zhang", "Iris Nakamura", "Jake Brown",
}

var pyFakeAuthors = []string{
	"Anna Kovac", "Ben Torres", "Clara Singh", "David Müller", "Elena Petrov",
	"Felix Okafor", "Gina Yamamoto", "Hassan Ali", "Isla Campbell", "Jan Nowak",
}

var commitPrefixes = []string{
	"feat", "fix", "refactor", "chore", "docs", "test", "perf", "style", "ci",
}

var staleTaskNames = []string{
	"task", "feature", "bugfix", "hotfix", "release", "chore",
}

var peakDays = []string{"Mon", "Tue", "Wed", "Thu", "Fri"}

var branchingModels = []struct {
	name       string
	confidence int
	note       string
}{
	{"GitHub Flow", 80, "Simple feature-branch workflow — no integration/release/hotfix branches"},
	{"Git Flow", 65, "Uses feature, release, and hotfix branch conventions alongside main/develop"},
	{"Trunk-Based", 72, "Short-lived branches merged directly to trunk with frequent releases"},
	{"Scaled Trunk", 58, "Trunk-based with feature flags — branches live < 2 days on average"},
}

var fakeCommitMessages = []string{
	"fix: resolve nil pointer dereference in request handler",
	"feat: add pagination support to list endpoints",
	"refactor: extract validation logic into separate package",
	"chore: update dependencies to latest versions",
	"fix: correct off-by-one error in range calculation",
	"feat: implement retry mechanism for external calls",
	"docs: update API documentation for new endpoints",
	"test: add integration tests for auth service",
	"perf: optimise database query with proper indexing",
	"fix: handle edge case when user session expires",
	"refactor: simplify error handling in middleware",
	"feat: add metrics collection for request duration",
	"chore: remove unused imports and dead code",
	"fix: correct timezone handling in date comparison",
	"feat: add graceful shutdown with context cancellation",
}

func langLabel(lang string) string {
	switch lang {
	case "go":
		return "Go"
	case "py":
		return "Python"
	}
	return lang
}

func pickN(rng *rand.Rand, items []string, n int) []string {
	if n >= len(items) {
		return items
	}
	picked := make([]string, len(items))
	copy(picked, items)
	rng.Shuffle(len(picked), func(i, j int) { picked[i], picked[j] = picked[j], picked[i] })
	return picked[:n]
}

// BuildContributors returns the contributor list to use: either the ones
// already in cfg.Contributors or a freshly generated set.  Call this before
// Generate() if you need to share the same contributor list with other
// subsystems (e.g. gitgen).
func BuildContributors(cfg Config) []parser.Contributor {
	if len(cfg.Contributors) > 0 {
		return cfg.Contributors
	}
	var authorPool []string
	switch cfg.Lang {
	case "go":
		authorPool = goFakeAuthors
	default:
		authorPool = pyFakeAuthors
	}
	count := cfg.Rng.IntN(4) + 3
	authors := pickN(cfg.Rng, authorPool, count)

	contributors := make([]parser.Contributor, len(authors))
	for i, a := range authors {
		commits := cfg.Rng.IntN(120) + 5
		files := cfg.Rng.IntN(500) + 10
		contributors[i] = parser.Contributor{Name: a, Commits: commits, Files: files}
	}
	sort.Slice(contributors, func(i, j int) bool {
		return contributors[i].Commits > contributors[j].Commits
	})
	return contributors
}

// renderTypedDecls emits the indented "  - kind: Name1, Name2" lines for
// a Key Files entry.  For files with no TypedDecls we fall back to the flat
// plain-name list (backward compatible with non-Go renderers).
func renderTypedDecls(f FileInfo, limit int) string {
	if len(f.TypedDecls) == 0 {
		// Legacy flat list.
		decls := f.Decls
		if len(decls) > limit {
			decls = decls[:limit]
		}
		if len(decls) == 0 {
			return ""
		}
		return " — " + strings.Join(decls, ", ")
	}

	// Emit one indented line per kind in canonical order.
	var sb strings.Builder
	for _, kind := range []string{"interface", "struct", "func"} {
		names := f.TypedDecls[kind]
		if len(names) == 0 {
			continue
		}
		if len(names) > limit {
			names = names[:limit]
		}
		fmt.Fprintf(&sb, "\n  - %s: %s", kind, strings.Join(names, ", "))
	}
	return sb.String()
}

// Generate returns the archscope-format prompt document plus metadata that
// callers (e.g. gitgen) can reuse for consistency.
func Generate(cfg Config) GenerateResult {
	label := langLabel(cfg.Lang)

	// Aggregate stats
	totalLines := 0
	totalDecls := 0
	for _, f := range cfg.Files {
		totalLines += f.Lines
		if len(f.TypedDecls) > 0 {
			for _, names := range f.TypedDecls {
				totalDecls += len(names)
			}
		} else {
			totalDecls += len(f.Decls)
		}
	}

	// Sort files by line count descending for Key Files section
	sorted := make([]FileInfo, len(cfg.Files))
	copy(sorted, cfg.Files)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Lines > sorted[j].Lines
	})
	keyFiles := sorted
	if len(keyFiles) > 15 {
		keyFiles = keyFiles[:15]
	}

	// Pick tech stack
	var techPool []string
	switch cfg.Lang {
	case "go":
		techPool = goTechStack
	default:
		techPool = pyTechStack
	}
	techCount := cfg.Rng.IntN(5) + 3
	tech := pickN(cfg.Rng, techPool, techCount)
	sort.Strings(tech)

	contributors := BuildContributors(cfg)

	// Total commits = sum of contributor commits
	totalCommits := 0
	for _, c := range contributors {
		totalCommits += c.Commits
	}
	typedCommits := totalCommits * (cfg.Rng.IntN(40) + 60) / 100

	// Hot files: pick top files by random change count
	type hotFile struct {
		path    string
		changes int
	}
	var hotFiles []hotFile
	for _, f := range cfg.Files {
		hotFiles = append(hotFiles, hotFile{
			path:    f.Path,
			changes: cfg.Rng.IntN(160) + 1,
		})
	}
	sort.Slice(hotFiles, func(i, j int) bool {
		return hotFiles[i].changes > hotFiles[j].changes
	})
	if len(hotFiles) > 15 {
		hotFiles = hotFiles[:15]
	}

	// Releases / tags
	tagCount := cfg.Rng.IntN(80) + 20
	latestMajor := cfg.Rng.IntN(2)
	latestMinor := cfg.Rng.IntN(8)
	latestPatch := cfg.Rng.IntN(10) + 1
	latestTag := fmt.Sprintf("%d.%d.%d", latestMajor, latestMinor, latestPatch)
	semverCount := tagCount - cfg.Rng.IntN(5)
	if semverCount < 0 {
		semverCount = tagCount
	}

	// Stale branches
	branchTotal := cfg.Rng.IntN(4) + 2
	staleBranchCount := cfg.Rng.IntN(branchTotal) + 1
	if staleBranchCount > branchTotal-1 {
		staleBranchCount = branchTotal - 1
	}

	// Commit type distribution
	commitTypes := map[string]int{}
	remaining := typedCommits
	prefixOrder := pickN(cfg.Rng, commitPrefixes, len(commitPrefixes))
	for i, p := range prefixOrder {
		if remaining <= 0 {
			break
		}
		var n int
		if i == len(prefixOrder)-1 {
			n = remaining
		} else {
			n = cfg.Rng.IntN(remaining/2+1) + 1
		}
		commitTypes[p] += n
		remaining -= n
	}

	// Recent commit messages (pick 5-8 random ones)
	msgCount := cfg.Rng.IntN(4) + 5
	recentMsgs := pickN(cfg.Rng, fakeCommitMessages, msgCount)

	// Branching model
	bm := branchingModels[cfg.Rng.IntN(len(branchingModels))]

	// Author names for inline Top authors line
	authorNames := make([]string, len(contributors))
	for i, c := range contributors {
		authorNames[i] = c.Name
	}

	var b strings.Builder

	fmt.Fprintf(&b, "# %s — Architecture Context\n\n", label)
	b.WriteString("This document was generated by fakecodegen. ")
	b.WriteString("Use it as context when asking an AI assistant questions about this codebase.\n\n")

	b.WriteString("## Overview\n\n")
	fmt.Fprintf(&b, "- **Platform:** %s\n", label)
	fmt.Fprintf(&b, "- **Files:** %d  |  **Lines:** %s  |  **Declarations:** %d\n",
		len(cfg.Files), fmtNum(totalLines), totalDecls)
	fmt.Fprintf(&b, "- **Modules:** %s\n\n", cfg.FolderName)

	b.WriteString("## Tech Stack\n\n")
	b.WriteString(strings.Join(tech, " · "))
	b.WriteString("\n\n")

	// ── Key Files ────────────────────────────────────────────────────────────
	b.WriteString("## Key Files\n\n")
	for _, f := range keyFiles {
		declSuffix := renderTypedDecls(f, 12)
		fmt.Fprintf(&b, "- `%s` (%d lines)%s\n",
			filepath.ToSlash(f.Path), f.Lines, declSuffix)
	}
	b.WriteByte('\n')

	// ── Longest Functions ────────────────────────────────────────────────────
	type funcEntry struct {
		name string
		file string
		lines int
	}
	var funcEntries []funcEntry
	for _, f := range cfg.Files {
		for fn, lc := range f.FuncLines {
			funcEntries = append(funcEntries, funcEntry{name: fn, file: filepath.ToSlash(f.Path), lines: lc})
		}
	}
	if len(funcEntries) > 0 {
		sort.Slice(funcEntries, func(i, j int) bool {
			return funcEntries[i].lines > funcEntries[j].lines
		})
		if len(funcEntries) > 15 {
			funcEntries = funcEntries[:15]
		}
		b.WriteString("## Longest Functions\n\n")
		b.WriteString("| Function | Lines | File |\n")
		b.WriteString("|----------|------:|------|\n")
		for _, fe := range funcEntries {
			fmt.Fprintf(&b, "| %s | %d | %s |\n", fe.name, fe.lines, fe.file)
		}
		b.WriteByte('\n')
	}

	// ── Git Stats ─────────────────────────────────────────────────────────────
	b.WriteString("## Git Stats\n\n")
	fmt.Fprintf(&b, "- **Total commits:** %d  |  **Typed:** %d/%d\n",
		totalCommits, typedCommits, totalCommits)
	fmt.Fprintf(&b, "- **Latest release:** %s\n", latestTag)
	fmt.Fprintf(&b, "- **Top authors:** %s\n", strings.Join(authorNames, ", "))
	b.WriteString("- **Hot files (by change count):**\n")
	for _, hf := range hotFiles {
		fmt.Fprintf(&b, "  - %s (%d changes)\n", filepath.ToSlash(hf.path), hf.changes)
	}
	b.WriteByte('\n')

	// Commit Types
	b.WriteString("### Commit Types\n\n")
	// Sort by count descending.
	type ctEntry struct{ prefix string; count int }
	ctList := make([]ctEntry, 0, len(commitTypes))
	for p, n := range commitTypes {
		ctList = append(ctList, ctEntry{p, n})
	}
	sort.Slice(ctList, func(i, j int) bool {
		if ctList[i].count != ctList[j].count {
			return ctList[i].count > ctList[j].count
		}
		return ctList[i].prefix < ctList[j].prefix
	})
	for _, ct := range ctList {
		fmt.Fprintf(&b, "- %s: %d\n", ct.prefix, ct.count)
	}
	b.WriteByte('\n')

	// Recent commit messages
	b.WriteString("**Recent commit messages:**\n\n")
	for _, msg := range recentMsgs {
		fmt.Fprintf(&b, "- %s\n", msg)
	}
	b.WriteByte('\n')

	// Branching Model
	b.WriteString("### Branching Model\n\n")
	fmt.Fprintf(&b, "**%s** · %d%% confidence · primary branch: `main`\n\n",
		bm.name, bm.confidence)
	fmt.Fprintf(&b, "- %s\n\n", bm.note)

	// Contributors table
	b.WriteString("### Contributors\n\n")
	b.WriteString("| Author | Commits | Files |\n")
	b.WriteString("|--------|--------:|------:|\n")
	for _, c := range contributors {
		fmt.Fprintf(&b, "| %s | %d | %d |\n", c.Name, c.Commits, c.Files)
	}
	b.WriteByte('\n')

	// File Churn table
	b.WriteString("### File Churn\n\n")
	b.WriteString("| File | Changes |\n")
	b.WriteString("|------|--------:|\n")
	for _, hf := range hotFiles {
		fmt.Fprintf(&b, "| %s | %d |\n", filepath.ToSlash(hf.path), hf.changes)
	}
	b.WriteByte('\n')

	// Releases
	tagSlice := buildTagSlice(tagCount, latestMajor, latestMinor, latestPatch, cfg.Rng)
	b.WriteString("### Releases\n\n")
	fmt.Fprintf(&b, "**%d tags** · %d semver · latest `%s`\n\n", tagCount, semverCount, latestTag)
	b.WriteString("Tags: ")
	b.WriteString(strings.Join(tagSlice, " · "))
	b.WriteString("\n\n")

	// Branches
	b.WriteString("### Branches\n\n")
	peak := peakDays[cfg.Rng.IntN(len(peakDays))]
	fmt.Fprintf(&b, "**%d total** · peak commit day: %s\n\n", branchTotal, peak)
	if staleBranchCount > 0 {
		b.WriteString("| Stale branch | Days idle |\n")
		b.WriteString("|--------------|----------:|\n")
		for range staleBranchCount {
			prefix := staleTaskNames[cfg.Rng.IntN(len(staleTaskNames))]
			num := cfg.Rng.IntN(900) + 100
			idle := cfg.Rng.IntN(180) + 14
			fmt.Fprintf(&b, "| %s/%d | %d |\n", prefix, num, idle)
		}
		b.WriteByte('\n')
	}

	return GenerateResult{
		Doc:            b.String(),
		CommitMessages: recentMsgs,
		Tags:           tagSlice,
	}
}

// buildTagSlice generates the ordered list of version tag strings.
func buildTagSlice(total, major, minor, patch int, rng *rand.Rand) []string {
	type ver struct{ major, minor, patch int }
	var versions []ver

	m, n, p := major, minor, patch
	for i := 0; i < total && (m > 0 || n > 0 || p > 0); i++ {
		versions = append(versions, ver{m, n, p})
		if p > 0 {
			p--
		} else if n > 0 {
			n--
			p = rng.IntN(10) + 5
		} else if m > 0 {
			m--
			n = rng.IntN(5) + 3
			p = rng.IntN(10) + 5
		}
	}
	for i, j := 0, len(versions)-1; i < j; i, j = i+1, j-1 {
		versions[i], versions[j] = versions[j], versions[i]
	}

	tags := make([]string, 0, len(versions))
	for _, v := range versions {
		base := fmt.Sprintf("%d.%d.%d", v.major, v.minor, v.patch)
		if rng.IntN(2) == 0 {
			tags = append(tags, "v"+base)
		} else {
			tags = append(tags, base)
		}
	}
	return tags
}

// generateTagList produces a space-separated · list of version tags.
func generateTagList(total, major, minor, patch int, rng *rand.Rand) string {
	return strings.Join(buildTagSlice(total, major, minor, patch, rng), " · ")
}

func fmtNum(n int) string {
	s := fmt.Sprintf("%d", n)
	if n < 1000 {
		return s
	}
	var out []byte
	for i, ch := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			out = append(out, ',')
		}
		out = append(out, byte(ch))
	}
	return string(out)
}
